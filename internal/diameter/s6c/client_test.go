package s6c

import (
	"testing"

	dcodec "github.com/svinson1121/vectorcore-smsc/internal/diameter/codec"
)

func TestEncodeDecodeTBCDRoundTrip(t *testing.T) {
	input := "+15550000000"
	encoded := encodeTBCD(input)
	if got, want := decodeTBCD(encoded), "15550000000"; got != want {
		t.Fatalf("decodeTBCD(encodeTBCD(%q)) = %q, want %q", input, got, want)
	}
}

func TestParseSRIAnswerAttachedSubscriber(t *testing.T) {
	servingNode, err := dcodec.NewGrouped(
		dcodec.CodeServingNode,
		dcodec.Vendor3GPP,
		dcodec.FlagMandatory|dcodec.FlagVendorSpecific,
		[]*dcodec.AVP{
			dcodec.NewString(dcodec.CodeMMEName, dcodec.Vendor3GPP, dcodec.FlagMandatory|dcodec.FlagVendorSpecific, "s6a-vpc-si-01.epc.mnc435.mcc311.3gppnetwork.org"),
			dcodec.NewString(dcodec.CodeMMERealm, dcodec.Vendor3GPP, dcodec.FlagMandatory|dcodec.FlagVendorSpecific, "epc.mnc435.mcc311.3gppnetwork.org"),
		},
	)
	if err != nil {
		t.Fatalf("NewGrouped() error = %v", err)
	}

	msg := &dcodec.Message{
		Header: dcodec.Header{
			CommandCode: dcodec.CmdSendRoutingInfoSM,
			AppID:       dcodec.App3GPP_S6c,
		},
		AVPs: []*dcodec.AVP{
			dcodec.NewString(dcodec.CodeSessionID, 0, dcodec.FlagMandatory, "smsc.example;123;1"),
			dcodec.NewUint32(dcodec.CodeResultCode, 0, dcodec.FlagMandatory, dcodec.DiameterSuccess),
			dcodec.NewString(dcodec.CodeUserName, 0, dcodec.FlagMandatory, "311435000070570"),
			dcodec.NewOctetString(dcodec.CodeMSISDN, dcodec.Vendor3GPP, dcodec.FlagMandatory|dcodec.FlagVendorSpecific, encodeTBCD("3342012832")),
			servingNode,
		},
	}

	info, err := parseSRIAnswer(msg)
	if err != nil {
		t.Fatalf("parseSRIAnswer() error = %v", err)
	}
	if got, want := info.IMSI, "311435000070570"; got != want {
		t.Fatalf("IMSI = %q, want %q", got, want)
	}
	if got, want := info.MSISDN, "3342012832"; got != want {
		t.Fatalf("MSISDN = %q, want %q", got, want)
	}
	if !info.Attached {
		t.Fatal("Attached = false, want true")
	}
	if got, want := info.MMEName, "s6a-vpc-si-01.epc.mnc435.mcc311.3gppnetwork.org"; got != want {
		t.Fatalf("MMEName = %q, want %q", got, want)
	}
	if got, want := info.MMERealm, "epc.mnc435.mcc311.3gppnetwork.org"; got != want {
		t.Fatalf("MMERealm = %q, want %q", got, want)
	}
}

func TestParseSRIAnswerUnattachedSubscriber(t *testing.T) {
	msg := &dcodec.Message{
		Header: dcodec.Header{
			CommandCode: dcodec.CmdSendRoutingInfoSM,
			AppID:       dcodec.App3GPP_S6c,
		},
		AVPs: []*dcodec.AVP{
			dcodec.NewString(dcodec.CodeSessionID, 0, dcodec.FlagMandatory, "smsc.example;123;2"),
			dcodec.NewUint32(dcodec.CodeResultCode, 0, dcodec.FlagMandatory, dcodec.DiameterSuccess),
			dcodec.NewString(dcodec.CodeUserName, 0, dcodec.FlagMandatory, "311435000070570"),
			dcodec.NewUint32(dcodec.CodeMWDStatusS6c, dcodec.Vendor3GPP, dcodec.FlagMandatory|dcodec.FlagVendorSpecific, dcodec.MWDStatusMNRF),
		},
	}

	info, err := parseSRIAnswer(msg)
	if err != nil {
		t.Fatalf("parseSRIAnswer() error = %v", err)
	}
	if info.Attached {
		t.Fatal("Attached = true, want false")
	}
	if got, want := info.MWDStatus, dcodec.MWDStatusMNRF; got != want {
		t.Fatalf("MWDStatus = %d, want %d", got, want)
	}
}
