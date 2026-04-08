package main

import (
	"testing"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/diameter"
	dcodec "github.com/svinson1121/vectorcore-smsc/internal/diameter/codec"
	"github.com/svinson1121/vectorcore-smsc/internal/diameter/sgd"
)

func TestAggregateDiameterRuntimePeersMergesSharedShAndSGdPeer(t *testing.T) {
	connectedAt := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	activeHSS := diameter.NewPeer(diameter.Config{
		Name:        "mme01",
		Application: "sh",
		AppIDs:      []uint32{dcodec.App3GPP_Sh, dcodec.App3GPP_SGd},
	})

	peers := aggregateDiameterRuntimePeers([]sgd.PeerStatus{{
		Name:        "mme01",
		Application: "sh",
		State:       "OPEN",
		ConnectedAt: &connectedAt,
	}}, []*diameter.Peer{activeHSS})

	if got, want := len(peers), 1; got != want {
		t.Fatalf("len(peers) = %d, want %d", got, want)
	}
	if got, want := peers[0].Name, "mme01"; got != want {
		t.Fatalf("peer name = %q, want %q", got, want)
	}
	if got, want := peers[0].Applications, []string{"sh", "sgd"}; !equalStrings(got, want) {
		t.Fatalf("applications = %v, want %v", got, want)
	}
}

func TestAggregateDiameterRuntimePeersCountsDistinctPeersSeparately(t *testing.T) {
	peers := aggregateDiameterRuntimePeers([]sgd.PeerStatus{
		{Name: "mme01", Application: "sgd", State: "OPEN"},
		{Name: "mme02", Application: "sgd", State: "OPEN"},
	}, nil)

	if got, want := len(peers), 2; got != want {
		t.Fatalf("len(peers) = %d, want %d", got, want)
	}
}

func TestAggregateDiameterRuntimePeersIncludesS6cFromSharedPeerAppIDs(t *testing.T) {
	activeHSS := diameter.NewPeer(diameter.Config{
		Name:        "mme01",
		Application: "sh",
		AppID:       dcodec.App3GPP_Sh,
		AppIDs:      []uint32{dcodec.App3GPP_Sh, dcodec.App3GPP_SGd, dcodec.App3GPP_S6c},
	})

	peers := aggregateDiameterRuntimePeers([]sgd.PeerStatus{{
		Name:        "mme01",
		Application: "sgd",
		State:       "OPEN",
	}}, []*diameter.Peer{activeHSS})

	if got, want := len(peers), 1; got != want {
		t.Fatalf("len(peers) = %d, want %d", got, want)
	}
	if got, want := peers[0].Applications, []string{"sh", "sgd", "s6c"}; !equalStrings(got, want) {
		t.Fatalf("applications = %v, want %v", got, want)
	}
}

func TestAggregateDiameterRuntimePeersIncludesS6cOnlyHSSPeer(t *testing.T) {
	activeHSS := diameter.NewPeer(diameter.Config{
		Name:        "hss01",
		Application: "s6c",
		AppID:       dcodec.App3GPP_S6c,
		AppIDs:      []uint32{dcodec.App3GPP_S6c},
	})

	peers := aggregateDiameterRuntimePeers(nil, []*diameter.Peer{activeHSS})

	if got, want := len(peers), 1; got != want {
		t.Fatalf("len(peers) = %d, want %d", got, want)
	}
	if got, want := peers[0].Applications, []string{"s6c"}; !equalStrings(got, want) {
		t.Fatalf("applications = %v, want %v", got, want)
	}
}

func TestAggregateDiameterRuntimePeersKeepsDistinctShAndS6cPeers(t *testing.T) {
	shPeer := diameter.NewPeer(diameter.Config{
		Name:        "hss-sh",
		Application: "sh",
		AppID:       dcodec.App3GPP_Sh,
		AppIDs:      []uint32{dcodec.App3GPP_Sh},
	})
	s6cPeer := diameter.NewPeer(diameter.Config{
		Name:        "hss-s6c",
		Application: "s6c",
		AppID:       dcodec.App3GPP_S6c,
		AppIDs:      []uint32{dcodec.App3GPP_S6c},
	})

	peers := aggregateDiameterRuntimePeers(nil, []*diameter.Peer{shPeer, s6cPeer})

	if got, want := len(peers), 2; got != want {
		t.Fatalf("len(peers) = %d, want %d", got, want)
	}
}

func TestDiameterRuntimePeerInfoFormatsMergedApplications(t *testing.T) {
	peerInfo := diameterRuntimePeerInfo([]diameterRuntimePeer{{
		Name:         "mme01",
		State:        "OPEN",
		Applications: []string{"sh", "sgd", "s6c"},
	}})

	if got, want := len(peerInfo), 1; got != want {
		t.Fatalf("len(peerInfo) = %d, want %d", got, want)
	}
	if got, want := peerInfo[0].Type, "diameter_peer"; got != want {
		t.Fatalf("peer type = %q, want %q", got, want)
	}
	if got, want := peerInfo[0].Application, "Sh SGd S6c"; got != want {
		t.Fatalf("application = %q, want %q", got, want)
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
