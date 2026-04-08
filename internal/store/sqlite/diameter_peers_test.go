package sqlite

import (
	"testing"
)

type stubSQLRow struct {
	values []any
}

func (r stubSQLRow) Scan(dest ...any) error {
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case *int:
			*d = r.values[i].(int)
		default:
			panic("unsupported scan type")
		}
	}
	return nil
}

func TestScanDiameterPeerFallsBackToLegacyApplication(t *testing.T) {
	row := stubSQLRow{values: []any{
		"id-1",
		"hss01",
		"10.0.0.10",
		"example.com",
		3868,
		"tcp",
		"sh",
		"[]",
		1,
		"2026-04-01T00:00:00Z",
		"2026-04-01T00:00:00Z",
	}}

	peer, err := scanDiameterPeer(row)
	if err != nil {
		t.Fatalf("scanDiameterPeer: %v", err)
	}
	if got, want := len(peer.Applications), 1; got != want {
		t.Fatalf("applications len = %d, want %d", got, want)
	}
	if got, want := peer.Applications[0], "sh"; got != want {
		t.Fatalf("applications[0] = %q, want %q", got, want)
	}
}
