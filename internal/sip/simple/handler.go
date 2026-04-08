package simple

import (
	"log/slog"
	"strings"

	"github.com/emiago/sipgo/sip"
	"github.com/svinson1121/vectorcore-smsc/internal/codec"
	"github.com/svinson1121/vectorcore-smsc/internal/codec/sipsimple"
)

// MessageHandler handles inbound SIP MESSAGE requests from inter-site SIP SIMPLE peers.
type MessageHandler struct {
	// OnMessage is called after successful decode.
	OnMessage func(msg *codec.Message)
}

// Handle is the sipgo request handler for inbound SIP SIMPLE messages.
func (h *MessageHandler) Handle(req *sip.Request, tx sip.ServerTransaction) {
	ct := ""
	if ctHdr := req.GetHeader("Content-Type"); ctHdr != nil {
		ct = strings.TrimSpace(ctHdr.Value())
	}

	// Accept text/plain and message/cpim; reject others.
	ctLower := strings.ToLower(strings.SplitN(ct, ";", 2)[0])
	if ctLower != sipsimple.ContentTypePlain && ctLower != sipsimple.ContentTypeCPIM {
		respond(tx, req, sip.StatusUnsupportedMediaType, "Unsupported Media Type")
		return
	}

	body := req.Body()
	if len(body) == 0 {
		respond(tx, req, sip.StatusBadRequest, "Empty body")
		return
	}

	// Extract MSISDN hints from SIP layer.
	srcMSISDN := ""
	if from := req.From(); from != nil {
		srcMSISDN = msisdnFromURI(from.Address.String())
	}
	dstMSISDN := msisdnFromURI(req.Recipient.String())

	msg, err := sipsimple.Decode(body, ct, srcMSISDN, dstMSISDN)
	if err != nil {
		slog.Error("SIMPLE decode failed", "err", err, "from", req.From())
		respond(tx, req, sip.StatusBadRequest, "Bad Request")
		return
	}

	msg.IngressInterface = codec.InterfaceSIPSimple
	msg.IngressPeer = peerFromVia(req)

	// Assign message ID from the Call-ID header for IMDN correlation.
	if callID := req.CallID(); callID != nil {
		msg.ID = callID.Value()
	}

	slog.Info("SIMPLE MO SMS received",
		"src", msg.Source.MSISDN,
		"dst", msg.Destination.MSISDN,
		"encoding", msg.Encoding,
		"text_len", len(msg.Text),
		"peer", msg.IngressPeer,
	)

	// Acknowledge immediately with 200 OK.
	respond(tx, req, sip.StatusOK, "OK")

	if h.OnMessage != nil {
		h.OnMessage(msg)
	}
}

func respond(tx sip.ServerTransaction, req *sip.Request, code sip.StatusCode, reason string) {
	resp := sip.NewResponseFromRequest(req, code, reason, nil)
	if err := tx.Respond(resp); err != nil {
		slog.Warn("SIMPLE respond error", "code", code, "err", err)
	}
}

func peerFromVia(req *sip.Request) string {
	via := req.Via()
	if via == nil {
		return ""
	}
	if via.Port > 0 {
		return via.Host + ":" + itoa(via.Port)
	}
	return via.Host
}

func msisdnFromURI(uri string) string {
	uri = strings.TrimSpace(uri)
	uri = strings.Trim(uri, "<>")
	for _, pfx := range []string{"sip:", "sips:", "tel:"} {
		if strings.HasPrefix(strings.ToLower(uri), pfx) {
			uri = uri[len(pfx):]
			break
		}
	}
	if at := strings.Index(uri, "@"); at >= 0 {
		uri = uri[:at]
	}
	uri = strings.TrimPrefix(uri, "+")
	if semi := strings.Index(uri, ";"); semi >= 0 {
		uri = uri[:semi]
	}
	return uri
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
