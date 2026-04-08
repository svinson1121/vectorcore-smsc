package postgres

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestIntervalToDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   pgtype.Interval
		want    time.Duration
		wantErr bool
	}{
		{
			name:  "seconds",
			input: pgtype.Interval{Microseconds: 10 * int64(time.Second/time.Microsecond), Valid: true},
			want:  10 * time.Second,
		},
		{
			name:  "days and micros",
			input: pgtype.Interval{Days: 2, Microseconds: 90 * int64(time.Minute/time.Microsecond), Valid: true},
			want:  49*time.Hour + 30*time.Minute,
		},
		{
			name:    "months unsupported",
			input:   pgtype.Interval{Months: 1, Valid: true},
			wantErr: true,
		},
		{
			name:  "null interval",
			input: pgtype.Interval{},
			want:  0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := intervalToDuration(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFormatInterval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   time.Duration
		want string
	}{
		{name: "milliseconds", in: 1500 * time.Millisecond, want: "1500000 microseconds"},
		{name: "negative", in: -2 * time.Second, want: "-2000000 microseconds"},
		{name: "zero", in: 0, want: "0 microseconds"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := formatInterval(tc.in)
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
