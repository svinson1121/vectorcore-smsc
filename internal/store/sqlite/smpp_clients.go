package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func (db *DB) GetSMPPClient(ctx context.Context, id string) (*store.SMPPClient, error) {
	const q = `
			SELECT id, name, host, port, transport, verify_server_cert, system_id, password,
			       bind_type, reconnect_interval, throughput_limit,
			       source_addr_ton, source_addr_npi, dest_addr_ton, dest_addr_npi,
			       enabled, created_at, updated_at
			FROM smpp_clients WHERE id = ?`
	row := db.db.QueryRowContext(ctx, q, id)
	c, err := scanSMPPClient(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := db.db.ExecContext(ctx, q,
		newUUID(), c.Name, c.Host, c.Port, c.Transport, boolInt(c.VerifyServerCert),
		c.SystemID, c.Password, c.BindType, c.ReconnectInterval.String(), c.ThroughputLimit,
		c.SourceAddrTON, c.SourceAddrNPI, c.DestAddrTON, c.DestAddrNPI, boolInt(c.Enabled))
	if err != nil {
		return fmt.Errorf("create smpp_client: %w", err)
	}
	return nil
}

func (db *DB) UpdateSMPPClient(ctx context.Context, c store.SMPPClient) error {
	const q = `
		UPDATE smpp_clients SET
			name               = ?,
			host               = ?,
			port               = ?,
			transport          = ?,
			verify_server_cert = ?,
			system_id          = ?,
			password           = ?,
				bind_type          = ?,
				reconnect_interval = ?,
				throughput_limit   = ?,
				source_addr_ton    = ?,
				source_addr_npi    = ?,
				dest_addr_ton      = ?,
				dest_addr_npi      = ?,
				enabled            = ?,
				updated_at         = datetime('now')
			WHERE id = ?`
	_, err := db.db.ExecContext(ctx, q,
		c.Name, c.Host, c.Port, c.Transport, boolInt(c.VerifyServerCert),
		c.SystemID, c.Password, c.BindType,
		c.ReconnectInterval.String(), c.ThroughputLimit, c.SourceAddrTON, c.SourceAddrNPI,
		c.DestAddrTON, c.DestAddrNPI, boolInt(c.Enabled), c.ID)
	if err != nil {
		return fmt.Errorf("update smpp_client %s: %w", c.ID, err)
	}
	return nil
}

func (db *DB) DeleteSMPPClient(ctx context.Context, id string) error {
	_, err := db.db.ExecContext(ctx, `DELETE FROM smpp_clients WHERE id = ?`, id)
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

	rows, err := db.db.QueryContext(ctx, q)
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

func scanSMPPClient(row sqlScanner) (*store.SMPPClient, error) {
	var c store.SMPPClient
	var enabled, verifyServerCert int
	var reconnectStr, createdStr, updatedStr string
	var sourceAddrTON, sourceAddrNPI, destAddrTON, destAddrNPI sql.NullInt64
	err := row.Scan(
		&c.ID, &c.Name, &c.Host, &c.Port, &c.Transport, &verifyServerCert, &c.SystemID, &c.Password,
		&c.BindType, &reconnectStr, &c.ThroughputLimit,
		&sourceAddrTON, &sourceAddrNPI, &destAddrTON, &destAddrNPI,
		&enabled, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}
	c.Enabled = enabled != 0
	c.VerifyServerCert = verifyServerCert != 0
	if c.Transport == "" {
		c.Transport = "tcp"
	}
	c.SourceAddrTON = intPtrFromNull(sourceAddrTON)
	c.SourceAddrNPI = intPtrFromNull(sourceAddrNPI)
	c.DestAddrTON = intPtrFromNull(destAddrTON)
	c.DestAddrNPI = intPtrFromNull(destAddrNPI)
	c.ReconnectInterval = parseSQLiteInterval(reconnectStr)
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return &c, nil
}

func intPtrFromNull(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}

// parseSQLiteInterval parses a duration string stored in SQLite (e.g. "10s", "1m").
func parseSQLiteInterval(s string) time.Duration {
	if s == "" {
		return 10 * time.Second
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	return 10 * time.Second
}
