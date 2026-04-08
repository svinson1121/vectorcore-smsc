// Package sgdmap provides hot-reloadable translation of MME hostnames returned
// by S6c (S6a FQDNs) to the corresponding SGd FQDNs used for Diameter delivery.
package sgdmap

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
	"unsafe"

	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

// Mapper holds an in-memory lookup table that is atomically swapped on reload.
// Only enabled entries are kept in the active map.
type Mapper struct {
	// m is an atomically-swapped *map[string]string (lowercase s6c_result → sgd_host).
	m unsafe.Pointer
}

// NewMapper creates a Mapper, loads the initial mapping set from st, and starts
// a background goroutine that hot-reloads on sgd_mme_mappings_changed events.
func NewMapper(ctx context.Context, st store.Store) (*Mapper, error) {
	mp := &Mapper{}
	empty := map[string]string{}
	atomic.StorePointer(&mp.m, unsafe.Pointer(&empty))

	if err := mp.load(ctx, st); err != nil {
		return nil, err
	}

	ch := make(chan store.ChangeEvent, 16)
	if err := st.Subscribe(ctx, "sgd_mme_mappings", ch); err != nil {
		return nil, err
	}
	go mp.watch(ctx, st, ch)
	return mp, nil
}

// Map translates mmeHost using the active mapping table.
// If an enabled entry is found the mapped SGd host is returned; otherwise
// mmeHost is returned unchanged.
// Comparison is case-insensitive (FQDNs are case-insensitive by spec).
func (mp *Mapper) Map(mmeHost string) string {
	m := *(*map[string]string)(atomic.LoadPointer(&mp.m))
	key := strings.ToLower(mmeHost)
	if mapped, ok := m[key]; ok {
		slog.Debug("sgdmap: remapped MME host",
			"original", mmeHost,
			"mapped", mapped,
		)
		return mapped
	}
	return mmeHost
}

// Reload rebuilds the active map from rows, keeping only enabled entries.
func (mp *Mapper) Reload(rows []store.SGDMMEMapping) {
	m := make(map[string]string, len(rows))
	for _, r := range rows {
		if r.Enabled {
			m[strings.ToLower(r.S6CResult)] = r.SGDHost
		}
	}
	atomic.StorePointer(&mp.m, unsafe.Pointer(&m))
	slog.Info("sgdmap: mapping table reloaded", "entries", len(m))
}

func (mp *Mapper) load(ctx context.Context, st store.Store) error {
	rows, err := st.ListSGDMMEMappings(ctx)
	if err != nil {
		return err
	}
	mp.Reload(rows)
	return nil
}

func (mp *Mapper) watch(ctx context.Context, st store.Store, ch <-chan store.ChangeEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ch:
			if !ok {
				return
			}
			if err := mp.load(ctx, st); err != nil {
				slog.Error("sgdmap: reload failed", "err", err)
			}
		}
	}
}
