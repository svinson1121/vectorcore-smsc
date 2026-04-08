package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func (db *DB) GetSubscriber(ctx context.Context, msisdn string) (*store.Subscriber, error) {
	const q = `
		SELECT id, msisdn, COALESCE(imsi,''), ims_registered, lte_attached,
		       COALESCE(mme_host,''), mwd_set, created_at, updated_at
		FROM subscribers
		WHERE msisdn = ?`

	row := db.db.QueryRowContext(ctx, q, msisdn)
	sub, err := scanSubscriber(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get subscriber %s: %w", msisdn, err)
	}
	return sub, nil
}

func (db *DB) UpsertSubscriber(ctx context.Context, sub store.Subscriber) error {
	if sub.ID == "" {
		sub.ID = newUUID()
	}
	const q = `
		INSERT INTO subscribers (id, msisdn, imsi, ims_registered, lte_attached, mme_host, mwd_set)
		VALUES (?, ?, NULLIF(?,''), ?, ?, NULLIF(?,''), ?)
		ON CONFLICT (msisdn) DO UPDATE SET
			imsi           = excluded.imsi,
			ims_registered = excluded.ims_registered,
			lte_attached   = excluded.lte_attached,
			mme_host       = excluded.mme_host,
			mwd_set        = excluded.mwd_set,
			updated_at     = datetime('now')`

	_, err := db.db.ExecContext(ctx, q,
		sub.ID, sub.MSISDN, sub.IMSI,
		boolInt(sub.IMSRegistered), boolInt(sub.LTEAttached),
		sub.MMEHost, boolInt(sub.MWDSet))
	if err != nil {
		return fmt.Errorf("upsert subscriber %s: %w", sub.MSISDN, err)
	}
	return nil
}

func (db *DB) ListSubscribers(ctx context.Context) ([]store.Subscriber, error) {
	const q = `
		SELECT id, msisdn, COALESCE(imsi,''), ims_registered, lte_attached,
		       COALESCE(mme_host,''), mwd_set, created_at, updated_at
		FROM subscribers ORDER BY msisdn`
	rows, err := db.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list subscribers: %w", err)
	}
	defer rows.Close()
	var subs []store.Subscriber
	for rows.Next() {
		s, err := scanSubscriber(rows)
		if err != nil {
			return nil, err
		}
		subs = append(subs, *s)
	}
	return subs, rows.Err()
}

func (db *DB) GetSubscriberByID(ctx context.Context, id string) (*store.Subscriber, error) {
	const q = `
		SELECT id, msisdn, COALESCE(imsi,''), ims_registered, lte_attached,
		       COALESCE(mme_host,''), mwd_set, created_at, updated_at
		FROM subscribers WHERE id = ?`
	row := db.db.QueryRowContext(ctx, q, id)
	sub, err := scanSubscriber(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get subscriber by id %s: %w", id, err)
	}
	return sub, nil
}

func (db *DB) DeleteSubscriber(ctx context.Context, id string) error {
	_, err := db.db.ExecContext(ctx, `DELETE FROM subscribers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete subscriber %s: %w", id, err)
	}
	return nil
}

func scanSubscriber(row sqlScanner) (*store.Subscriber, error) {
	var s store.Subscriber
	var imsReg, lteAtt, mwd int
	var createdStr, updatedStr string
	err := row.Scan(
		&s.ID, &s.MSISDN, &s.IMSI,
		&imsReg, &lteAtt, &s.MMEHost, &mwd,
		&createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}
	s.IMSRegistered = imsReg != 0
	s.LTEAttached = lteAtt != 0
	s.MWDSet = mwd != 0
	s.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return &s, nil
}
