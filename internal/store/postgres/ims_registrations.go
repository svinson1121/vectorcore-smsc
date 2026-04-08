package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func (db *DB) GetIMSRegistration(ctx context.Context, msisdn string) (*store.IMSRegistration, error) {
	const q = `
		SELECT id, msisdn, sip_aor, contact_uri, s_cscf, registered, expiry, updated_at
		FROM ims_registrations
		WHERE msisdn = $1`

	row := db.pool.QueryRow(ctx, q, msisdn)
	reg, err := scanIMSRegistration(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get ims_registration %s: %w", msisdn, err)
	}
	return reg, nil
}

func (db *DB) UpsertIMSRegistration(ctx context.Context, reg store.IMSRegistration) error {
	const q = `
		INSERT INTO ims_registrations (id, msisdn, sip_aor, contact_uri, s_cscf, registered, expiry, updated_at)
		VALUES (COALESCE(NULLIF($1,''), gen_random_uuid()::text)::uuid, $2, $3, $4, $5, $6, $7, now())
		ON CONFLICT (msisdn) DO UPDATE SET
			sip_aor     = EXCLUDED.sip_aor,
			contact_uri = EXCLUDED.contact_uri,
			s_cscf      = EXCLUDED.s_cscf,
			registered  = EXCLUDED.registered,
			expiry      = EXCLUDED.expiry,
			updated_at  = now()`

	_, err := db.pool.Exec(ctx, q,
		reg.ID, reg.MSISDN, reg.SIPAOR, reg.ContactURI, reg.SCSCF, reg.Registered, reg.Expiry)
	if err != nil {
		return fmt.Errorf("upsert ims_registration %s: %w", reg.MSISDN, err)
	}
	return nil
}

func (db *DB) DeleteIMSRegistration(ctx context.Context, msisdn string) error {
	const q = `DELETE FROM ims_registrations WHERE msisdn = $1`
	_, err := db.pool.Exec(ctx, q, msisdn)
	if err != nil {
		return fmt.Errorf("delete ims_registration %s: %w", msisdn, err)
	}
	return nil
}

func (db *DB) ListIMSRegistrations(ctx context.Context) ([]store.IMSRegistration, error) {
	const q = `
		SELECT id, msisdn, sip_aor, contact_uri, s_cscf, registered, expiry, updated_at
		FROM ims_registrations
		ORDER BY msisdn`

	rows, err := db.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list ims_registrations: %w", err)
	}
	defer rows.Close()

	var regs []store.IMSRegistration
	for rows.Next() {
		reg, err := scanIMSRegistration(rows)
		if err != nil {
			return nil, err
		}
		regs = append(regs, *reg)
	}
	return regs, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanIMSRegistration(row scanner) (*store.IMSRegistration, error) {
	var r store.IMSRegistration
	var expiry time.Time
	var updatedAt time.Time
	err := row.Scan(&r.ID, &r.MSISDN, &r.SIPAOR, &r.ContactURI, &r.SCSCF,
		&r.Registered, &expiry, &updatedAt)
	if err != nil {
		return nil, err
	}
	r.Expiry = expiry
	r.UpdatedAt = updatedAt
	return &r, nil
}
