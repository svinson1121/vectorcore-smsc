package postgres

import (
	"testing"
	"time"
)

type stubScanRow struct {
	values []any
}

func (r stubScanRow) Scan(dest ...any) error {
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case *int:
			*d = r.values[i].(int)
		case *bool:
			*d = r.values[i].(bool)
		case *time.Time:
			*d = r.values[i].(time.Time)
		default:
			panic("unsupported scan type")
		}
	}
	return nil
}

func TestScanDiameterPeerFallsBackToLegacyApplication(t *testing.T) {
	now := time.Now().UTC().Round(time.Second)
	row := stubScanRow{values: []any{
		"id-1",
		"hss01",
		"10.0.0.10",
		"example.com",
		3868,
		"tcp",
		"sh",
		"[]",
		true,
		now,
		now,
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
