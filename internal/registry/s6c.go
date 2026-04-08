package registry

import (
	"context"
	"log/slog"

	"github.com/svinson1121/vectorcore-smsc/internal/diameter/s6c"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

// S6cLookup checks the local subscriber cache first. If no usable attached MME
// is already known, it queries the HSS via SRI-SM, updates the subscriber
// record, and returns the refreshed state.
func (r *Registry) S6cLookup(ctx context.Context, msisdn string) (*store.Subscriber, error) {
	sub, err := r.st.GetSubscriber(ctx, msisdn)
	if err != nil {
		return nil, err
	}
	if sub != nil && sub.LTEAttached && sub.MMEHost != "" && sub.IMSI != "" {
		return sub, nil
	}

	r.mu.RLock()
	s6cc := r.s6cClient
	r.mu.RUnlock()
	if s6cc == nil {
		return sub, nil
	}

	slog.Debug("registry: S6c fallback lookup", "msisdn", msisdn)

	info, err := s6cc.LookupRouting(msisdn)
	if err != nil {
		var unknown *s6c.ErrUnknownUser
		if isS6cUnknown(err, &unknown) {
			slog.Debug("registry: S6c lookup unknown subscriber", "msisdn", msisdn)
			return sub, nil
		}
		return nil, err
	}

	refreshed := store.Subscriber{MSISDN: msisdn}
	if sub != nil {
		refreshed = *sub
	}
	refreshed.MSISDN = msisdn
	refreshed.IMSI = info.IMSI
	refreshed.LTEAttached = info.Attached
	refreshed.MMEHost = info.MMEName
	refreshed.MWDSet = info.MWDStatus != 0

	if err := r.st.UpsertSubscriber(ctx, refreshed); err != nil {
		slog.Warn("registry: S6c fallback upsert failed", "msisdn", msisdn, "err", err)
	} else {
		slog.Info("registry: S6c fallback populated",
			"msisdn", msisdn,
			"imsi", refreshed.IMSI,
			"mme", refreshed.MMEHost,
			"attached", refreshed.LTEAttached,
			"mwd_set", refreshed.MWDSet,
		)
	}

	return &refreshed, nil
}

func isS6cUnknown(err error, target **s6c.ErrUnknownUser) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*s6c.ErrUnknownUser); ok {
		*target = e
		return true
	}
	return false
}
