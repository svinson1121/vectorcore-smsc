package isc

import (
	"testing"

	"github.com/emiago/sipgo/sip"
)

func TestBuildSubmitReportRequestIncludesInReplyToAndRPAck(t *testing.T) {
	req, err := buildSubmitReportRequest(
		"sip:3342012834@ims.mnc435.mcc311.3gppnetwork.org",
		"10.90.250.52:5060",
		"sip:smsc@smsc.ims.mnc435.mcc311.3gppnetwork.org",
		"3026596545_2340267036@10.150.3.42",
		0x01,
		Settings{
			AcceptContact:           "*;+g.3gpp.smsip",
			SubmitReportDisposition: "no-fork",
		},
	)
	if err != nil {
		t.Fatalf("build submit report request: %v", err)
	}

	if got := headerValue(req, "In-Reply-To"); got != "3026596545_2340267036@10.150.3.42" {
		t.Fatalf("unexpected In-Reply-To: %q", got)
	}
	if got := headerValue(req, "Request-Disposition"); got != "no-fork" {
		t.Fatalf("unexpected Request-Disposition: %q", got)
	}
	if body := req.Body(); len(body) != 2 || body[0] != 0x03 || body[1] != 0x01 {
		t.Fatalf("unexpected RP-ACK body: %v", body)
	}
	to := req.To()
	if to == nil || to.Address.String() != "sip:3342012834@ims.mnc435.mcc311.3gppnetwork.org" {
		t.Fatalf("unexpected To header: %#v", to)
	}
}

func TestReplyTargetURIPrefersPAI(t *testing.T) {
	var target sip.Uri
	if err := sip.ParseUri("sip:0015555@ims.mnc435.mcc311.3gppnetwork.org", &target); err != nil {
		t.Fatalf("parse target: %v", err)
	}
	req := sip.NewRequest(sip.MESSAGE, target)
	var fromURI sip.Uri
	if err := sip.ParseUri("sip:fromuser@ims.mnc435.mcc311.3gppnetwork.org", &fromURI); err != nil {
		t.Fatalf("parse from: %v", err)
	}
	req.AppendHeader(&sip.FromHeader{Address: fromURI})
	req.AppendHeader(sip.NewHeader("P-Asserted-Identity", "<sip:3342012834@ims.mnc435.mcc311.3gppnetwork.org>"))

	got, err := replyTargetURI(req)
	if err != nil {
		t.Fatalf("replyTargetURI: %v", err)
	}
	if got != "sip:3342012834@ims.mnc435.mcc311.3gppnetwork.org" {
		t.Fatalf("unexpected reply target: %q", got)
	}
}

func TestHandleInvokesOnResultForRPAck(t *testing.T) {
	var target sip.Uri
	if err := sip.ParseUri("sip:3342012834@ims.mnc435.mcc311.3gppnetwork.org", &target); err != nil {
		t.Fatalf("parse target: %v", err)
	}
	req := sip.NewRequest(sip.MESSAGE, target)
	req.AppendHeader(&sip.ViaHeader{Params: sip.NewParams(), Host: "10.90.250.52", Port: 5060, Transport: "UDP"})
	req.AppendHeader(&sip.FromHeader{Address: target, Params: sip.NewParams()})
	req.AppendHeader(&sip.ToHeader{Address: target, Params: sip.NewParams()})
	callID := sip.CallIDHeader("reply-test")
	req.AppendHeader(&callID)
	req.AppendHeader(&sip.CSeqHeader{SeqNo: 1, MethodName: sip.MESSAGE})
	req.AppendHeader(sip.NewHeader("Content-Type", "application/vnd.3gpp.sms"))
	req.AppendHeader(sip.NewHeader("In-Reply-To", "message-123"))
	req.SetBody([]byte{0x03, 0x01})

	var gotReplyTo string
	var gotBody []byte
	h := &MessageHandler{
		OnResult: func(inReplyTo string, body []byte) {
			gotReplyTo = inReplyTo
			gotBody = append([]byte(nil), body...)
		},
	}

	tx := &stubServerTx{}
	h.Handle(req, tx)

	if gotReplyTo != "message-123" {
		t.Fatalf("OnResult inReplyTo = %q, want %q", gotReplyTo, "message-123")
	}
	if len(gotBody) != 2 || gotBody[0] != 0x03 || gotBody[1] != 0x01 {
		t.Fatalf("OnResult body = %v, want [3 1]", gotBody)
	}
	if tx.last == nil || tx.last.StatusCode != sip.StatusOK {
		t.Fatalf("unexpected response: %#v", tx.last)
	}
}

type stubServerTx struct {
	last *sip.Response
}

func (s *stubServerTx) Respond(resp *sip.Response) error {
	s.last = resp
	return nil
}

func (s *stubServerTx) Acks() <-chan *sip.Request    { return nil }
func (s *stubServerTx) Cancels() <-chan *sip.Request { return nil }
func (s *stubServerTx) Terminate()                   {}
func (s *stubServerTx) Done() <-chan struct{}        { return nil }
func (s *stubServerTx) Err() error                   { return nil }
