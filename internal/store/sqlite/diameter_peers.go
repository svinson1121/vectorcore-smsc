package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func (db *DB) ListAllDiameterPeers(ctx context.Context) ([]store.DiameterPeer, error) {
	rows, err := db.db.QueryContext(ctx,
		`SELECT `+sqliteDiamPeerCols+` FROM diameter_peers ORDER BY name`)
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
	row := db.db.QueryRowContext(ctx,
		`SELECT `+sqliteDiamPeerCols+` FROM diameter_peers WHERE id = ?`, id)
	p, err := scanDiameterPeer(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get diameter_peer by id %q: %w", id, err)
	}
	return &p, nil
}

func (db *DB) CreateDiameterPeer(ctx context.Context, p store.DiameterPeer) error {
	appsJSON, _ := json.Marshal(p.Applications)
	const q = `
		INSERT INTO diameter_peers (id, name, host, realm, port, transport, application, applications, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	legacyApp := ""
	if len(p.Applications) > 0 {
		legacyApp = p.Applications[0]
	}
	_, err := db.db.ExecContext(ctx, q,
		newUUID(), p.Name, p.Host, p.Realm, p.Port, p.Transport, legacyApp, string(appsJSON), boolInt(p.Enabled))
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
			name         = ?, host        = ?, realm       = ?,
			port         = ?, transport   = ?, application = ?,
			applications = ?, enabled     = ?, updated_at  = datetime('now')
		WHERE id = ?`
	_, err := db.db.ExecContext(ctx, q,
		p.Name, p.Host, p.Realm, p.Port, p.Transport, legacyApp,
		string(appsJSON), boolInt(p.Enabled), p.ID)
	if err != nil {
		return fmt.Errorf("update diameter_peer %s: %w", p.ID, err)
	}
	return nil
}

func (db *DB) DeleteDiameterPeer(ctx context.Context, id string) error {
	_, err := db.db.ExecContext(ctx, `DELETE FROM diameter_peers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete diameter_peer %s: %w", id, err)
	}
	return nil
}

const sqliteDiamPeerCols = `id, name, host, realm, port, transport, application, applications,
	enabled, created_at, updated_at`

func scanDiameterPeer(row sqlScanner) (store.DiameterPeer, error) {
	var p store.DiameterPeer
	var enabledInt int
	var createdStr, updatedStr, legacyApp, appsJSON string
	err := row.Scan(
		&p.ID, &p.Name, &p.Host, &p.Realm, &p.Port, &p.Transport, &legacyApp, &appsJSON,
		&enabledInt, &createdStr, &updatedStr,
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
	p.Enabled = enabledInt != 0
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return p, nil
}

func (db *DB) ListDiameterPeers(ctx context.Context) ([]store.DiameterPeer, error) {
	rows, err := db.db.QueryContext(ctx,
		`SELECT `+sqliteDiamPeerCols+` FROM diameter_peers WHERE enabled = 1 ORDER BY name`)
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
	row := db.db.QueryRowContext(ctx,
		`SELECT `+sqliteDiamPeerCols+` FROM diameter_peers WHERE name = ?`, name)
	p, err := scanDiameterPeer(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get diameter_peer %q: %w", name, err)
	}
	return &p, nil
}
