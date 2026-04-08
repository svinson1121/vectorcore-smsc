package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func (db *DB) GetSMPPServerAccountByID(ctx context.Context, id string) (*store.SMPPServerAccount, error) {
	const q = `
		SELECT id, COALESCE(name,''), system_id, password_hash, COALESCE(allowed_ip,''),
		       bind_type, throughput_limit, COALESCE(default_route_id,''),
		       enabled, created_at, updated_at
		FROM smpp_server_accounts WHERE id = ?`
	row := db.db.QueryRowContext(ctx, q, id)
	acc, err := scanSMPPServerAccount(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get smpp_server_account by id %q: %w", id, err)
	}
	return acc, nil
}

func (db *DB) CreateSMPPServerAccount(ctx context.Context, a store.SMPPServerAccount) error {
	const q = `
		INSERT INTO smpp_server_accounts
			(id, name, system_id, password_hash, allowed_ip, bind_type, throughput_limit, default_route_id, enabled)
		VALUES (?, ?, ?, ?, NULLIF(?,''), ?, ?, NULLIF(?,''), ?)`
	_, err := db.db.ExecContext(ctx, q,
		newUUID(), a.Name, a.SystemID, a.PasswordHash, a.AllowedIP,
		a.BindType, a.ThroughputLimit, a.DefaultRouteID, boolInt(a.Enabled))
	if err != nil {
		return fmt.Errorf("create smpp_server_account: %w", err)
	}
	return nil
}

func (db *DB) UpdateSMPPServerAccount(ctx context.Context, a store.SMPPServerAccount) error {
	const q = `
		UPDATE smpp_server_accounts SET
			name             = ?,
			system_id        = ?,
			password_hash    = ?,
			allowed_ip       = NULLIF(?,''),
			bind_type        = ?,
			throughput_limit = ?,
			default_route_id = NULLIF(?,''),
			enabled          = ?,
			updated_at       = datetime('now')
	WHERE id = ?`
	_, err := db.db.ExecContext(ctx, q,
		a.Name, a.SystemID, a.PasswordHash, a.AllowedIP,
		a.BindType, a.ThroughputLimit, a.DefaultRouteID,
		boolInt(a.Enabled), a.ID)
	if err != nil {
		return fmt.Errorf("update smpp_server_account %s: %w", a.ID, err)
	}
	return nil
}

func (db *DB) DeleteSMPPServerAccount(ctx context.Context, id string) error {
	_, err := db.db.ExecContext(ctx, `DELETE FROM smpp_server_accounts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete smpp_server_account %s: %w", id, err)
	}
	return nil
}

func (db *DB) GetSMPPServerAccount(ctx context.Context, systemID string) (*store.SMPPServerAccount, error) {
	const q = `
		SELECT id, COALESCE(name,''), system_id, password_hash, COALESCE(allowed_ip,''),
		       bind_type, throughput_limit, COALESCE(default_route_id,''),
		       enabled, created_at, updated_at
		FROM smpp_server_accounts
		WHERE system_id = ?`

	row := db.db.QueryRowContext(ctx, q, systemID)
	acc, err := scanSMPPServerAccount(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get smpp_server_account %q: %w", systemID, err)
	}
	return acc, nil
}

func (db *DB) ListSMPPServerAccounts(ctx context.Context) ([]store.SMPPServerAccount, error) {
	const q = `
		SELECT id, COALESCE(name,''), system_id, password_hash, COALESCE(allowed_ip,''),
		       bind_type, throughput_limit, COALESCE(default_route_id,''),
		       enabled, created_at, updated_at
		FROM smpp_server_accounts
		ORDER BY name, system_id`

	rows, err := db.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list smpp_server_accounts: %w", err)
	}
	defer rows.Close()

	var accs []store.SMPPServerAccount
	for rows.Next() {
		acc, err := scanSMPPServerAccount(rows)
		if err != nil {
			return nil, err
		}
		accs = append(accs, *acc)
	}
	return accs, rows.Err()
}

func scanSMPPServerAccount(row sqlScanner) (*store.SMPPServerAccount, error) {
	var a store.SMPPServerAccount
	var enabled int
	var createdStr, updatedStr string
	err := row.Scan(
		&a.ID, &a.Name, &a.SystemID, &a.PasswordHash, &a.AllowedIP,
		&a.BindType, &a.ThroughputLimit, &a.DefaultRouteID,
		&enabled, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}
	a.Enabled = enabled != 0
	return &a, nil
}
