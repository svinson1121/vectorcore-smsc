package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

const sgdMappingCols = `id, s6c_result, sgd_host, enabled, created_at, updated_at`

func scanSGDMMEMapping(row interface{ Scan(...any) error }) (store.SGDMMEMapping, error) {
	var m store.SGDMMEMapping
	err := row.Scan(&m.ID, &m.S6CResult, &m.SGDHost, &m.Enabled, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}

func (db *DB) ListSGDMMEMappings(ctx context.Context) ([]store.SGDMMEMapping, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT `+sgdMappingCols+` FROM sgd_mme_mappings ORDER BY s6c_result`)
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
	row := db.pool.QueryRow(ctx,
		`SELECT `+sgdMappingCols+` FROM sgd_mme_mappings WHERE id = $1::uuid`, id)
	m, err := scanSGDMMEMapping(row)
	if errors.Is(err, pgx.ErrNoRows) {
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
		VALUES (gen_random_uuid(), $1, $2, $3)`
	_, err := db.pool.Exec(ctx, q, m.S6CResult, m.SGDHost, m.Enabled)
	if err != nil {
		return fmt.Errorf("create sgd_mme_mapping: %w", err)
	}
	return nil
}

func (db *DB) UpdateSGDMMEMapping(ctx context.Context, m store.SGDMMEMapping) error {
	const q = `
		UPDATE sgd_mme_mappings SET
			s6c_result = $2,
			sgd_host   = $3,
			enabled    = $4,
			updated_at = now()
		WHERE id = $1::uuid`
	_, err := db.pool.Exec(ctx, q, m.ID, m.S6CResult, m.SGDHost, m.Enabled)
	if err != nil {
		return fmt.Errorf("update sgd_mme_mapping %s: %w", m.ID, err)
	}
	return nil
}

func (db *DB) DeleteSGDMMEMapping(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM sgd_mme_mappings WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete sgd_mme_mapping %s: %w", id, err)
	}
	return nil
}
