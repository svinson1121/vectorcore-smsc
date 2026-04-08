package main

import (
	"testing"

	"github.com/svinson1121/vectorcore-smsc/internal/diameter"
	dcodec "github.com/svinson1121/vectorcore-smsc/internal/diameter/codec"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func TestShouldReplaceHSSPeerWhenApplicationsChange(t *testing.T) {
	current := diameter.NewPeer(diameter.Config{
		Name:        "hss01",
		Host:        "10.0.0.10",
		Port:        3868,
		Transport:   "tcp",
		Application: "sh",
		AppID:       dcodec.App3GPP_Sh,
		AppIDs:      []uint32{dcodec.App3GPP_Sh},
		PeerRealm:   "example.com",
	})

	want := &store.DiameterPeer{
		Name:         "hss01",
		Host:         "10.0.0.10",
		Realm:        "example.com",
		Port:         3868,
		Transport:    "tcp",
		Applications: []string{"sh", "s6c"},
		Enabled:      true,
	}

	if !shouldReplaceHSSPeer(current, false, want, false) {
		t.Fatal("expected HSS peer replacement when application set changes")
	}
}
