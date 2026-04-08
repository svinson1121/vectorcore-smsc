// Package registry maintains an in-memory IMS registration table backed by
// the database.  The SIP ISC handler writes to it on 3rd-party REGISTER and
// NOTIFY; the routing engine reads it to decide whether to deliver via SIP.
package registry

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/diameter/s6c"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

// Registration holds a single IMS registration entry.
type Registration struct {
	MSISDN     string
	IMSI       string // IMSI-derived IMPU user part, e.g. "311435000070570"
	SIPAOR     string // sip:+14155551234@ims.example.com
	ContactURI string // sip:UE-IP:port
	SCSCF      string // sip:scscf.ims.example.com
	Registered bool
	Expiry     time.Time
}

// Registry is safe for concurrent use from multiple goroutines.
type Registry struct {
	mu        sync.RWMutex
	cache     map[string]*Registration // keyed by MSISDN
	st        store.Store
	shClient  ShClient // set via SetShClient; nil until wired
	s6cClient S6cClient
}

// S6cClient is reserved for S6c routing and notification flows.
// Implemented by *s6c.Client.
type S6cClient interface {
	LookupRouting(msisdn string) (*s6c.RoutingInfo, error)
}

// New creates a Registry, loads all current registrations from the store,
// and subscribes to future changes.
func New(ctx context.Context, st store.Store) (*Registry, error) {
	r := &Registry{
		cache: make(map[string]*Registration),
		st:    st,
	}
	if err := r.loadAll(ctx); err != nil {
		return nil, err
	}
	ch := make(chan store.ChangeEvent, 32)
	if err := st.Subscribe(ctx, "ims_registrations", ch); err != nil {
		return nil, err
	}
	go r.watchChanges(ctx, ch)
	return r, nil
}

// Lookup returns the registration for the given MSISDN, or false if not found.
func (r *Registry) Lookup(msisdn string) (*Registration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	reg, ok := r.cache[msisdn]
	if !ok {
		return nil, false
	}
	// Treat expired entries as absent
	if time.Now().After(reg.Expiry) {
		return nil, false
	}
	return reg, true
}

// Upsert inserts or updates a registration in the in-memory cache and the
// backing store.
func (r *Registry) Upsert(ctx context.Context, reg Registration) error {
	r.mu.Lock()
	r.cache[reg.MSISDN] = &reg
	r.mu.Unlock()

	return r.st.UpsertIMSRegistration(ctx, store.IMSRegistration{
		MSISDN:     reg.MSISDN,
		IMSI:       reg.IMSI,
		SIPAOR:     reg.SIPAOR,
		ContactURI: reg.ContactURI,
		SCSCF:      reg.SCSCF,
		Registered: reg.Registered,
		Expiry:     reg.Expiry,
	})
}

// Delete removes a registration from the cache and the backing store.
func (r *Registry) Delete(ctx context.Context, msisdn string) error {
	r.mu.Lock()
	delete(r.cache, msisdn)
	r.mu.Unlock()
	return r.st.DeleteIMSRegistration(ctx, msisdn)
}

// loadAll populates the cache from the database.
func (r *Registry) loadAll(ctx context.Context) error {
	regs, err := r.st.ListIMSRegistrations(ctx)
	if err != nil {
		return err
	}
	r.mu.Lock()
	for _, sr := range regs {
		reg := storeToReg(sr)
		r.cache[reg.MSISDN] = &reg
	}
	r.mu.Unlock()
	slog.Info("registry loaded", "entries", len(regs))
	return nil
}

// watchChanges reloads the full cache whenever the ims_registrations table changes.
// This is acceptable because the table is small (one row per subscriber).
func (r *Registry) watchChanges(ctx context.Context, ch <-chan store.ChangeEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ch:
			if !ok {
				return
			}
			if err := r.loadAll(ctx); err != nil {
				slog.Error("registry reload failed", "err", err)
			}
		}
	}
}

func storeToReg(s store.IMSRegistration) Registration {
	return Registration{
		MSISDN:     s.MSISDN,
		IMSI:       s.IMSI,
		SIPAOR:     s.SIPAOR,
		ContactURI: s.ContactURI,
		SCSCF:      s.SCSCF,
		Registered: s.Registered,
		Expiry:     s.Expiry,
	}
}
