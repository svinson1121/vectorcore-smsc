package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

// ChangeEvent is re-exported so callers only import this package.
type ChangeEvent = store.ChangeEvent

// notifier holds a dedicated PostgreSQL connection for LISTEN/NOTIFY.
// It is separate from the pool so notifications are never lost during
// query traffic.
type notifier struct {
	conn   *pgx.Conn
	mu     sync.Mutex
	subs   map[string][]chan<- ChangeEvent // table → subscriber channels
	cancel context.CancelFunc
	listen chan listenRequest
}

type listenRequest struct {
	channel string
	result  chan error
}

func newNotifier(ctx context.Context, dsn string) (*notifier, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("notify connection: %w", err)
	}

	nctx, cancel := context.WithCancel(ctx)
	n := &notifier{
		conn:   conn,
		subs:   make(map[string][]chan<- ChangeEvent),
		cancel: cancel,
		listen: make(chan listenRequest),
	}
	go n.loop(nctx)
	return n, nil
}

// subscribe registers ch for notifications on the given table.
// The first subscriber for a table issues a LISTEN command.
func (n *notifier) subscribe(ctx context.Context, table string, ch chan<- ChangeEvent) error {
	channel := table + "_changed"

	n.mu.Lock()
	existing := n.subs[table]
	n.subs[table] = append(existing, ch)
	first := len(existing) == 0
	n.mu.Unlock()

	if first {
		req := listenRequest{
			channel: channel,
			result:  make(chan error, 1),
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case n.listen <- req:
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-req.result:
			if err != nil {
				return fmt.Errorf("LISTEN %s: %w", channel, err)
			}
		}
	}
	return nil
}

// loop waits for notifications and fans them out to subscribers.
func (n *notifier) loop(ctx context.Context) {
	for {
		if n.handleListen(ctx) {
			continue
		}

		waitCtx, cancel := context.WithTimeout(ctx, time.Second)
		notification, err := n.conn.WaitForNotification(waitCtx)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return // cancelled, clean shutdown
			}
			if errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			slog.Error("notify wait error", "err", err)
			return
		}
		n.dispatch(notification)
	}
}

func (n *notifier) handleListen(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case req := <-n.listen:
		_, err := n.conn.Exec(ctx, "LISTEN "+pgx.Identifier{req.channel}.Sanitize())
		req.result <- err
		close(req.result)
		for {
			select {
			case req := <-n.listen:
				_, err := n.conn.Exec(ctx, "LISTEN "+pgx.Identifier{req.channel}.Sanitize())
				req.result <- err
				close(req.result)
			default:
				return true
			}
		}
	default:
		return false
	}
}

func (n *notifier) dispatch(notif *pgconn.Notification) {
	// channel name is "<table>_changed"; payload is the SQL operation
	table := tableName(notif.Channel)
	event := ChangeEvent{Table: table, Operation: notif.Payload}

	n.mu.Lock()
	chs := n.subs[table]
	n.mu.Unlock()

	for _, ch := range chs {
		select {
		case ch <- event:
		default:
			// drop if subscriber is not keeping up
		}
	}
}

func (n *notifier) close() {
	n.cancel()
	n.conn.Close(context.Background())
}

// tableName strips the "_changed" suffix from a PostgreSQL NOTIFY channel name.
func tableName(channel string) string {
	const suffix = "_changed"
	if len(channel) > len(suffix) && channel[len(channel)-len(suffix):] == suffix {
		return channel[:len(channel)-len(suffix)]
	}
	return channel
}
