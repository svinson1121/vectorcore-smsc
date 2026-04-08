// Package sip initialises the sipgo UA and shared listener used by both
// the ISC handler and the SIP SIMPLE handler.
package sip

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

// Server wraps the sipgo UserAgent and Server, providing a single shared listener.
type Server struct {
	UA  *sipgo.UserAgent
	Srv *sipgo.Server
}

// NewServer creates and configures the SIP UA and server.
func NewServer(transport, listen, identityHost string) (*Server, error) {
	host := strings.TrimSpace(identityHost)
	if host == "" {
		host = hostFromAddr(listen)
	}
	ua, err := sipgo.NewUA(
		sipgo.WithUserAgentHostname(host),
	)
	if err != nil {
		return nil, fmt.Errorf("create SIP UA: %w", err)
	}

	srv, err := sipgo.NewServer(ua)
	if err != nil {
		return nil, fmt.Errorf("create SIP server: %w", err)
	}

	return &Server{UA: ua, Srv: srv}, nil
}

// ListenAndServe starts the SIP listener.  It blocks until ctx is cancelled.
func (s *Server) ListenAndServe(ctx context.Context, transport, listen string) error {
	slog.Info("SIP server listening", "transport", transport, "addr", listen)
	err := s.Srv.ListenAndServe(ctx, strings.ToUpper(transport), listen)
	if err != nil && ctx.Err() != nil {
		return nil // clean shutdown
	}
	return err
}

// NewClient creates a SIP client from the shared UA.
func (s *Server) NewClient() (*sipgo.Client, error) {
	client, err := sipgo.NewClient(s.UA)
	if err != nil {
		return nil, fmt.Errorf("create SIP client: %w", err)
	}
	return client, nil
}

// OnRequest registers a request handler on the underlying sipgo server.
func (s *Server) OnRequest(method sip.RequestMethod, handler func(*sip.Request, sip.ServerTransaction)) {
	s.Srv.OnRequest(method, handler)
}

// hostFromAddr returns only the host part of a "host:port" string.
func hostFromAddr(addr string) string {
	host := addr
	if h, _, err := net.SplitHostPort(addr); err == nil {
		host = h
	}
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	switch host {
	case "", "::", "0.0.0.0", "*":
		return "localhost"
	default:
		return host
	}
}

// DomainFromAddr returns the host part of a "host:port" address for use as a SIP domain.
func DomainFromAddr(addr string) string {
	return hostFromAddr(addr)
}
