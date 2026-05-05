package sip3gpp

import (
	"testing"

	"github.com/svinson1121/vectorcore-smsc/internal/codec"
	"github.com/svinson1121/vectorcore-smsc/internal/codec/tpdu"
	"github.com/svinson1121/vectorcore-smsc/internal/numbering"
)

func TestEncodeRPAddressStripsLeadingPlus(t *testing.T) {
	got := encodeRPAddress("+15550000000")
	if len(got) < 3 {
		t.Fatalf("encoded RP address too short: %v", got)
	}
	if got[0] == 0x00 {
		t.Fatalf("expected non-empty RP address for +E.164 input: %v", got)
	}
	if got[1] != 0x91 {
		t.Fatalf("unexpected TOA: got 0x%02x want 0x91", got[1])
	}
}

func TestEncodeRPAddressRejectsNonDigits(t *testing.T) {
	got := encodeRPAddress("smsc")
	if len(got) != 1 || got[0] != 0x00 {
		t.Fatalf("expected empty RP address for invalid input, got %v", got)
	}
}

func TestDecodeRPDataUsesTPDestinationAddressForRecipient(t *testing.T) {
	tpData, err := tpdu.EncodeSubmit(&codec.Message{
		Destination: codec.Address{
			MSISDN: "6752012860",
			TON:    numbering.TONUnknown,
			NPI:    0x01,
		},
		Text:     "hello",
		Encoding: codec.EncodingGSM7,
	})
	if err != nil {
		t.Fatalf("EncodeSubmit() error = %v", err)
	}

	body := []byte{rpMTIDataMStoSC, 0x7a}
	body = append(body, encodeRPAddress("")...)
	body = append(body, encodeRPAddress("15550000000")...)
	body = append(body, byte(len(tpData)))
	body = append(body, tpData...)

	msg, err := Decode(body)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got, want := msg.Destination.MSISDN, "6752012860"; got != want {
		t.Fatalf("Destination.MSISDN = %q, want TP-DA %q", got, want)
	}
	if got, want := msg.Destination.TON, numbering.TONUnknown; got != want {
		t.Fatalf("Destination.TON = %d, want %d", got, want)
	}
	if got, want := msg.Destination.NPI, byte(0x01); got != want {
		t.Fatalf("Destination.NPI = %d, want %d", got, want)
	}
}
