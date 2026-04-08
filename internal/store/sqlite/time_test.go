package sqlite

import (
	"testing"
	"time"
)

func TestParseDBTimeAcceptsSQLiteDatetime(t *testing.T) {
	got := parseDBTime("2026-04-07 09:17:40")
	if got.IsZero() {
		t.Fatal("expected SQLite datetime to parse")
	}
	if got.Year() != 2026 || got.Month() != time.April || got.Day() != 7 {
		t.Fatalf("unexpected parsed date: %v", got)
	}
	if got.Hour() != 9 || got.Minute() != 17 || got.Second() != 40 {
		t.Fatalf("unexpected parsed time: %v", got)
	}
}
