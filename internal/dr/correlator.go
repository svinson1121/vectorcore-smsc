// Package dr handles cross-interface delivery report correlation.
// When a message originates on interface A and is delivered on interface B,
// the correlator generates the appropriate DR format for interface A and
// arranges for it to be sent back to the originating peer.
package dr

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"time"

	smppcodec "github.com/svinson1121/vectorcore-smsc/internal/codec/smpp"
	dcodec "github.com/svinson1121/vectorcore-smsc/internal/diameter/codec"
	s6cdiam "github.com/svinson1121/vectorcore-smsc/internal/diameter/s6c"
	"github.com/svinson1121/vectorcore-smsc/internal/smpp"
	smppClient "github.com/svinson1121/vectorcore-smsc/internal/smpp/client"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

// SMPPRegistry is used to send DR deliver_sm back to the originating ESME.
type SMPPRegistry interface {
	GetByName(name string) *smpp.Link
}

// S6cReporter sends terminal MT outcomes to the HSS over S6c.
type S6cReporter interface {
	ReportDelivery(msisdn, imsi, scAddr string, cause uint32, diagnostic *uint32) (*s6cdiam.ReportDeliveryResult, error)
}

// Correlator generates delivery reports for delivered or failed messages
// and routes them back to the originating interface.
type Correlator struct {
	st      store.Store
	smppMgr *smppClient.Manager
	smppReg SMPPRegistry
	s6c     S6cReporter
	scAddr  string
}

// New creates a Correlator.
func New(st store.Store, smppMgr *smppClient.Manager, smppReg SMPPRegistry, scAddr string) *Correlator {
	return &Correlator{st: st, smppMgr: smppMgr, smppReg: smppReg, scAddr: scAddr}
}

// SetS6cReporter swaps the active S6c reporter as peer selection changes.
func (c *Correlator) SetS6cReporter(r S6cReporter) { c.s6c = r }

// DrStatus values passed to Report.
const (
	StatusDelivered = "DELIVRD"
	StatusFailed    = "FAILED"
	StatusExpired   = "EXPIRED"
	StatusUndeliv   = "UNDELIV"
)

// Report records a delivery report for the given message and, if the origin
// interface supports it, sends the DR back to the originating peer.
func (c *Correlator) Report(ctx context.Context, m store.Message, status string) {
	dr := store.DeliveryReport{
		ID:          newUUID(),
		MessageID:   m.ID,
		Status:      status,
		EgressIface: m.EgressIface,
		RawReceipt:  fmt.Sprintf("id:%s stat:%s", m.ID, status),
		ReportedAt:  time.Now(),
	}
	if err := c.st.SaveDeliveryReport(ctx, dr); err != nil {
		slog.Error("correlator: save delivery report", "id", m.ID, "err", err)
	}
	c.reportS6c(ctx, m, status)

	switch m.OriginIface {
	case "smpp":
		c.reportSMPP(ctx, m, status)
	default:
		// SIP 3GPP STATUS-REPORT and SIP SIMPLE IMDN generation require the
		// original TP-DATA and the originating peer's SIP contact — those are
		// handled by the ISC/SIMPLE senders when they complete delivery.
		// Log only for now; full cross-interface DR for SIP origins is Phase 7.
		slog.Info("correlator: DR recorded",
			"id", m.ID,
			"origin", m.OriginIface,
			"egress", m.EgressIface,
			"status", status,
		)
	}
}

func (c *Correlator) reportS6c(ctx context.Context, m store.Message, status string) {
	if c.s6c == nil || c.scAddr == "" {
		return
	}
	if m.EgressIface != "sgd" {
		return
	}

	cause := s6cCauseForStatus(status)
	sub, err := c.st.GetSubscriber(ctx, m.DstMSISDN)
	if err != nil {
		slog.Warn("correlator: S6c subscriber lookup failed", "id", m.ID, "dst", m.DstMSISDN, "err", err)
		return
	}

	var imsi string
	if sub != nil {
		imsi = sub.IMSI
	}

	if _, err := c.s6c.ReportDelivery(m.DstMSISDN, imsi, c.scAddr, cause, nil); err != nil {
		slog.Warn("correlator: S6c RSDS failed",
			"id", m.ID,
			"dst", m.DstMSISDN,
			"imsi", imsi,
			"status", status,
			"cause", cause,
			"err", err,
		)
		return
	}

	slog.Info("correlator: S6c RSDS sent",
		"id", m.ID,
		"dst", m.DstMSISDN,
		"imsi", imsi,
		"sc_addr", c.scAddr,
		"status", status,
		"cause", cause,
	)
}

func s6cCauseForStatus(status string) uint32 {
	switch status {
	case StatusDelivered:
		return dcodec.SMDeliveryCauseSuccessfulTransfer
	default:
		return dcodec.SMDeliveryCauseAbsentUser
	}
}

// reportSMPP sends an SMPP deliver_sm delivery receipt back to the originating ESME.
func (c *Correlator) reportSMPP(ctx context.Context, m store.Message, status string) {
	if c.smppMgr == nil && c.smppReg == nil {
		return
	}
	if m.OriginPeer == "" {
		return
	}

	msgID := m.SMPPMsgID
	if msgID == "" {
		msgID = m.ID
	}

	pdu := smppcodec.EncodeDeliveryReceipt(msgID, m.SrcMSISDN, m.DstMSISDN, status, "000")
	pdu.SequenceNumber = 0

	// Try the named outbound client link first.
	if c.smppMgr != nil {
		if _, err := c.smppMgr.SendViaPeer(m.OriginPeer, pdu); err == nil {
			slog.Info("correlator: SMPP DR sent via client", "peer", m.OriginPeer, "id", m.ID)
			return
		}
	}

	// Fall back to the inbound server link (the ESME that submitted the message).
	if c.smppReg != nil {
		link := c.smppReg.GetByName(m.OriginPeer)
		if link != nil {
			if err := link.Send(pdu); err != nil {
				slog.Warn("correlator: SMPP DR via server link failed",
					"peer", m.OriginPeer, "id", m.ID, "err", err)
			} else {
				slog.Info("correlator: SMPP DR queued via server link", "peer", m.OriginPeer, "id", m.ID)
			}
			return
		}
	}

	slog.Warn("correlator: no SMPP link found for DR", "peer", m.OriginPeer, "id", m.ID)
}

func newUUID() string {
	var b [16]byte
	rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
