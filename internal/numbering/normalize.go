package numbering

import "strings"

const (
	TONUnknown       byte = 0x00
	TONInternational byte = 0x01
	TONNational      byte = 0x02
)

func NormalizeRecipientMSISDN(digits string, ton byte, defaultCountryCode string, localNationalLength int) string {
	n := strings.TrimSpace(digits)
	n = strings.TrimPrefix(n, "+")
	cc := strings.TrimPrefix(strings.TrimSpace(defaultCountryCode), "+")

	if ton == TONInternational {
		return n
	}

	if ton == TONUnknown || ton == TONNational {
		if localNationalLength > 0 && len(n) == localNationalLength {
			return cc + n
		}

		if cc != "" && strings.HasPrefix(n, cc) {
			return n
		}
	}

	return n
}
