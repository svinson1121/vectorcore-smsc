package diameter

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	dcodec "github.com/svinson1121/vectorcore-smsc/internal/diameter/codec"
)

func TestSendCERAdvertisesAllConfiguredAppIDs(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	p := NewPeer(Config{
		LocalFQDN:  "smsc.example.net",
		LocalRealm: "example.net",
		AppIDs:     []uint32{dcodec.App3GPP_Sh, dcodec.App3GPP_SGd, dcodec.App3GPP_S6c},
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- p.sendCER(client)
	}()

	msg, err := dcodec.DecodeMessage(server)
	if err != nil {
		t.Fatalf("decode CER: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("send CER: %v", err)
	}

	vsaids := msg.FindAVPs(dcodec.CodeVendorSpecificApplicationID, 0)
	if got := len(vsaids); got != 3 {
		t.Fatalf("expected 3 Vendor-Specific-Application-Id AVPs, got %d", got)
	}

	want := map[uint32]bool{
		dcodec.App3GPP_S6c: false,
		dcodec.App3GPP_Sh:  false,
		dcodec.App3GPP_SGd: false,
	}
	for _, vsaid := range vsaids {
		grouped, err := dcodec.DecodeGrouped(vsaid)
		if err != nil {
			t.Fatalf("decode grouped AVP: %v", err)
		}
		var found bool
		for _, avp := range grouped {
			if avp.Code != dcodec.CodeAuthApplicationID || avp.VendorID != 0 {
				continue
			}
			appID, err := avp.Uint32()
			if err != nil {
				t.Fatalf("decode Auth-Application-Id: %v", err)
			}
			if _, ok := want[appID]; ok {
				want[appID] = true
			}
			found = true
		}
		if !found {
			t.Fatal("Vendor-Specific-Application-Id missing Auth-Application-Id")
		}
	}
	for appID, seen := range want {
		if !seen {
			t.Fatalf("expected app ID %d to be advertised", appID)
		}
	}
}

func TestWriteFullHandlesShortWrites(t *testing.T) {
	conn := &shortWriteConn{maxChunk: 5}
	payload := []byte("abcdefghijklmnopqrstuvwxyz")

	if err := writeFull(conn, payload); err != nil {
		t.Fatalf("writeFull: %v", err)
	}
	if got := conn.buf.Bytes(); !bytes.Equal(got, payload) {
		t.Fatalf("payload mismatch: got %q want %q", string(got), string(payload))
	}
}

func TestDiameterCommandName(t *testing.T) {
	if got, want := diameterCommandName(dcodec.CmdSendRoutingInfoSM), "SRI-SM"; got != want {
		t.Fatalf("diameterCommandName(SRI-SM) = %q, want %q", got, want)
	}
	if got, want := diameterCommandName(dcodec.CmdReportSMDeliveryStatus), "RSDS"; got != want {
		t.Fatalf("diameterCommandName(RSDS) = %q, want %q", got, want)
	}
}

func TestDiameterResultCode(t *testing.T) {
	msg := &dcodec.Message{
		AVPs: []*dcodec.AVP{
			dcodec.NewUint32(dcodec.CodeResultCode, 0, dcodec.FlagMandatory, dcodec.DiameterSuccess),
		},
	}
	if got, ok := diameterResultCode(msg); !ok || got != dcodec.DiameterSuccess {
		t.Fatalf("diameterResultCode() = (%d, %v), want (%d, true)", got, ok, dcodec.DiameterSuccess)
	}
}

func TestDiameterResultCodeExperimentalResult(t *testing.T) {
	exp, err := dcodec.NewGrouped(
		dcodec.CodeExperimentalResult, 0, dcodec.FlagMandatory,
		[]*dcodec.AVP{
			dcodec.NewUint32(dcodec.CodeVendorID, 0, dcodec.FlagMandatory, dcodec.Vendor3GPP),
			dcodec.NewUint32(dcodec.CodeExperimentalResultCode, 0, dcodec.FlagMandatory, 5001),
		},
	)
	if err != nil {
		t.Fatalf("NewGrouped() error = %v", err)
	}
	msg := &dcodec.Message{AVPs: []*dcodec.AVP{exp}}
	if got, ok := diameterResultCode(msg); !ok || got != 5001 {
		t.Fatalf("diameterResultCode() = (%d, %v), want (5001, true)", got, ok)
	}
}

type shortWriteConn struct {
	buf      bytes.Buffer
	maxChunk int
}

func (c *shortWriteConn) Read(_ []byte) (int, error)       { return 0, io.EOF }
func (c *shortWriteConn) Close() error                     { return nil }
func (c *shortWriteConn) LocalAddr() net.Addr              { return dummyAddr("local") }
func (c *shortWriteConn) RemoteAddr() net.Addr             { return dummyAddr("remote") }
func (c *shortWriteConn) SetDeadline(time.Time) error      { return nil }
func (c *shortWriteConn) SetReadDeadline(time.Time) error  { return nil }
func (c *shortWriteConn) SetWriteDeadline(time.Time) error { return nil }
func (c *shortWriteConn) Write(p []byte) (int, error) {
	n := len(p)
	if c.maxChunk > 0 && n > c.maxChunk {
		n = c.maxChunk
	}
	return c.buf.Write(p[:n])
}

type dummyAddr string

func (a dummyAddr) Network() string { return "test" }
func (a dummyAddr) String() string  { return string(a) }
