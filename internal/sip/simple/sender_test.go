package simple

import (
	"testing"

	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func TestSipPeerURIIncludesTransport(t *testing.T) {
	peer := store.SIPPeer{
		Address:   "peer.example.net",
		Port:      5061,
		Transport: "tcp",
	}

	got := sipPeerURI("+15551234567", peer)
	want := "sip:+15551234567@peer.example.net:5061;transport=tcp"
	if got != want {
		t.Fatalf("unexpected peer URI: got %q want %q", got, want)
	}
}
