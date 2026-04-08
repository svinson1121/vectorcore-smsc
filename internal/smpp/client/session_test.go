package client

import (
	"context"
	"net"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/codec"
	"github.com/svinson1121/vectorcore-smsc/internal/smpp"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func TestSessionClientTLSConfigLoadsCAAndServerName(t *testing.T) {
	sess := newSession(store.SMPPClient{
		Host:             "localhost",
		Transport:        "tls",
		VerifyServerCert: true,
	}, nil, nil, TLSOptions{
		OutboundCAFile: testCertPath(t, "ca.crt"),
	})

	tlsCfg, err := sess.clientTLSConfig()
	if err != nil {
		t.Fatalf("clientTLSConfig() error = %v", err)
	}
	if tlsCfg.InsecureSkipVerify {
		t.Fatal("expected server verification to remain enabled")
	}
	if tlsCfg.ServerName != "localhost" {
		t.Fatalf("expected ServerName localhost, got %q", tlsCfg.ServerName)
	}
	if tlsCfg.RootCAs == nil {
		t.Fatal("expected RootCAs to be populated")
	}
}

func TestSessionClientTLSConfigDoesNotSetServerNameForIP(t *testing.T) {
	sess := newSession(store.SMPPClient{
		Host:             "127.0.0.1",
		Transport:        "tls",
		VerifyServerCert: true,
	}, nil, nil, TLSOptions{})

	tlsCfg, err := sess.clientTLSConfig()
	if err != nil {
		t.Fatalf("clientTLSConfig() error = %v", err)
	}
	if tlsCfg.ServerName != "" {
		t.Fatalf("expected empty ServerName for IP host, got %q", tlsCfg.ServerName)
	}
}

func TestSessionReadLoopRespondsToSubmitSM(t *testing.T) {
	serverConn, peerConn := net.Pipe()
	defer peerConn.Close()

	sess := newSession(store.SMPPClient{
		Name:     "smpp_router",
		SystemID: "smpp_router",
	}, smpp.NewRegistry(), func(msg *codec.Message, _ *smpp.Link, clientName string) {
		if clientName != "smpp_router" {
			t.Errorf("unexpected client name %q", clientName)
		}
		if msg.Source.MSISDN != "15552221234" {
			t.Errorf("unexpected source %q", msg.Source.MSISDN)
		}
		if msg.Destination.MSISDN != "3342012832" {
			t.Errorf("unexpected destination %q", msg.Destination.MSISDN)
		}
		if msg.SMPPMsgID == "" {
			t.Error("expected SMPP message id to be set")
		}
	}, TLSOptions{})

	conn := smpp.NewConn(serverConn)
	link := smpp.NewLink("smpp_router", "smpp_router", "transceiver", "client", "tcp", "10.90.250.186:2775", conn, smpp.StateBound)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- sess.readLoop(ctx, conn, link)
	}()

	peer := smpp.NewConn(peerConn)
	req := &smpp.PDU{
		CommandID:       smpp.CmdSubmitSM,
		SequenceNumber:  7,
		SourceAddr:      "15552221234",
		DestinationAddr: "3342012832",
		DataCoding:      0x08,
		ShortMessage:    []byte{0x00, 0x53, 0x00, 0x4d},
	}
	if err := peer.WritePDU(req); err != nil {
		t.Fatalf("WritePDU: %v", err)
	}

	if err := peerConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	resp, err := peer.ReadPDU()
	if err != nil {
		t.Fatalf("ReadPDU: %v", err)
	}
	if resp.CommandID != smpp.CmdSubmitSMResp {
		t.Fatalf("expected submit_sm_resp, got %#x", resp.CommandID)
	}
	if resp.CommandStatus != smpp.ESME_ROK {
		t.Fatalf("expected ESME_ROK, got %#x", resp.CommandStatus)
	}
	if resp.SequenceNumber != req.SequenceNumber {
		t.Fatalf("expected sequence %d, got %d", req.SequenceNumber, resp.SequenceNumber)
	}
	if resp.MessageID == "" {
		t.Fatal("expected message_id in submit_sm_resp")
	}
	if err := peerConn.SetReadDeadline(time.Time{}); err != nil {
		t.Fatalf("clear read deadline: %v", err)
	}

	cancel()
	serverConn.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("readLoop returned error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("readLoop did not exit")
	}
}

func testCertPath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "test-certs", name)
}
