package smpp

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func FuzzDecode(f *testing.F) {
	addPDU := func(pdu *PDU) {
		f.Helper()
		wire, err := Encode(pdu)
		if err != nil {
			f.Fatalf("encode seed: %v", err)
		}
		f.Add(wire)
	}

	f.Add([]byte{})
	f.Add(make([]byte, HeaderLen))
	addPDU(&PDU{CommandID: CmdEnquireLink, SequenceNumber: 1})
	addPDU(&PDU{
		CommandID:       CmdSubmitSM,
		SequenceNumber:  2,
		SourceAddr:      "15551230001",
		DestinationAddr: "15551230002",
		ShortMessage:    []byte("hello"),
		TLVs:            map[uint16][]byte{TLVReceiptedMessageID: []byte("abc\x00")},
	})

	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) >= 4 {
			declared := binary.BigEndian.Uint32(wire[:4])
			if declared > uint32(len(wire)+1024) && declared <= MaxPDULen {
				return
			}
		}
		pdu, err := Decode(bytes.NewReader(wire))
		if err != nil {
			return
		}
		encoded, err := Encode(pdu)
		if err != nil {
			t.Fatalf("re-encode decoded PDU: %v", err)
		}
		if _, err := Decode(bytes.NewReader(encoded)); err != nil {
			t.Fatalf("decode re-encoded PDU: %v", err)
		}
	})
}
