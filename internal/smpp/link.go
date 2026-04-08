package smpp

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrTimeout is returned by SendAndWait when no response arrives in time.
var ErrTimeout = errors.New("smpp: request timeout")

// LinkState represents the lifecycle state of a link.
type LinkState int32

const (
	StateDisconnected LinkState = 0
	StateConnecting   LinkState = 1
	StateBound        LinkState = 2
)

func (s LinkState) String() string {
	switch s {
	case StateDisconnected:
		return "DISCONNECTED"
	case StateConnecting:
		return "CONNECTING"
	case StateBound:
		return "BOUND"
	default:
		return "UNKNOWN"
	}
}

// Link represents a single live SMPP connection (server-side or client-side).
type Link struct {
	Name        string // friendly name (client links only; empty for server links)
	SystemID    string
	BindType    string // "transceiver" | "transmitter" | "receiver"
	Mode        string // "client" | "server"
	Transport   string
	RemoteAddr  string
	ConnectedAt time.Time

	mu    sync.Mutex
	state LinkState
	conn  *Conn

	pendMu  sync.Mutex
	pending map[uint32]chan *PDU
}

func NewLink(name, systemID, bindType, mode, transport, remoteAddr string, conn *Conn, state LinkState) *Link {
	return &Link{
		Name:        name,
		SystemID:    systemID,
		BindType:    bindType,
		Mode:        mode,
		Transport:   transport,
		RemoteAddr:  remoteAddr,
		ConnectedAt: time.Now(),
		conn:        conn,
		state:       state,
		pending:     make(map[uint32]chan *PDU),
	}
}

func (l *Link) State() LinkState {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.state
}

func (l *Link) SetState(s LinkState) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.state = s
}

// GetConn returns the underlying SMPP connection.
func (l *Link) GetConn() *Conn {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.conn
}

// Close forcibly closes the underlying connection, causing the read loop to exit.
func (l *Link) Close() {
	l.mu.Lock()
	conn := l.conn
	l.mu.Unlock()
	if conn != nil {
		conn.Close() //nolint:errcheck
	}
}

// SendAndWait writes pdu to this link (assigning a fresh sequence number) and
// blocks until the matching response arrives or timeout elapses.
func (l *Link) SendAndWait(pdu *PDU, timeout time.Duration) (*PDU, error) {
	conn := l.GetConn()
	if conn == nil {
		return nil, fmt.Errorf("smpp: link not connected")
	}

	seq := conn.NextSeq()
	pdu.SequenceNumber = seq

	ch := make(chan *PDU, 1)
	l.pendMu.Lock()
	l.pending[seq] = ch
	l.pendMu.Unlock()

	defer func() {
		l.pendMu.Lock()
		delete(l.pending, seq)
		l.pendMu.Unlock()
	}()

	if err := conn.WritePDU(pdu); err != nil {
		return nil, err
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(timeout):
		return nil, ErrTimeout
	}
}

// Send writes pdu to this link after assigning a fresh sequence number.
// It does not wait for a response.
func (l *Link) Send(pdu *PDU) error {
	conn := l.GetConn()
	if conn == nil {
		return fmt.Errorf("smpp: link not connected")
	}
	pdu.SequenceNumber = conn.NextSeq()
	return conn.WritePDU(pdu)
}

// DispatchPending delivers pdu to the pending SendAndWait waiter for its
// sequence number. Returns true if the PDU was consumed.
func (l *Link) DispatchPending(pdu *PDU) bool {
	l.pendMu.Lock()
	ch, ok := l.pending[pdu.SequenceNumber]
	if ok {
		delete(l.pending, pdu.SequenceNumber)
	}
	l.pendMu.Unlock()
	if ok {
		ch <- pdu
	}
	return ok
}

// ---- Registry ----

// Registry holds all active links keyed by SystemID, with optional name aliases
// for client links.  Thread-safe.
type Registry struct {
	mu    sync.RWMutex
	links map[string][]*Link // key: SystemID
	names map[string]string  // connection name → SystemID (client links)
}

func NewRegistry() *Registry {
	return &Registry{
		links: make(map[string][]*Link),
		names: make(map[string]string),
	}
}

// Add registers a link.
func (r *Registry) Add(l *Link) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.links[l.SystemID] = append(r.links[l.SystemID], l)
	if l.Name != "" && l.Name != l.SystemID {
		r.names[l.Name] = l.SystemID
	}
}

// Remove deregisters a link.
func (r *Registry) Remove(l *Link) {
	r.mu.Lock()
	defer r.mu.Unlock()
	list := r.links[l.SystemID]
	for i, c := range list {
		if c == l {
			r.links[l.SystemID] = append(list[:i], list[i+1:]...)
			break
		}
	}
	if l.Name != "" && l.Name != l.SystemID {
		remaining := r.links[l.SystemID]
		for _, other := range remaining {
			if other.Name == l.Name {
				return
			}
		}
		delete(r.names, l.Name)
	}
}

// GetByName returns the first BOUND link matching key (SystemID or name alias).
func (r *Registry) GetByName(key string) *Link {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if l := firstBound(r.links[key]); l != nil {
		return l
	}
	if sysID, ok := r.names[key]; ok {
		return firstBound(r.links[sysID])
	}
	return nil
}

// GetBySystemID returns all links (any state) for the given system ID.
func (r *Registry) GetBySystemID(systemID string) []*Link {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := r.links[systemID]
	out := make([]*Link, len(list))
	copy(out, list)
	return out
}

// All returns a snapshot of every registered link.
func (r *Registry) All() []*Link {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []*Link
	for _, list := range r.links {
		all = append(all, list...)
	}
	return all
}

// CloseAllForSystemID closes every active link for the given systemID.
func (r *Registry) CloseAllForSystemID(systemID string) {
	r.mu.RLock()
	links := append([]*Link{}, r.links[systemID]...)
	r.mu.RUnlock()
	for _, l := range links {
		l.Close()
	}
}

func firstBound(list []*Link) *Link {
	for _, l := range list {
		if l.State() == StateBound {
			return l
		}
	}
	return nil
}
