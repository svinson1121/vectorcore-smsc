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
			dcodec.NewOctetString(dcodec.CodeMMENumberForMTSMSServing, dcodec.Vendor3GPP, dcodec.FlagVendorSpecific, encodeTBCD("15550000001")),
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
	if got, want := info.MMENumber, "15550000001"; got != want {
		t.Fatalf("MMENumber = %q, want %q", got, want)
	}
}

func TestParseSRIAnswerServingNodeMMENumberUsesE164Digits(t *testing.T) {
	servingNode, err := dcodec.NewGrouped(
		dcodec.CodeServingNode,
		dcodec.Vendor3GPP,
		dcodec.FlagMandatory|dcodec.FlagVendorSpecific,
		[]*dcodec.AVP{
			dcodec.NewString(dcodec.CodeMMEName, dcodec.Vendor3GPP, dcodec.FlagMandatory|dcodec.FlagVendorSpecific, "s6a-vpc-si-01.epc.mnc435.mcc311.3gppnetwork.org"),
			dcodec.NewOctetString(dcodec.CodeMMENumberForMTSMSServing, dcodec.Vendor3GPP, dcodec.FlagVendorSpecific, []byte{0x51, 0x55, 0x00, 0x00, 0x00, 0xF1}),
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
			dcodec.NewString(dcodec.CodeSessionID, 0, dcodec.FlagMandatory, "smsc.example;123;3"),
			dcodec.NewUint32(dcodec.CodeResultCode, 0, dcodec.FlagMandatory, dcodec.DiameterSuccess),
			servingNode,
		},
	}

	info, err := parseSRIAnswer(msg)
	if err != nil {
		t.Fatalf("parseSRIAnswer() error = %v", err)
	}
	if got, want := info.MMENumber, "15550000001"; got != want {
		t.Fatalf("MMENumber = %q, want %q", got, want)
	}
}

func TestNormalizeE164Address(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "digits", input: "15550000001", want: "15550000001"},
		{name: "plus digits", input: "+15550000001", want: "15550000001"},
		{name: "tbcd hex text", input: "5155000000f1", want: "15550000001"},
		{name: "raw tbcd bytes", input: string([]byte{0x51, 0x55, 0x00, 0x00, 0x00, 0xF1}), want: "15550000001"},
	}

	for _, tt := range tests {
		if got := NormalizeE164Address(tt.input); got != tt.want {
			t.Fatalf("%s: NormalizeE164Address(%q) = %q, want %q", tt.name, tt.input, got, tt.want)
		}
	}
}

func TestDecodeTBCDForSCAddress(t *testing.T) {
	if got, want := decodeTBCD([]byte{0x51, 0x55, 0x00, 0x00, 0x00, 0xF1}), "15550000001"; got != want {
		t.Fatalf("decodeTBCD(sc-address) = %q, want %q", got, want)
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
