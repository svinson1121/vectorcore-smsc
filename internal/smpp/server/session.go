package server

import (
	"fmt"
	"log/slog"
	"net"
	"runtime/debug"
	"sync/atomic"

	"github.com/svinson1121/vectorcore-smsc/internal/codec"
	smppcodec "github.com/svinson1121/vectorcore-smsc/internal/codec/smpp"
	"github.com/svinson1121/vectorcore-smsc/internal/smpp"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

// PDUHandler is called for every non-housekeeping PDU received from a bound ESME.
type PDUHandler func(link *smpp.Link, pdu *smpp.PDU)

// OnMessageFunc is the higher-level callback invoked with a decoded Message.
type OnMessageFunc func(msg *codec.Message, link *smpp.Link, account store.SMPPServerAccount)

// session runs the SMPP session lifecycle for a single accepted TCP connection.
// It is created and owned by the Listener's accept loop.
type session struct {
	conn  net.Conn
	auth  *Authenticator
	reg   *smpp.Registry
	cfg   SessionConfig
	onMsg OnMessageFunc
}

// SessionConfig holds per-session settings derived from the server config.
type SessionConfig struct {
	MaxConnections int
}

func newSession(conn net.Conn, auth *Authenticator, reg *smpp.Registry,
	cfg SessionConfig, onMsg OnMessageFunc) *session {
	return &session{
		conn:  conn,
		auth:  auth,
		reg:   reg,
		cfg:   cfg,
		onMsg: onMsg,
	}
}

// run executes the bind phase then the read loop.
// It is called in its own goroutine.
func (s *session) run() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("smpp session panic recovered",
				"remote", s.conn.RemoteAddr().String(),
				"panic", fmt.Sprint(r),
				"stack", string(debug.Stack()),
			)
		}
		s.conn.Close()
	}()
	remote := s.conn.RemoteAddr().String()
	sc := smpp.NewConn(s.conn)

	// --- Bind phase ---
	pdu, err := sc.ReadPDU()
	if err != nil {
		slog.Warn("smpp bind read failed", "remote", remote, "err", err)
		return
	}

	respCmdID, ok := bindRespCmd(pdu.CommandID)
	if !ok {
		slog.Warn("smpp expected bind PDU", "remote", remote, "cmd", pdu.CommandID)
		_ = sc.WritePDU(&smpp.PDU{
			CommandID:      smpp.CmdGenericNack,
			CommandStatus:  smpp.ESME_RINVBNDSTS,
			SequenceNumber: pdu.SequenceNumber,
		})
		return
	}

	result := s.auth.Authenticate(pdu.SystemID, pdu.Password, remote)
	if !result.Allowed {
		slog.Warn("smpp bind rejected",
			"system_id", pdu.SystemID,
			"remote", remote,
			"reason", result.Reason,
		)
		_ = sc.WritePDU(&smpp.PDU{
			CommandID:      respCmdID,
			CommandStatus:  smpp.ESME_RINVPASWD,
			SequenceNumber: pdu.SequenceNumber,
		})
		return
	}

	if err := sc.WritePDU(&smpp.PDU{
		CommandID:      respCmdID,
		CommandStatus:  smpp.ESME_ROK,
		SequenceNumber: pdu.SequenceNumber,
		SystemID:       pdu.SystemID,
	}); err != nil {
		slog.Warn("smpp bind resp write failed", "err", err)
		return
	}

	bindType := bindTypeStr(pdu.CommandID)
	slog.Info("smpp peer bound",
		"system_id", pdu.SystemID,
		"remote", remote,
		"bind_type", bindType,
	)

	linkName := result.Account.Name
	if linkName == "" {
		linkName = pdu.SystemID
	}
	link := smpp.NewLink(linkName, pdu.SystemID, bindType, "server", remote, sc, smpp.StateBound)
	s.reg.Add(link)
	defer func() {
		link.SetState(smpp.StateDisconnected)
		s.reg.Remove(link)
		slog.Info("smpp peer disconnected",
			"system_id", pdu.SystemID,
			"remote", remote,
		)
	}()

	account := result.Account

	// --- Read loop ---
	for {
		pdu, err := sc.ReadPDU()
		if err != nil {
			return
		}

		switch pdu.CommandID {
		case smpp.CmdEnquireLink:
			_ = sc.WritePDU(&smpp.PDU{
				CommandID:      smpp.CmdEnquireLinkResp,
				CommandStatus:  smpp.ESME_ROK,
				SequenceNumber: pdu.SequenceNumber,
			})

		case smpp.CmdUnbind:
			_ = sc.WritePDU(&smpp.PDU{
				CommandID:      smpp.CmdUnbindResp,
				CommandStatus:  smpp.ESME_ROK,
				SequenceNumber: pdu.SequenceNumber,
			})
			return

		case smpp.CmdSubmitSM:
			s.handleSubmit(sc, link, pdu, account)

		default:
			// Dispatch to any pending SendAndWait caller first
			if !link.DispatchPending(pdu) {
				slog.Debug("smpp unhandled PDU",
					"system_id", pdu.SystemID,
					"cmd", pdu.CommandID,
				)
			}
		}
	}
}

// handleSubmit decodes a submit_sm, responds immediately, then fires OnMessage.
func (s *session) handleSubmit(conn *smpp.Conn, link *smpp.Link, pdu *smpp.PDU, account store.SMPPServerAccount) {
	// Always acknowledge the ESME — message is accepted
	resp := &smpp.PDU{
		CommandID:      smpp.CmdSubmitSMResp,
		CommandStatus:  smpp.ESME_ROK,
		SequenceNumber: pdu.SequenceNumber,
		MessageID:      generateMsgID(),
	}
	if err := conn.WritePDU(resp); err != nil {
		slog.Warn("smpp submit_sm_resp write failed", "err", err)
		return
	}

	// Check if this is a delivery receipt
	if pdu.ESMClass&smpp.ESMClassDeliverReceipt != 0 {
		// DR coming in via server connection — unusual but handle it
		msg, err := smppcodec.DecodeDeliveryReceipt(pdu)
		if err == nil {
			msg.IngressInterface = codec.InterfaceSMPP
			msg.IngressPeer = account.SystemID
			if s.onMsg != nil {
				s.dispatchMessage(msg, link, account)
			}
		}
		return
	}

	msg, err := smppcodec.DecodeSM(pdu)
	if err != nil {
		slog.Error("smpp submit_sm decode failed",
			"system_id", account.SystemID,
			"err", err,
		)
		return
	}

	msg.IngressInterface = codec.InterfaceSMPP
	msg.IngressPeer = account.SystemID
	msg.SMPPMsgID = resp.MessageID

	slog.Info("smpp MO received",
		"system_id", account.SystemID,
		"src", msg.Source.MSISDN,
		"dst", msg.Destination.MSISDN,
		"encoding", encodingLabel(msg.Encoding),
		"text_len", len(msg.Text),
		"binary_len", len(msg.Binary),
		"udh_len", rawUDHLen(msg),
	)

	if s.onMsg != nil {
		s.dispatchMessage(msg, link, account)
	}
}

func (s *session) dispatchMessage(msg *codec.Message, link *smpp.Link, account store.SMPPServerAccount) {
	if s.onMsg == nil {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("smpp async message handler panic recovered",
					"system_id", account.SystemID,
					"panic", fmt.Sprint(r),
					"stack", string(debug.Stack()),
				)
			}
		}()
		s.onMsg(msg, link, account)
	}()
}

// bindRespCmd maps a bind command ID to its response counterpart.
func bindRespCmd(cmdID uint32) (uint32, bool) {
	switch cmdID {
	case smpp.CmdBindTransceiver:
		return smpp.CmdBindTransceiverResp, true
	case smpp.CmdBindReceiver:
		return smpp.CmdBindReceiverResp, true
	case smpp.CmdBindTransmitter:
		return smpp.CmdBindTransmitterResp, true
	default:
		return 0, false
	}
}

func bindTypeStr(cmdID uint32) string {
	switch cmdID {
	case smpp.CmdBindTransmitter:
		return "transmitter"
	case smpp.CmdBindReceiver:
		return "receiver"
	default:
		return "transceiver"
	}
}

func encodingLabel(enc codec.Encoding) string {
	switch enc {
	case codec.EncodingGSM7:
		return "gsm7"
	case codec.EncodingUCS2:
		return "ucs2"
	case codec.EncodingUTF8:
		return "utf8"
	case codec.EncodingBinary:
		return "binary"
	default:
		return fmt.Sprintf("unknown(%d)", enc)
	}
}

func rawUDHLen(msg *codec.Message) int {
	if msg == nil || msg.UDH == nil {
		return 0
	}
	return len(msg.UDH.Raw)
}

// generateMsgID generates a simple monotonically increasing message ID.
// In Phase 6 this is replaced by the UUID from the messages table.
var smppMsgIDCounter atomic.Uint64

func generateMsgID() string {
	id := smppMsgIDCounter.Add(1)
	return msgIDHex(id)
}

func msgIDHex(v uint64) string {
	const hex = "0123456789abcdef"
	b := make([]byte, 16)
	for i := 15; i >= 0; i-- {
		b[i] = hex[v&0xF]
		v >>= 4
	}
	return string(b)
}
