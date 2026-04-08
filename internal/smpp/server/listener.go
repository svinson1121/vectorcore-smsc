package server

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/smpp"
)

// Listener accepts inbound SMPP connections and spawns a session per connection.
type Listener struct {
	listenAddr string
	auth       *Authenticator
	reg        *smpp.Registry
	cfg        SessionConfig
	onMsg      OnMessageFunc
	tlsConfig  *tls.Config
	transport  string

	listener net.Listener
	active   atomic.Int32 // count of live sessions
}

// NewListener creates a Listener.
func NewListener(listenAddr string, auth *Authenticator, reg *smpp.Registry,
	cfg SessionConfig, onMsg OnMessageFunc) *Listener {
	cfg.applyDefaults("tcp")
	return &Listener{
		listenAddr: listenAddr,
		auth:       auth,
		reg:        reg,
		cfg:        cfg,
		onMsg:      onMsg,
		transport:  cfg.Transport,
	}
}

// NewTLSListener creates a TLS-enabled Listener.
func NewTLSListener(listenAddr string, auth *Authenticator, reg *smpp.Registry,
	cfg SessionConfig, tlsCfg *tls.Config, onMsg OnMessageFunc) *Listener {
	cfg.applyDefaults("tls")
	return &Listener{
		listenAddr: listenAddr,
		auth:       auth,
		reg:        reg,
		cfg:        cfg,
		onMsg:      onMsg,
		tlsConfig:  tlsCfg,
		transport:  cfg.Transport,
	}
}

// ListenAndServe binds the TCP socket and starts the accept loop.
// It blocks until ctx is cancelled or a fatal accept error occurs.
func (l *Listener) ListenAndServe(ctx context.Context) error {
	transport := strings.ToLower(strings.TrimSpace(l.transport))
	if transport == "" {
		transport = "tcp"
	}

	var (
		ln  net.Listener
		err error
	)
	if transport == "tls" {
		ln, err = tls.Listen("tcp", l.listenAddr, l.tlsConfig)
	} else {
		ln, err = net.Listen("tcp", l.listenAddr)
	}
	if err != nil {
		return err
	}
	l.listener = ln
	slog.Info("smpp server listening", "addr", l.listenAddr, "transport", transport)

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil // clean shutdown
			}
			slog.Error("smpp accept error", "err", err)
			return err
		}

		if l.cfg.MaxConnections > 0 && int(l.active.Load()) >= l.cfg.MaxConnections {
			slog.Warn("smpp max connections reached, rejecting",
				"remote", conn.RemoteAddr(),
				"limit", l.cfg.MaxConnections,
			)
			conn.Close()
			continue
		}

		l.active.Add(1)
		s := newSession(conn, l.auth, l.reg, l.cfg, l.onMsg)
		go func() {
			defer l.active.Add(-1)
			s.run()
		}()
	}
}

// ReloadAccounts refreshes the auth cache from the database.
// Called by main on smpp_server_accounts_changed NOTIFY.
func (l *Listener) ReloadAccounts(ctx context.Context) {
	l.auth.Reload(ctx)
}

// ActiveSessions returns the number of currently bound sessions.
func (l *Listener) ActiveSessions() int {
	return int(l.active.Load())
}

const sendTimeout = 30 * time.Second

// SendMT delivers a PDU to an ESME and waits for the response.
func SendMT(link *smpp.Link, pdu *smpp.PDU) (*smpp.PDU, error) {
	return link.SendAndWait(pdu, sendTimeout)
}
