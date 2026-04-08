package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

const sqliteSGDMappingCols = `id, s6c_result, sgd_host, enabled, created_at, updated_at`

func scanSGDMMEMapping(row sqlScanner) (store.SGDMMEMapping, error) {
	var m store.SGDMMEMapping
	var enabledInt int
	var createdStr, updatedStr string
	err := row.Scan(&m.ID, &m.S6CResult, &m.SGDHost, &enabledInt, &createdStr, &updatedStr)
	if err != nil {
		return m, err
	}
	m.Enabled = enabledInt != 0
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return m, nil
}

func (db *DB) ListSGDMMEMappings(ctx context.Context) ([]store.SGDMMEMapping, error) {
	rows, err := db.db.QueryContext(ctx,
		`SELECT `+sqliteSGDMappingCols+` FROM sgd_mme_mappings ORDER BY s6c_result`)
	if err != nil {
		return nil, fmt.Errorf("list sgd_mme_mappings: %w", err)
	}
	defer rows.Close()
	var out []store.SGDMMEMapping
	for rows.Next() {
		m, err := scanSGDMMEMapping(rows)
		if err != nil {
			return nil, fmt.Errorf("scan sgd_mme_mapping: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (db *DB) GetSGDMMEMappingByID(ctx context.Context, id string) (*store.SGDMMEMapping, error) {
	row := db.db.QueryRowContext(ctx,
		`SELECT `+sqliteSGDMappingCols+` FROM sgd_mme_mappings WHERE id = ?`, id)
	m, err := scanSGDMMEMapping(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get sgd_mme_mapping %s: %w", id, err)
	}
	return &m, nil
}

func (db *DB) CreateSGDMMEMapping(ctx context.Context, m store.SGDMMEMapping) error {
	const q = `
		INSERT INTO sgd_mme_mappings (id, s6c_result, sgd_host, enabled)
		VALUES (?, ?, ?, ?)`
	_, err := db.db.ExecContext(ctx, q, newUUID(), m.S6CResult, m.SGDHost, boolInt(m.Enabled))
	if err != nil {
		return fmt.Errorf("create sgd_mme_mapping: %w", err)
	}
	return nil
}

func (db *DB) UpdateSGDMMEMapping(ctx context.Context, m store.SGDMMEMapping) error {
	const q = `
		UPDATE sgd_mme_mappings SET
			s6c_result = ?,
			sgd_host   = ?,
			enabled    = ?,
			updated_at = datetime('now')
		WHERE id = ?`
	_, err := db.db.ExecContext(ctx, q, m.S6CResult, m.SGDHost, boolInt(m.Enabled), m.ID)
	if err != nil {
		return fmt.Errorf("update sgd_mme_mapping %s: %w", m.ID, err)
	}
	return nil
}

func (db *DB) DeleteSGDMMEMapping(ctx context.Context, id string) error {
	_, err := db.db.ExecContext(ctx, `DELETE FROM sgd_mme_mappings WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete sgd_mme_mapping %s: %w", id, err)
	}
	return nil
}
