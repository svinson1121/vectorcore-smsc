package dr

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/smpp"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

type captureConn struct {
	writes chan *smpp.PDU
}

func (c *captureConn) Read(_ []byte) (int, error) { return 0, nil }
func (c *captureConn) Write(b []byte) (int, error) {
	pdu, err := smpp.Decode(bytes.NewReader(b))
	if err != nil {
		return 0, err
	}
	c.writes <- pdu
	return len(b), nil
}
func (c *captureConn) Close() error                     { return nil }
func (c *captureConn) LocalAddr() net.Addr              { return dummyAddr("local") }
func (c *captureConn) RemoteAddr() net.Addr             { return dummyAddr("remote") }
func (c *captureConn) SetDeadline(time.Time) error      { return nil }
func (c *captureConn) SetReadDeadline(time.Time) error  { return nil }
func (c *captureConn) SetWriteDeadline(time.Time) error { return nil }

type dummyAddr string

func (d dummyAddr) Network() string { return "tcp" }
func (d dummyAddr) String() string  { return string(d) }

func TestReportSMPPServerLinkDoesNotWaitForResponse(t *testing.T) {
	conn := &captureConn{writes: make(chan *smpp.PDU, 1)}
	link := smpp.NewLink("peer-a", "peer-a", "transceiver", "server", "tcp", "remote", smpp.NewConn(conn), smpp.StateBound)
	reg := smpp.NewRegistry()
	reg.Add(link)

	c := New(nil, nil, reg, "")
	msg := store.Message{
		ID:          "msg-1",
		SMPPMsgID:   "smpp-1",
		OriginIface: "smpp",
		OriginPeer:  "peer-a",
		SrcMSISDN:   "15551230001",
		DstMSISDN:   "15551230002",
	}

	done := make(chan struct{})
	go func() {
		c.reportSMPP(context.Background(), msg, StatusDelivered)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("reportSMPP blocked waiting for response")
	}

	select {
	case pdu := <-conn.writes:
		if pdu.CommandID != smpp.CmdDeliverSM {
			t.Fatalf("unexpected PDU command: 0x%08x", pdu.CommandID)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected DR PDU to be written")
	}
}
