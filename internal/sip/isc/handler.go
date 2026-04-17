// Package isc handles the ISC (IP Multimedia Subsystem Service Control)
// interface between the S-CSCF and this SMS-AS.
// It receives SIP MESSAGE requests carrying application/vnd.3gpp.sms bodies
// (MO SMS from IMS UEs) and dispatches them into the routing engine.
package isc

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/svinson1121/vectorcore-smsc/internal/codec"
	"github.com/svinson1121/vectorcore-smsc/internal/codec/sip3gpp"
)

// MessageHandler is called for each inbound SIP MESSAGE on the ISC interface.
// In Phase 1 it decodes the TP-DATA and logs the canonical Message.
// In Phase 6 it will hand off to the routing engine.
type MessageHandler struct {
	// OnMessage is called after successful decode.  nil is valid in Phase 1.
	OnMessage func(msg *codec.Message)
	// OnResult is called for inbound RP-ACK / RP-ERROR carrying the body linked
	// by In-Reply-To to a previously sent MESSAGE.
	OnResult func(inReplyTo string, body []byte)
	Client   *sipgo.Client
	SIPLocal string
	Settings Settings
}

// Handle is the sipgo request handler for SIP MESSAGE on the ISC interface.
func (h *MessageHandler) Handle(req *sip.Request, tx sip.ServerTransaction) {
	ct := req.GetHeader("Content-Type")
	if ct == nil || !strings.EqualFold(strings.TrimSpace(ct.Value()), sip3gpp.ContentType) {
		respond(tx, req, sip.StatusUnsupportedMediaType, "Unsupported Media Type")
		return
	}

	body := req.Body()
	if len(body) == 0 {
		respond(tx, req, sip.StatusBadRequest, "Empty body")
		return
	}

	msg, err := sip3gpp.Decode(body)
	if err != nil {
		slog.Error("ISC decode failed", "err", err,
			"from", req.From(), "to", req.To())
		respond(tx, req, sip.StatusBadRequest, "Bad Request")
		return
	}
	if msg == nil {
		// RP-ACK, RP-ERROR, or RP-SMMA — acknowledge and done.
		inReplyTo := ""
		if hdr := req.GetHeader("In-Reply-To"); hdr != nil {
			inReplyTo = strings.TrimSpace(hdr.Value())
		}
		slog.Debug("ISC RP-ACK/RP-ERROR received", "from", req.From(), "in_reply_to", inReplyTo)
		if h.OnResult != nil && inReplyTo != "" && len(body) > 0 {
			h.OnResult(inReplyTo, append([]byte(nil), body...))
		}
		respond(tx, req, sip.StatusOK, "OK")
		return
	}

	msg.IngressInterface = codec.InterfaceSIP3GPP
	msg.IngressPeer = peerFromVia(req)

	if msg.Source.MSISDN == "" {
		if pai := req.GetHeader("P-Asserted-Identity"); pai != nil {
			msg.Source.MSISDN = msisdnFromURI(pai.Value())
			msg.Source.SIPURI = strings.TrimSpace(pai.Value())
		}
	}
	if msg.Destination.MSISDN == "" {
		msg.Destination.MSISDN = msisdnFromURI(req.Recipient.String())
		msg.Destination.SIPURI = req.Recipient.String()
	}

	slog.Info("ISC MO SMS received",
		"src", msg.Source.MSISDN,
		"dst", msg.Destination.MSISDN,
		"encoding", msg.Encoding,
		"text_len", len(msg.Text),
		"peer", msg.IngressPeer,
	)

	respond(tx, req, sip.StatusOK, "OK")

	// For store-and-forward MO handling we acknowledge the SIP transaction
	// immediately, then continue routing and RP-ACK generation asynchronously.
	if h.Client != nil {
		go func(msg *codec.Message) {
			// The originating UE expects a separate MESSAGE carrying RP-ACK/RP-ERROR,
			// linked by In-Reply-To to the original submit.
			if err := h.sendSubmitReport(context.Background(), req, msg.RPMR); err != nil {
				slog.Warn("ISC submit report send failed", "err", err, "src", msg.Source.MSISDN)
			}
		}(msg)
	}

	if h.OnMessage != nil {
		go h.OnMessage(msg)
	}
}

func respond(tx sip.ServerTransaction, req *sip.Request, code sip.StatusCode, reason string) {
	resp := sip.NewResponseFromRequest(req, code, reason, nil)
	if err := tx.Respond(resp); err != nil {
		slog.Warn("ISC respond error", "code", code, "err", err)
	}
}

// peerFromVia extracts the first Via sent-by value.
func peerFromVia(req *sip.Request) string {
	via := req.Via()
	if via == nil {
		return ""
	}
	if via.Port > 0 {
		return fmt.Sprintf("%s:%d", via.Host, via.Port)
	}
	return via.Host
}

// msisdnFromURI strips scheme and domain, returning the E.164 digit string.
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

func (h *MessageHandler) sendSubmitReport(ctx context.Context, req *sip.Request, rpMR byte) error {
	target, err := replyTargetURI(req)
	if err != nil {
		return err
	}

	routeHost := peerFromVia(req)
	reportReq, err := buildSubmitReportRequest(target, routeHost, strings.TrimSpace(h.SIPLocal), req.CallID().Value(), rpMR, h.Settings)
	if err != nil {
		return err
	}

	tx, err := h.Client.TransactionRequest(ctx, reportReq)
	if err != nil {
		return fmt.Errorf("send submit report MESSAGE: %w", err)
	}

	for {
		select {
		case resp := <-tx.Responses():
			if resp == nil {
				return fmt.Errorf("no response to submit report")
			}
			if resp.StatusCode < 200 {
				continue
			}
			if resp.StatusCode >= 300 {
				return fmt.Errorf("submit report rejected: %d %s", resp.StatusCode, resp.Reason)
			}
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func replyTargetURI(req *sip.Request) (string, error) {
	if pai := req.GetHeader("P-Asserted-Identity"); pai != nil {
		if uri := strings.Trim(strings.TrimSpace(pai.Value()), "<>"); uri != "" {
			return uri, nil
		}
	}
	from := req.From()
	if from == nil {
		return "", fmt.Errorf("missing From header on inbound MO MESSAGE")
	}
	return from.Address.String(), nil
}

func buildSubmitReportRequest(targetURI, routeHost, sipLocal, inReplyTo string, rpMR byte, cfg Settings) (*sip.Request, error) {
	var target sip.Uri
	if err := sip.ParseUri(targetURI, &target); err != nil {
		return nil, fmt.Errorf("parse submit-report target %q: %w", targetURI, err)
	}
	req := sip.NewRequest(sip.MESSAGE, target)

	fromURI, fromValue, err := buildFromURI(strings.TrimSpace(sipLocal), target.Host, routeHost)
	if err != nil {
		return nil, err
	}
	fromParams := sip.NewParams()
	fromParams.Add("tag", newTag())
	req.AppendHeader(&sip.FromHeader{Address: fromURI, Params: fromParams})
	req.AppendHeader(&sip.ToHeader{Address: target})
	req.AppendHeader(sip.NewHeader("Content-Type", sip3gpp.ContentType))
	if v := strings.TrimSpace(cfg.SubmitReportDisposition); v != "" {
		req.AppendHeader(sip.NewHeader("Request-Disposition", v))
	}
	if v := strings.TrimSpace(cfg.AcceptContact); v != "" {
		req.AppendHeader(sip.NewHeader("Accept-Contact", v))
	}
	req.AppendHeader(sip.NewHeader("In-Reply-To", inReplyTo))
	req.AppendHeader(sip.NewHeader("P-Asserted-Identity", fmt.Sprintf("<%s>", fromValue)))
	req.SetBody(sip3gpp.EncodeRPAck(rpMR, true))

	if routeHost != "" {
		routeStr := fmt.Sprintf("sip:%s;lr", routeHost)
		var routeURI sip.Uri
		if err := sip.ParseUri(routeStr, &routeURI); err == nil {
			req.AppendHeader(&sip.RouteHeader{Address: routeURI})
		}
	}
	return req, nil
}
