package sqlite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func (db *DB) SaveMessage(ctx context.Context, msg store.Message) error {
	const q = `
		INSERT INTO messages (
			id, tp_mr, smpp_msgid, origin_iface, origin_peer,
			egress_iface, egress_peer, src_msisdn, dst_msisdn,
			payload, udh, encoding, dcs, status, retry_count,
			next_retry_at, dr_required, submitted_at, expiry_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
	_, err := db.db.ExecContext(ctx, q,
		msg.ID, msg.TPMR, msg.SMPPMsgID, msg.OriginIface, msg.OriginPeer,
		msg.EgressIface, msg.EgressPeer, msg.SrcMSISDN, msg.DstMSISDN,
		msg.Payload, msg.UDH, msg.Encoding, msg.DCS, msg.Status, msg.RetryCount,
		nullableTime(msg.NextRetryAt),
		boolInt(msg.DRRequired),
		msg.SubmittedAt.UTC().Format(time.RFC3339),
		nullableTime(msg.ExpiryAt),
	)
	if err != nil {
		return fmt.Errorf("save message %s: %w", msg.ID, err)
	}
	return nil
}

func (db *DB) UpdateMessageRouting(ctx context.Context, id, egressIface, egressPeer string) error {
	_, err := db.db.ExecContext(ctx,
		`UPDATE messages SET egress_iface=?, egress_peer=? WHERE id=?`,
		egressIface, egressPeer, id)
	if err != nil {
		return fmt.Errorf("update message routing %s: %w", id, err)
	}
	return nil
}

func (db *DB) UpdateMessageStatus(ctx context.Context, id, status string) error {
	var err error
	if status == store.MessageStatusDelivered {
		now := time.Now().UTC().Format(time.RFC3339)
		_, err = db.db.ExecContext(ctx,
			`UPDATE messages SET status=?, delivered_at=? WHERE id=?`,
			status, now, id)
	} else {
		_, err = db.db.ExecContext(ctx,
			`UPDATE messages SET status=? WHERE id=?`, status, id)
	}
	if err != nil {
		return fmt.Errorf("update message status %s: %w", id, err)
	}
	return nil
}

func (db *DB) UpdateMessageRetry(ctx context.Context, id string, retryCount int, nextRetryAt time.Time) error {
	_, err := db.db.ExecContext(ctx,
		`UPDATE messages SET status=?, retry_count=?, next_retry_at=? WHERE id=?`,
		store.MessageStatusQueued, retryCount,
		nextRetryAt.UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("update message retry %s: %w", id, err)
	}
	return nil
}

func (db *DB) ListRetryableMessages(ctx context.Context) ([]store.Message, error) {
	const q = `
		SELECT id, tp_mr, smpp_msgid, origin_iface, origin_peer,
		       egress_iface, egress_peer, src_msisdn, dst_msisdn,
		       payload, udh, encoding, dcs, status, retry_count, next_retry_at,
		       dr_required, submitted_at, expiry_at
		FROM messages
		WHERE status='QUEUED' AND (next_retry_at IS NULL OR next_retry_at <= datetime('now'))
		ORDER BY next_retry_at ASC LIMIT 100`
	return sqliteScanMessages(db, ctx, q)
}

func (db *DB) ListExpiredMessages(ctx context.Context) ([]store.Message, error) {
	const q = `
		SELECT id, tp_mr, smpp_msgid, origin_iface, origin_peer,
		       egress_iface, egress_peer, src_msisdn, dst_msisdn,
		       payload, udh, encoding, dcs, status, retry_count, next_retry_at,
		       dr_required, submitted_at, expiry_at
		FROM messages
		WHERE status='QUEUED' AND expiry_at IS NOT NULL AND expiry_at <= datetime('now')
		LIMIT 100`
	return sqliteScanMessages(db, ctx, q)
}

func (db *DB) GetMessage(ctx context.Context, id string) (*store.Message, error) {
	const q = `
		SELECT id, tp_mr, smpp_msgid, origin_iface, origin_peer,
		       egress_iface, egress_peer, src_msisdn, dst_msisdn,
		       payload, dcs, status, retry_count, next_retry_at,
		       dr_required, submitted_at, expiry_at
		FROM messages WHERE id=?`
	msgs, err := sqliteScanMessages(db, ctx, q, id)
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return nil, nil
	}
	return &msgs[0], nil
}

func (db *DB) DeleteMessage(ctx context.Context, id string) error {
	_, err := db.db.ExecContext(ctx, `DELETE FROM messages WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("delete message %s: %w", id, err)
	}
	return nil
}

func sqliteScanMessages(db *DB, ctx context.Context, q string, args ...any) ([]store.Message, error) {
	rows, err := db.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var msgs []store.Message
	for rows.Next() {
		var m store.Message
		var drInt int
		var nextRetryStr, submittedStr, expiryStr *string
		var tpmr *int
		var encoding *int
		var dcs *int

		err := rows.Scan(
			&m.ID, &tpmr, &m.SMPPMsgID, &m.OriginIface, &m.OriginPeer,
			&m.EgressIface, &m.EgressPeer, &m.SrcMSISDN, &m.DstMSISDN,
			&m.Payload, &m.UDH, &encoding, &dcs, &m.Status,
			&m.RetryCount, &nextRetryStr,
			&drInt, &submittedStr, &expiryStr,
		)
		if err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		m.TPMR = tpmr
		if encoding != nil {
			m.Encoding = *encoding
		}
		if dcs != nil {
			m.DCS = *dcs
		}
		m.DRRequired = drInt != 0
		if submittedStr != nil {
			m.SubmittedAt, _ = time.Parse(time.RFC3339, *submittedStr)
		}
		if nextRetryStr != nil {
			t, _ := time.Parse(time.RFC3339, *nextRetryStr)
			m.NextRetryAt = &t
		}
		if expiryStr != nil {
			t, _ := time.Parse(time.RFC3339, *expiryStr)
			m.ExpiryAt = &t
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (db *DB) CountMessagesByStatus(ctx context.Context) (map[string]int64, error) {
	rows, err := db.db.QueryContext(ctx,
		`SELECT status, COUNT(*) FROM messages GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("count messages by status: %w", err)
	}
	defer rows.Close()
	counts := make(map[string]int64)
	for rows.Next() {
		var status string
		var n int64
		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}
		counts[status] = n
	}
	return counts, rows.Err()
}

func (db *DB) ListMessages(ctx context.Context, limit int) ([]store.Message, error) {
	if limit <= 0 {
		limit = 100
	}
	const q = `
		SELECT id, tp_mr, smpp_msgid, origin_iface, origin_peer,
		       egress_iface, egress_peer, src_msisdn, dst_msisdn,
		       payload, udh, encoding, dcs, status, retry_count, next_retry_at,
		       dr_required, submitted_at, expiry_at
	FROM messages ORDER BY submitted_at DESC LIMIT ?`
	return sqliteScanMessages(db, ctx, q, limit)
}

func (db *DB) ListFilteredMessages(ctx context.Context, filter store.MessageFilter) ([]store.Message, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}

	conds := make([]string, 0, 4)
	args := make([]any, 0, 8)

	if len(filter.Statuses) > 0 {
		placeholders := make([]string, 0, len(filter.Statuses))
		for _, status := range filter.Statuses {
			if status == "" {
				continue
			}
			placeholders = append(placeholders, "?")
			args = append(args, status)
		}
		if len(placeholders) > 0 {
			conds = append(conds, "status IN ("+strings.Join(placeholders, ",")+")")
		}
	}
	if filter.SrcMSISDN != "" {
		conds = append(conds, "src_msisdn LIKE ?")
		args = append(args, "%"+filter.SrcMSISDN+"%")
	}
	if filter.DstMSISDN != "" {
		conds = append(conds, "dst_msisdn LIKE ?")
		args = append(args, "%"+filter.DstMSISDN+"%")
	}
	if filter.OriginPeer != "" {
		conds = append(conds, "origin_peer LIKE ?")
		args = append(args, "%"+filter.OriginPeer+"%")
	}

	q := `
		SELECT id, tp_mr, smpp_msgid, origin_iface, origin_peer,
		       egress_iface, egress_peer, src_msisdn, dst_msisdn,
		       payload, dcs, status, retry_count, next_retry_at,
		       dr_required, submitted_at, expiry_at
		FROM messages`
	if len(conds) > 0 {
		q += "\n\t\tWHERE " + strings.Join(conds, " AND ")
	}
	q += "\n\t\tORDER BY COALESCE(next_retry_at, submitted_at) ASC, submitted_at DESC LIMIT ?"
	args = append(args, limit)
	return sqliteScanMessages(db, ctx, q, args...)
}

func (db *DB) ListDeliveryReports(ctx context.Context, limit int) ([]store.DeliveryReport, error) {
	if limit <= 0 {
		limit = 100
	}
	const q = `
		SELECT id, message_id, status, egress_iface, COALESCE(raw_receipt,''), reported_at
		FROM delivery_reports ORDER BY reported_at DESC LIMIT ?`
	return sqliteScanDeliveryReports(db, ctx, q, limit)
}

func (db *DB) GetDeliveryReport(ctx context.Context, id string) (*store.DeliveryReport, error) {
	const q = `
		SELECT id, message_id, status, egress_iface, COALESCE(raw_receipt,''), reported_at
		FROM delivery_reports WHERE id = ?`
	drs, err := sqliteScanDeliveryReports(db, ctx, q, id)
	if err != nil {
		return nil, err
	}
	if len(drs) == 0 {
		return nil, nil
	}
	return &drs[0], nil
}

func sqliteScanDeliveryReports(db *DB, ctx context.Context, q string, args ...any) ([]store.DeliveryReport, error) {
	rows, err := db.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query delivery_reports: %w", err)
	}
	defer rows.Close()
	var drs []store.DeliveryReport
	for rows.Next() {
		var dr store.DeliveryReport
		var reportedStr string
		if err := rows.Scan(&dr.ID, &dr.MessageID, &dr.Status, &dr.EgressIface, &dr.RawReceipt, &reportedStr); err != nil {
			return nil, fmt.Errorf("scan delivery_report: %w", err)
		}
		dr.ReportedAt, _ = time.Parse(time.RFC3339, reportedStr)
		drs = append(drs, dr)
	}
	return drs, rows.Err()
}

func (db *DB) SaveDeliveryReport(ctx context.Context, dr store.DeliveryReport) error {
	const q = `
		INSERT INTO delivery_reports (id, message_id, status, egress_iface, raw_receipt, reported_at)
		VALUES (?,?,?,?,?,?)`
	_, err := db.db.ExecContext(ctx, q,
		dr.ID, dr.MessageID, dr.Status, dr.EgressIface, dr.RawReceipt,
		dr.ReportedAt.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save delivery_report: %w", err)
	}
	return nil
}

// nullableTime converts a *time.Time to a string for SQLite, or nil.
func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}
