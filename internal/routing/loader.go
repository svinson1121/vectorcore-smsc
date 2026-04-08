package routing

import (
	"context"
	"log/slog"

	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

// Loader keeps the Engine's rule table in sync with the database.
type Loader struct {
	st     store.Store
	engine *Engine
}

// NewLoader creates a Loader and does an initial rule load.
func NewLoader(ctx context.Context, st store.Store, engine *Engine) (*Loader, error) {
	l := &Loader{st: st, engine: engine}
	if err := l.load(ctx); err != nil {
		return nil, err
	}
	ch := make(chan store.ChangeEvent, 16)
	if err := st.Subscribe(ctx, "routing_rules", ch); err != nil {
		return nil, err
	}
	go l.watch(ctx, ch)
	return l, nil
}

func (l *Loader) load(ctx context.Context) error {
	rules, err := l.st.ListRoutingRules(ctx)
	if err != nil {
		return err
	}
	l.engine.Reload(rules)
	return nil
}

func (l *Loader) watch(ctx context.Context, ch <-chan store.ChangeEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ch:
			if !ok {
				return
			}
			if err := l.load(ctx); err != nil {
				slog.Error("routing reload failed", "err", err)
			}
		}
	}
}
