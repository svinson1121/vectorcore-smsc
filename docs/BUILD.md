# Build Guide

## Requirements

- Linux environment (recommended)
- Go `1.25.x` (project `go.mod` is `go 1.25.0`)
- Node.js + npm (for UI build)
- `make`

Optional/runtime dependencies:

- SQLite (embedded driver used by default config) or PostgreSQL server
- SCTP support if using Diameter SCTP transport
- systemd if using `make install`

## Project Outputs

- Go binary: `bin/smsc`
- Web UI static bundle: `web/dist/`

## Quick Build

From repository root:

```bash
make
```

This runs:

1. UI dependency install + UI production build (`web/dist`)
2. Go build with version ldflags to `bin/smsc`

## Run Locally

```bash
bin/smsc -d -c smsc.yaml
```

- `-d` enables debug logging
- `-c` selects config file

## Test

```bash
make test
```

Equivalent:

```bash
go test ./...
```

If cache path permissions are restricted, use:

```bash
GOCACHE=/tmp/go-build-cache go test ./...
```

## UI Development

Run backend separately, then start Vite dev server:

```bash
make dev-ui
```

## Clean

```bash
make clean
```

Removes:

- `bin/`
- `web/dist/`

## Install as Service (systemd)

```bash
make install
```

Installs binary/config/service unit and enables service.

Uninstall:

```bash
make uninstall
```

## Configuration Notes

- Main runtime config: `smsc.yaml`
- SIP identity should be explicitly set with:
  - `sip.fqdn: "smsc.ims.mnc435.mcc311.3gppnetwork.org"`
- Default sample config also includes SMPP, Diameter, DB, API, and log sections.

