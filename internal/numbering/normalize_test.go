package numbering

import "testing"

func TestNormalizeRecipientMSISDN(t *testing.T) {
	tests := []struct {
		name                string
		digits              string
		ton                 byte
		defaultCountryCode  string
		localNationalLength int
		want                string
	}{
		{
			name:                "international strips leading plus",
			digits:              "+16752012860",
			ton:                 TONInternational,
			defaultCountryCode:  "1",
			localNationalLength: 10,
			want:                "16752012860",
		},
		{
			name:                "international preserves e164 digits",
			digits:              "16752012860",
			ton:                 TONInternational,
			defaultCountryCode:  "1",
			localNationalLength: 10,
			want:                "16752012860",
		},
		{
			name:                "unknown NANP national becomes canonical",
			digits:              "6752012860",
			ton:                 TONUnknown,
			defaultCountryCode:  "1",
			localNationalLength: 10,
			want:                "16752012860",
		},
		{
			name:                "national NANP national becomes canonical",
			digits:              "6752012860",
			ton:                 TONNational,
			defaultCountryCode:  "1",
			localNationalLength: 10,
			want:                "16752012860",
		},
		{
			name:                "unknown NANP e164 digits stay canonical",
			digits:              "16752012860",
			ton:                 TONUnknown,
			defaultCountryCode:  "1",
			localNationalLength: 10,
			want:                "16752012860",
		},
		{
			name:                "unknown French national becomes canonical",
			digits:              "612345678",
			ton:                 TONUnknown,
			defaultCountryCode:  "33",
			localNationalLength: 9,
			want:                "33612345678",
		},
		{
			name:                "international French e164 digits stay canonical",
			digits:              "33612345678",
			ton:                 TONInternational,
			defaultCountryCode:  "33",
			localNationalLength: 9,
			want:                "33612345678",
		},
		{
			name:                "unknown non French national length stays unchanged",
			digits:              "6752012860",
			ton:                 TONUnknown,
			defaultCountryCode:  "33",
			localNationalLength: 9,
			want:                "6752012860",
		},
		{
			name:                "eleven digit NANP never strips leading one",
			digits:              "16752012860",
			ton:                 TONUnknown,
			defaultCountryCode:  "1",
			localNationalLength: 10,
			want:                "16752012860",
		},
		{
			name:                "eleven digit NANP never prepends country code",
			digits:              "16752012860",
			ton:                 TONNational,
			defaultCountryCode:  "1",
			localNationalLength: 10,
			want:                "16752012860",
		},
		{
			name:                "config country code strips leading plus",
			digits:              "6752012860",
			ton:                 TONUnknown,
			defaultCountryCode:  "+1",
			localNationalLength: 10,
			want:                "16752012860",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeRecipientMSISDN(tt.digits, tt.ton, tt.defaultCountryCode, tt.localNationalLength)
			if got != tt.want {
				t.Fatalf("NormalizeRecipientMSISDN(%q, %d, %q, %d) = %q, want %q",
					tt.digits, tt.ton, tt.defaultCountryCode, tt.localNationalLength, got, tt.want)
			}
		})
	}
}
