package registry

import (
	"context"
	"log/slog"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/diameter/sh"
)

// ShClient is the interface the registry uses to query IMS state from the HSS.
// Implemented by *sh.Client.
type ShClient interface {
	LookupIMSState(msisdn string) (*sh.UserDataResult, error)
}

// SetShClient attaches a Sh client to the registry for fallback lookups.
// Call this after creating the Registry and before handling any messages.
func (r *Registry) SetShClient(c ShClient) {
	r.mu.Lock()
	r.shClient = c
	r.mu.Unlock()
}

// SetS6cClient attaches an S6c client for routing lookup and alert handling.
func (r *Registry) SetS6cClient(c S6cClient) {
	r.mu.Lock()
	r.s6cClient = c
	r.mu.Unlock()
}

// ShLookup checks the local cache first.  On a cache miss (or expired entry)
// it queries the HSS via Sh UDR, populates the cache with the result, and
// returns the registration.
//
// Returns (nil, nil) if the subscriber is not IMS-registered according to the HSS.
func (r *Registry) ShLookup(ctx context.Context, msisdn string) (*Registration, error) {
	// Fast path: cache hit
	if reg, ok := r.Lookup(msisdn); ok {
		return reg, nil
	}

	r.mu.RLock()
	shc := r.shClient
	r.mu.RUnlock()

	if shc == nil {
		return nil, nil
	}

	slog.Debug("registry: Sh fallback lookup", "msisdn", msisdn)

	result, err := shc.LookupIMSState(msisdn)
	if err != nil {
		// ErrUnknownUser is not fatal — subscriber just isn't in HSS
		var unknown *sh.ErrUnknownUser
		if isUnknown(err, &unknown) {
			slog.Debug("registry: Sh lookup unknown subscriber", "msisdn", msisdn)
			return nil, nil
		}
		var unsupported *sh.ErrUnsupportedUserData
		if isUnsupportedUserData(err, &unsupported) {
			slog.Debug("registry: Sh lookup unsupported user data", "msisdn", msisdn, "result_code", unsupported.ResultCode)
			return nil, err
		}
		return nil, err
	}

	if !result.Registered {
		slog.Debug("registry: Sh lookup — subscriber unregistered", "msisdn", msisdn)
		return nil, nil
	}

	// Populate cache from HSS data
	reg := Registration{
		MSISDN:     msisdn,
		SIPAOR:     result.SIPAoR,
		ContactURI: result.SIPAoR, // best available; Contact not in Sh response
		SCSCF:      result.SCSCFName,
		Registered: true,
		Expiry:     time.Now().Add(shCacheTTL),
	}

	if err := r.Upsert(ctx, reg); err != nil {
		// Cache upsert failure is non-fatal; return the result anyway
		slog.Warn("registry: Sh fallback upsert failed", "msisdn", msisdn, "err", err)
	}

	slog.Info("registry: Sh fallback populated",
		"msisdn", msisdn,
		"scscf", reg.SCSCF,
		"aor", reg.SIPAOR,
	)

	return &reg, nil
}

// shCacheTTL is how long a Sh-sourced registration is cached before re-querying.
// Kept short so that IMS state changes (re-register, de-register) propagate
// within a reasonable window without hammering the HSS.
const shCacheTTL = 5 * time.Minute

// isUnknown checks whether err wraps *sh.ErrUnknownUser.
func isUnknown(err error, target **sh.ErrUnknownUser) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*sh.ErrUnknownUser); ok {
		*target = e
		return true
	}
	return false
}

func isUnsupportedUserData(err error, target **sh.ErrUnsupportedUserData) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*sh.ErrUnsupportedUserData); ok {
		*target = e
		return true
	}
	return false
}
