# VectorCore SMSC

VectorCore SMSC is a multi-interface SMS core written in Go. It accepts SMS from SMPP, SIP/3GPP ISC, SIP SIMPLE, and Diameter SGd, applies built-in and fallback routing, and forwards traffic across interfaces with store-and-forward persistence, retry handling, expiry control, and delivery reporting.

## Current Runtime Surface

- Ingress:
  - SMPP server over TCP
  - optional SMPP server over TLS
  - SIP `MESSAGE`, `REGISTER`, and `NOTIFY`
  - Diameter SGd server
- Egress:
  - SMPP outbound clients over TCP or TLS
  - SIP/3GPP ISC delivery
  - SIP SIMPLE peer delivery
  - Diameter SGd delivery with S6c-assisted target resolution
- Persistence and state:
  - PostgreSQL or SQLite backends
  - hot-reloaded routing rules and peer configuration
  - retry scheduler, expiry sweeper, and delivery-report correlation
- Operations:
  - REST API under `/api/v1`
  - OpenAPI document at `/api/v1/openapi.json`
  - Swagger-style docs at `/api/v1/docs`
  - Prometheus metrics at `/metrics`
  - health probe at `/health`
  - embedded React UI at `/ui/`

## Quick Start

Build the UI bundle and the Go binary:

```bash
make
```

Run the SMSC with the sample config shipped in this tree:

```bash
bin/smsc -c config/smsc.yaml
```

Useful flags:

- `-d` enables debug logging
- `-v` prints the embedded version and exits

Note: the binary's default config path is `config.yaml`, but the sample config in this repository lives at `config/smsc.yaml`, so pass `-c config/smsc.yaml` unless you provide your own root-level `config.yaml`.

## Documentation

- API reference and endpoint inventory: `docs/API.md`
- Routing order, retry states, and SF policy behavior: `docs/ROUTING.md`
- Build, run, test, UI, and install notes: `docs/BUILD.md`
