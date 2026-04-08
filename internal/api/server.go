// Package api implements the VectorCore SMSC REST API and web UI server.
package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

// PeerInfo describes the live connection state of a peer.
type PeerInfo struct {
	Name        string     `json:"name"`
	Type        string     `json:"type"`  // smpp_server | smpp_client | diameter_peer
	State       string     `json:"state"` // BOUND | DISCONNECTED | CONNECTING | OPEN | CLOSED | etc.
	SystemID    string     `json:"system_id,omitempty"`
	BindType    string     `json:"bind_type,omitempty"`
	RemoteAddr  string     `json:"remote_addr,omitempty"`
	Application string     `json:"application,omitempty"`
	ConnectedAt *time.Time `json:"connected_at,omitempty"`
	ExpiryAt    *time.Time `json:"expiry_at,omitempty"`
}

// Server is the HTTP management API and web UI server.
type Server struct {
	st             store.Store
	version        string
	startAt        time.Time
	peerStatusFunc func() []PeerInfo
}

// New creates an API server.
func New(st store.Store, version string) *Server {
	return &Server{
		st:      st,
		version: version,
		startAt: time.Now(),
	}
}

// SetPeerStatusFunc registers the callback used by GET /api/v1/status/peers.
func (s *Server) SetPeerStatusFunc(fn func() []PeerInfo) {
	s.peerStatusFunc = fn
}

// Handler builds and returns the HTTP handler with all routes registered.
func (s *Server) Handler() http.Handler {
	mux := chi.NewRouter()
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.RealIP)
	mux.Use(requestLogger)

	// Huma OpenAPI at /api/v1/
	humaConfig := huma.DefaultConfig("VectorCore SMSC API", s.version)
	humaConfig.OpenAPIPath = "/api/v1/openapi.json"
	humaConfig.DocsPath = "/api/v1/docs"
	humaConfig.SchemasPath = "/api/v1/schemas"
	api := humachi.New(mux, humaConfig)

	// Register all resource groups
	registerSMPPAccounts(api, s.st)
	registerSMPPClients(api, s.st)
	registerSIPPeers(api, s.st)
	registerDiameterPeers(api, s.st)
	registerRoutingRules(api, s.st)
	registerSFPolicies(api, s.st)
	registerSubscribers(api, s.st)
	registerMessages(api, s.st)

	// OAM status endpoint
	type msgCounts struct {
		Queued     int64 `json:"queued"`
		Dispatched int64 `json:"dispatched"`
		Delivered  int64 `json:"delivered"`
		Failed     int64 `json:"failed"`
		Expired    int64 `json:"expired"`
	}
	type statusBody struct {
		Version       string    `json:"version"`
		Uptime        string    `json:"uptime"`
		UptimeSec     float64   `json:"uptime_sec"`
		MessageCounts msgCounts `json:"message_counts"`
	}
	huma.Register(api, huma.Operation{
		OperationID: "get-status",
		Method:      http.MethodGet,
		Path:        "/api/v1/status",
		Summary:     "Get system status",
		Tags:        []string{"OAM"},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body statusBody }, error) {
		up := time.Since(s.startAt)
		body := statusBody{
			Version:   s.version,
			Uptime:    up.Round(time.Second).String(),
			UptimeSec: up.Seconds(),
		}
		if counts, err := s.st.CountMessagesByStatus(ctx); err == nil {
			body.MessageCounts = msgCounts{
				Queued:     counts["QUEUED"],
				Dispatched: counts["DISPATCHED"],
				Delivered:  counts["DELIVERED"],
				Failed:     counts["FAILED"],
				Expired:    counts["EXPIRED"],
			}
		}
		return &struct{ Body statusBody }{Body: body}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-peer-status",
		Method:      http.MethodGet,
		Path:        "/api/v1/status/peers",
		Summary:     "Get live peer connection status",
		Tags:        []string{"OAM"},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []PeerInfo }, error) {
		var peers []PeerInfo
		if s.peerStatusFunc != nil {
			peers = s.peerStatusFunc()
		}
		if peers == nil {
			peers = []PeerInfo{}
		}
		return &struct{ Body []PeerInfo }{peers}, nil
	})

	// Prometheus metrics
	mux.Handle("/metrics", promhttp.Handler())

	// Health probe
	mux.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	// Embedded React UI at /ui/
	mux.Handle("/ui", http.RedirectHandler("/ui/", http.StatusMovedPermanently))
	ui := uiHandler()
	mux.Handle("/ui/", ui)
	mux.Handle("/ui/*", ui)

	// Redirect root → UI
	mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/", http.StatusFound)
	})

	return mux
}

// Start runs the HTTP server until ctx is cancelled.
func (s *Server) Start(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("API server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutCtx)
	case err := <-errCh:
		return err
	}
}

// requestLogger is a minimal slog-based request logger middleware.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		slog.Debug("api",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"ms", time.Since(start).Milliseconds(),
		)
	})
}
