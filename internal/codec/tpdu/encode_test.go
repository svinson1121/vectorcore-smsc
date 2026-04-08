package tpdu

import (
	"bytes"
	"testing"

	"github.com/svinson1121/vectorcore-smsc/internal/codec"
)

func TestEncodeDeliverPreservesRawUDHForBinaryPayload(t *testing.T) {
	rawUDH := []byte{0x0b, 0x05, 0x04, 0x0b, 0x84, 0x23, 0xf0, 0x00, 0x03, 0x42, 0x02, 0x01}
	payload := []byte{0x01, 0x06, 0x03, 0xbe, 0xaf}
	msg := &codec.Message{
		Source:   codec.Address{MSISDN: "3342012832"},
		UDH:      &codec.UDH{Raw: rawUDH},
		Concat:   &codec.ConcatInfo{Ref: 0x0042, Total: 2, Sequence: 1},
		Binary:   payload,
		DCS:      0x04,
		Encoding: codec.EncodingBinary,
	}

	tpdu, err := EncodeDeliver(msg)
	if err != nil {
		t.Fatalf("EncodeDeliver returned error: %v", err)
	}

	gotUDL := int(tpdu[len(tpdu)-len(rawUDH)-len(payload)-1])
	if gotUDL != len(rawUDH)+len(payload) {
		t.Fatalf("TP-UDL = %d, want %d", gotUDL, len(rawUDH)+len(payload))
	}
	gotUD := tpdu[len(tpdu)-len(rawUDH)-len(payload):]
	wantUD := append(append([]byte(nil), rawUDH...), payload...)
	if !bytes.Equal(gotUD, wantUD) {
		t.Fatalf("TP-UD = %x, want %x", gotUD, wantUD)
	}
}

func TestDecodeUCS2SurrogatePair(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "single emoji surrogate pair (U+1F633 😳)",
			input: []byte{0xD8, 0x3D, 0xDE, 0x33},
			want:  "😳",
		},
		{
			name:  "BMP chars pass through unchanged",
			input: []byte{0x26, 0x3A, 0xFE, 0x0F},
			want:  "☺️",
		},
		{
			name:  "mixed BMP and surrogate pair",
			input: []byte{0x26, 0x3A, 0xD8, 0x3D, 0xDE, 0x33},
			want:  "☺😳",
		},
		{
			name:  "multiple surrogate pairs",
			input: []byte{0xD8, 0x3D, 0xDE, 0x02, 0xD8, 0x3D, 0xDE, 0x00},
			want:  "😂😀",
		},
		{
			name:  "U+1F914 🤔 (D83E DD14)",
			input: []byte{0xD8, 0x3E, 0xDD, 0x14},
			want:  "🤔",
		},
		{
			name:  "unpaired high surrogate replaced with U+FFFD",
			input: []byte{0xD8, 0x3D, 0x00, 0x41},
			want:  "\uFFFDA",
		},
		{
			name:  "unpaired low surrogate replaced with U+FFFD",
			input: []byte{0xDE, 0x33, 0x00, 0x41},
			want:  "\uFFFDA",
		},
		{
			name:  "eight surrogate pairs (multi-emoji message from pcap)",
			input: []byte{0xD8, 0x3E, 0xDE, 0xE5, 0xD8, 0x3D, 0xDE, 0x33, 0xD8, 0x3D, 0xDE, 0x02, 0xD8, 0x3D, 0xDE, 0x00},
			want:  "🫥😳😂😀",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecodeUCS2(tt.input)
			if got != tt.want {
				t.Errorf("DecodeUCS2(%x) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEncodeUCS2SurrogatePair(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []byte
	}{
		{
			name:  "single emoji (U+1F633 😳)",
			input: "😳",
			want:  []byte{0xD8, 0x3D, 0xDE, 0x33},
		},
		{
			name:  "BMP chars unchanged",
			input: "☺️",
			want:  []byte{0x26, 0x3A, 0xFE, 0x0F},
		},
		{
			name:  "U+1F914 🤔",
			input: "🤔",
			want:  []byte{0xD8, 0x3E, 0xDD, 0x14},
		},
		{
			name:  "mixed BMP and SMP",
			input: "☺😳",
			want:  []byte{0x26, 0x3A, 0xD8, 0x3D, 0xDE, 0x33},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeUCS2(tt.input)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("EncodeUCS2(%q) = %x, want %x", tt.input, got, tt.want)
			}
		})
	}
}

func TestDecodeEncodeUCS2RoundTrip(t *testing.T) {
	// Every emoji from the pcap that was being mangled.
	emojis := "😳😒😊😏😌😔😂😑😀😉😕🙁😣🧐😋😟😶🤔😰😨"
	encoded := EncodeUCS2(emojis)
	decoded := DecodeUCS2(encoded)
	if decoded != emojis {
		t.Errorf("round-trip failed:\n  got  %q\n  want %q", decoded, emojis)
	}
}
