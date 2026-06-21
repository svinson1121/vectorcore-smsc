package codec

import (
	"bytes"
	"testing"
)

func FuzzDecodeAVPs(f *testing.F) {
	f.Add([]byte{})
	for _, seed := range []*AVP{
		NewString(CodeOriginHost, 0, FlagMandatory, "smsc.example.net"),
		NewUint32(CodeResultCode, 0, FlagMandatory, DiameterSuccess),
		NewOctetString(1, Vendor3GPP, FlagMandatory, []byte{0x00, 0x01, 0x02}),
	} {
		encoded, err := Encode(seed)
		if err != nil {
			f.Fatalf("encode seed: %v", err)
		}
		f.Add(encoded)
	}

	f.Fuzz(func(t *testing.T, wire []byte) {
		avps, err := DecodeAll(wire)
		if err != nil {
			return
		}
		var encoded []byte
		for _, avp := range avps {
			b, err := Encode(avp)
			if err != nil {
				t.Fatalf("re-encode decoded AVP: %v", err)
			}
			encoded = append(encoded, b...)
		}
		if len(encoded) != len(wire) {
			t.Fatalf("decoded AVPs re-encode to %d bytes, input used %d", len(encoded), len(wire))
		}
		if _, err := DecodeAll(encoded); err != nil {
			t.Fatalf("decode re-encoded AVPs: %v", err)
		}
	})
}

func FuzzDecodeMessage(f *testing.F) {
	f.Add([]byte{})
	msg := NewRequest(CmdDeviceWatchdog, AppDiameterCommon).
		Add(NewString(CodeOriginHost, 0, FlagMandatory, "smsc.example.net")).
		Build()
	encoded, err := msg.Encode()
	if err != nil {
		f.Fatalf("encode seed: %v", err)
	}
	f.Add(encoded)

	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) >= 4 {
			declared := uint32(wire[1])<<16 | uint32(wire[2])<<8 | uint32(wire[3])
			if declared > uint32(len(wire)+1024) {
				return
			}
		}
		msg, err := DecodeMessage(bytes.NewReader(wire))
		if err != nil {
			return
		}
		encoded, err := msg.Encode()
		if err != nil {
			t.Fatalf("re-encode decoded message: %v", err)
		}
		if _, err := DecodeMessage(bytes.NewReader(encoded)); err != nil {
			t.Fatalf("decode re-encoded message: %v", err)
		}
	})
}
