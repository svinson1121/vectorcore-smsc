// Package client manages outbound SMPP connections to remote SMSCs.
package client

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/codec"
	smppcodec "github.com/svinson1121/vectorcore-smsc/internal/codec/smpp"
	"github.com/svinson1121/vectorcore-smsc/internal/smpp"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

const (
	enquireLinkInterval = 30 * time.Second
	enquireLinkTimeout  = 10 * time.Second
	dialTimeout         = 10 * time.Second
	sendTimeout         = 30 * time.Second
	unbindTimeout       = 5 * time.Second
)

// OnMessageFunc is called when a deliver_sm or delivery receipt arrives
// on an outbound client connection.
type OnMessageFunc func(msg *codec.Message, link *smpp.Link, clientName string)

// Session manages a single outbound SMPP connection with exponential-backoff reconnect.
// Adapted from VectorCore SMPP Router client.go.
type Session struct {
	cfg   store.SMPPClient
	reg   *smpp.Registry
	onMsg OnMessageFunc

	stopCh   chan struct{}
	doneCh   chan struct{}
	stopOnce sync.Once
}

// newSession creates a Session for the given client configuration.
func newSession(cfg store.SMPPClient, reg *smpp.Registry, onMsg OnMessageFunc) *Session {
	return &Session{
		cfg:    cfg,
		reg:    reg,
		onMsg:  onMsg,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

// start launches the connect loop in a goroutine.
func (s *Session) start(ctx context.Context) {
	go s.run(ctx)
}

// stop signals the connect loop to exit.
func (s *Session) stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
}

func (s *Session) stopGraceful(timeout time.Duration) bool {
	s.stop()
	if timeout <= 0 {
		<-s.doneCh
		return true
	}
	select {
	case <-s.doneCh:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (s *Session) run(ctx context.Context) {
	defer close(s.doneCh)
	delay := time.Second
	maxDelay := s.cfg.ReconnectInterval
	if maxDelay <= 0 {
		maxDelay = 60 * time.Second
	}

	for {
		if err := s.connect(ctx); err != nil {
			select {
			case <-ctx.Done():
				return
			case <-s.stopCh:
				return
			default:
			}
			slog.Warn("smpp client connection lost, reconnecting",
				"name", s.cfg.Name,
				"err", err,
				"delay", delay,
			)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return
			case <-s.stopCh:
				return
			}
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		} else {
			return // context cancelled or stop requested
		}
	}
}

func (s *Session) connect(ctx context.Context) error {
	addr := net.JoinHostPort(s.cfg.Host, fmt.Sprintf("%d", s.cfg.Port))
	slog.Info("smpp client connecting", "name", s.cfg.Name, "addr", addr)

	netConn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	conn := smpp.NewConn(netConn)

	reqCmd, respCmd := bindCmds(s.cfg.BindType)
	bindPDU := &smpp.PDU{
		CommandID:        reqCmd,
		CommandStatus:    smpp.ESME_ROK,
		SequenceNumber:   conn.NextSeq(),
		SystemID:         s.cfg.SystemID,
		Password:         s.cfg.Password,
		InterfaceVersion: 0x34,
	}
	if err := conn.WritePDU(bindPDU); err != nil {
		netConn.Close()
		return fmt.Errorf("write bind: %w", err)
	}

	resp, err := conn.ReadPDU()
	if err != nil {
		netConn.Close()
		return fmt.Errorf("read bind resp: %w", err)
	}
	if resp.CommandID != respCmd {
		netConn.Close()
		return fmt.Errorf("expected bind resp 0x%08X, got 0x%08X", respCmd, resp.CommandID)
	}
	if resp.CommandStatus != smpp.ESME_ROK {
		netConn.Close()
		return fmt.Errorf("bind rejected: status 0x%08X", resp.CommandStatus)
	}

	bindType := s.cfg.BindType
	if bindType == "" {
		bindType = "transceiver"
	}
	slog.Info("smpp client bound",
		"name", s.cfg.Name,
		"system_id", s.cfg.SystemID,
		"bind_type", bindType,
	)

	link := smpp.NewLink(s.cfg.Name, s.cfg.SystemID, bindType, "client", addr, conn, smpp.StateBound)
	s.reg.Add(link)
	defer func() {
		link.SetState(smpp.StateDisconnected)
		s.reg.Remove(link)
		netConn.Close()
		slog.Info("smpp client disconnected", "name", s.cfg.Name)
	}()

	return s.readLoop(ctx, conn, link)
}

func (s *Session) readLoop(ctx context.Context, conn *smpp.Conn, link *smpp.Link) error {
	pduCh := make(chan *smpp.PDU, 16)
	errCh := make(chan error, 1)

	go func() {
		for {
			pdu, err := conn.ReadPDU()
			if err != nil {
				errCh <- err
				return
			}
			pduCh <- pdu
		}
	}()

	enquireTicker := time.NewTicker(enquireLinkInterval)
	defer enquireTicker.Stop()

	enquireTimer := time.NewTimer(0)
	if !enquireTimer.Stop() {
		<-enquireTimer.C
	}
	defer enquireTimer.Stop()
	pendingEnquire := false

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-s.stopCh:
			return s.gracefulUnbind(conn, link, pduCh, errCh)
		case err := <-errCh:
			return err

		case pdu := <-pduCh:
			switch pdu.CommandID {
			case smpp.CmdEnquireLinkResp:
				if pendingEnquire {
					pendingEnquire = false
					if !enquireTimer.Stop() {
						select {
						case <-enquireTimer.C:
						default:
						}
					}
				}

			case smpp.CmdEnquireLink:
				_ = conn.WritePDU(&smpp.PDU{
					CommandID:      smpp.CmdEnquireLinkResp,
					CommandStatus:  smpp.ESME_ROK,
					SequenceNumber: pdu.SequenceNumber,
				})

			case smpp.CmdUnbind:
				_ = conn.WritePDU(&smpp.PDU{
					CommandID:      smpp.CmdUnbindResp,
					CommandStatus:  smpp.ESME_ROK,
					SequenceNumber: pdu.SequenceNumber,
				})
				return fmt.Errorf("remote requested unbind")

			case smpp.CmdDeliverSM:
				s.handleIncoming(conn, link, pdu)

			default:
				if !link.DispatchPending(pdu) {
					slog.Debug("smpp client unhandled PDU",
						"name", s.cfg.Name,
						"cmd", pdu.CommandID,
					)
				}
			}

		case <-enquireTicker.C:
			if pendingEnquire {
				return fmt.Errorf("enquire_link timeout (no response)")
			}
			if err := conn.WritePDU(&smpp.PDU{
				CommandID:      smpp.CmdEnquireLink,
				CommandStatus:  smpp.ESME_ROK,
				SequenceNumber: conn.NextSeq(),
			}); err != nil {
				return err
			}
			pendingEnquire = true
			enquireTimer.Reset(enquireLinkTimeout)

		case <-enquireTimer.C:
			return fmt.Errorf("enquire_link timeout: no response within %s", enquireLinkTimeout)
		}
	}
}

func (s *Session) gracefulUnbind(conn *smpp.Conn, link *smpp.Link, pduCh <-chan *smpp.PDU, errCh <-chan error) error {
	if err := conn.WritePDU(&smpp.PDU{
		CommandID:      smpp.CmdUnbind,
		CommandStatus:  smpp.ESME_ROK,
		SequenceNumber: conn.NextSeq(),
	}); err != nil {
		return err
	}

	timer := time.NewTimer(unbindTimeout)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return fmt.Errorf("unbind timeout after %s", unbindTimeout)
		case err := <-errCh:
			return err
		case pdu := <-pduCh:
			switch pdu.CommandID {
			case smpp.CmdUnbindResp:
				return nil
			case smpp.CmdEnquireLink:
				_ = conn.WritePDU(&smpp.PDU{
					CommandID:      smpp.CmdEnquireLinkResp,
					CommandStatus:  smpp.ESME_ROK,
					SequenceNumber: pdu.SequenceNumber,
				})
			case smpp.CmdDeliverSM:
				s.handleIncoming(conn, link, pdu)
			default:
				if !link.DispatchPending(pdu) {
					slog.Debug("smpp client unhandled PDU during unbind",
						"name", s.cfg.Name,
						"cmd", pdu.CommandID,
					)
				}
			}
		}
	}
}

// handleIncoming decodes a deliver_sm (incoming message or DR from remote SMSC)
// and fires the OnMessage callback.
func (s *Session) handleIncoming(conn *smpp.Conn, link *smpp.Link, pdu *smpp.PDU) {
	// Respond immediately
	_ = conn.WritePDU(&smpp.PDU{
		CommandID:      smpp.CmdDeliverSMResp,
		CommandStatus:  smpp.ESME_ROK,
		SequenceNumber: pdu.SequenceNumber,
	})

	if s.onMsg == nil {
		return
	}

	var msg *codec.Message
	var err error

	if pdu.ESMClass&smpp.ESMClassDeliverReceipt != 0 {
		msg, err = smppcodec.DecodeDeliveryReceipt(pdu)
	} else {
		msg, err = smppcodec.DecodeSM(pdu)
	}
	if err != nil {
		slog.Error("smpp client deliver_sm decode failed",
			"name", s.cfg.Name, "err", err)
		return
	}

	msg.IngressInterface = codec.InterfaceSMPP
	msg.IngressPeer = s.cfg.Name
	s.onMsg(msg, link, s.cfg.Name)
}

// Send sends a PDU and waits for the response (e.g. submit_sm_resp).
func (s *Session) Send(link *smpp.Link, pdu *smpp.PDU) (*smpp.PDU, error) {
	return link.SendAndWait(pdu, sendTimeout)
}

func bindCmds(bindType string) (reqCmd, respCmd uint32) {
	switch bindType {
	case "transmitter":
		return smpp.CmdBindTransmitter, smpp.CmdBindTransmitterResp
	case "receiver":
		return smpp.CmdBindReceiver, smpp.CmdBindReceiverResp
	default:
		return smpp.CmdBindTransceiver, smpp.CmdBindTransceiverResp
	}
}
