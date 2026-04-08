package server

import (
	"net"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/svinson1121/vectorcore-smsc/internal/codec"
	"github.com/svinson1121/vectorcore-smsc/internal/smpp"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func TestSessionContinuesReadingWhileSubmitHandlerBlocked(t *testing.T) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword: %v", err)
	}

	auth := &Authenticator{
		accounts: map[string]store.SMPPServerAccount{
			"esme-1": {
				SystemID:     "esme-1",
				PasswordHash: string(passwordHash),
				Enabled:      true,
			},
		},
	}
	reg := smpp.NewRegistry()

	firstHandlerStarted := make(chan struct{})
	releaseHandler := make(chan struct{})

	var mu sync.Mutex
	var seen []*codec.Message
	onMsg := func(msg *codec.Message, _ *smpp.Link, _ store.SMPPServerAccount) {
		mu.Lock()
		seen = append(seen, msg)
		count := len(seen)
		mu.Unlock()
		if count == 1 {
			close(firstHandlerStarted)
			<-releaseHandler
		}
	}

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	sess := newSession(serverConn, auth, reg, SessionConfig{}, onMsg)
	done := make(chan struct{})
	go func() {
		sess.run()
		close(done)
	}()

	sc := smpp.NewConn(clientConn)

	bindResp := writeAndReadPDU(t, clientConn, sc, &smpp.PDU{
		CommandID:        smpp.CmdBindTransceiver,
		SequenceNumber:   1,
		SystemID:         "esme-1",
		Password:         "secret",
		InterfaceVersion: 0x34,
	})
	if bindResp.CommandID != smpp.CmdBindTransceiverResp || bindResp.CommandStatus != smpp.ESME_ROK {
		t.Fatalf("unexpected bind response: %#v", bindResp)
	}

	firstResp := writeAndReadPDU(t, clientConn, sc, multipartSegmentPDU(2, 1))
	if firstResp.CommandID != smpp.CmdSubmitSMResp || firstResp.CommandStatus != smpp.ESME_ROK {
		t.Fatalf("unexpected first submit response: %#v", firstResp)
	}

	select {
	case <-firstHandlerStarted:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("first submit handler did not start")
	}

	secondResp := writeAndReadPDU(t, clientConn, sc, multipartSegmentPDU(3, 2))
	if secondResp.CommandID != smpp.CmdSubmitSMResp || secondResp.CommandStatus != smpp.ESME_ROK {
		t.Fatalf("unexpected second submit response: %#v", secondResp)
	}

	enquireResp := writeAndReadPDU(t, clientConn, sc, &smpp.PDU{
		CommandID:      smpp.CmdEnquireLink,
		SequenceNumber: 4,
	})
	if enquireResp.CommandID != smpp.CmdEnquireLinkResp || enquireResp.CommandStatus != smpp.ESME_ROK {
		t.Fatalf("unexpected enquire_link response: %#v", enquireResp)
	}

	close(releaseHandler)

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		mu.Lock()
		count := len(seen)
		mu.Unlock()
		if count == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected 2 messages dispatched, got %d", count)
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("session did not exit after client close")
	}
}

func multipartSegmentPDU(seq uint32, segment byte) *smpp.PDU {
	payload := []byte{0x06, 0x08, 0x04, 0x12, 0x34, 0x02, segment, 0xaa, segment}
	return &smpp.PDU{
		CommandID:       smpp.CmdSubmitSM,
		SequenceNumber:  seq,
		SourceAddrTON:   0x01,
		SourceAddrNPI:   0x01,
		SourceAddr:      "15551230001",
		DestAddrTON:     0x01,
		DestAddrNPI:     0x01,
		DestinationAddr: "15551230002",
		ESMClass:        smpp.ESMClassUDHI,
		DataCoding:      0x04,
		ShortMessage:    payload,
	}
}

func writeAndReadPDU(t *testing.T, conn net.Conn, sc *smpp.Conn, req *smpp.PDU) *smpp.PDU {
	t.Helper()
	if err := sc.WritePDU(req); err != nil {
		t.Fatalf("WritePDU: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	defer conn.SetReadDeadline(time.Time{})
	resp, err := sc.ReadPDU()
	if err != nil {
		t.Fatalf("ReadPDU: %v", err)
	}
	return resp
}
