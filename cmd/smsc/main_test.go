package main

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/svinson1121/vectorcore-smsc/internal/config"
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

func TestBuildInboundSMPPTLSConfigRequiresKeypair(t *testing.T) {
	cfg := &config.Config{}
	cfg.SMPP.ServerTLS.Address = "127.0.0.1"
	cfg.SMPP.ServerTLS.Port = 3550
	cfg.SMPP.ServerTLS.Listen = "127.0.0.1:3550"

	if _, err := buildInboundSMPPTLSConfig(cfg); err == nil {
		t.Fatal("expected error when TLS listener is enabled without keypair")
	}
}

func TestBuildInboundSMPPTLSConfigLoadsClientVerification(t *testing.T) {
	cfg := &config.Config{}
	cfg.SMPP.ServerTLS.Address = "127.0.0.1"
	cfg.SMPP.ServerTLS.Port = 3550
	cfg.SMPP.ServerTLS.Listen = "127.0.0.1:3550"
	cfg.SMPP.ServerTLS.VerifyClientCert = true
	cfg.SMPP.ServerTLS.ClientCAFile = testCertPath(t, "ca.crt")
	cfg.SMPP.ServerTLS.CertFile = testCertPath(t, "server.crt")
	cfg.SMPP.ServerTLS.KeyFile = testCertPath(t, "server.key")

	tlsCfg, err := buildInboundSMPPTLSConfig(cfg)
	if err != nil {
		t.Fatalf("buildInboundSMPPTLSConfig() error = %v", err)
	}
	if tlsCfg == nil {
		t.Fatal("expected TLS config")
	}
	if tlsCfg.ClientAuth == 0 {
		t.Fatal("expected client certificate verification mode to be configured")
	}
	if tlsCfg.ClientCAs == nil {
		t.Fatal("expected client CA pool to be loaded")
	}
	if len(tlsCfg.Certificates) != 1 {
		t.Fatalf("expected 1 server certificate, got %d", len(tlsCfg.Certificates))
	}
}

func testCertPath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "test-certs", name)
}
