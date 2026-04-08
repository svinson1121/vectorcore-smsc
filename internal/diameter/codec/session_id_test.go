package codec

import "testing"

func TestNewSessionIDIsUniqueAcrossCalls(t *testing.T) {
	first := NewSessionID("smsc.example.net")
	second := NewSessionID("smsc.example.net")
	if first == second {
		t.Fatalf("NewSessionID() collision: %q == %q", first, second)
	}
}

func TestNewSessionIDIncludesOriginHost(t *testing.T) {
	id := NewSessionID("smsc.example.net")
	if len(id) == 0 || id[:len("smsc.example.net")] != "smsc.example.net" {
		t.Fatalf("session ID %q does not start with origin host", id)
	}
}
