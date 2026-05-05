# Routing Logic

This document describes the routing order and deferred-delivery behavior implemented by the current forwarder.

## High-Level Flow

Every inbound message goes through this sequence:

1. decode into canonical `codec.Message`
2. persist the new record to the message store as `DISPATCHED`
3. run the initial routing pass
4. finalize as `DELIVERED`, `WAIT_TIMER`, `WAIT_EVENT`, `WAIT_TIMER_EVENT`, or `FAILED`
5. later expire as `EXPIRED` if the queue lifetime cap is reached

## Candidate Order

The forwarder always evaluates candidates in this order:

1. `ims-local`
   - uses the local IMS registration cache
   - routes to `sip3gpp` when the destination subscriber is already registered locally

2. `ims-sh`
   - performs a live `Sh` refresh through the active `Sh` Diameter client
   - routes to `sip3gpp` when `Sh` confirms IMS registration and returns a usable S-CSCF

3. `sgd-built-in`
   - not a routing rule
   - performs an `S6c` lookup
   - requires LTE attach state, IMSI, and `MME-Number-for-MT-SMS`
   - applies the configured `S6c -> SGd` MME hostname mapping before sending over `SGd`

4. fallback routing rules
   - evaluated in ascending `priority`
   - only enabled rules are used at runtime
   - only `smpp` and `sipsimple` are valid routing-rule egress interfaces

## Routing Rules

Routing rules are fallback-only policy. The API accepts:

- `egress_iface: "smpp"`
- `egress_iface: "sipsimple"`

`sgd` is no longer a routing-rule option. Legacy `sgd` rules are deleted during store startup cleanup, and the runtime loader ignores any non-`smpp`/`sipsimple` fallback rule that still appears.

A rule matches when every non-empty match field succeeds:

- `match_src_iface`
- `match_src_peer`
- `match_dst_prefix`
- `match_msisdn_min`
- `match_msisdn_max`

Matching details:

- leading `+` is stripped before prefix and range checks
- MSISDN range comparison is string comparison on normalized digits
- rules are hot-reloaded when `routing_rules` changes in the database

## Route Outcome Classification

Each route attempt is classified internally as one of:

- delivered
- try next candidate
- wait for timer
- wait for external event
- permanent failure

Current behavior by interface:

- `sip3gpp` and `sipsimple`
  - SIP `404` is treated as a permanent failure for that candidate
  - other send errors become timer-based retry candidates
- `smpp`
  - send failures are retried on timer
- `sgd`
  - `DIAMETER_UNABLE_TO_DELIVER` becomes `WAIT_EVENT`
  - other failures become timer-based retry candidates

Routing-pass finalization precedence is:

1. any candidate delivers -> `DELIVERED`
2. else any event-wait candidate -> `WAIT_EVENT` or `WAIT_TIMER_EVENT`
3. else any timer-retry candidate -> `WAIT_TIMER`
4. else -> `FAILED`

## Stored Message States

Persisted message statuses are:

- `QUEUED`
- `DISPATCHED`
- `WAIT_TIMER`
- `WAIT_EVENT`
- `WAIT_TIMER_EVENT`
- `DELIVERED`
- `FAILED`
- `EXPIRED`

These are real database states, not UI-only labels.

## Deferred Delivery Behavior

When delivery cannot complete immediately, the forwarder stores deferred metadata:

- `alert_correlation_id`
- `deferred_reason`
- `deferred_interface`
- `serving_node_at_deferral`
- `route_cursor`

Current deferral patterns:

- generic routing failure uses `deferred_reason: "route_lookup"`
- SGd lookup failure uses `deferred_reason: "sgd_lookup"`
- SGd delivery failure uses `deferred_reason: "sgd_delivery"`
- SGd alert-triggered requeue uses `deferred_reason: "sgd_alert_retry"`

Route cursor behavior matters:

- non-SGd deferrals resume from the next candidate
- SGd deferrals keep the SGd candidate cursor so alert-triggered retry can re-attempt the same serving node path

## Retry And Alert Ownership

The retry scheduler and alert-triggered wakeup path both use guarded state transitions:

- timer retry can only claim messages still in `QUEUED`, `WAIT_TIMER`, or `WAIT_TIMER_EVENT`
- alert requeue only updates messages still in queued or waiting states
- stale alerts received after delivery or after another worker has already claimed the message are skipped

This prevents duplicate dispatch when timers and alerts race.

## Store-And-Forward Policies

`sf_policy_id` on a matched fallback rule controls retry and lifetime behavior for that route.

Policy fields:

- `max_retries`
- `retry_schedule`
- `max_ttl`
- optional `vp_override`

Semantics:

- `max_ttl` overrides the global `smsc.max_queue_lifetime` once the message enters a waiting state
- `vp_override` is applied as an internal validity hint on the attempted route
- if `vp_override` is shorter than the queue lifetime cap, it becomes the earlier expiry cap

Retry scheduling details:

- the initial deferred retry from the first routing pass is scheduled `30s` later
- subsequent retries use the matched SF policy
- if no SF policy is attached, the built-in schedule is:
  - `[30, 300, 1800, 3600, 3600, 3600, 3600, 3600]`

If the next retry would exceed `max_retries`, the message is marked `FAILED`.

## Queue Lifetime

Global queue lifetime is configured by:

- `smsc.max_queue_lifetime`

Default:

- `168h` (`7 days`)

Expiry is capped from the original submission timestamp. Retries do not extend the base lifetime.

## Queue Semantics In Metrics And UI

Queue depth is treated as backlog across these active states:

- `QUEUED`
- `DISPATCHED`
- `WAIT_TIMER`
- `WAIT_EVENT`
- `WAIT_TIMER_EVENT`

`/api/v1/messages/queue` and the OAM queue view use this same operational definition of "in queue".

## Egress Peer Semantics

- `sip3gpp`: peer is derived from the local IMS cache or a live `Sh` result
- `sipsimple`: peer name must exist in the SIP peers table
- `smpp`: peer name must match an active outbound SMPP client session
- `sgd`: peer is derived from `S6c` serving-node data plus optional `S6c -> SGd` hostname mapping
