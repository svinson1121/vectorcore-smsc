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
			SELECT id, name, host, port, transport, verify_server_cert, system_id, password,
			       bind_type, reconnect_interval, throughput_limit,
			       source_addr_ton, source_addr_npi, dest_addr_ton, dest_addr_npi,
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
				(id, name, host, port, transport, verify_server_cert, system_id, password, bind_type,
				 reconnect_interval, throughput_limit, source_addr_ton, source_addr_npi,
				 dest_addr_ton, dest_addr_npi, enabled)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9::interval, $10, $11, $12, $13, $14, $15)`
	_, err := db.pool.Exec(ctx, q,
		c.Name, c.Host, c.Port, c.Transport, c.VerifyServerCert, c.SystemID, c.Password, c.BindType,
		formatInterval(c.ReconnectInterval), c.ThroughputLimit, c.SourceAddrTON, c.SourceAddrNPI,
		c.DestAddrTON, c.DestAddrNPI, c.Enabled)
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
			transport          = $5,
			verify_server_cert = $6,
			system_id          = $7,
			password           = $8,
				bind_type          = $9,
				reconnect_interval = $10::interval,
				throughput_limit   = $11,
				source_addr_ton    = $12,
				source_addr_npi    = $13,
				dest_addr_ton      = $14,
				dest_addr_npi      = $15,
				enabled            = $16,
				updated_at         = now()
			WHERE id = $1::uuid`
	_, err := db.pool.Exec(ctx, q,
		c.ID, c.Name, c.Host, c.Port, c.Transport, c.VerifyServerCert, c.SystemID, c.Password, c.BindType,
		formatInterval(c.ReconnectInterval), c.ThroughputLimit, c.SourceAddrTON, c.SourceAddrNPI,
		c.DestAddrTON, c.DestAddrNPI, c.Enabled)
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
			SELECT id, name, host, port, transport, verify_server_cert, system_id, password,
			       bind_type, reconnect_interval, throughput_limit,
			       source_addr_ton, source_addr_npi, dest_addr_ton, dest_addr_npi,
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
	var sourceAddrTON, sourceAddrNPI, destAddrTON, destAddrNPI pgtype.Int4
	err := row.Scan(
		&c.ID, &c.Name, &c.Host, &c.Port, &c.Transport, &c.VerifyServerCert, &c.SystemID, &c.Password,
		&c.BindType, &reconnectInterval, &c.ThroughputLimit,
		&sourceAddrTON, &sourceAddrNPI, &destAddrTON, &destAddrNPI,
		&c.Enabled, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.ReconnectInterval, err = intervalToDuration(reconnectInterval)
	if err != nil {
		return nil, fmt.Errorf("scan smpp_client reconnect_interval: %w", err)
	}
	if c.Transport == "" {
		c.Transport = "tcp"
	}
	c.SourceAddrTON = intPtrFromPG(sourceAddrTON)
	c.SourceAddrNPI = intPtrFromPG(sourceAddrNPI)
	c.DestAddrTON = intPtrFromPG(destAddrTON)
	c.DestAddrNPI = intPtrFromPG(destAddrNPI)
	return &c, nil
}

func intPtrFromPG(v pgtype.Int4) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int32)
	return &n
}
