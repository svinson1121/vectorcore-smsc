package smppcodec

import (
	"testing"

	"github.com/svinson1121/vectorcore-smsc/internal/codec"
	"github.com/svinson1121/vectorcore-smsc/internal/smpp"
)

func TestDecodeSMRejectsTruncatedUDH(t *testing.T) {
	pdu := &smpp.PDU{
		CommandID:       smpp.CmdSubmitSM,
		ESMClass:        smpp.ESMClassUDHI,
		DataCoding:      0x04,
		ShortMessage:    []byte{0x06, 0x08, 0x04},
		SourceAddr:      "15551230001",
		DestinationAddr: "15551230002",
	}

	_, err := DecodeSM(pdu)
	if err == nil {
		t.Fatal("expected truncated UDH error")
	}
}

func TestDecodeSMDecodesBinaryPayloadWithUDH(t *testing.T) {
	rawUDH := []byte{0x06, 0x08, 0x04, 0x12, 0x34, 0x02, 0x01}
	pdu := &smpp.PDU{
		CommandID:       smpp.CmdSubmitSM,
		ESMClass:        smpp.ESMClassUDHI,
		DataCoding:      0x04,
		ShortMessage:    append(append([]byte(nil), rawUDH...), 0xde, 0xad),
		SourceAddr:      "15551230001",
		DestinationAddr: "15551230002",
	}

	msg, err := DecodeSM(pdu)
	if err != nil {
		t.Fatalf("DecodeSM returned error: %v", err)
	}
	if msg.Encoding != codec.EncodingBinary {
		t.Fatalf("unexpected encoding: got %v", msg.Encoding)
	}
	if got := msg.Concat; got == nil || got.Ref != 0x1234 || got.Total != 2 || got.Sequence != 1 {
		t.Fatalf("unexpected concat info: %#v", got)
	}
	if msg.UDH == nil || string(msg.UDH.Raw) != string(rawUDH) {
		t.Fatalf("unexpected raw UDH: %x", msg.UDH.Raw)
	}
	if string(msg.Binary) != string([]byte{0xde, 0xad}) {
		t.Fatalf("unexpected binary payload: %x", msg.Binary)
	}
}
