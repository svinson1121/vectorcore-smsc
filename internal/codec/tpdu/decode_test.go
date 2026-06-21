package tpdu

import "testing"

func TestDecodeStatusReportRejectsMissingStatus(t *testing.T) {
	// TP-MTI, TP-MR, empty recipient address, SCTS, and discharge time.
	// TP-ST is missing.
	data := []byte{
		0x02, 0x00, 0x00, 0x81,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	if _, err := Decode(data); err == nil {
		t.Fatal("Decode accepted a status report without TP-ST")
	}
}

func TestDecodeRejectsUDHLargerThanUDL(t *testing.T) {
	// SMS-DELIVER with UDHI, empty originator, binary DCS, UDL=1, but a
	// two-octet UDH (UDHL plus one content octet).
	data := []byte{
		0x40, 0x00, 0x81, 0x00, 0x04,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x01, 0x01, 0x00,
	}

	if _, err := Decode(data); err == nil {
		t.Fatal("Decode accepted a UDH larger than TP-UDL")
	}
}

func TestDecodeRejectsTruncatedUserData(t *testing.T) {
	// SMS-DELIVER declaring a four-octet binary payload with one octet present.
	data := []byte{
		0x00, 0x00, 0x81, 0x00, 0x04,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x04, 0xff,
	}

	if _, err := Decode(data); err == nil {
		t.Fatal("Decode accepted truncated TP-UD")
	}
}
