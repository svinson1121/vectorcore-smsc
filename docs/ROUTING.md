# Routing Logic

This document describes the real routing behavior used by the forwarder and routing engine.

## High-Level Flow

Every inbound message goes through:

1. Ingress decode to canonical `codec.Message`
2. Forwarder egress selection
3. Persist + dispatch attempt
4. Success finalize or retry scheduling

## Egress Selection Precedence

The forwarder chooses egress in this order:

1. IMS registration check (Sh lookup)
- If destination MSISDN is currently IMS-registered, route to:
  - `egress_iface = sip3gpp`
  - `egress_peer = registration.s_cscf`

2. LTE attach check (subscriber state)
- If subscriber exists with `lte_attached=true` and `mme_host` set, route to:
  - `egress_iface = sgd`
  - `egress_peer = mme_host`

3. Routing rules engine
- Evaluate configured rules in priority order (lowest number first)
- First match wins
- Use rule-defined:
  - `egress_iface`
  - `egress_peer`
  - `sf_policy_id`

If no decision is found, message is persisted as `FAILED`.

## Rule Matching

A rule matches when all non-empty match fields pass:

- `match_src_iface` equals ingress interface
- `match_src_peer` equals ingress peer
- `match_dst_prefix` is a prefix of destination MSISDN
- `match_msisdn_min` and `match_msisdn_max` bound destination MSISDN

Notes:

- Leading `+` is stripped before prefix/range comparisons.
- Range comparison is string comparison on normalized MSISDN digits.
- Rules are hot-reloaded from DB when `routing_rules` changes.

## Persistence and Dispatch Lifecycle

For each message:

1. Assign internal UUID if missing
2. Save `messages` row as `QUEUED`
3. Update status to `DISPATCHED`
4. Attempt network delivery on selected egress
5. On success, mark `DELIVERED`
6. On failure, schedule retry (`next_retry_at`, `retry_count`)

## Store-and-Forward Policies

`sf_policy_id` from matched routing rule controls retry behavior.

Policy fields:

- `max_retries`
- `retry_schedule` (seconds between attempts)
- `max_ttl`
- optional `vp_override`

If no policy is set, default retry schedule is used.

## Queue Semantics in Metrics/UI

Current queue depth is treated as in-flight backlog:

- `QUEUED + DISPATCHED`

This is reflected in Prometheus queue gauge and dashboard queue card.

## Egress Peer Semantics

- `sip3gpp`: peer is managed by live IMS registration (`s_cscf`), UI peer field is informational
- `sipsimple`: peer name must exist in SIP peers table
- `smpp`: peer name references SMPP outbound client session
- `sgd`: peer name references Diameter SGd peer/MME host

