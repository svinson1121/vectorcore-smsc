package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func (db *DB) ListAllRoutingRules(ctx context.Context) ([]store.RoutingRule, error) {
	const q = `
		SELECT id, name, priority,
		       COALESCE(match_src_iface,''), COALESCE(match_src_peer,''),
		       COALESCE(match_dst_prefix,''), COALESCE(match_msisdn_min,''), COALESCE(match_msisdn_max,''),
		       egress_iface, COALESCE(egress_peer,''), COALESCE(sf_policy_id,''),
		       enabled, created_at, updated_at
		FROM routing_rules ORDER BY priority ASC`
	rows, err := db.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list all routing_rules: %w", err)
	}
	defer rows.Close()
	var rules []store.RoutingRule
	for rows.Next() {
		var r store.RoutingRule
		var enabledInt int
		var createdStr, updatedStr string
		if err := rows.Scan(
			&r.ID, &r.Name, &r.Priority,
			&r.MatchSrcIface, &r.MatchSrcPeer,
			&r.MatchDstPrefix, &r.MatchMSISDNMin, &r.MatchMSISDNMax,
			&r.EgressIface, &r.EgressPeer, &r.SFPolicyID,
			&enabledInt, &createdStr, &updatedStr,
		); err != nil {
			return nil, fmt.Errorf("scan routing_rule: %w", err)
		}
		r.Enabled = enabledInt != 0
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		r.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

func (db *DB) GetRoutingRule(ctx context.Context, id string) (*store.RoutingRule, error) {
	const q = `
		SELECT id, name, priority,
		       COALESCE(match_src_iface,''), COALESCE(match_src_peer,''),
		       COALESCE(match_dst_prefix,''), COALESCE(match_msisdn_min,''), COALESCE(match_msisdn_max,''),
		       egress_iface, COALESCE(egress_peer,''), COALESCE(sf_policy_id,''),
		       enabled, created_at, updated_at
		FROM routing_rules WHERE id = ?`
	var r store.RoutingRule
	var enabledInt int
	var createdStr, updatedStr string
	err := db.db.QueryRowContext(ctx, q, id).Scan(
		&r.ID, &r.Name, &r.Priority,
		&r.MatchSrcIface, &r.MatchSrcPeer,
		&r.MatchDstPrefix, &r.MatchMSISDNMin, &r.MatchMSISDNMax,
		&r.EgressIface, &r.EgressPeer, &r.SFPolicyID,
		&enabledInt, &createdStr, &updatedStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get routing_rule %s: %w", id, err)
	}
	r.Enabled = enabledInt != 0
	r.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	r.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return &r, nil
}

func (db *DB) CreateRoutingRule(ctx context.Context, r store.RoutingRule) error {
	const q = `
		INSERT INTO routing_rules
			(id, name, priority, match_src_iface, match_src_peer,
			 match_dst_prefix, match_msisdn_min, match_msisdn_max,
			 egress_iface, egress_peer, sf_policy_id, enabled)
		VALUES (?,?,?,NULLIF(?,''),NULLIF(?,''),NULLIF(?,''),NULLIF(?,''),NULLIF(?,''),?,NULLIF(?,''),NULLIF(?,''),?)`
	_, err := db.db.ExecContext(ctx, q,
		newUUID(), r.Name, r.Priority,
		r.MatchSrcIface, r.MatchSrcPeer,
		r.MatchDstPrefix, r.MatchMSISDNMin, r.MatchMSISDNMax,
		r.EgressIface, r.EgressPeer, r.SFPolicyID, boolInt(r.Enabled))
	if err != nil {
		return fmt.Errorf("create routing_rule: %w", err)
	}
	return nil
}

func (db *DB) UpdateRoutingRule(ctx context.Context, r store.RoutingRule) error {
	const q = `
		UPDATE routing_rules SET
			name             = ?, priority         = ?,
			match_src_iface  = NULLIF(?,''), match_src_peer   = NULLIF(?,''),
			match_dst_prefix = NULLIF(?,''), match_msisdn_min = NULLIF(?,''),
			match_msisdn_max = NULLIF(?,''), egress_iface     = ?,
			egress_peer   = NULLIF(?,''), sf_policy_id     = NULLIF(?,''),
			enabled          = ?,            updated_at       = datetime('now')
		WHERE id = ?`
	_, err := db.db.ExecContext(ctx, q,
		r.Name, r.Priority,
		r.MatchSrcIface, r.MatchSrcPeer,
		r.MatchDstPrefix, r.MatchMSISDNMin, r.MatchMSISDNMax,
		r.EgressIface, r.EgressPeer, r.SFPolicyID,
		boolInt(r.Enabled), r.ID)
	if err != nil {
		return fmt.Errorf("update routing_rule %s: %w", r.ID, err)
	}
	return nil
}

func (db *DB) DeleteRoutingRule(ctx context.Context, id string) error {
	_, err := db.db.ExecContext(ctx, `DELETE FROM routing_rules WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete routing_rule %s: %w", id, err)
	}
	return nil
}

func (db *DB) ListSFPolicies(ctx context.Context) ([]store.SFPolicy, error) {
	const q = `
		SELECT id, name, max_retries, retry_schedule, max_ttl,
		       COALESCE(vp_override,''), created_at, updated_at
		FROM sf_policies ORDER BY name`
	rows, err := db.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list sf_policies: %w", err)
	}
	defer rows.Close()
	var policies []store.SFPolicy
	for rows.Next() {
		var p store.SFPolicy
		var schedJSON, maxTTLStr, vpOverrideStr, createdStr, updatedStr string
		if err := rows.Scan(
			&p.ID, &p.Name, &p.MaxRetries, &schedJSON, &maxTTLStr,
			&vpOverrideStr, &createdStr, &updatedStr,
		); err != nil {
			return nil, fmt.Errorf("scan sf_policy: %w", err)
		}
		if err := json.Unmarshal([]byte(schedJSON), &p.RetrySchedule); err != nil {
			return nil, fmt.Errorf("parse retry_schedule: %w", err)
		}
		p.MaxTTL = parseSQLiteInterval(maxTTLStr)
		if vpOverrideStr != "" {
			d := parseSQLiteInterval(vpOverrideStr)
			p.VPOverride = &d
		}
		p.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

func (db *DB) CreateSFPolicy(ctx context.Context, p store.SFPolicy) error {
	sched, _ := json.Marshal(p.RetrySchedule)
	vpStr := ""
	if p.VPOverride != nil {
		vpStr = p.VPOverride.String()
	}
	const q = `
		INSERT INTO sf_policies (id, name, max_retries, retry_schedule, max_ttl, vp_override)
		VALUES (?, ?, ?, ?, ?, NULLIF(?,''))`
	_, err := db.db.ExecContext(ctx, q,
		newUUID(), p.Name, p.MaxRetries, string(sched), p.MaxTTL.String(), vpStr)
	if err != nil {
		return fmt.Errorf("create sf_policy: %w", err)
	}
	return nil
}

func (db *DB) UpdateSFPolicy(ctx context.Context, p store.SFPolicy) error {
	sched, _ := json.Marshal(p.RetrySchedule)
	vpStr := ""
	if p.VPOverride != nil {
		vpStr = p.VPOverride.String()
	}
	const q = `
		UPDATE sf_policies SET
			name           = ?,
			max_retries    = ?,
			retry_schedule = ?,
			max_ttl        = ?,
			vp_override    = NULLIF(?,''),
			updated_at     = datetime('now')
		WHERE id = ?`
	_, err := db.db.ExecContext(ctx, q,
		p.Name, p.MaxRetries, string(sched), p.MaxTTL.String(), vpStr, p.ID)
	if err != nil {
		return fmt.Errorf("update sf_policy %s: %w", p.ID, err)
	}
	return nil
}

func (db *DB) DeleteSFPolicy(ctx context.Context, id string) error {
	_, err := db.db.ExecContext(ctx, `DELETE FROM sf_policies WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete sf_policy %s: %w", id, err)
	}
	return nil
}

func (db *DB) ListRoutingRules(ctx context.Context) ([]store.RoutingRule, error) {
	const q = `
		SELECT id, name, priority,
		       COALESCE(match_src_iface,''), COALESCE(match_src_peer,''),
		       COALESCE(match_dst_prefix,''), COALESCE(match_msisdn_min,''), COALESCE(match_msisdn_max,''),
		       egress_iface, COALESCE(egress_peer,''), COALESCE(sf_policy_id,''),
		       enabled, created_at, updated_at
		FROM routing_rules WHERE enabled = 1 ORDER BY priority ASC`

	rows, err := db.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list routing_rules: %w", err)
	}
	defer rows.Close()

	var rules []store.RoutingRule
	for rows.Next() {
		var r store.RoutingRule
		var enabledInt int
		var createdStr, updatedStr string
		err := rows.Scan(
			&r.ID, &r.Name, &r.Priority,
			&r.MatchSrcIface, &r.MatchSrcPeer,
			&r.MatchDstPrefix, &r.MatchMSISDNMin, &r.MatchMSISDNMax,
			&r.EgressIface, &r.EgressPeer, &r.SFPolicyID,
			&enabledInt, &createdStr, &updatedStr,
		)
		if err != nil {
			return nil, fmt.Errorf("scan routing_rule: %w", err)
		}
		r.Enabled = enabledInt != 0
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		r.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

func (db *DB) GetSFPolicy(ctx context.Context, id string) (*store.SFPolicy, error) {
	const q = `
		SELECT id, name, max_retries, retry_schedule, max_ttl,
		       COALESCE(vp_override,''), created_at, updated_at
		FROM sf_policies WHERE id = ?`

	row := db.db.QueryRowContext(ctx, q, id)
	var p store.SFPolicy
	var schedJSON, maxTTLStr, vpOverrideStr string
	var createdStr, updatedStr string

	err := row.Scan(
		&p.ID, &p.Name, &p.MaxRetries, &schedJSON, &maxTTLStr,
		&vpOverrideStr, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, fmt.Errorf("get sf_policy %s: %w", id, err)
	}

	if err := json.Unmarshal([]byte(schedJSON), &p.RetrySchedule); err != nil {
		return nil, fmt.Errorf("parse retry_schedule: %w", err)
	}

	p.MaxTTL = parseSQLiteInterval(maxTTLStr)
	if vpOverrideStr != "" {
		d := parseSQLiteInterval(vpOverrideStr)
		p.VPOverride = &d
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return &p, nil
}
