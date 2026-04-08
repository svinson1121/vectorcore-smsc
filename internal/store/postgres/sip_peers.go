package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func (db *DB) ListAllSIPPeers(ctx context.Context) ([]store.SIPPeer, error) {
	rows, err := db.pool.Query(ctx, `SELECT `+sipPeerCols+` FROM sip_peers ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list all sip_peers: %w", err)
	}
	defer rows.Close()
	var peers []store.SIPPeer
	for rows.Next() {
		p, err := scanSIPPeer(rows)
		if err != nil {
			return nil, err
		}
		peers = append(peers, p)
	}
	return peers, rows.Err()
}

func (db *DB) GetSIPPeerByID(ctx context.Context, id string) (*store.SIPPeer, error) {
	row := db.pool.QueryRow(ctx, `SELECT `+sipPeerCols+` FROM sip_peers WHERE id = $1::uuid`, id)
	p, err := scanSIPPeer(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get sip_peer by id %q: %w", id, err)
	}
	return &p, nil
}

func (db *DB) CreateSIPPeer(ctx context.Context, p store.SIPPeer) error {
	const q = `
		INSERT INTO sip_peers (id, name, address, port, transport, domain, auth_user, auth_pass, enabled)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, NULLIF($6,''), NULLIF($7,''), $8)`
	_, err := db.pool.Exec(ctx, q,
		p.Name, p.Address, p.Port, p.Transport, p.Domain, p.AuthUser, p.AuthPass, p.Enabled)
	if err != nil {
		return fmt.Errorf("create sip_peer: %w", err)
	}
	return nil
}

func (db *DB) UpdateSIPPeer(ctx context.Context, p store.SIPPeer) error {
	const q = `
		UPDATE sip_peers SET
			name      = $2, address   = $3, port      = $4,
			transport = $5, domain    = $6,
			auth_user = NULLIF($7,''), auth_pass = NULLIF($8,''),
			enabled   = $9, updated_at = now()
		WHERE id = $1::uuid`
	_, err := db.pool.Exec(ctx, q,
		p.ID, p.Name, p.Address, p.Port, p.Transport, p.Domain,
		p.AuthUser, p.AuthPass, p.Enabled)
	if err != nil {
		return fmt.Errorf("update sip_peer %s: %w", p.ID, err)
	}
	return nil
}

func (db *DB) DeleteSIPPeer(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM sip_peers WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete sip_peer %s: %w", id, err)
	}
	return nil
}

const sipPeerCols = `id, name, address, port, transport, domain,
	COALESCE(auth_user,'') AS auth_user,
	COALESCE(auth_pass,'') AS auth_pass,
	enabled, created_at, updated_at`

func scanSIPPeer(row interface{ Scan(...any) error }) (store.SIPPeer, error) {
	var p store.SIPPeer
	err := row.Scan(
		&p.ID, &p.Name, &p.Address, &p.Port, &p.Transport, &p.Domain,
		&p.AuthUser, &p.AuthPass,
		&p.Enabled, &p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

func (db *DB) ListSIPPeers(ctx context.Context) ([]store.SIPPeer, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT `+sipPeerCols+` FROM sip_peers WHERE enabled = true ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list sip_peers: %w", err)
	}
	defer rows.Close()

	var peers []store.SIPPeer
	for rows.Next() {
		p, err := scanSIPPeer(rows)
		if err != nil {
			return nil, fmt.Errorf("scan sip_peer: %w", err)
		}
		peers = append(peers, p)
	}
	return peers, rows.Err()
}

func (db *DB) GetSIPPeer(ctx context.Context, name string) (*store.SIPPeer, error) {
	row := db.pool.QueryRow(ctx,
		`SELECT `+sipPeerCols+` FROM sip_peers WHERE name = $1`, name)
	p, err := scanSIPPeer(row)
	if err != nil {
		return nil, fmt.Errorf("get sip_peer %q: %w", name, err)
	}
	return &p, nil
}
