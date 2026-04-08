# VectorCore SMSC

VectorCore SMSC is a multi-interface SMS core written in Go. It accepts SMS from SMPP, SIP/3GPP, SIP SIMPLE, and Diameter SGd, applies routing decisions, and forwards traffic across interfaces with store-and-forward handling, retry, and delivery reporting.

## What This Project Is

- A production-style SMSC runtime (`bin/smsc`) with REST API, Prometheus metrics, and web UI.
- A routing + forwarding engine that supports:
  - IMS registration-aware delivery (SIP/3GPP ISC)
  - LTE attach-aware delivery (Diameter SGd)
  - Rule-based egress selection (SMPP/SIP/Diameter)
- A persistence-backed message pipeline with retry/expiry sweeps.

## Core Features

- Multi-protocol ingress/egress
  - SMPP server (inbound ESME binds + submit_sm)
  - SMPP outbound clients
  - SIP/3GPP ISC messaging
  - SIP SIMPLE inter-site messaging
  - Diameter SGd and Sh peer support
- Routing engine
  - Priority-ordered routing rules (first match wins)
  - Source/destination interface/peer filters
  - MSISDN prefix and range matching
  - Egress peer and SF policy selection
- Store-and-forward
  - QUEUED/DISPATCHED/DELIVERED/FAILED/EXPIRED lifecycle
  - Retry scheduler with policy-based schedules
  - Expiry sweeper and DR correlation
- Operations
  - REST API (`/api/v1/...`) for CRUD and observability
  - Prometheus metrics (`/metrics`)
  - React web UI (`/ui`)
  - OpenAPI docs (`/api/v1/docs`)

## Documentation

- API reference and curl examples: `docs/API.md`
- Routing logic and rule behavior: `docs/ROUTING.md`
- Build requirements and build steps: `docs/BUILD.md`
