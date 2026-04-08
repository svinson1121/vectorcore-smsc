package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func (db *DB) GetIMSRegistration(ctx context.Context, msisdn string) (*store.IMSRegistration, error) {
	const q = `
		SELECT id, msisdn, imsi, sip_aor, contact_uri, s_cscf, registered, expiry, updated_at
		FROM ims_registrations
		WHERE msisdn = ?`

	row := db.db.QueryRowContext(ctx, q, msisdn)
	reg, err := scanIMSRegistration(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get ims_registration %s: %w", msisdn, err)
	}
	return reg, nil
}

func (db *DB) UpsertIMSRegistration(ctx context.Context, reg store.IMSRegistration) error {
	if reg.ID == "" {
		reg.ID = newUUID()
	}
	const q = `
		INSERT INTO ims_registrations (id, msisdn, imsi, sip_aor, contact_uri, s_cscf, registered, expiry, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT (msisdn) DO UPDATE SET
			imsi        = excluded.imsi,
			sip_aor     = excluded.sip_aor,
			contact_uri = excluded.contact_uri,
			s_cscf      = excluded.s_cscf,
			registered  = excluded.registered,
			expiry      = excluded.expiry,
			updated_at  = datetime('now')`

	_, err := db.db.ExecContext(ctx, q,
		reg.ID, reg.MSISDN, reg.IMSI, reg.SIPAOR, reg.ContactURI, reg.SCSCF,
		boolInt(reg.Registered), reg.Expiry.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("upsert ims_registration %s: %w", reg.MSISDN, err)
	}
	return nil
}

func (db *DB) DeleteIMSRegistration(ctx context.Context, msisdn string) error {
	const q = `DELETE FROM ims_registrations WHERE msisdn = ?`
	_, err := db.db.ExecContext(ctx, q, msisdn)
	if err != nil {
		return fmt.Errorf("delete ims_registration %s: %w", msisdn, err)
	}
	return nil
}

func (db *DB) ListIMSRegistrations(ctx context.Context) ([]store.IMSRegistration, error) {
	const q = `
		SELECT id, msisdn, imsi, sip_aor, contact_uri, s_cscf, registered, expiry, updated_at
		FROM ims_registrations
		ORDER BY msisdn`

	rows, err := db.db.QueryContext(ctx, q)
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

type sqlScanner interface {
	Scan(dest ...any) error
}

func scanIMSRegistration(row sqlScanner) (*store.IMSRegistration, error) {
	var r store.IMSRegistration
	var registered int
	var imsi sql.NullString
	var expiryStr, updatedStr string
	err := row.Scan(&r.ID, &r.MSISDN, &imsi, &r.SIPAOR, &r.ContactURI, &r.SCSCF,
		&registered, &expiryStr, &updatedStr)
	if err != nil {
		return nil, err
	}
	r.IMSI = imsi.String
	r.Registered = registered != 0
	r.Expiry = parseDBTime(expiryStr)
	r.UpdatedAt = parseDBTime(updatedStr)
	return &r, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func newUUID() string {
	var b [16]byte
	rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
