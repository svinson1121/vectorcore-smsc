// Package forwarder dispatches canonical Messages to egress interfaces based
// on routing decisions, persists them for store-and-forward, and triggers
// delivery report generation on completion.
package forwarder

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/codec"
	smppcodec "github.com/svinson1121/vectorcore-smsc/internal/codec/smpp"
	"github.com/svinson1121/vectorcore-smsc/internal/metrics"
	"github.com/svinson1121/vectorcore-smsc/internal/registry"
	"github.com/svinson1121/vectorcore-smsc/internal/routing"
	"github.com/svinson1121/vectorcore-smsc/internal/smpp"
	smppClient "github.com/svinson1121/vectorcore-smsc/internal/smpp/client"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
	"github.com/svinson1121/vectorcore-smsc/internal/sgdmap"
)

// ISCSender sends MT SMS to IMS UEs via the SIP ISC interface.
type ISCSender interface {
	Send(ctx context.Context, msg *codec.Message, reg *registry.Registration) error
}

// SimpleSender sends MT SMS to inter-site SIP SIMPLE peers.
type SimpleSender interface {
	Send(ctx context.Context, msg *codec.Message, peer store.SIPPeer) error
}

// SGdSender sends MT SMS to an LTE UE via the Diameter SGd interface.
type SGdSender interface {
	SendOFR(ctx context.Context, msg *codec.Message, mmeHost, scAddr string) error
}

// DeliveryReporter records terminal outcomes once delivery is complete.
type DeliveryReporter interface {
	Report(ctx context.Context, m store.Message, status string)
}

type selectedRoute struct {
	egressIface string
	egressPeer  string
	sgdIMSI     string
	imsReg      *registry.Registration
	sfPolicyID  string
	ruleName    string
	priority    int
}

// Forwarder dispatches messages to the correct egress interface.
type Forwarder struct {
	reg       *registry.Registry
	engine    *routing.Engine
	st        store.Store
	scAddr    string // Our SMSC address for RP-OA / SGd SC-Address
	m         *metrics.M
	sgdMapper *sgdmap.Mapper // optional; nil = no MME name translation

	// Egress senders — all optional; nil means that interface is not available.
	iscSender    ISCSender
	simpleSender SimpleSender
	smppMgr      *smppClient.Manager
	sgdSender    SGdSender
	reporter     DeliveryReporter
}

// Config holds the dependencies wired into the Forwarder.
type Config struct {
	Registry     *registry.Registry
	Engine       *routing.Engine
	Store        store.Store
	SCAddr       string
	Metrics      *metrics.M
	ISCSender    ISCSender
	SimpleSender SimpleSender
	SMPPManager  *smppClient.Manager
	SGdSender    SGdSender
	Reporter     DeliveryReporter
	SGDMapper    *sgdmap.Mapper // optional S6c→SGd MME name translator
}

// New creates a Forwarder.
func New(cfg Config) *Forwarder {
	return &Forwarder{
		reg:          cfg.Registry,
		engine:       cfg.Engine,
		st:           cfg.Store,
		scAddr:       cfg.SCAddr,
		m:            cfg.Metrics,
		sgdMapper:    cfg.SGDMapper,
		iscSender:    cfg.ISCSender,
		simpleSender: cfg.SimpleSender,
		smppMgr:      cfg.SMPPManager,
		sgdSender:    cfg.SGdSender,
		reporter:     cfg.Reporter,
	}
}

// Dispatch routes msg to the correct egress interface.
// Every message is written to the database before the network send, so a crash
// between receipt and delivery leaves the message in QUEUED state for retry.
// On success the record is updated to DELIVERED; on failure the retry scheduler
// picks it up based on next_retry_at.
func (f *Forwarder) Dispatch(ctx context.Context, msg *codec.Message) {
	if msg.ID == "" {
		msg.ID = newUUID()
	}
	if f.m != nil {
		f.m.MessagesIn.WithLabelValues(string(msg.IngressInterface)).Inc()
	}

	now := time.Now()
	route, err := f.selectRoute(ctx, msg)
	if err != nil {
		slog.Warn("forwarder: no usable route",
			"dst", msg.Destination.MSISDN,
			"src_iface", msg.IngressInterface,
			"err", err,
		)
		f.persistMessage(ctx, msg, "", "", store.MessageStatusFailed, now)
		return
	}
	msg.EgressInterface = codec.InterfaceType(route.egressIface)

	// ── Step 2: Write to DB before sending ───────────────────────────────────
	// Save as QUEUED first, then immediately mark DISPATCHED. This ordering
	// means a crash between save and send leaves the row in QUEUED; startup
	// recovery will move any stuck DISPATCHED rows back to QUEUED.
	f.persistMessage(ctx, msg, route.egressIface, route.egressPeer, store.MessageStatusQueued, now)
	if err := f.st.UpdateMessageStatus(ctx, msg.ID, store.MessageStatusDispatched); err != nil {
		slog.Error("forwarder: mark dispatched failed", "id", msg.ID, "err", err)
		// Continue — we'll still attempt delivery; worst case the retry
		// scheduler also attempts it (double-deliver is safer than losing it).
	}

	// ── Step 3: Attempt delivery ─────────────────────────────────────────────
	var delivErr error
	if f.m != nil {
		// Count every outbound dispatch attempt (including attempts that fail and are retried).
		f.m.MessagesOut.WithLabelValues(route.egressIface).Inc()
	}
	switch route.egressIface {
	case string(codec.InterfaceSIP3GPP):
		delivErr = f.deliverISC(ctx, msg, route.imsReg)
	case string(codec.InterfaceSIPSimple):
		delivErr = f.deliverSimple(ctx, msg, route.egressPeer)
	case string(codec.InterfaceSMPP):
		delivErr = f.deliverSMPP(ctx, msg, route.egressPeer)
	case string(codec.InterfaceSGd):
		delivErr = f.deliverSGd(ctx, msg, route.egressPeer)
	default:
		slog.Error("forwarder: unknown egress interface", "iface", route.egressIface)
		_ = f.st.UpdateMessageStatus(ctx, msg.ID, store.MessageStatusFailed)
		return
	}

	if delivErr == nil {
		_ = f.st.UpdateMessageStatus(ctx, msg.ID, store.MessageStatusDelivered)
		if f.reporter != nil {
			if stored, err := f.st.GetMessage(ctx, msg.ID); err == nil && stored != nil {
				f.reporter.Report(ctx, *stored, "DELIVRD")
			}
		}
		return
	}

	// ── Step 4: Schedule retry ────────────────────────────────────────────────
	slog.Warn("forwarder: initial delivery failed, queuing for retry",
		"dst", msg.Destination.MSISDN, "egress", route.egressIface, "peer", route.egressPeer, "err", delivErr)
	next := now.Add(30 * time.Second)
	if err := f.st.UpdateMessageRetry(ctx, msg.ID, 1, next); err != nil {
		slog.Error("forwarder: schedule retry failed", "id", msg.ID, "err", err)
	}
}

func (f *Forwarder) selectRoute(ctx context.Context, msg *codec.Message) (*selectedRoute, error) {
	reg, err := f.reg.ShLookup(ctx, msg.Destination.MSISDN)
	if err != nil {
		slog.Warn("forwarder: Sh lookup error, continuing to routing rules",
			"dst", msg.Destination.MSISDN, "err", err)
	}
	if reg != nil && reg.Registered {
		slog.Debug("forwarder: selected IMS route from local/Sh registration",
			"dst", msg.Destination.MSISDN,
			"egress", codec.InterfaceSIP3GPP,
			"peer", reg.SCSCF,
		)
		return &selectedRoute{
			egressIface: string(codec.InterfaceSIP3GPP),
			egressPeer:  reg.SCSCF,
			imsReg:      reg,
		}, nil
	}

	decisions, err := f.engine.RouteAll(msg)
	if err != nil {
		return nil, err
	}
	slog.Debug("forwarder: routing candidates matched",
		"dst", msg.Destination.MSISDN,
		"count", len(decisions),
	)

	var skipped []string
	for _, decision := range decisions {
		slog.Debug("forwarder: evaluating routing candidate",
			"dst", msg.Destination.MSISDN,
			"rule", decision.RuleName,
			"priority", decision.Priority,
			"egress", decision.EgressIface,
			"peer", decision.EgressPeer,
		)
		route, reason := f.resolveCandidate(ctx, msg, decision, reg)
		if reason != "" {
			slog.Debug("forwarder: routing candidate skipped",
				"dst", msg.Destination.MSISDN,
				"rule", decision.RuleName,
				"priority", decision.Priority,
				"egress", decision.EgressIface,
				"reason", reason,
			)
			skipped = append(skipped, fmt.Sprintf("%s[%d]: %s", decision.RuleName, decision.Priority, reason))
			continue
		}
		slog.Debug("forwarder: routing candidate selected",
			"dst", msg.Destination.MSISDN,
			"rule", decision.RuleName,
			"priority", decision.Priority,
			"egress", route.egressIface,
			"peer", route.egressPeer,
		)
		return route, nil
	}

	slog.Debug("forwarder: no more matching rules",
		"dst", msg.Destination.MSISDN,
		"matched", len(decisions),
	)
	return nil, fmt.Errorf("no usable matching route: %s", strings.Join(skipped, "; "))
}

func (f *Forwarder) resolveCandidate(ctx context.Context, msg *codec.Message, decision routing.Decision, reg *registry.Registration) (*selectedRoute, string) {
	route := &selectedRoute{
		egressIface: decision.EgressIface,
		egressPeer:  decision.EgressPeer,
		sfPolicyID:  decision.SFPolicyID,
		ruleName:    decision.RuleName,
		priority:    decision.Priority,
		imsReg:      reg,
	}

	switch decision.EgressIface {
	case string(codec.InterfaceSIP3GPP):
		if reg == nil || !reg.Registered || reg.SCSCF == "" {
			return nil, "subscriber not IMS registered"
		}
		route.egressPeer = reg.SCSCF
		return route, ""
	case string(codec.InterfaceSIPSimple):
		if f.simpleSender == nil {
			return nil, "SIP SIMPLE sender not configured"
		}
		if decision.EgressPeer == "" {
			return nil, "SIP SIMPLE peer not specified"
		}
		peer, err := f.st.GetSIPPeer(ctx, decision.EgressPeer)
		if err != nil || peer == nil {
			return nil, fmt.Sprintf("SIP SIMPLE peer %q not found", decision.EgressPeer)
		}
		return route, ""
	case string(codec.InterfaceSMPP):
		if f.smppMgr == nil {
			return nil, "SMPP manager not configured"
		}
		if decision.EgressPeer == "" {
			return nil, "SMPP peer not specified"
		}
		if !f.smppMgr.HasPeer(decision.EgressPeer) {
			return nil, fmt.Sprintf("SMPP peer %q not active", decision.EgressPeer)
		}
		return route, ""
	case string(codec.InterfaceSGd):
		mmeHost, imsi, reason := f.resolveSGdTarget(ctx, msg.Destination.MSISDN)
		if reason != "" {
			return nil, reason
		}
		if checker, ok := f.sgdSender.(interface{ HasPeerForMME(string) bool }); ok && !checker.HasPeerForMME(mmeHost) {
			return nil, fmt.Sprintf("no active SGd peer for MME %q", mmeHost)
		}
		if inspector, ok := f.sgdSender.(interface {
			RoutePeerForMME(string) (string, bool, bool)
		}); ok {
			if peerName, viaProxy, found := inspector.RoutePeerForMME(mmeHost); found {
				slog.Debug("forwarder: SGd route resolved",
					"dst", msg.Destination.MSISDN,
					"mme", mmeHost,
					"peer", peerName,
					"via_proxy", viaProxy,
				)
			}
		}
		msg.Destination.IMSI = imsi
		route.egressPeer = mmeHost
		route.sgdIMSI = imsi
		return route, ""
	default:
		return nil, fmt.Sprintf("unsupported egress interface %q", decision.EgressIface)
	}
}

func (f *Forwarder) resolveSGdTarget(ctx context.Context, msisdn string) (string, string, string) {
	if f.sgdSender == nil {
		return "", "", "SGd sender not configured"
	}
	sub, err := f.st.GetSubscriber(ctx, msisdn)
	if err == nil && sub != nil && sub.LTEAttached && sub.MMEHost != "" && sub.IMSI != "" {
		return f.applyMMEMapping(sub.MMEHost), sub.IMSI, ""
	}
	s6cSub, s6cErr := f.reg.S6cLookup(ctx, msisdn)
	if s6cErr != nil {
		return "", "", fmt.Sprintf("S6c lookup failed: %v", s6cErr)
	}
	if s6cSub == nil || !s6cSub.LTEAttached || s6cSub.MMEHost == "" {
		return "", "", "S6c did not return a serving MME"
	}
	if s6cSub.IMSI == "" {
		return "", "", "S6c did not return an IMSI"
	}
	return f.applyMMEMapping(s6cSub.MMEHost), s6cSub.IMSI, ""
}

// applyMMEMapping translates an MME hostname via the S6c→SGd mapping table.
// If no mapper is configured or no enabled entry matches, the original host is returned.
func (f *Forwarder) applyMMEMapping(mmeHost string) string {
	if f.sgdMapper == nil {
		return mmeHost
	}
	return f.sgdMapper.Map(mmeHost)
}

// persistMessage writes a new message record to the store.
// Errors are logged but do not abort delivery — best effort on DB write failure.
func (f *Forwarder) persistMessage(ctx context.Context, msg *codec.Message,
	egressIface, egressPeer, status string, now time.Time) {

	var expiry *time.Time
	if msg.ValidityPeriod != nil {
		t := now.Add(*msg.ValidityPeriod)
		expiry = &t
	}

	sm := store.Message{
		ID:          msg.ID,
		OriginIface: string(msg.IngressInterface),
		OriginPeer:  msg.IngressPeer,
		SMPPMsgID:   msg.SMPPMsgID,
		EgressIface: egressIface,
		EgressPeer:  egressPeer,
		SrcMSISDN:   msg.Source.MSISDN,
		DstMSISDN:   msg.Destination.MSISDN,
		Encoding:    int(msg.Encoding),
		DCS:         int(msg.DCS),
		Status:      status,
		RetryCount:  0,
		DRRequired:  msg.TPSRRequired,
		SubmittedAt: now,
		ExpiryAt:    expiry,
	}
	if len(msg.Binary) > 0 {
		sm.Payload = append([]byte(nil), msg.Binary...)
	}
	if msg.UDH != nil && len(msg.UDH.Raw) > 0 {
		sm.UDH = append([]byte(nil), msg.UDH.Raw...)
	}
	if err := f.st.SaveMessage(ctx, sm); err != nil {
		slog.Error("forwarder: persist message failed", "id", msg.ID, "err", err)
	}
}

// ── Egress helpers ────────────────────────────────────────────────────────────

func (f *Forwarder) deliverISC(ctx context.Context, msg *codec.Message, reg *registry.Registration) error {
	if f.iscSender == nil {
		return fmt.Errorf("ISC sender not configured")
	}
	if err := f.iscSender.Send(ctx, msg, reg); err != nil {
		return err
	}
	return nil
}

func (f *Forwarder) deliverSimple(ctx context.Context, msg *codec.Message, peerName string) error {
	if f.simpleSender == nil {
		return fmt.Errorf("SIP SIMPLE sender not configured")
	}
	peer, err := f.st.GetSIPPeer(ctx, peerName)
	if err != nil || peer == nil {
		return fmt.Errorf("SIP SIMPLE peer %q not found", peerName)
	}
	if err := f.simpleSender.Send(ctx, msg, *peer); err != nil {
		return err
	}
	return nil
}

func (f *Forwarder) deliverSMPP(ctx context.Context, msg *codec.Message, peerName string) error {
	if f.smppMgr == nil {
		return fmt.Errorf("SMPP manager not configured")
	}
	pdu, err := smppcodec.EncodeDeliverSM(msg)
	if err != nil {
		return fmt.Errorf("encode deliver_sm: %w", err)
	}
	pdu.SequenceNumber = 0 // assigned by Link.SendAndWait
	resp, err := f.smppMgr.SendViaPeer(peerName, pdu)
	if err != nil {
		return err
	}
	if resp.CommandStatus != smpp.ESME_ROK {
		return fmt.Errorf("SMPP peer %s returned status 0x%08X", peerName, resp.CommandStatus)
	}
	return nil
}

func (f *Forwarder) deliverSGd(ctx context.Context, msg *codec.Message, mmeHost string) error {
	if f.sgdSender == nil {
		return fmt.Errorf("SGd sender not configured")
	}
	if err := f.sgdSender.SendOFR(ctx, msg, mmeHost, f.scAddr); err != nil {
		return err
	}
	return nil
}

func newUUID() string {
	var b [16]byte
	rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
