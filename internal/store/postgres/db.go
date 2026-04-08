// Package postgres implements the store.Store interface using pgx/v5.
package postgres

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schema string

// DB holds the connection pool and implements store.Store.
type DB struct {
	pool   *pgxpool.Pool
	notify *notifier
}

// Open creates a pgx connection pool and starts the LISTEN/NOTIFY notifier.
func Open(ctx context.Context, dsn string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, fmt.Errorf("apply postgres schema: %w", err)
	}

	n, err := newNotifier(ctx, dsn)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("notifier: %w", err)
	}

	db := &DB{pool: pool, notify: n}
	// Additive migrations — ignore errors if already applied.
	pool.Exec(ctx, `ALTER TABLE smpp_server_accounts ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT ''`)
	pool.Exec(ctx, `UPDATE smpp_server_accounts SET name = system_id WHERE COALESCE(name,'') = ''`)
	pool.Exec(ctx, `ALTER TABLE routing_rules RENAME COLUMN egress_peer_id TO egress_peer`)
	pool.Exec(ctx, `ALTER TABLE routing_rules ALTER COLUMN egress_peer TYPE TEXT USING egress_peer::text`)
	// diameter_peers: add applications (JSON array text) alongside legacy application column.
	pool.Exec(ctx, `ALTER TABLE diameter_peers ADD COLUMN IF NOT EXISTS applications TEXT NOT NULL DEFAULT '[]'`)
	pool.Exec(ctx, `UPDATE diameter_peers SET applications = json_build_array(application)::text WHERE applications = '[]'`)
	pool.Exec(ctx, `ALTER TABLE messages ADD COLUMN IF NOT EXISTS udh BYTEA`)
	pool.Exec(ctx, `ALTER TABLE messages ADD COLUMN IF NOT EXISTS encoding SMALLINT`)
	// Crash recovery: messages stuck in DISPATCHED state (server died mid-send)
	// are reset to QUEUED so the retry scheduler re-attempts them.
	pool.Exec(ctx, `UPDATE messages SET status='QUEUED', next_retry_at=now() WHERE status='DISPATCHED'`)
	return db, nil
}

// Close shuts down the pool and notifier.
func (db *DB) Close() error {
	db.notify.close()
	db.pool.Close()
	return nil
}

// Subscribe registers ch to receive ChangeEvents for table.
func (db *DB) Subscribe(ctx context.Context, table string, ch chan<- ChangeEvent) error {
	return db.notify.subscribe(ctx, table, ch)
}
