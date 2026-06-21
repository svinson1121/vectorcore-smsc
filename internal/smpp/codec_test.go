package smpp

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestDecodeRejectsOversizedPDU(t *testing.T) {
	header := make([]byte, HeaderLen)
	binary.BigEndian.PutUint32(header[:4], MaxPDULen+1)

	if _, err := Decode(bytes.NewReader(header)); err == nil {
		t.Fatal("Decode accepted an oversized command_length")
	}
}

func TestEncodeRejectsOversizedShortMessage(t *testing.T) {
	pdu := &PDU{CommandID: CmdSubmitSM, ShortMessage: make([]byte, 256)}
	if _, err := Encode(pdu); err == nil {
		t.Fatal("Encode accepted a short_message longer than 255 octets")
	}
}

func TestEncodeRejectsOversizedTLV(t *testing.T) {
	pdu := &PDU{
		CommandID: CmdSubmitSM,
		TLVs:      map[uint16][]byte{TLVMessagePayload: make([]byte, 65536)},
	}
	if _, err := Encode(pdu); err == nil {
		t.Fatal("Encode accepted a TLV value longer than 65535 octets")
	}
}

func TestDecodeRejectsTrailingPartialTLV(t *testing.T) {
	pdu := &PDU{CommandID: CmdSubmitSM}
	wire, err := Encode(pdu)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	wire = append(wire, 0x00)
	binary.BigEndian.PutUint32(wire[:4], uint32(len(wire)))

	if _, err := Decode(bytes.NewReader(wire)); err == nil {
		t.Fatal("Decode accepted a partial trailing TLV")
	}
}
