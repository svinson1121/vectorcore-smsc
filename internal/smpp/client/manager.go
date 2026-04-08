package client

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/smpp"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

const gracefulStopTimeout = 5 * time.Second

// Manager owns all outbound SMPP client sessions.
// It loads the initial client list from the database and hot-reloads on
// smpp_clients_changed NOTIFY.
type Manager struct {
	st    store.Store
	reg   *smpp.Registry
	onMsg OnMessageFunc
	tls   TLSOptions

	mu       sync.Mutex
	sessions map[string]*Session // keyed by client ID
}

// NewManager creates a Manager.
func NewManager(st store.Store, reg *smpp.Registry, onMsg OnMessageFunc, tlsOpts TLSOptions) *Manager {
	return &Manager{
		st:       st,
		reg:      reg,
		onMsg:    onMsg,
		tls:      tlsOpts,
		sessions: make(map[string]*Session),
	}
}

// SetOnMessage replaces the message callback after creation.
func (m *Manager) SetOnMessage(fn OnMessageFunc) {
	m.mu.Lock()
	m.onMsg = fn
	m.mu.Unlock()
}

// Start loads all enabled clients and starts their connect loops.
// It also subscribes to hot-reload notifications.
func (m *Manager) Start(ctx context.Context) error {
	clients, err := m.st.ListSMPPClients(ctx)
	if err != nil {
		return err
	}
	for _, c := range clients {
		if c.Enabled {
			m.startSession(ctx, c)
		}
	}

	ch := make(chan store.ChangeEvent, 16)
	if err := m.st.Subscribe(ctx, "smpp_clients", ch); err != nil {
		return err
	}
	go m.watchChanges(ctx, ch)

	slog.Info("smpp client manager started", "clients", len(clients))
	return nil
}

// SendViaPeer sends a PDU via the named outbound client (by Name or SystemID).
// Returns the response PDU or an error if no BOUND session exists.
func (m *Manager) SendViaPeer(name string, pdu *smpp.PDU) (*smpp.PDU, error) {
	link := m.reg.GetByName(name)
	if link == nil {
		return nil, &ErrNoPeer{Name: name}
	}
	return link.SendAndWait(pdu, sendTimeout)
}

func (m *Manager) HasPeer(name string) bool {
	return m.reg.GetByName(name) != nil
}

// watchChanges reloads client sessions whenever the smpp_clients table changes.
func (m *Manager) watchChanges(ctx context.Context, ch <-chan store.ChangeEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ch:
			if !ok {
				return
			}
			m.reload(ctx)
		}
	}
}

func (m *Manager) reload(ctx context.Context) {
	clients, err := m.st.ListSMPPClients(ctx)
	if err != nil {
		slog.Error("smpp client reload failed", "err", err)
		return
	}

	// Build desired state map
	desired := make(map[string]store.SMPPClient, len(clients))
	for _, c := range clients {
		if c.Enabled {
			desired[c.ID] = c
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop sessions that are no longer desired
	for id, sess := range m.sessions {
		if _, ok := desired[id]; !ok {
			slog.Info("smpp client stopping removed session", "id", id)
			sess.stopGraceful(gracefulStopTimeout)
			delete(m.sessions, id)
		}
	}

	// Restart sessions whose runtime configuration changed.
	for id, c := range desired {
		if sess, exists := m.sessions[id]; exists && !smppClientRuntimeEqual(sess.cfg, c) {
			slog.Info("smpp client restarting session after config change", "name", c.Name)
			sess.stopGraceful(gracefulStopTimeout)
			delete(m.sessions, id)
		}
	}

	// Start sessions for new clients
	for id, c := range desired {
		if _, exists := m.sessions[id]; !exists {
			slog.Info("smpp client starting new session", "name", c.Name)
			m.startSessionLocked(ctx, c)
		}
	}
}

func smppClientRuntimeEqual(a, b store.SMPPClient) bool {
	return a.ID == b.ID &&
		a.Name == b.Name &&
		a.Host == b.Host &&
		a.Port == b.Port &&
		a.Transport == b.Transport &&
		a.VerifyServerCert == b.VerifyServerCert &&
		a.SystemID == b.SystemID &&
		a.Password == b.Password &&
		a.BindType == b.BindType &&
		a.ReconnectInterval == b.ReconnectInterval &&
		a.ThroughputLimit == b.ThroughputLimit &&
		a.Enabled == b.Enabled
}

// startSession starts a new Session (acquires lock internally).
func (m *Manager) startSession(ctx context.Context, c store.SMPPClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startSessionLocked(ctx, c)
}

func (m *Manager) startSessionLocked(ctx context.Context, c store.SMPPClient) {
	sess := newSession(c, m.reg, m.onMsg, m.tls)
	m.sessions[c.ID] = sess
	sess.start(ctx)
}

// ErrNoPeer is returned when no BOUND link exists for the given peer name.
type ErrNoPeer struct {
	Name string
}

func (e *ErrNoPeer) Error() string {
	return "smpp: no bound peer " + e.Name
}
