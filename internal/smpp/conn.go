package smpp

import (
	"net"
	"sync"
	"sync/atomic"
)

// Conn wraps a net.Conn with PDU framing and per-connection sequence numbering.
// WritePDU is safe for concurrent use; ReadPDU must be called from a single goroutine.
type Conn struct {
	conn net.Conn
	seq  uint32
	wmu  sync.Mutex
}

func NewConn(c net.Conn) *Conn {
	return &Conn{conn: c}
}

// ReadPDU reads exactly one PDU from the underlying connection.
func (c *Conn) ReadPDU() (*PDU, error) {
	return Decode(c.conn)
}

// WritePDU encodes and writes a PDU. Thread-safe.
func (c *Conn) WritePDU(pdu *PDU) error {
	b, err := Encode(pdu)
	if err != nil {
		return err
	}
	c.wmu.Lock()
	defer c.wmu.Unlock()
	_, err = c.conn.Write(b)
	return err
}

// NextSeq returns the next monotonically increasing sequence number.
func (c *Conn) NextSeq() uint32 {
	return atomic.AddUint32(&c.seq, 1)
}

// Close closes the underlying connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// RemoteAddr returns the remote network address.
func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}
