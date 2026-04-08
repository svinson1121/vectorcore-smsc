package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func (db *DB) GetSMPPServerAccountByID(ctx context.Context, id string) (*store.SMPPServerAccount, error) {
	const q = `
		SELECT id, COALESCE(name,''), system_id, password_hash, COALESCE(allowed_ip::text,''),
		       bind_type, throughput_limit, COALESCE(default_route_id::text,''),
		       enabled, created_at, updated_at
		FROM smpp_server_accounts WHERE id = $1::uuid`
	row := db.pool.QueryRow(ctx, q, id)
	acc, err := scanSMPPServerAccount(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
		VALUES
			(gen_random_uuid(), $1, $2, $3, NULLIF($4,'')::inet, $5, $6, NULLIF($7,'')::uuid, $8)`
	_, err := db.pool.Exec(ctx, q,
		a.Name, a.SystemID, a.PasswordHash, a.AllowedIP,
		a.BindType, a.ThroughputLimit, a.DefaultRouteID, a.Enabled)
	if err != nil {
		return fmt.Errorf("create smpp_server_account: %w", err)
	}
	return nil
}

func (db *DB) UpdateSMPPServerAccount(ctx context.Context, a store.SMPPServerAccount) error {
	const q = `
		UPDATE smpp_server_accounts SET
			name             = $2,
			system_id        = $3,
			password_hash    = $4,
			allowed_ip       = NULLIF($5,'')::inet,
			bind_type        = $6,
			throughput_limit = $7,
			default_route_id = NULLIF($8,'')::uuid,
			enabled          = $9,
			updated_at       = now()
		WHERE id = $1::uuid`
	_, err := db.pool.Exec(ctx, q,
		a.ID, a.Name, a.SystemID, a.PasswordHash, a.AllowedIP,
		a.BindType, a.ThroughputLimit, a.DefaultRouteID, a.Enabled)
	if err != nil {
		return fmt.Errorf("update smpp_server_account %s: %w", a.ID, err)
	}
	return nil
}

func (db *DB) DeleteSMPPServerAccount(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM smpp_server_accounts WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete smpp_server_account %s: %w", id, err)
	}
	return nil
}

func (db *DB) GetSMPPServerAccount(ctx context.Context, systemID string) (*store.SMPPServerAccount, error) {
	const q = `
		SELECT id, COALESCE(name,''), system_id, password_hash, COALESCE(allowed_ip::text,''),
		       bind_type, throughput_limit, COALESCE(default_route_id::text,''),
		       enabled, created_at, updated_at
		FROM smpp_server_accounts
		WHERE system_id = $1`

	row := db.pool.QueryRow(ctx, q, systemID)
	acc, err := scanSMPPServerAccount(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get smpp_server_account %q: %w", systemID, err)
	}
	return acc, nil
}

func (db *DB) ListSMPPServerAccounts(ctx context.Context) ([]store.SMPPServerAccount, error) {
	const q = `
		SELECT id, COALESCE(name,''), system_id, password_hash, COALESCE(allowed_ip::text,''),
		       bind_type, throughput_limit, COALESCE(default_route_id::text,''),
		       enabled, created_at, updated_at
		FROM smpp_server_accounts
		ORDER BY name, system_id`

	rows, err := db.pool.Query(ctx, q)
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

func scanSMPPServerAccount(row scanner) (*store.SMPPServerAccount, error) {
	var a store.SMPPServerAccount
	err := row.Scan(
		&a.ID, &a.Name, &a.SystemID, &a.PasswordHash, &a.AllowedIP,
		&a.BindType, &a.ThroughputLimit, &a.DefaultRouteID,
		&a.Enabled, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}
