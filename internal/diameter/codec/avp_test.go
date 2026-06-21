package codec

import "testing"

func TestDecodeRejectsTruncatedPadding(t *testing.T) {
	// AVP length 15 requires one padding octet on the wire.
	wire := []byte{
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x0f,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	if _, _, err := Decode(wire); err == nil {
		t.Fatal("Decode accepted an AVP with truncated padding")
	}
}

func TestDecodeRejectsZeroVendorIDWithVendorFlag(t *testing.T) {
	wire := []byte{
		0x00, 0x00, 0x00, 0x01,
		FlagVendorSpecific, 0x00, 0x00, 0x0c,
		0x00, 0x00, 0x00, 0x00,
	}

	if _, _, err := Decode(wire); err == nil {
		t.Fatal("Decode accepted the vendor flag with Vendor-ID zero")
	}
}
