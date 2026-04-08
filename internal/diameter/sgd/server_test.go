package sgd

import (
	"testing"

	"github.com/svinson1121/vectorcore-smsc/internal/diameter"
	dcodec "github.com/svinson1121/vectorcore-smsc/internal/diameter/codec"
)

func TestSelectPeerForMMELockedPrefersDirectMMEPeer(t *testing.T) {
	s := NewServer("tcp", ":0", "smsc.example.net", "example.net", nil)
	direct := diameter.NewPeer(diameter.Config{Name: "mme-direct", Application: "sgd"})
	proxy := diameter.NewPeer(diameter.Config{Name: "dra01", Application: "sgd"})
	proxy.RemoteFQDN = "dra01.example.net"

	s.peers["mme01.example.net"] = direct
	s.peers[proxy.RemoteFQDN] = proxy
	s.outboundPeers["dra01"] = proxy

	got, viaProxy := s.selectPeerForMMELocked("mme01.example.net")
	if got != direct {
		t.Fatalf("selected peer = %p, want direct %p", got, direct)
	}
	if viaProxy {
		t.Fatal("viaProxy = true, want false")
	}
}

func TestSelectPeerForMMELockedFallsBackToActiveOutboundProxy(t *testing.T) {
	s := NewServer("tcp", ":0", "smsc.example.net", "example.net", nil)
	proxy := diameter.NewPeer(diameter.Config{Name: "dra01", Application: "sgd"})
	proxy.RemoteFQDN = "dra01.example.net"
	proxy.RemoteRealm = "example.net"

	s.outboundPeers["dra01"] = proxy
	s.peers[proxy.RemoteFQDN] = proxy

	got, viaProxy := s.selectPeerForMMELocked("mme01.example.net")
	if got != proxy {
		t.Fatalf("selected peer = %p, want proxy %p", got, proxy)
	}
	if !viaProxy {
		t.Fatal("viaProxy = false, want true")
	}
	if !s.HasPeerForMME("mme01.example.net") {
		t.Fatal("HasPeerForMME() = false, want true via proxy fallback")
	}
}

func TestDestinationRealmForPeerFallsBackToMMEHostRealm(t *testing.T) {
	if got, want := destinationRealmForPeer("mme01.epc.example.net", nil), "epc.example.net"; got != want {
		t.Fatalf("destinationRealmForPeer() = %q, want %q", got, want)
	}
}

func TestRoutePeerForMMEReturnsProxyDetails(t *testing.T) {
	s := NewServer("tcp", ":0", "smsc.example.net", "example.net", nil)
	proxy := diameter.NewPeer(diameter.Config{Name: "dra01", Application: "sgd"})
	proxy.RemoteFQDN = "dra01.example.net"

	s.outboundPeers["dra01"] = proxy
	s.peers[proxy.RemoteFQDN] = proxy

	peerName, viaProxy, ok := s.RoutePeerForMME("mme01.example.net")
	if !ok {
		t.Fatal("RoutePeerForMME() ok = false, want true")
	}
	if got, want := peerName, "dra01.example.net"; got != want {
		t.Fatalf("peerName = %q, want %q", got, want)
	}
	if !viaProxy {
		t.Fatal("viaProxy = false, want true")
	}
}

func TestDispatchRoutesOFAAnswerToPendingSender(t *testing.T) {
	s := NewServer("tcp", ":0", "smsc.example.net", "example.net", nil)
	replyCh := make(chan *dcodec.Message, 1)
	hopByHop := uint32(12345)
	s.trackPendingOFR(hopByHop, replyCh)
	defer s.untrackPendingOFR(hopByHop)

	ans := &dcodec.Message{
		Header: dcodec.Header{
			Flags:       dcodec.FlagProxiable,
			CommandCode: dcodec.CmdMTForwardShortMessage,
			AppID:       dcodec.App3GPP_SGd,
			HopByHop:    hopByHop,
		},
		AVPs: []*dcodec.AVP{
			dcodec.NewUint32(dcodec.CodeResultCode, 0, dcodec.FlagMandatory, dcodec.DiameterAVPUnsupported),
		},
	}

	s.dispatch(nil, ans)

	select {
	case got := <-replyCh:
		if got != ans {
			t.Fatal("pending OFA mismatch")
		}
	default:
		t.Fatal("pending OFA was not delivered to sender")
	}
}
