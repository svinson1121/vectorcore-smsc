package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func (db *DB) GetSMPPClient(ctx context.Context, id string) (*store.SMPPClient, error) {
	const q = `
		SELECT id, name, host, port, system_id, password,
		       bind_type, reconnect_interval, throughput_limit,
		       enabled, created_at, updated_at
		FROM smpp_clients WHERE id = $1::uuid`
	row := db.pool.QueryRow(ctx, q, id)
	c, err := scanSMPPClient(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get smpp_client %s: %w", id, err)
	}
	return c, nil
}

func (db *DB) CreateSMPPClient(ctx context.Context, c store.SMPPClient) error {
	const q = `
		INSERT INTO smpp_clients
			(id, name, host, port, system_id, password, bind_type,
			 reconnect_interval, throughput_limit, enabled)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7::interval, $8, $9)`
	_, err := db.pool.Exec(ctx, q,
		c.Name, c.Host, c.Port, c.SystemID, c.Password, c.BindType,
		formatInterval(c.ReconnectInterval), c.ThroughputLimit, c.Enabled)
	if err != nil {
		return fmt.Errorf("create smpp_client: %w", err)
	}
	return nil
}

func (db *DB) UpdateSMPPClient(ctx context.Context, c store.SMPPClient) error {
	const q = `
		UPDATE smpp_clients SET
			name               = $2,
			host               = $3,
			port               = $4,
			system_id          = $5,
			password           = $6,
			bind_type          = $7,
			reconnect_interval = $8::interval,
			throughput_limit   = $9,
			enabled            = $10,
			updated_at         = now()
		WHERE id = $1::uuid`
	_, err := db.pool.Exec(ctx, q,
		c.ID, c.Name, c.Host, c.Port, c.SystemID, c.Password, c.BindType,
		formatInterval(c.ReconnectInterval), c.ThroughputLimit, c.Enabled)
	if err != nil {
		return fmt.Errorf("update smpp_client %s: %w", c.ID, err)
	}
	return nil
}

func (db *DB) DeleteSMPPClient(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM smpp_clients WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete smpp_client %s: %w", id, err)
	}
	return nil
}

func (db *DB) ListSMPPClients(ctx context.Context) ([]store.SMPPClient, error) {
	const q = `
		SELECT id, name, host, port, system_id, password,
		       bind_type, reconnect_interval, throughput_limit,
		       enabled, created_at, updated_at
		FROM smpp_clients
		ORDER BY name`

	rows, err := db.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list smpp_clients: %w", err)
	}
	defer rows.Close()

	var clients []store.SMPPClient
	for rows.Next() {
		c, err := scanSMPPClient(rows)
		if err != nil {
			return nil, err
		}
		clients = append(clients, *c)
	}
	return clients, rows.Err()
}

func scanSMPPClient(row scanner) (*store.SMPPClient, error) {
	var c store.SMPPClient
	var reconnectInterval pgtype.Interval
	err := row.Scan(
		&c.ID, &c.Name, &c.Host, &c.Port, &c.SystemID, &c.Password,
		&c.BindType, &reconnectInterval, &c.ThroughputLimit,
		&c.Enabled, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.ReconnectInterval, err = intervalToDuration(reconnectInterval)
	if err != nil {
		return nil, fmt.Errorf("scan smpp_client reconnect_interval: %w", err)
	}
	return &c, nil
}
