package sip3gpp

import "testing"

func FuzzDecode(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{rpMTIACKMStoSC})
	f.Add([]byte{rpMTIDataMStoSC, 0x01, 0x00, 0x00, 0x00})
	f.Add([]byte{0x07, 0x01})

	f.Fuzz(func(t *testing.T, body []byte) {
		_, _ = Decode(body)
	})
}
