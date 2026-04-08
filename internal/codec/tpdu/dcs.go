// Package tpdu implements encoding and decoding of GSM TP-DATA (SMS PDU layer)
// as defined in 3GPP TS 23.040.
package tpdu

import (
	"github.com/svinson1121/vectorcore-smsc/internal/codec"
)

// ParseDCS extracts the character encoding from a Data Coding Scheme byte
// (3GPP TS 23.038 §4).
func ParseDCS(dcs byte) codec.Encoding {
	upper := dcs >> 4

	switch {
	case upper <= 0x03:
		// General Data Coding group (bits 7-6 = 00)
		// Bit 5: compressed (ignore for encoding purposes)
		// Bits 3-2: charset
		return charsetBits((dcs >> 2) & 0x03)

	case upper >= 0x04 && upper <= 0x0B:
		// Reserved / auto-delete groups — treat charset same as general group
		return charsetBits((dcs >> 2) & 0x03)

	case upper == 0x0C:
		// Message Waiting Indication — discard message, GSM7
		return codec.EncodingGSM7

	case upper == 0x0D:
		// Message Waiting Indication — store, GSM7
		return codec.EncodingGSM7

	case upper == 0x0E:
		// Message Waiting Indication — store, UCS2
		return codec.EncodingUCS2

	case upper == 0x0F:
		// Data coding / message class (bits 7-4 = 1111)
		// Bit 2: 0 = GSM7, 1 = 8-bit binary
		if dcs&0x04 != 0 {
			return codec.EncodingBinary
		}
		return codec.EncodingGSM7

	default:
		return codec.EncodingGSM7
	}
}

// BuildDCS returns a minimal DCS byte for the given encoding with no message
// class.  It is intentionally lossy — it encodes only the character set.
// Do not use this when forwarding a received message; use dcsForEgress instead
// so that message class and coding-group flags (e.g. 0xF5 = binary class 1
// for WAP push) are preserved from the original inbound DCS.
func BuildDCS(enc codec.Encoding) byte {
	switch enc {
	case codec.EncodingGSM7:
		return 0x00
	case codec.EncodingBinary:
		return 0x04
	case codec.EncodingUCS2:
		return 0x08
	default:
		return 0x00
	}
}

func charsetBits(b byte) codec.Encoding {
	switch b & 0x03 {
	case 0x01:
		return codec.EncodingBinary
	case 0x02:
		return codec.EncodingUCS2
	default: // 0x00 and 0x03 (reserved → treat as GSM7)
		return codec.EncodingGSM7
	}
}

// --- GSM 7-bit default alphabet (3GPP TS 23.038 Table 1) ---

// gsm7Basic maps GSM 7-bit positions 0x00–0x7F to Unicode runes.
// Position 0x1B is the ESC character used with the extension table.
var gsm7Basic = [128]rune{
	'@', '£', '$', '¥', 'è', 'é', 'ù', 'ì', 'ò', 'Ç', '\n', 'Ø', 'ø', '\r', 'Å', 'å',
	'Δ', '_', 'Φ', 'Γ', 'Λ', 'Ω', 'Π', 'Ψ', 'Σ', 'Θ', 'Ξ', 0x1B, 'Æ', 'æ', 'ß', 'É',
	' ', '!', '"', '#', '¤', '%', '&', '\'', '(', ')', '*', '+', ',', '-', '.', '/',
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', ':', ';', '<', '=', '>', '?',
	'¡', 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O',
	'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z', 'Ä', 'Ö', 'Ñ', 'Ü', '§',
	'¿', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o',
	'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', 'ä', 'ö', 'ñ', 'ü', 'à',
}

// gsm7Ext maps GSM 7-bit extension table positions to Unicode runes.
// Only populated positions are listed; zero means "not defined".
var gsm7Ext = [128]rune{
	// 0x0A: FF (form feed)
	0x0A: '\f',
	// 0x14: ^
	0x14: '^',
	// 0x28: {
	0x28: '{',
	// 0x29: }
	0x29: '}',
	// 0x2F: \ (backslash)
	0x2F: '\\',
	// 0x3C: [
	0x3C: '[',
	// 0x3D: ~
	0x3D: '~',
	// 0x3E: ]
	0x3E: ']',
	// 0x40: |
	0x40: '|',
	// 0x65: €
	0x65: '€',
}

// runeToGSM7 is the reverse map: rune → GSM7 byte.
// Built at init time from gsm7Basic.
var runeToGSM7 map[rune]byte

// runeToGSM7Ext is the reverse map for extension characters: rune → ext byte.
var runeToGSM7Ext map[rune]byte

func init() {
	runeToGSM7 = make(map[rune]byte, 128)
	for i, r := range gsm7Basic {
		if r != 0 && r != 0x1B {
			runeToGSM7[r] = byte(i)
		}
	}
	runeToGSM7Ext = make(map[rune]byte, 16)
	for i, r := range gsm7Ext {
		if r != 0 {
			runeToGSM7Ext[r] = byte(i)
		}
	}
}

// DecodeGSM7 unpacks a GSM 7-bit packed byte slice into a UTF-8 string.
// udlBits is the number of septets (characters) in the payload.
// If hasUDH is true the first few bits are consumed by the UDH padding.
func DecodeGSM7(packed []byte, udlSeptets int, udhOctets int) string {
	// Unpack septets
	septets := unpackGSM7(packed, udlSeptets, udhOctets)

	runes := make([]rune, 0, len(septets))
	for i := 0; i < len(septets); i++ {
		b := septets[i]
		if b == 0x1B {
			// Extension table — next character
			i++
			if i < len(septets) {
				ext := septets[i]
				if ext < 128 && gsm7Ext[ext] != 0 {
					runes = append(runes, gsm7Ext[ext])
				} else {
					runes = append(runes, ' ') // undefined extension → space
				}
			}
		} else if b < 128 {
			runes = append(runes, gsm7Basic[b])
		}
	}
	return string(runes)
}

// EncodeGSM7 converts a UTF-8 string to a GSM 7-bit packed byte slice.
// Returns the packed bytes and the septet count (TP-UDL value).
// Characters not representable in GSM7 are replaced with '?'.
func EncodeGSM7(text string) (packed []byte, septets int) {
	raw := make([]byte, 0, len(text)*2)
	for _, r := range text {
		if b, ok := runeToGSM7[r]; ok {
			raw = append(raw, b)
		} else if b, ok := runeToGSM7Ext[r]; ok {
			raw = append(raw, 0x1B, b)
		} else {
			raw = append(raw, runeToGSM7['?'])
		}
	}
	return packGSM7(raw), len(raw)
}

// unpackGSM7 converts packed 7-bit octets to a septet slice.
// udhOctets is the number of UDH octets that precede the text (for bit alignment).
func unpackGSM7(packed []byte, septets int, udhOctets int) []byte {
	out := make([]byte, 0, septets)
	// Calculate the fill bits: UDH octets * 8 must align to septet boundary
	fillBits := 0
	if udhOctets > 0 {
		fillBits = (7 - (udhOctets*8)%7) % 7
	}

	bitPos := udhOctets*8 + fillBits
	for i := 0; i < septets; i++ {
		byteIdx := bitPos / 8
		bitIdx := bitPos % 8
		if byteIdx >= len(packed) {
			break
		}
		var septet byte
		septet = (packed[byteIdx] >> uint(bitIdx)) & 0x7F
		if bitIdx > 1 && byteIdx+1 < len(packed) {
			septet |= (packed[byteIdx+1] << uint(8-bitIdx)) & 0x7F
		}
		out = append(out, septet)
		bitPos += 7
	}
	return out
}

// packGSM7 converts a septet slice to packed 7-bit octets.
func packGSM7(septets []byte) []byte {
	if len(septets) == 0 {
		return nil
	}
	packed := make([]byte, (len(septets)*7+7)/8)
	for i, s := range septets {
		bitPos := i * 7
		byteIdx := bitPos / 8
		bitIdx := bitPos % 8
		packed[byteIdx] |= s << uint(bitIdx)
		if bitIdx > 1 && byteIdx+1 < len(packed) {
			packed[byteIdx+1] |= s >> uint(8-bitIdx)
		}
	}
	return packed
}

// DecodeUCS2 converts a big-endian UCS-2/UTF-16 byte slice to a UTF-8 string.
// Surrogate pairs (U+D800–U+DBFF followed by U+DC00–U+DFFF) are combined into
// a single supplementary code point, matching how phones encode emoji above U+FFFF.
// An unpaired surrogate is replaced with U+FFFD.
func DecodeUCS2(b []byte) string {
	if len(b)%2 != 0 {
		b = b[:len(b)-1]
	}
	words := make([]uint16, len(b)/2)
	for i := range words {
		words[i] = uint16(b[i*2])<<8 | uint16(b[i*2+1])
	}
	runes := make([]rune, 0, len(words))
	for i := 0; i < len(words); {
		w := words[i]
		if w >= 0xD800 && w <= 0xDBFF {
			// High surrogate — must be followed by a low surrogate.
			if i+1 < len(words) {
				if lo := words[i+1]; lo >= 0xDC00 && lo <= 0xDFFF {
					runes = append(runes, rune(0x10000+(rune(w-0xD800)<<10)+rune(lo-0xDC00)))
					i += 2
					continue
				}
			}
			runes = append(runes, '\uFFFD') // unpaired high surrogate
		} else if w >= 0xDC00 && w <= 0xDFFF {
			runes = append(runes, '\uFFFD') // unexpected low surrogate
		} else {
			runes = append(runes, rune(w))
		}
		i++
	}
	return string(runes)
}

// EncodeUCS2 converts a UTF-8 string to big-endian UCS-2/UTF-16 bytes.
// Characters above U+FFFF are encoded as surrogate pairs so that emoji
// round-trip correctly through UCS2 TP-DATA.
func EncodeUCS2(text string) []byte {
	// Pre-calculate output size: BMP runes = 2 bytes, SMP runes = 4 bytes.
	size := 0
	for _, r := range text {
		if r < 0x10000 {
			size += 2
		} else {
			size += 4
		}
	}
	out := make([]byte, 0, size)
	for _, r := range text {
		if r < 0x10000 {
			out = append(out, byte(r>>8), byte(r))
		} else {
			// Encode as UTF-16 surrogate pair.
			r -= 0x10000
			hi := uint16(0xD800 + (r >> 10))
			lo := uint16(0xDC00 + (r & 0x3FF))
			out = append(out, byte(hi>>8), byte(hi), byte(lo>>8), byte(lo))
		}
	}
	return out
}
