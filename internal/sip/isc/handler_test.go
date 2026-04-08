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
