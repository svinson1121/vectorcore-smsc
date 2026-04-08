// Package diameter provides a Diameter peer FSM for the SMSC.
// It handles CER/CEA capability exchange, DWR/DWA watchdog, and DPR/DPA disconnect,
// then dispatches all other messages to the application callback.
package diameter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/ishidawataru/sctp"
	dcodec "github.com/svinson1121/vectorcore-smsc/internal/diameter/codec"
)

// State is the RFC 6733 peer FSM state.
type State int

const (
	StateClosed  State = iota
	StateWaitCEA       // outbound: CER sent, awaiting CEA
	StateOpen          // capability exchange complete
	StateClosing       // DPR in flight
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateWaitCEA:
		return "WAIT_CEA"
	case StateOpen:
		return "OPEN"
	case StateClosing:
		return "CLOSING"
	default:
		return "UNKNOWN"
	}
}

const (
	watchdogInterval = 30 * time.Second
	watchdogTimeout  = 10 * time.Second
	cerTimeout       = 10 * time.Second
	dialTimeout      = 10 * time.Second
	reconnectMax     = 60 * time.Second
)

// Config holds per-peer configuration.
type Config struct {
	Name        string
	Host        string // remote IP or hostname
	Port        int
	Transport   string   // "tcp" (only TCP for now)
	Application string   // "sgd" | "sh" | "s6c"
	AppID       uint32   // Diameter application ID
	AppIDs      []uint32 // optional multi-application advertisement for a shared peer

	LocalFQDN  string
	LocalRealm string
	PeerFQDN   string // expected peer FQDN (informational)
	PeerRealm  string // expected realm
}

// Peer manages a single Diameter peer connection.
type Peer struct {
	cfg Config

	mu          sync.RWMutex
	state       State
	conn        net.Conn
	connectedAt time.Time // set when state transitions to OPEN

	writeCh   chan *dcodec.Message
	stopCh    chan struct{}
	doneCh    chan struct{}
	stopped   bool
	ceaCh     chan error
	dprDoneCh chan struct{}
	wdReplyCh chan struct{}

	// PeerFQDN / PeerRealm populated after CER/CEA
	RemoteFQDN  string
	RemoteRealm string

	// OnOpen is called once after the peer transitions to OPEN state.
	OnOpen         func(p *Peer)
	onOpenHandlers []func(p *Peer)

	// OnMessage is called for every non-FSM message when in OPEN state.
	OnMessage         func(p *Peer, msg *dcodec.Message)
	onMessageHandlers []func(p *Peer, msg *dcodec.Message)

	// OnClose is called when the peer session ends and the peer transitions
	// away from an active connection.
	OnClose         func(p *Peer)
	onCloseHandlers []func(p *Peer)
}

// NewPeer creates a Peer.
func NewPeer(cfg Config) *Peer {
	return &Peer{
		cfg:       cfg,
		state:     StateClosed,
		writeCh:   make(chan *dcodec.Message, 64),
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
		ceaCh:     make(chan error, 1),
		dprDoneCh: make(chan struct{}, 1),
		wdReplyCh: make(chan struct{}, 1),
	}
}

// Start begins the outbound connect loop.
func (p *Peer) Name() string   { return p.cfg.Name }
func (p *Peer) Config() Config { return p.cfg }

func (p *Peer) Start(ctx context.Context) {
	go p.runOutbound(ctx)
}

// AddOnOpenHandler registers an additional callback invoked when the peer opens.
func (p *Peer) AddOnOpenHandler(fn func(p *Peer)) {
	if fn == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onOpenHandlers = append(p.onOpenHandlers, fn)
}

// AddOnMessageHandler registers an additional callback for application messages.
func (p *Peer) AddOnMessageHandler(fn func(p *Peer, msg *dcodec.Message)) {
	if fn == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onMessageHandlers = append(p.onMessageHandlers, fn)
}

// AddOnCloseHandler registers an additional callback invoked when the peer closes.
func (p *Peer) AddOnCloseHandler(fn func(p *Peer)) {
	if fn == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onCloseHandlers = append(p.onCloseHandlers, fn)
}

// RunInbound handles an already-accepted inbound connection.
// It waits for the inbound CER, responds with CEA, then runs the session.
// Blocks until the session ends; intended to be called in a goroutine.
func (p *Peer) RunInbound(ctx context.Context, conn net.Conn) {
	p.mu.Lock()
	p.conn = conn
	p.mu.Unlock()

	if err := p.runInboundSession(ctx, conn); err != nil && ctx.Err() == nil {
		slog.Warn("diameter inbound session ended",
			"peer", p.cfg.Name, "err", err)
	}
	p.mu.Lock()
	p.conn = nil
	p.mu.Unlock()
	p.setState(StateClosed)
	p.fireOnClose()
}

func (p *Peer) runInboundSession(ctx context.Context, conn net.Conn) error {
	sessionDone := make(chan struct{})
	defer close(sessionDone)

	select {
	case <-p.ceaCh:
	default:
	}
	select {
	case <-p.wdReplyCh:
	default:
	}

	readErrCh := make(chan error, 1)
	writeErrCh := make(chan error, 1)
	go p.writeLoop(ctx, conn, writeErrCh, sessionDone)
	go p.readLoop(ctx, conn, readErrCh)

	// Wait for inbound CER (handled in dispatch → handleCER → ceaCh)
	p.setState(StateWaitCEA)
	select {
	case err := <-p.ceaCh:
		if err != nil {
			conn.Close()
			return fmt.Errorf("inbound CER/CEA: %w", err)
		}
	case err := <-readErrCh:
		conn.Close()
		return fmt.Errorf("CER/CEA: connection closed: %w", err)
	case <-time.After(cerTimeout):
		conn.Close()
		return fmt.Errorf("timeout waiting for CER")
	case <-ctx.Done():
		conn.Close()
		return ctx.Err()
	}

	p.setState(StateOpen)
	slog.Info("diameter inbound peer OPEN",
		"peer", p.cfg.Name,
		"remote_fqdn", p.RemoteFQDN,
	)
	p.fireOnOpen()

	wdDone := make(chan struct{})
	go p.watchdogLoop(ctx, conn, wdDone)

	var sessionErr error
	select {
	case <-ctx.Done():
		sessionErr = ctx.Err()
	case <-p.stopCh:
		p.setState(StateClosing)
		p.sendDPR(conn)
		select {
		case <-p.dprDoneCh:
		case <-time.After(5 * time.Second):
		}
	case err := <-readErrCh:
		sessionErr = err
	case err := <-writeErrCh:
		sessionErr = err
	}

	close(wdDone)
	conn.Close()
	return sessionErr
}

// Stop gracefully shuts down the peer.
func (p *Peer) Stop() {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return
	}
	p.stopped = true
	p.mu.Unlock()
	close(p.stopCh)
}

// StopGraceful requests a graceful disconnect and waits for the peer loop to exit.
func (p *Peer) StopGraceful(timeout time.Duration) bool {
	p.Stop()
	if timeout <= 0 {
		<-p.doneCh
		return true
	}
	select {
	case <-p.doneCh:
		return true
	case <-time.After(timeout):
		return false
	}
}

// Send queues a message for delivery.
func (p *Peer) Send(msg *dcodec.Message) error {
	p.mu.RLock()
	st := p.state
	p.mu.RUnlock()
	if st != StateOpen {
		return fmt.Errorf("diameter peer %s: not OPEN (state=%s)", p.cfg.Name, st)
	}
	select {
	case p.writeCh <- msg:
		return nil
	default:
		return fmt.Errorf("diameter peer %s: write channel full", p.cfg.Name)
	}
}

// Cfg returns the peer's configuration.
func (p *Peer) Cfg() Config { return p.cfg }

// State returns the current FSM state.
func (p *Peer) State() State {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// ConnectedAt returns the time the peer last entered OPEN state, or nil if not yet connected.
func (p *Peer) ConnectedAt() *time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.connectedAt.IsZero() {
		return nil
	}
	t := p.connectedAt
	return &t
}

func (p *Peer) setState(s State) {
	p.mu.Lock()
	if s == StateOpen {
		p.connectedAt = time.Now()
	}
	p.state = s
	p.mu.Unlock()
	slog.Debug("diameter peer state", "peer", p.cfg.Name, "state", s.String())
}

// runOutbound manages reconnect with exponential backoff.
func (p *Peer) runOutbound(ctx context.Context) {
	defer close(p.doneCh)
	delay := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		default:
		}

		addr := net.JoinHostPort(p.cfg.Host, fmt.Sprintf("%d", p.cfg.Port))
		slog.Info("diameter peer connecting",
			"peer", p.cfg.Name,
			"transport", normalizeTransport(p.cfg.Transport),
			"addr", addr,
			"app_ids", p.advertisedAppIDs(),
		)

		conn, err := DialConn(p.cfg.Transport, addr, dialTimeout)
		if err != nil {
			slog.Warn("diameter peer connect failed",
				"peer", p.cfg.Name, "transport", p.cfg.Transport, "err", err, "retry_in", delay)
			select {
			case <-ctx.Done():
				return
			case <-p.stopCh:
				return
			case <-time.After(delay):
			}
			delay = min2(delay*2, reconnectMax)
			continue
		}

		p.mu.Lock()
		p.conn = conn
		p.mu.Unlock()

		if err := p.runSession(ctx, conn); err != nil && ctx.Err() == nil {
			slog.Warn("diameter peer session ended",
				"peer", p.cfg.Name, "err", err, "retry_in", delay)
		}

		p.mu.Lock()
		p.conn = nil
		p.mu.Unlock()
		p.setState(StateClosed)
		p.fireOnClose()

		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-time.After(delay):
		}
		delay = min2(delay*2, reconnectMax)
	}
}

func (p *Peer) runSession(ctx context.Context, conn net.Conn) error {
	sessionDone := make(chan struct{})
	defer close(sessionDone)

	// Drain stale signals
	select {
	case <-p.ceaCh:
	default:
	}
	select {
	case <-p.wdReplyCh:
	default:
	}

	readErrCh := make(chan error, 1)
	writeErrCh := make(chan error, 1)
	go p.writeLoop(ctx, conn, writeErrCh, sessionDone)
	go p.readLoop(ctx, conn, readErrCh)

	// Send CER
	p.setState(StateWaitCEA)
	if err := p.sendCER(conn); err != nil {
		conn.Close()
		return fmt.Errorf("send CER: %w", err)
	}

	// Await CEA — also watch readErrCh so a remote RST/FIN is detected immediately
	// rather than waiting for the full cerTimeout.
	select {
	case err := <-p.ceaCh:
		if err != nil {
			conn.Close()
			return fmt.Errorf("CEA: %w", err)
		}
	case err := <-readErrCh:
		conn.Close()
		return fmt.Errorf("CER/CEA: connection closed: %w", err)
	case <-time.After(cerTimeout):
		conn.Close()
		return fmt.Errorf("CEA timeout")
	case <-ctx.Done():
		conn.Close()
		return ctx.Err()
	case <-p.stopCh:
		conn.Close()
		return nil
	}

	p.setState(StateOpen)
	slog.Info("diameter peer OPEN",
		"peer", p.cfg.Name,
		"remote_fqdn", p.RemoteFQDN,
		"remote_realm", p.RemoteRealm,
	)
	p.fireOnOpen()

	// Watchdog loop
	wdDone := make(chan struct{})
	go p.watchdogLoop(ctx, conn, wdDone)

	var sessionErr error
	select {
	case <-ctx.Done():
		sessionErr = ctx.Err()
	case <-p.stopCh:
		p.setState(StateClosing)
		p.sendDPR(conn)
		select {
		case <-p.dprDoneCh:
		case <-time.After(5 * time.Second):
		}
	case err := <-readErrCh:
		sessionErr = err
	case err := <-writeErrCh:
		sessionErr = err
	}

	close(wdDone)
	conn.Close()
	return sessionErr
}

func (p *Peer) readLoop(ctx context.Context, conn net.Conn, errCh chan<- error) {
	for {
		select {
		case <-ctx.Done():
			errCh <- ctx.Err()
			return
		case <-p.stopCh:
			return
		default:
		}
		msg, err := dcodec.DecodeMessage(conn)
		if err != nil {
			if errors.Is(err, io.EOF) || isClosedErr(err) {
				errCh <- fmt.Errorf("connection closed: %w", err)
			} else {
				errCh <- err
			}
			return
		}
		p.logDiameterMessage("in", msg)
		p.dispatch(msg, conn)
	}
}

func (p *Peer) writeLoop(ctx context.Context, conn net.Conn, errCh chan<- error, done <-chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			errCh <- ctx.Err()
			return
		case <-p.stopCh:
			return
		case <-done:
			return
		case msg := <-p.writeCh:
			enc, err := msg.Encode()
			if err != nil {
				slog.Error("diameter encode error", "peer", p.cfg.Name, "err", err)
				continue
			}
			if err := writeFull(conn, enc); err != nil {
				errCh <- err
				return
			}
			p.logDiameterMessage("out", msg)
		}
	}
}

func (p *Peer) watchdogLoop(ctx context.Context, conn net.Conn, done <-chan struct{}) {
	ticker := time.NewTicker(watchdogInterval)
	defer ticker.Stop()
	pending := false
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()

	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			return
		case <-timer.C:
			slog.Warn("diameter watchdog timeout, closing", "peer", p.cfg.Name)
			conn.Close()
			return
		case <-p.wdReplyCh:
			pending = false
			timer.Stop()
		case <-ticker.C:
			if pending {
				slog.Warn("diameter DWR unanswered, closing", "peer", p.cfg.Name)
				conn.Close()
				return
			}
			dwr := dcodec.NewRequest(dcodec.CmdDeviceWatchdog, dcodec.AppDiameterCommon)
			dwr.NonProxiable()
			dwr.Add(
				dcodec.NewString(dcodec.CodeOriginHost, 0, dcodec.FlagMandatory, p.cfg.LocalFQDN),
				dcodec.NewString(dcodec.CodeOriginRealm, 0, dcodec.FlagMandatory, p.cfg.LocalRealm),
				dcodec.NewUint32(dcodec.CodeOriginStateID, 0, dcodec.FlagMandatory, dcodec.OriginStateID()),
			)
			if enc, err := dwr.Build().Encode(); err == nil {
				_ = writeFull(conn, enc)
			}
			pending = true
			timer.Reset(watchdogTimeout)
		}
	}
}

func (p *Peer) dispatch(msg *dcodec.Message, conn net.Conn) {
	cmd := msg.Header.CommandCode
	isReq := msg.IsRequest()

	switch cmd {
	case dcodec.CmdCapabilitiesExchange:
		if isReq {
			p.handleCER(msg, conn)
		} else {
			p.handleCEA(msg)
		}
	case dcodec.CmdDeviceWatchdog:
		if isReq {
			p.handleDWR(msg, conn)
		} else {
			select {
			case p.wdReplyCh <- struct{}{}:
			default:
			}
		}
	case dcodec.CmdDisconnectPeer:
		if isReq {
			p.handleDPR(msg, conn)
		} else {
			select {
			case p.dprDoneCh <- struct{}{}:
			default:
			}
		}
	default:
		if p.State() == StateOpen {
			p.fireOnMessage(msg)
		}
	}
}

func (p *Peer) sendCER(conn net.Conn) error {
	appIDs := p.advertisedAppIDs()
	slog.Debug("diameter sending CER",
		"peer", p.cfg.Name,
		"transport", normalizeTransport(p.cfg.Transport),
		"local_addr", conn.LocalAddr().String(),
		"remote_addr", conn.RemoteAddr().String(),
		"app_ids", appIDs,
	)

	b := dcodec.NewRequest(dcodec.CmdCapabilitiesExchange, dcodec.AppDiameterCommon)
	b.NonProxiable()
	b.Add(
		dcodec.NewString(dcodec.CodeOriginHost, 0, dcodec.FlagMandatory, p.cfg.LocalFQDN),
		dcodec.NewString(dcodec.CodeOriginRealm, 0, dcodec.FlagMandatory, p.cfg.LocalRealm),
		dcodec.NewUint32(dcodec.CodeOriginStateID, 0, dcodec.FlagMandatory, dcodec.OriginStateID()),
		dcodec.NewUint32(dcodec.CodeVendorID, 0, dcodec.FlagMandatory, 0),
		dcodec.NewString(dcodec.CodeProductName, 0, 0, "VectorCore SMSC"),
		dcodec.NewUint32(dcodec.CodeFirmwareRevision, 0, 0, 1),
	)
	for _, ip := range diameterHostIPs(conn.LocalAddr()) {
		b.Add(dcodec.NewAddress(dcodec.CodeHostIPAddress, 0, dcodec.FlagMandatory, ip))
	}
	// 3GPP vendor-specific apps (SGd, Sh, S6c) must be advertised via
	// one Vendor-Specific-Application-Id (AVP 260) per application, not bare
	// Auth-Application-Id values.
	if len(appIDs) > 0 {
		b.Add(dcodec.NewUint32(dcodec.CodeSupportedVendorID, 0, dcodec.FlagMandatory, dcodec.Vendor3GPP))
		for _, appID := range appIDs {
			vsaid, err := dcodec.NewGrouped(dcodec.CodeVendorSpecificApplicationID, 0, dcodec.FlagMandatory, []*dcodec.AVP{
				dcodec.NewUint32(dcodec.CodeVendorID, 0, dcodec.FlagMandatory, dcodec.Vendor3GPP),
				dcodec.NewUint32(dcodec.CodeAuthApplicationID, 0, dcodec.FlagMandatory, appID),
			})
			if err != nil {
				return fmt.Errorf("build Vendor-Specific-Application-Id: %w", err)
			}
			b.Add(vsaid)
		}
	} else {
		b.Add(dcodec.NewUint32(dcodec.CodeAuthApplicationID, 0, dcodec.FlagMandatory, 0))
	}
	enc, err := b.Build().Encode()
	if err != nil {
		return err
	}
	return writeFull(conn, enc)
}

func (p *Peer) handleCER(req *dcodec.Message, conn net.Conn) {
	p.extractPeerCaps(req)
	b := dcodec.NewAnswer(req)
	b.NonProxiable()
	b.Add(
		dcodec.NewUint32(dcodec.CodeResultCode, 0, dcodec.FlagMandatory, dcodec.DiameterSuccess),
		dcodec.NewString(dcodec.CodeOriginHost, 0, dcodec.FlagMandatory, p.cfg.LocalFQDN),
		dcodec.NewString(dcodec.CodeOriginRealm, 0, dcodec.FlagMandatory, p.cfg.LocalRealm),
		dcodec.NewUint32(dcodec.CodeOriginStateID, 0, dcodec.FlagMandatory, dcodec.OriginStateID()),
	)
	if appIDs := p.advertisedAppIDs(); len(appIDs) > 0 {
		b.Add(dcodec.NewUint32(dcodec.CodeSupportedVendorID, 0, dcodec.FlagMandatory, dcodec.Vendor3GPP))
		for _, appID := range appIDs {
			vsaid, err := dcodec.NewGrouped(dcodec.CodeVendorSpecificApplicationID, 0, dcodec.FlagMandatory, []*dcodec.AVP{
				dcodec.NewUint32(dcodec.CodeVendorID, 0, dcodec.FlagMandatory, dcodec.Vendor3GPP),
				dcodec.NewUint32(dcodec.CodeAuthApplicationID, 0, dcodec.FlagMandatory, appID),
			})
			if err != nil {
				p.ceaCh <- fmt.Errorf("build Vendor-Specific-Application-Id: %w", err)
				return
			}
			b.Add(vsaid)
		}
	} else {
		b.Add(dcodec.NewUint32(dcodec.CodeAuthApplicationID, 0, dcodec.FlagMandatory, 0))
	}
	enc, err := b.Build().Encode()
	if err != nil {
		p.ceaCh <- fmt.Errorf("encode CEA: %w", err)
		return
	}
	if err := writeFull(conn, enc); err != nil {
		p.ceaCh <- fmt.Errorf("write CEA: %w", err)
		return
	}
	p.ceaCh <- nil
}

func (p *Peer) handleCEA(cea *dcodec.Message) {
	rcAVP := cea.FindAVP(dcodec.CodeResultCode, 0)
	if rcAVP == nil {
		p.ceaCh <- fmt.Errorf("CEA missing Result-Code")
		return
	}
	rc, err := rcAVP.Uint32()
	if err != nil || rc != dcodec.DiameterSuccess {
		p.ceaCh <- fmt.Errorf("CEA result %d", rc)
		return
	}
	p.extractPeerCaps(cea)
	p.ceaCh <- nil
}

func (p *Peer) handleDWR(req *dcodec.Message, conn net.Conn) {
	b := dcodec.NewAnswer(req)
	b.NonProxiable()
	b.Add(
		dcodec.NewUint32(dcodec.CodeResultCode, 0, dcodec.FlagMandatory, dcodec.DiameterSuccess),
		dcodec.NewString(dcodec.CodeOriginHost, 0, dcodec.FlagMandatory, p.cfg.LocalFQDN),
		dcodec.NewString(dcodec.CodeOriginRealm, 0, dcodec.FlagMandatory, p.cfg.LocalRealm),
	)
	if enc, err := b.Build().Encode(); err == nil {
		_ = writeFull(conn, enc)
	}
}

func diameterHostIPs(addr net.Addr) []net.IP {
	switch a := addr.(type) {
	case *net.TCPAddr:
		if a == nil || a.IP == nil {
			return nil
		}
		return []net.IP{a.IP}
	case *sctp.SCTPAddr:
		if a == nil {
			return nil
		}
		ips := make([]net.IP, 0, len(a.IPAddrs))
		for _, ipAddr := range a.IPAddrs {
			if ipAddr.IP == nil {
				continue
			}
			ips = append(ips, ipAddr.IP)
		}
		return ips
	default:
		return nil
	}
}

func (p *Peer) sendDPR(conn net.Conn) {
	b := dcodec.NewRequest(dcodec.CmdDisconnectPeer, dcodec.AppDiameterCommon)
	b.NonProxiable()
	b.Add(
		dcodec.NewString(dcodec.CodeOriginHost, 0, dcodec.FlagMandatory, p.cfg.LocalFQDN),
		dcodec.NewString(dcodec.CodeOriginRealm, 0, dcodec.FlagMandatory, p.cfg.LocalRealm),
		dcodec.NewUint32(dcodec.CodeDisconnectCause, 0, dcodec.FlagMandatory, 0), // REBOOTING=0
	)
	if enc, err := b.Build().Encode(); err == nil {
		_ = writeFull(conn, enc)
	}
}

func (p *Peer) advertisedAppIDs() []uint32 {
	seen := make(map[uint32]struct{}, len(p.cfg.AppIDs)+1)
	appIDs := make([]uint32, 0, len(p.cfg.AppIDs)+1)
	if p.cfg.AppID != 0 {
		seen[p.cfg.AppID] = struct{}{}
		appIDs = append(appIDs, p.cfg.AppID)
	}
	for _, appID := range p.cfg.AppIDs {
		if appID == 0 {
			continue
		}
		if _, ok := seen[appID]; ok {
			continue
		}
		seen[appID] = struct{}{}
		appIDs = append(appIDs, appID)
	}
	return appIDs
}

func (p *Peer) logDiameterMessage(direction string, msg *dcodec.Message) {
	fields := []any{
		"direction", direction,
		"peer", p.cfg.Name,
		"app", diameterAppName(msg.Header.AppID),
		"app_id", msg.Header.AppID,
		"command", diameterCommandName(msg.Header.CommandCode),
		"command_code", msg.Header.CommandCode,
		"request", msg.IsRequest(),
		"hop_by_hop", msg.Header.HopByHop,
		"end_to_end", msg.Header.EndToEnd,
	}
	if p.RemoteFQDN != "" {
		fields = append(fields, "remote_fqdn", p.RemoteFQDN)
	}
	if sessionID := diameterSessionID(msg); sessionID != "" {
		fields = append(fields, "session_id", sessionID)
	}
	if !msg.IsRequest() {
		if resultCode, ok := diameterResultCode(msg); ok {
			fields = append(fields, "result_code", resultCode)
		}
	}
	slog.Debug("diameter message", fields...)
}

func diameterAppName(appID uint32) string {
	switch appID {
	case dcodec.AppDiameterCommon:
		return "base"
	case dcodec.App3GPP_Sh:
		return "sh"
	case dcodec.App3GPP_S6c:
		return "s6c"
	case dcodec.App3GPP_SGd:
		return "sgd"
	default:
		return "unknown"
	}
}

func diameterCommandName(commandCode uint32) string {
	switch commandCode {
	case dcodec.CmdCapabilitiesExchange:
		return "CER/CEA"
	case dcodec.CmdDeviceWatchdog:
		return "DWR/DWA"
	case dcodec.CmdDisconnectPeer:
		return "DPR/DPA"
	case dcodec.CmdUserData:
		return "UDR/UDA"
	case dcodec.CmdMOForwardShortMessage:
		return "OFR/OFA"
	case dcodec.CmdMTForwardShortMessage:
		return "TFR/TFA"
	case dcodec.CmdSendRoutingInfoSM:
		return "SRI-SM"
	case dcodec.CmdAlertServiceCentre:
		return "ALSC"
	case dcodec.CmdReportSMDeliveryStatus:
		return "RSDS"
	default:
		return "unknown"
	}
}

func diameterSessionID(msg *dcodec.Message) string {
	if avp := msg.FindAVP(dcodec.CodeSessionID, 0); avp != nil {
		if sessionID, err := avp.String(); err == nil {
			return sessionID
		}
	}
	return ""
}

func diameterResultCode(msg *dcodec.Message) (uint32, bool) {
	if avp := msg.FindAVP(dcodec.CodeResultCode, 0); avp != nil {
		resultCode, err := avp.Uint32()
		if err == nil {
			return resultCode, true
		}
	}
	if avp := msg.FindAVP(dcodec.CodeExperimentalResult, 0); avp != nil {
		children, err := dcodec.DecodeGrouped(avp)
		if err == nil {
			for _, child := range children {
				if child.Code != dcodec.CodeExperimentalResultCode || child.VendorID != 0 {
					continue
				}
				resultCode, err := child.Uint32()
				if err == nil {
					return resultCode, true
				}
			}
		}
	}
	return 0, false
}

func (p *Peer) fireOnOpen() {
	p.mu.RLock()
	onOpen := p.OnOpen
	handlers := append([]func(*Peer){}, p.onOpenHandlers...)
	p.mu.RUnlock()
	if onOpen != nil {
		onOpen(p)
	}
	for _, fn := range handlers {
		fn(p)
	}
}

func (p *Peer) fireOnMessage(msg *dcodec.Message) {
	p.mu.RLock()
	onMessage := p.OnMessage
	handlers := append([]func(*Peer, *dcodec.Message){}, p.onMessageHandlers...)
	p.mu.RUnlock()
	if onMessage != nil {
		onMessage(p, msg)
	}
	for _, fn := range handlers {
		fn(p, msg)
	}
}

func (p *Peer) fireOnClose() {
	p.mu.RLock()
	onClose := p.OnClose
	handlers := append([]func(*Peer){}, p.onCloseHandlers...)
	p.mu.RUnlock()
	if onClose != nil {
		onClose(p)
	}
	for _, fn := range handlers {
		fn(p)
	}
}

func (p *Peer) handleDPR(req *dcodec.Message, conn net.Conn) {
	b := dcodec.NewAnswer(req)
	b.NonProxiable()
	b.Add(
		dcodec.NewUint32(dcodec.CodeResultCode, 0, dcodec.FlagMandatory, dcodec.DiameterSuccess),
		dcodec.NewString(dcodec.CodeOriginHost, 0, dcodec.FlagMandatory, p.cfg.LocalFQDN),
		dcodec.NewString(dcodec.CodeOriginRealm, 0, dcodec.FlagMandatory, p.cfg.LocalRealm),
	)
	if enc, err := b.Build().Encode(); err == nil {
		_ = writeFull(conn, enc)
	}
	conn.Close()
}

func (p *Peer) extractPeerCaps(msg *dcodec.Message) {
	if a := msg.FindAVP(dcodec.CodeOriginHost, 0); a != nil {
		if s, err := a.String(); err == nil {
			p.RemoteFQDN = s
		}
	}
	if a := msg.FindAVP(dcodec.CodeOriginRealm, 0); a != nil {
		if s, err := a.String(); err == nil {
			p.RemoteRealm = s
		}
	}
}

func isClosedErr(err error) bool {
	var netErr *net.OpError
	return errors.As(err, &netErr) || errors.Is(err, net.ErrClosed)
}

func min2(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func writeFull(conn net.Conn, buf []byte) error {
	for len(buf) > 0 {
		n, err := conn.Write(buf)
		if err != nil {
			return err
		}
		if n <= 0 {
			return io.ErrShortWrite
		}
		buf = buf[n:]
	}
	return nil
}
