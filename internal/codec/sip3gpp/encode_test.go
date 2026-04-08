package sip3gpp

import "testing"

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
