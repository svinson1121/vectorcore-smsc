package sgd

import (
	"testing"

	"github.com/svinson1121/vectorcore-smsc/internal/codec"
	sgdcodec "github.com/svinson1121/vectorcore-smsc/internal/codec/sgd"
	"github.com/svinson1121/vectorcore-smsc/internal/codec/tpdu"
	"github.com/svinson1121/vectorcore-smsc/internal/diameter"
	dcodec "github.com/svinson1121/vectorcore-smsc/internal/diameter/codec"
)

func TestBuildAlertSCRequestIncludesRequiredAVPs(t *testing.T) {
	msg := buildAlertSCRequest(
		diameter.Config{LocalFQDN: "smsc.example.net", LocalRealm: "example.net"},
		"mme01.epc.example.net",
		"epc.example.net",
		"3342012832",
		"+15550000000",
	)

	if !msg.IsRequest() {
		t.Fatal("expected Alert-SC to be a request")
	}
	if got, want := msg.Header.CommandCode, dcodec.CmdAlertServiceCentre; got != want {
		t.Fatalf("command = %d, want %d", got, want)
	}
	if got, want := msg.Header.AppID, dcodec.App3GPP_SGd; got != want {
		t.Fatalf("app_id = %d, want %d", got, want)
	}
	requiredBase := []uint32{
		dcodec.CodeSessionID,
		dcodec.CodeAuthSessionState,
		dcodec.CodeOriginHost,
		dcodec.CodeOriginRealm,
		dcodec.CodeDestinationHost,
		dcodec.CodeDestinationRealm,
	}
	for _, code := range requiredBase {
		if msg.FindAVP(code, 0) == nil {
			t.Fatalf("missing required base AVP %d", code)
		}
	}
	if msg.FindAVP(dcodec.CodeMSISDN, dcodec.Vendor3GPP) == nil {
		t.Fatal("missing MSISDN AVP")
	}
	if msg.FindAVP(dcodec.CodeSCAddress, dcodec.Vendor3GPP) == nil {
		t.Fatal("missing SC-Address AVP")
	}
}

func TestBuildOFRRequestIncludesRequiredAVPs(t *testing.T) {
	canonical := &codec.Message{}
	canonical.Source.MSISDN = "15551234567"
	canonical.Destination.MSISDN = "3342012832"
	canonical.Destination.IMSI = "311435000070570"
	canonical.TPMR = 7
	canonical.DCS = 0
	tpData, err := tpdu.EncodeDeliver(canonical)
	if err != nil {
		t.Fatalf("EncodeDeliver() error = %v", err)
	}
	avps, err := sgdcodec.EncodeOFR(canonical, "+15550000000")
	if err != nil {
		t.Fatalf("EncodeOFR() error = %v", err)
	}
	msg := buildOFRRequest("smsc.example.net", "example.net", "mme01.epc.example.net", "epc.example.net", avps)

	if !msg.IsRequest() {
		t.Fatal("expected OFR to be a request")
	}
	if got, want := msg.Header.CommandCode, dcodec.CmdMTForwardShortMessage; got != want {
		t.Fatalf("command = %d, want %d", got, want)
	}
	if got, want := msg.Header.AppID, dcodec.App3GPP_SGd; got != want {
		t.Fatalf("app_id = %d, want %d", got, want)
	}
	requiredBase := []uint32{
		dcodec.CodeSessionID,
		dcodec.CodeAuthSessionState,
		dcodec.CodeOriginHost,
		dcodec.CodeOriginRealm,
		dcodec.CodeDestinationHost,
		dcodec.CodeDestinationRealm,
	}
	for _, code := range requiredBase {
		if msg.FindAVP(code, 0) == nil {
			t.Fatalf("missing required base AVP %d", code)
		}
	}
	if avp := msg.FindAVP(dcodec.CodeSMRPUI, dcodec.Vendor3GPP); avp == nil {
		t.Fatal("missing SM-RP-UI")
	} else if string(avp.Data) != string(tpData) {
		t.Fatal("SM-RP-UI payload mismatch")
	}
	if msg.FindAVP(dcodec.CodeSMRPMTI, dcodec.Vendor3GPP) != nil {
		t.Fatal("unexpected SM-RP-MTI AVP on normal MT SGd request")
	}
	if msg.FindAVP(dcodec.CodeSCAddress, dcodec.Vendor3GPP) == nil {
		t.Fatal("missing SC-Address AVP")
	}
	if avp := msg.FindAVP(dcodec.CodeUserName, 0); avp == nil {
		t.Fatal("missing User-Name AVP")
	} else if got, want := string(avp.Data), canonical.Destination.IMSI; got != want {
		t.Fatalf("User-Name = %q, want %q", got, want)
	}
	if msg.FindAVP(dcodec.CodeMSISDN, dcodec.Vendor3GPP) != nil {
		t.Fatal("unexpected MSISDN AVP on MT SGd request")
	}
}
