package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func (db *DB) ListAllDiameterPeers(ctx context.Context) ([]store.DiameterPeer, error) {
	rows, err := db.pool.Query(ctx, `SELECT `+diamPeerCols+` FROM diameter_peers ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list all diameter_peers: %w", err)
	}
	defer rows.Close()
	var peers []store.DiameterPeer
	for rows.Next() {
		p, err := scanDiameterPeer(rows)
		if err != nil {
			return nil, err
		}
		peers = append(peers, p)
	}
	return peers, rows.Err()
}

func (db *DB) GetDiameterPeerByID(ctx context.Context, id string) (*store.DiameterPeer, error) {
	row := db.pool.QueryRow(ctx, `SELECT `+diamPeerCols+` FROM diameter_peers WHERE id = $1::uuid`, id)
	p, err := scanDiameterPeer(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get diameter_peer by id %q: %w", id, err)
	}
	return &p, nil
}

func (db *DB) CreateDiameterPeer(ctx context.Context, p store.DiameterPeer) error {
	appsJSON, _ := json.Marshal(p.Applications)
	legacyApp := ""
	if len(p.Applications) > 0 {
		legacyApp = p.Applications[0]
	}
	const q = `
		INSERT INTO diameter_peers (id, name, host, realm, port, transport, application, applications, enabled)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := db.pool.Exec(ctx, q,
		p.Name, p.Host, p.Realm, p.Port, p.Transport, legacyApp, string(appsJSON), p.Enabled)
	if err != nil {
		return fmt.Errorf("create diameter_peer: %w", err)
	}
	return nil
}

func (db *DB) UpdateDiameterPeer(ctx context.Context, p store.DiameterPeer) error {
	appsJSON, _ := json.Marshal(p.Applications)
	legacyApp := ""
	if len(p.Applications) > 0 {
		legacyApp = p.Applications[0]
	}
	const q = `
		UPDATE diameter_peers SET
			name         = $2, host        = $3, realm        = $4,
			port         = $5, transport   = $6, application  = $7,
			applications = $8, enabled     = $9, updated_at   = now()
		WHERE id = $1::uuid`
	_, err := db.pool.Exec(ctx, q,
		p.ID, p.Name, p.Host, p.Realm, p.Port, p.Transport, legacyApp, string(appsJSON), p.Enabled)
	if err != nil {
		return fmt.Errorf("update diameter_peer %s: %w", p.ID, err)
	}
	return nil
}

func (db *DB) DeleteDiameterPeer(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM diameter_peers WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete diameter_peer %s: %w", id, err)
	}
	return nil
}

const diamPeerCols = `id, name, host, realm, port, transport, application, applications,
	enabled, created_at, updated_at`

func scanDiameterPeer(row interface{ Scan(...any) error }) (store.DiameterPeer, error) {
	var p store.DiameterPeer
	var legacyApp, appsJSON string
	err := row.Scan(
		&p.ID, &p.Name, &p.Host, &p.Realm, &p.Port, &p.Transport, &legacyApp, &appsJSON,
		&p.Enabled, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return p, err
	}
	if appsJSON != "" && appsJSON != "[]" {
		_ = json.Unmarshal([]byte(appsJSON), &p.Applications)
	}
	if len(p.Applications) == 0 && legacyApp != "" {
		p.Applications = []string{legacyApp}
	}
	if p.Applications == nil {
		p.Applications = []string{}
	}
	return p, nil
}

func (db *DB) ListDiameterPeers(ctx context.Context) ([]store.DiameterPeer, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT `+diamPeerCols+` FROM diameter_peers WHERE enabled = true ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list diameter_peers: %w", err)
	}
	defer rows.Close()

	var peers []store.DiameterPeer
	for rows.Next() {
		p, err := scanDiameterPeer(rows)
		if err != nil {
			return nil, fmt.Errorf("scan diameter_peer: %w", err)
		}
		peers = append(peers, p)
	}
	return peers, rows.Err()
}

func (db *DB) GetDiameterPeer(ctx context.Context, name string) (*store.DiameterPeer, error) {
	row := db.pool.QueryRow(ctx,
		`SELECT `+diamPeerCols+` FROM diameter_peers WHERE name = $1`, name)
	p, err := scanDiameterPeer(row)
	if err != nil {
		return nil, fmt.Errorf("get diameter_peer %q: %w", name, err)
	}
	return &p, nil
}
