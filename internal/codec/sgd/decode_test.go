package sgd

import (
	"bytes"
	"testing"

	"github.com/svinson1121/vectorcore-smsc/internal/codec"
	"github.com/svinson1121/vectorcore-smsc/internal/codec/tpdu"
	dcodec "github.com/svinson1121/vectorcore-smsc/internal/diameter/codec"
)

func TestDecodeTFRUsesUserIdentifierMSISDNAsSource(t *testing.T) {
	msg := &codec.Message{
		Destination: codec.Address{MSISDN: "15551230001"},
		Text:        "test",
		Encoding:    codec.EncodingGSM7,
	}
	tpData, err := tpdu.EncodeSubmit(msg)
	if err != nil {
		t.Fatalf("EncodeSubmit() error = %v", err)
	}

	uid, err := dcodec.NewGrouped(
		dcodec.CodeUserIdentifier,
		dcodec.Vendor3GPP,
		dcodec.FlagMandatory|dcodec.FlagVendorSpecific,
		[]*dcodec.AVP{
			dcodec.NewOctetString(dcodec.CodeMSISDN, dcodec.Vendor3GPP,
				dcodec.FlagMandatory|dcodec.FlagVendorSpecific, encodeBCDMSISDN("3342012832")),
		},
	)
	if err != nil {
		t.Fatalf("NewGrouped() error = %v", err)
	}

	wire := dcodec.NewRequest(dcodec.CmdMTForwardShortMessage, dcodec.App3GPP_SGd)
	wire.Add(
		dcodec.NewOctetString(dcodec.CodeSMRPUI, dcodec.Vendor3GPP,
			dcodec.FlagMandatory|dcodec.FlagVendorSpecific, tpData),
		uid,
		dcodec.NewOctetString(dcodec.CodeSCAddress, dcodec.Vendor3GPP,
			dcodec.FlagMandatory|dcodec.FlagVendorSpecific, []byte("15550000001")),
	)

	decoded, err := DecodeTFR(wire.Build())
	if err != nil {
		t.Fatalf("DecodeTFR() error = %v", err)
	}
	if got, want := decoded.Source.MSISDN, "3342012832"; got != want {
		t.Fatalf("Source.MSISDN = %q, want %q", got, want)
	}
	if got, want := decoded.Destination.MSISDN, "15551230001"; got != want {
		t.Fatalf("Destination.MSISDN = %q, want %q", got, want)
	}
}

func TestDecodeSCAddressAcceptsASCIIDigits(t *testing.T) {
	if got, want := decodeSCAddress([]byte("15550000001")), "15550000001"; got != want {
		t.Fatalf("decodeSCAddress() = %q, want %q", got, want)
	}
}

func TestDecodeBCDMSISDNLengthPrefixedWithoutTONNPI(t *testing.T) {
	data := []byte{0x05, 0x33, 0x24, 0x10, 0x82, 0x23}
	if got, want := decodeBCDMSISDN(data), "3342012832"; got != want {
		t.Fatalf("decodeBCDMSISDN() = %q, want %q", got, want)
	}
}

func TestEncodeSCAddressUsesASCIIDigitsWhenConfigured(t *testing.T) {
	got := encodeSCAddress("+15550000000", "ascii_digits")
	want := []byte("15550000000")
	if !bytes.Equal(got, want) {
		t.Fatalf("encodeSCAddress() = %x, want %x", got, want)
	}
}
