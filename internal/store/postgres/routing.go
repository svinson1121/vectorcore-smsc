package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func (db *DB) ListAllRoutingRules(ctx context.Context) ([]store.RoutingRule, error) {
	const q = `
		SELECT id, name, priority,
		       COALESCE(match_src_iface,''), COALESCE(match_src_peer,''),
		       COALESCE(match_dst_prefix,''), COALESCE(match_msisdn_min,''), COALESCE(match_msisdn_max,''),
		       egress_iface, COALESCE(egress_peer,''), COALESCE(sf_policy_id::text,''),
		       enabled, created_at, updated_at
		FROM routing_rules ORDER BY priority ASC`
	rows, err := db.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list all routing_rules: %w", err)
	}
	defer rows.Close()
	var rules []store.RoutingRule
	for rows.Next() {
		var r store.RoutingRule
		err := rows.Scan(
			&r.ID, &r.Name, &r.Priority,
			&r.MatchSrcIface, &r.MatchSrcPeer,
			&r.MatchDstPrefix, &r.MatchMSISDNMin, &r.MatchMSISDNMax,
			&r.EgressIface, &r.EgressPeer, &r.SFPolicyID,
			&r.Enabled, &r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan routing_rule: %w", err)
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

func (db *DB) GetRoutingRule(ctx context.Context, id string) (*store.RoutingRule, error) {
	const q = `
		SELECT id, name, priority,
		       COALESCE(match_src_iface,''), COALESCE(match_src_peer,''),
		       COALESCE(match_dst_prefix,''), COALESCE(match_msisdn_min,''), COALESCE(match_msisdn_max,''),
		       egress_iface, COALESCE(egress_peer,''), COALESCE(sf_policy_id::text,''),
		       enabled, created_at, updated_at
		FROM routing_rules WHERE id = $1::uuid`
	row := db.pool.QueryRow(ctx, q, id)
	var r store.RoutingRule
	err := row.Scan(
		&r.ID, &r.Name, &r.Priority,
		&r.MatchSrcIface, &r.MatchSrcPeer,
		&r.MatchDstPrefix, &r.MatchMSISDNMin, &r.MatchMSISDNMax,
		&r.EgressIface, &r.EgressPeer, &r.SFPolicyID,
		&r.Enabled, &r.CreatedAt, &r.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get routing_rule %s: %w", id, err)
	}
	return &r, nil
}

func (db *DB) CreateRoutingRule(ctx context.Context, r store.RoutingRule) error {
	const q = `
		INSERT INTO routing_rules
			(id, name, priority, match_src_iface, match_src_peer,
			 match_dst_prefix, match_msisdn_min, match_msisdn_max,
			 egress_iface, egress_peer, sf_policy_id, enabled)
		VALUES
			(gen_random_uuid(), $1, $2, NULLIF($3,''), NULLIF($4,''),
			 NULLIF($5,''), NULLIF($6,''), NULLIF($7,''),
			 $8, NULLIF($9,''), NULLIF($10,'')::uuid, $11)`
	_, err := db.pool.Exec(ctx, q,
		r.Name, r.Priority, r.MatchSrcIface, r.MatchSrcPeer,
		r.MatchDstPrefix, r.MatchMSISDNMin, r.MatchMSISDNMax,
		r.EgressIface, r.EgressPeer, r.SFPolicyID, r.Enabled)
	if err != nil {
		return fmt.Errorf("create routing_rule: %w", err)
	}
	return nil
}

func (db *DB) UpdateRoutingRule(ctx context.Context, r store.RoutingRule) error {
	const q = `
		UPDATE routing_rules SET
			name             = $2,  priority         = $3,
			match_src_iface  = NULLIF($4,''),  match_src_peer   = NULLIF($5,''),
			match_dst_prefix = NULLIF($6,''),  match_msisdn_min = NULLIF($7,''),
			match_msisdn_max = NULLIF($8,''),  egress_iface     = $9,
			egress_peer      = NULLIF($10,''),
			sf_policy_id     = NULLIF($11,'')::uuid,
			enabled          = $12, updated_at       = now()
		WHERE id = $1::uuid`
	_, err := db.pool.Exec(ctx, q,
		r.ID, r.Name, r.Priority, r.MatchSrcIface, r.MatchSrcPeer,
		r.MatchDstPrefix, r.MatchMSISDNMin, r.MatchMSISDNMax,
		r.EgressIface, r.EgressPeer, r.SFPolicyID, r.Enabled)
	if err != nil {
		return fmt.Errorf("update routing_rule %s: %w", r.ID, err)
	}
	return nil
}

func (db *DB) DeleteRoutingRule(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM routing_rules WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete routing_rule %s: %w", id, err)
	}
	return nil
}

func (db *DB) ListSFPolicies(ctx context.Context) ([]store.SFPolicy, error) {
	const q = `
		SELECT id, name, max_retries, retry_schedule, max_ttl,
		       COALESCE(vp_override, '0') AS vp_override_str,
		       (vp_override IS NOT NULL) AS has_vp_override,
		       created_at, updated_at
		FROM sf_policies ORDER BY name`
	rows, err := db.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list sf_policies: %w", err)
	}
	defer rows.Close()
	var policies []store.SFPolicy
	for rows.Next() {
		var p store.SFPolicy
		var schedJSON []byte
		var maxTTL pgtype.Interval
		var vpOverride pgtype.Interval
		var hasVPOverride bool
		err := rows.Scan(
			&p.ID, &p.Name, &p.MaxRetries, &schedJSON, &maxTTL,
			&vpOverride, &hasVPOverride,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan sf_policy: %w", err)
		}
		if err := json.Unmarshal(schedJSON, &p.RetrySchedule); err != nil {
			return nil, fmt.Errorf("parse retry_schedule: %w", err)
		}
		p.MaxTTL, err = intervalToDuration(maxTTL)
		if err != nil {
			return nil, fmt.Errorf("scan sf_policy max_ttl: %w", err)
		}
		if hasVPOverride {
			d, err := intervalToDuration(vpOverride)
			if err != nil {
				return nil, fmt.Errorf("scan sf_policy vp_override: %w", err)
			}
			p.VPOverride = &d
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

func (db *DB) CreateSFPolicy(ctx context.Context, p store.SFPolicy) error {
	sched, _ := json.Marshal(p.RetrySchedule)
	maxTTL := formatInterval(p.MaxTTL)
	var vpOverride interface{}
	if p.VPOverride != nil {
		vpOverride = formatInterval(*p.VPOverride)
	}
	const q = `
		INSERT INTO sf_policies (id, name, max_retries, retry_schedule, max_ttl, vp_override)
		VALUES (gen_random_uuid(), $1, $2, $3, $4::interval, $5::interval)`
	_, err := db.pool.Exec(ctx, q, p.Name, p.MaxRetries, sched, maxTTL, vpOverride)
	if err != nil {
		return fmt.Errorf("create sf_policy: %w", err)
	}
	return nil
}

func (db *DB) UpdateSFPolicy(ctx context.Context, p store.SFPolicy) error {
	sched, _ := json.Marshal(p.RetrySchedule)
	maxTTL := formatInterval(p.MaxTTL)
	var vpOverride interface{}
	if p.VPOverride != nil {
		vpOverride = formatInterval(*p.VPOverride)
	}
	const q = `
		UPDATE sf_policies SET
			name           = $2,
			max_retries    = $3,
			retry_schedule = $4,
			max_ttl        = $5::interval,
			vp_override    = $6::interval,
			updated_at     = now()
		WHERE id = $1::uuid`
	_, err := db.pool.Exec(ctx, q, p.ID, p.Name, p.MaxRetries, sched, maxTTL, vpOverride)
	if err != nil {
		return fmt.Errorf("update sf_policy %s: %w", p.ID, err)
	}
	return nil
}

func (db *DB) DeleteSFPolicy(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM sf_policies WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete sf_policy %s: %w", id, err)
	}
	return nil
}

// keep unused import happy
var _ = time.Second

func (db *DB) ListRoutingRules(ctx context.Context) ([]store.RoutingRule, error) {
	const q = `
		SELECT id, name, priority,
		       COALESCE(match_src_iface,''), COALESCE(match_src_peer,''),
		       COALESCE(match_dst_prefix,''), COALESCE(match_msisdn_min,''), COALESCE(match_msisdn_max,''),
		       egress_iface, COALESCE(egress_peer,''), COALESCE(sf_policy_id::text,''),
		       enabled, created_at, updated_at
		FROM routing_rules
		WHERE enabled = true
		ORDER BY priority ASC`

	rows, err := db.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list routing_rules: %w", err)
	}
	defer rows.Close()

	var rules []store.RoutingRule
	for rows.Next() {
		var r store.RoutingRule
		err := rows.Scan(
			&r.ID, &r.Name, &r.Priority,
			&r.MatchSrcIface, &r.MatchSrcPeer,
			&r.MatchDstPrefix, &r.MatchMSISDNMin, &r.MatchMSISDNMax,
			&r.EgressIface, &r.EgressPeer, &r.SFPolicyID,
			&r.Enabled, &r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan routing_rule: %w", err)
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

func (db *DB) GetSFPolicy(ctx context.Context, id string) (*store.SFPolicy, error) {
	const q = `
		SELECT id, name, max_retries, retry_schedule, max_ttl,
		       COALESCE(vp_override, '0') AS vp_override_str,
		       (vp_override IS NOT NULL) AS has_vp_override,
		       created_at, updated_at
		FROM sf_policies WHERE id = $1`

	row := db.pool.QueryRow(ctx, q, id)
	var p store.SFPolicy
	var schedJSON []byte
	var maxTTL pgtype.Interval
	var vpOverride pgtype.Interval
	var hasVPOverride bool

	err := row.Scan(
		&p.ID, &p.Name, &p.MaxRetries, &schedJSON, &maxTTL,
		&vpOverride, &hasVPOverride,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get sf_policy %s: %w", id, err)
	}

	if err := json.Unmarshal(schedJSON, &p.RetrySchedule); err != nil {
		return nil, fmt.Errorf("parse retry_schedule: %w", err)
	}

	p.MaxTTL, err = intervalToDuration(maxTTL)
	if err != nil {
		return nil, fmt.Errorf("scan sf_policy max_ttl: %w", err)
	}
	if hasVPOverride {
		d, err := intervalToDuration(vpOverride)
		if err != nil {
			return nil, fmt.Errorf("scan sf_policy vp_override: %w", err)
		}
		p.VPOverride = &d
	}
	return &p, nil
}
