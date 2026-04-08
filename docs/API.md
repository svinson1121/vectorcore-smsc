# API Reference

Base URL:

- `http://<host>:8080`

Primary API namespace:

- `/api/v1`

## Operational Endpoints

### Status

- `GET /api/v1/status`

```bash
curl -s http://localhost:8080/api/v1/status | jq
```

Sample response:

```json
{
  "version": "0.0.1d",
  "uptime": "2m34s",
  "uptime_sec": 154.1,
  "message_counts": {
    "queued": 0,
    "dispatched": 0,
    "delivered": 12,
    "failed": 1,
    "expired": 0
  }
}
```

- `GET /api/v1/status/peers`

```bash
curl -s http://localhost:8080/api/v1/status/peers | jq
```

Sample response:

```json
[
  {
    "name": "msc0",
    "type": "smpp_client",
    "state": "BOUND",
    "system_id": "smsc",
    "bind_type": "transceiver",
    "remote_addr": "10.90.250.42:2775",
    "connected_at": "2026-03-31T16:42:37Z"
  }
]
```

### Metrics and Health

- `GET /metrics`

```bash
curl -s http://localhost:8080/metrics
```

Sample excerpt:

```text
smsc_messages_in_total{interface="smpp"} 12
smsc_messages_out_total{interface="sip3gpp"} 12
smsc_store_forward_queued 0
```

- `GET /health`

```bash
curl -s http://localhost:8080/health
```

Sample response:

```json
{"status":"ok"}
```

### OpenAPI

- `GET /api/v1/openapi.json`
- `GET /api/v1/docs`

```bash
curl -s http://localhost:8080/api/v1/openapi.json | jq '.info'
```

## SMPP Server Accounts

- `GET /api/v1/smpp/server/accounts`
- `GET /api/v1/smpp/server/accounts/{id}`
- `POST /api/v1/smpp/server/accounts`
- `PUT /api/v1/smpp/server/accounts/{id}`
- `DELETE /api/v1/smpp/server/accounts/{id}`

List sample response:

```json
[
  {
    "id": "f3dc2f8e-1b0a-4f95-b851-9ab742e8d600",
    "name": "test-account",
    "system_id": "test",
    "password_hash": "$2a$10$...",
    "allowed_ip": "192.168.105.20",
    "bind_type": "transceiver",
    "throughput_limit": 0,
    "enabled": true
  }
]
```

Create:

```bash
curl -s -X POST http://localhost:8080/api/v1/smpp/server/accounts \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "test-account",
    "system_id": "test",
    "password": "secret",
    "allowed_ip": "192.168.105.20",
    "bind_type": "transceiver",
    "throughput_limit": 0,
    "enabled": true
  }'
```

Update:

```bash
curl -s -X PUT http://localhost:8080/api/v1/smpp/server/accounts/<id> \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "test-account",
    "system_id": "test",
    "password": "",
    "allowed_ip": "",
    "bind_type": "transceiver",
    "throughput_limit": 10,
    "enabled": true
  }'
```

## SMPP Outbound Clients

- `GET /api/v1/smpp/clients`
- `GET /api/v1/smpp/clients/{id}`
- `POST /api/v1/smpp/clients`
- `PUT /api/v1/smpp/clients/{id}`
- `DELETE /api/v1/smpp/clients/{id}`

List sample response:

```json
[
  {
    "id": "7aefb5d0-3498-44d2-a373-8f2f86ff3de3",
    "name": "msc0",
    "host": "10.90.250.42",
    "port": 2775,
    "system_id": "smsc",
    "password": "secret",
    "bind_type": "transceiver",
    "reconnect_interval": "10s",
    "throughput_limit": 0,
    "enabled": true
  }
]
```

Create:

```bash
curl -s -X POST http://localhost:8080/api/v1/smpp/clients \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "msc0",
    "host": "10.90.250.42",
    "port": 2775,
    "system_id": "smsc",
    "password": "secret",
    "bind_type": "transceiver",
    "reconnect_interval": "10s",
    "throughput_limit": 0,
    "enabled": true
  }'
```

## SIP SIMPLE Peers

- `GET /api/v1/sip/peers`
- `GET /api/v1/sip/peers/{id}`
- `POST /api/v1/sip/peers`
- `PUT /api/v1/sip/peers/{id}`
- `DELETE /api/v1/sip/peers/{id}`

List sample response:

```json
[
  {
    "id": "8f3e167f-8d31-4f2d-b7a7-0e2e9164a7a6",
    "name": "site-b",
    "address": "10.0.0.10",
    "port": 5060,
    "transport": "udp",
    "domain": "example.org",
    "auth_user": "",
    "auth_pass": "",
    "enabled": true
  }
]
```

Create:

```bash
curl -s -X POST http://localhost:8080/api/v1/sip/peers \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "site-b",
    "address": "10.0.0.10",
    "port": 5060,
    "transport": "udp",
    "domain": "example.org",
    "auth_user": "",
    "auth_pass": "",
    "enabled": true
  }'
```

## Diameter Peers

- `GET /api/v1/diameter/peers`
- `GET /api/v1/diameter/peers/{id}`
- `POST /api/v1/diameter/peers`
- `PUT /api/v1/diameter/peers/{id}`
- `DELETE /api/v1/diameter/peers/{id}`

List sample response:

```json
[
  {
    "id": "2dcb8fcb-8ff6-4b62-8a98-a8bc5f7a58be",
    "name": "dra01.epc.mnc435.mcc311.3gppnetwork.org",
    "host": "10.90.250.35",
    "realm": "epc.mnc435.mcc311.3gppnetwork.org",
    "port": 3868,
    "transport": "sctp",
    "applications": ["sh", "sgd"],
    "enabled": true
  }
]
```

Create:

```bash
curl -s -X POST http://localhost:8080/api/v1/diameter/peers \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "dra01.epc.mnc435.mcc311.3gppnetwork.org",
    "host": "10.90.250.35",
    "realm": "epc.mnc435.mcc311.3gppnetwork.org",
    "port": 3868,
    "transport": "sctp",
    "applications": ["sh", "sgd"],
    "enabled": true
  }'
```

## Routing Rules

- `GET /api/v1/routing/rules`
- `GET /api/v1/routing/rules/{id}`
- `POST /api/v1/routing/rules`
- `PUT /api/v1/routing/rules/{id}`
- `DELETE /api/v1/routing/rules/{id}`

List sample response:

```json
[
  {
    "id": "ddcbec7e-55c4-4e9f-9d13-a5fd464fb114",
    "name": "default-smpp-egress",
    "priority": 100,
    "match_src_iface": "smpp",
    "match_src_peer": "",
    "match_dst_prefix": "",
    "match_msisdn_min": "",
    "match_msisdn_max": "",
    "egress_iface": "smpp",
    "egress_peer": "msc0",
    "sf_policy_id": "",
    "enabled": true
  }
]
```

Create:

```bash
curl -s -X POST http://localhost:8080/api/v1/routing/rules \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "default-smpp-egress",
    "priority": 100,
    "match_src_iface": "smpp",
    "match_src_peer": "",
    "match_dst_prefix": "",
    "match_msisdn_min": "",
    "match_msisdn_max": "",
    "egress_iface": "smpp",
    "egress_peer": "msc0",
    "sf_policy_id": "",
    "enabled": true
  }'
```

## Store-and-Forward Policies

- `GET /api/v1/routing/policies`
- `GET /api/v1/routing/policies/{id}`
- `POST /api/v1/routing/policies`
- `PUT /api/v1/routing/policies/{id}`
- `DELETE /api/v1/routing/policies/{id}`

List sample response:

```json
[
  {
    "id": "ba4fda4f-67f6-42f9-b3d4-113a7ea62467",
    "name": "default",
    "max_retries": 8,
    "retry_schedule": [30, 300, 1800, 3600, 3600, 3600, 3600, 3600],
    "max_ttl": "48h0m0s",
    "vp_override": null
  }
]
```

Create:

```bash
curl -s -X POST http://localhost:8080/api/v1/routing/policies \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "default",
    "max_retries": 8,
    "retry_schedule": [30,300,1800,3600,3600,3600,3600,3600],
    "max_ttl": "48h",
    "vp_override": ""
  }'
```

## Subscribers

- `GET /api/v1/subscribers`
- `GET /api/v1/subscribers/{id}`
- `POST /api/v1/subscribers` (upsert)
- `PUT /api/v1/subscribers/{id}`
- `DELETE /api/v1/subscribers/{id}`

List sample response:

```json
[
  {
    "id": "c64e124a-5f4c-4b5a-9e2a-c6b6d8f0c983",
    "msisdn": "3342012832",
    "imsi": "",
    "ims_registered": true,
    "lte_attached": false,
    "mme_host": "",
    "mwd_set": false
  }
]
```

Upsert:

```bash
curl -s -X POST http://localhost:8080/api/v1/subscribers \
  -H 'Content-Type: application/json' \
  -d '{
    "msisdn": "3342012832",
    "imsi": "",
    "ims_registered": true,
    "lte_attached": false,
    "mme_host": "",
    "mwd_set": false
  }'
```

## Messages and Delivery Reports (Read-only)

- `GET /api/v1/messages?limit=100`
- `GET /api/v1/messages/{id}`
- `GET /api/v1/delivery-reports?limit=100`
- `GET /api/v1/delivery-reports/{id}`

```bash
curl -s "http://localhost:8080/api/v1/messages?limit=20" | jq
curl -s "http://localhost:8080/api/v1/delivery-reports?limit=20" | jq
```

Messages sample response:

```json
[
  {
    "id": "6ff75e5c-bd8e-468f-8c17-8396d6124e38",
    "smpp_msg_id": "000000000000001a",
    "origin_iface": "smpp",
    "origin_peer": "test",
    "egress_iface": "sip3gpp",
    "egress_peer": "10.90.250.52",
    "src_msisdn": "3342021234",
    "dst_msisdn": "3342012832",
    "dcs": 0,
    "status": "DELIVERED",
    "retry_count": 0,
    "dr_required": true,
    "submitted_at": "2026-03-31T16:51:50Z"
  }
]
```

Delivery reports sample response:

```json
[
  {
    "id": "747fcb2e-21f7-4739-a041-868c5b9ca50e",
    "message_id": "6ff75e5c-bd8e-468f-8c17-8396d6124e38",
    "status": "DELIVRD",
    "egress_iface": "sip3gpp",
    "raw_receipt": "id:6ff75e5c-bd8e-468f-8c17-8396d6124e38 stat:DELIVRD",
    "reported_at": "2026-03-31T16:51:51Z"
  }
]
```

## UI Endpoints

- `GET /ui/`
- `GET /ui/assets/...`
