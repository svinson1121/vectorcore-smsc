# Build Guide

## Requirements

- Linux environment recommended
- Go `1.25.x`
- Node.js + npm for the React UI build
- `make`

Optional/runtime dependencies:

- SQLite or PostgreSQL
- SCTP support if you use Diameter over SCTP
- systemd if you want service installation

## Outputs

- Go binary: `bin/smsc`
- UI bundle: `web/dist/`

The UI is embedded into the Go binary at build time. If the UI was not built when the binary was compiled, `/ui/` serves a placeholder page instead of the React app.

## Build

Build everything:

```bash
make
```

Equivalent explicit steps:

```bash
make ui
make build
```

What this does:

1. installs UI dependencies in `web/`
2. builds the React bundle into `web/dist/`
3. builds `bin/smsc` with the Makefile version string embedded via `-ldflags`

## Run Locally

Use the sample config that exists in this repository:

```bash
bin/smsc -c config/smsc.yaml
```

Useful flags:

- `-d` enables debug logging
- `-v` prints the embedded version and exits
- `-c` selects the config file

Important path note:

- the binary default is `-c config.yaml`
- this source tree ships its sample config as `config/smsc.yaml`

If you do not pass `-c config/smsc.yaml`, the binary will look for a root-level `config.yaml`.

## Test

```bash
make test
```

Equivalent:

```bash
go test ./...
```

If cache permissions are restrictive in your environment:

```bash
GOCACHE=/tmp/go-build-cache go test ./...
```

## UI Development

Run the backend separately, then start the Vite dev server:

```bash
make dev-ui
```

The Vite server proxies:

- `/api` to `http://localhost:8080`
- `/metrics` to `http://localhost:8080`
- `/health` to `http://localhost:8080`

## Clean

```bash
make clean
```

This removes:

- `bin/`
- `web/dist/`

## Install As Service

The shipped systemd unit runs:

```bash
/opt/vectorcore/bin/smsc -c /opt/vectorcore/etc/smsc.yaml
```

And `make install` is intended to:

- install the binary under `/opt/vectorcore/bin/`
- install the config under `/opt/vectorcore/etc/`
- install `systemd/vectorcore-smsc.service`
- enable and start the service

Current source-tree caveat:

- the sample config in this repo is `config/smsc.yaml`
- the current `make install` target copies `config.yaml`

Before relying on `make install`, make sure you either:

- provide a root-level `config.yaml`, or
- adjust the install step to copy `config/smsc.yaml` into `/opt/vectorcore/etc/smsc.yaml`

## Configuration Notes

The sample config at `config/smsc.yaml` includes:

- plain SMPP listener settings
- optional inbound SMPP/TLS listener settings
- optional outbound SMPP TLS CA settings
- SIP listen identity and ISC header controls
- Diameter transport and local identity
- database driver and DSN
- API listen address
- log file and log level
