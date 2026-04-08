package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func (db *DB) GetSubscriber(ctx context.Context, msisdn string) (*store.Subscriber, error) {
	const q = `
		SELECT id, msisdn, COALESCE(imsi,''), ims_registered, lte_attached,
		       COALESCE(mme_host,''), mwd_set, created_at, updated_at
		FROM subscribers
		WHERE msisdn = $1`

	row := db.pool.QueryRow(ctx, q, msisdn)
	sub, err := scanSubscriber(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get subscriber %s: %w", msisdn, err)
	}
	return sub, nil
}

func (db *DB) UpsertSubscriber(ctx context.Context, sub store.Subscriber) error {
	const q = `
		INSERT INTO subscribers (id, msisdn, imsi, ims_registered, lte_attached, mme_host, mwd_set)
		VALUES (COALESCE(NULLIF($1,''), gen_random_uuid()::text)::uuid, $2,
		        NULLIF($3,''), $4, $5, NULLIF($6,''), $7)
		ON CONFLICT (msisdn) DO UPDATE SET
			imsi           = EXCLUDED.imsi,
			ims_registered = EXCLUDED.ims_registered,
			lte_attached   = EXCLUDED.lte_attached,
			mme_host       = EXCLUDED.mme_host,
			mwd_set        = EXCLUDED.mwd_set,
			updated_at     = now()`

	_, err := db.pool.Exec(ctx, q,
		sub.ID, sub.MSISDN, sub.IMSI,
		sub.IMSRegistered, sub.LTEAttached, sub.MMEHost, sub.MWDSet)
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
	rows, err := db.pool.Query(ctx, q)
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
		FROM subscribers WHERE id = $1::uuid`
	row := db.pool.QueryRow(ctx, q, id)
	sub, err := scanSubscriber(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get subscriber by id %s: %w", id, err)
	}
	return sub, nil
}

func (db *DB) DeleteSubscriber(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM subscribers WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete subscriber %s: %w", id, err)
	}
	return nil
}

func scanSubscriber(row scanner) (*store.Subscriber, error) {
	var s store.Subscriber
	err := row.Scan(
		&s.ID, &s.MSISDN, &s.IMSI,
		&s.IMSRegistered, &s.LTEAttached, &s.MMEHost, &s.MWDSet,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
