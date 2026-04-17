# Routing Logic

This document describes the current routing behavior implemented by the SMSC forwarder.

## High-Level Flow

Every inbound message goes through:

1. Ingress decode to canonical `codec.Message`
2. Persist as a queued message
3. Single routing pass across built-in and fallback candidates
4. Finalize as `DELIVERED`, `WAIT_TIMER`, `WAIT_EVENT`, `WAIT_TIMER_EVENT`, `FAILED`, or later `EXPIRED`

## Built-In Route Order

The forwarder evaluates MT delivery in this order:

1. `ims-local`
   - use locally cached IMS registration
   - route to `sip3gpp` if the subscriber is already registered locally

2. `ims-sh`
   - use a live `Sh` lookup
   - route to `sip3gpp` if the subscriber is IMS-registered via `Sh`
   - skipped if no usable `Sh` Diameter path exists

3. `sgd`
   - built-in route, not a routing rule
   - uses `S6c` lookup to get serving MME state, then applies the existing `S6c -> SGd` MME mapping logic, then sends over `SGd`
   - skipped unless both required Diameter capabilities are usable:
     - `S6c`
     - `SGd`

4. Fallback routing rules
   - evaluated in priority order
   - only `smpp` and `sipsimple` are valid routing-rule egress interfaces

## Routing Rules

Routing rules are now fallback-only policy.

Valid `egress_iface` values:

- `smpp`
- `sipsimple`

`sgd` is no longer a routing-rule option. Legacy `sgd` rules are treated as obsolete and are removed during startup migration/cleanup.

A rule matches when all non-empty match fields pass:

- `match_src_iface` equals ingress interface
- `match_src_peer` equals ingress peer
- `match_dst_prefix` is a prefix of destination MSISDN
- `match_msisdn_min` and `match_msisdn_max` bound destination MSISDN

Notes:

- leading `+` is stripped before prefix/range comparisons
- range comparison is string comparison on normalized MSISDN digits
- rules are hot-reloaded from DB when `routing_rules` changes

## Message State Model

Persisted message statuses are:

- `QUEUED`
- `DISPATCHED`
- `WAIT_TIMER`
- `WAIT_EVENT`
- `WAIT_TIMER_EVENT`
- `DELIVERED`
- `FAILED`
- `EXPIRED`

These are real stored statuses, not derived UI-only labels.

## Routing Outcomes

Each route attempt is classified internally as one of:

- `DELIVERED`
- `TRY_NEXT`
- `WAIT_TIMER`
- `WAIT_EVENT`
- `ROUTE_PERMANENT`

The pass finalization precedence is:

1. any route delivers -> `DELIVERED`
2. else any event wait -> `WAIT_EVENT` or `WAIT_TIMER_EVENT`
3. else any timer-needed outcome -> `WAIT_TIMER`
4. else -> `FAILED`

Delivery is global. If a later route delivers, older deferred state is stale and later alert/timer wakeups are ignored.

## Trigger Ownership

Retry scheduler and alert-triggered wakeups use guarded store transitions:

- retry can only claim messages still in retryable waiting states
- alert requeue only updates messages still in queued/waiting states
- stale `ALR` after delivery or after another worker has claimed the message is skipped

This prevents timer and event triggers from dispatching the same message twice in parallel.

## Store-and-Forward Policies

`sf_policy_id` from a matched fallback rule controls retry and lifetime behavior for that route.

Policy fields:

- `max_retries`
- `retry_schedule`
- `max_ttl`
- optional `vp_override`

Semantics:

- `max_ttl` overrides the global `smsc.max_queue_lifetime` for that route when the message enters a waiting state
- `vp_override` is currently future-capable for full wire-level validity handling; today it is used as an internal validity/lifetime hint rather than a guaranteed on-the-wire validity field across all MT routes
- `vp_override` also acts as an earlier expiry cap for deferred queue lifetime if it is shorter than the other lifetime cap

If no policy is set, the forwarder uses the built-in retry schedule and the global `smsc.max_queue_lifetime`.

## Queue Lifetime

Global queue lifetime is configured by:

- `smsc.max_queue_lifetime`

Current default:

- `168h` (`7 days`)

Expiry is capped from original submission time, not extended by retries.

## Queue Semantics in Metrics/UI

Current queue depth is treated as backlog across active and waiting states:

- `QUEUED`
- `DISPATCHED`
- `WAIT_TIMER`
- `WAIT_EVENT`
- `WAIT_TIMER_EVENT`

## Egress Peer Semantics

- `sip3gpp`: peer is derived from IMS registration / `Sh`
- `sipsimple`: peer name must exist in SIP peers table
- `smpp`: peer name references an SMPP outbound client session
- `sgd`: peer is derived from `S6c` serving-node data plus the SMSC's `S6c -> SGd` mapping logic
