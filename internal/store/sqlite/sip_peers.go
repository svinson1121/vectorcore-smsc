package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func (db *DB) ListAllSIPPeers(ctx context.Context) ([]store.SIPPeer, error) {
	rows, err := db.db.QueryContext(ctx,
		`SELECT `+sqliteSIPPeerCols+` FROM sip_peers ORDER BY name`)
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
	row := db.db.QueryRowContext(ctx,
		`SELECT `+sqliteSIPPeerCols+` FROM sip_peers WHERE id = ?`, id)
	p, err := scanSIPPeer(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get sip_peer by id %q: %w", id, err)
	}
	return &p, nil
}

func (db *DB) CreateSIPPeer(ctx context.Context, p store.SIPPeer) error {
	const q = `
		INSERT INTO sip_peers (id, name, address, port, transport, domain, auth_user, auth_pass, enabled)
		VALUES (?, ?, ?, ?, ?, ?, NULLIF(?,''), NULLIF(?,''), ?)`
	_, err := db.db.ExecContext(ctx, q,
		newUUID(), p.Name, p.Address, p.Port, p.Transport, p.Domain,
		p.AuthUser, p.AuthPass, boolInt(p.Enabled))
	if err != nil {
		return fmt.Errorf("create sip_peer: %w", err)
	}
	return nil
}

func (db *DB) UpdateSIPPeer(ctx context.Context, p store.SIPPeer) error {
	const q = `
		UPDATE sip_peers SET
			name      = ?, address   = ?, port      = ?,
			transport = ?, domain    = ?,
			auth_user = NULLIF(?,''), auth_pass = NULLIF(?,''),
			enabled   = ?, updated_at = datetime('now')
		WHERE id = ?`
	_, err := db.db.ExecContext(ctx, q,
		p.Name, p.Address, p.Port, p.Transport, p.Domain,
		p.AuthUser, p.AuthPass, boolInt(p.Enabled), p.ID)
	if err != nil {
		return fmt.Errorf("update sip_peer %s: %w", p.ID, err)
	}
	return nil
}

func (db *DB) DeleteSIPPeer(ctx context.Context, id string) error {
	_, err := db.db.ExecContext(ctx, `DELETE FROM sip_peers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete sip_peer %s: %w", id, err)
	}
	return nil
}

const sqliteSIPPeerCols = `id, name, address, port, transport, domain,
	COALESCE(auth_user,'') AS auth_user,
	COALESCE(auth_pass,'') AS auth_pass,
	enabled, created_at, updated_at`

func scanSIPPeer(row sqlScanner) (store.SIPPeer, error) {
	var p store.SIPPeer
	var enabledInt int
	var createdStr, updatedStr string
	err := row.Scan(
		&p.ID, &p.Name, &p.Address, &p.Port, &p.Transport, &p.Domain,
		&p.AuthUser, &p.AuthPass,
		&enabledInt, &createdStr, &updatedStr,
	)
	if err != nil {
		return p, err
	}
	p.Enabled = enabledInt != 0
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return p, nil
}

func (db *DB) ListSIPPeers(ctx context.Context) ([]store.SIPPeer, error) {
	rows, err := db.db.QueryContext(ctx,
		`SELECT `+sqliteSIPPeerCols+` FROM sip_peers WHERE enabled = 1 ORDER BY name`)
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
	row := db.db.QueryRowContext(ctx,
		`SELECT `+sqliteSIPPeerCols+` FROM sip_peers WHERE name = ?`, name)
	p, err := scanSIPPeer(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get sip_peer %q: %w", name, err)
	}
	return &p, nil
}
