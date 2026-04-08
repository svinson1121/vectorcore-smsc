package isc

import (
	"testing"

	"github.com/emiago/sipgo/sip"
	"github.com/svinson1121/vectorcore-smsc/internal/codec"
	"github.com/svinson1121/vectorcore-smsc/internal/registry"
)

func TestBuildRequestIncludes3GPPHeadersAndFromTag(t *testing.T) {
	sender := &Sender{
		scAddr:   "+15550000000",
		sipLocal: "sip:smsc@smsc.ims.mnc435.mcc311.3gppnetwork.org",
		cfg: Settings{
			AcceptContact:        "*;+g.3gpp.smsip",
			MTRequestDisposition: "no-fork",
		},
	}
	msg := &codec.Message{}
	msg.Destination.MSISDN = "3342012832"
	msg.Source.MSISDN = "3342012834"

	var targetURI sip.Uri
	if err := sip.ParseUri("sip:3342012832@ims.mnc435.mcc311.3gppnetwork.org", &targetURI); err != nil {
		t.Fatalf("parse target URI: %v", err)
	}

	req, err := sender.buildRequest(msg, &registry.Registration{
		SIPAOR: targetURI.String(),
		SCSCF:  "10.90.250.52",
	}, []byte{0x01, 0x03, 0x00})
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	from := req.From()
	if from == nil {
		t.Fatal("expected From header")
	}
	if from.Params == nil || !from.Params.Has("tag") {
		t.Fatalf("expected From tag, got %#v", from.Params)
	}
	if got := headerValue(req, "Accept-Contact"); got != "*;+g.3gpp.smsip" {
		t.Fatalf("unexpected Accept-Contact: %q", got)
	}
	if got := headerValue(req, "Request-Disposition"); got != "no-fork" {
		t.Fatalf("unexpected Request-Disposition: %q", got)
	}
	if got := headerValue(req, "P-Asserted-Identity"); got != "<sip:smsc@smsc.ims.mnc435.mcc311.3gppnetwork.org>" {
		t.Fatalf("unexpected P-Asserted-Identity: %q", got)
	}
}

func headerValue(req *sip.Request, name string) string {
	h := req.GetHeader(name)
	if h == nil {
		return ""
	}
	return h.Value()
}
