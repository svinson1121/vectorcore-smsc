# SMSC Response Report for SMPP Router Incident

Date: 2026-04-08 UTC
Capture reviewed: `/tmp/SMPP_R-SMSC.pcap`
Router report reviewed: `/usr/src/vectorcore-smpp_router/SMPP_R-SMSC_REPORT.md`

## Conclusion

We found an SMSC bug in the current codebase that matches the router capture.

The affected path is not the inbound SMPP server listener. It is the outbound SMPP client session used when this SMSC connects to a downstream peer/router as a transceiver.

In the reviewed capture, `10.90.250.53` initiates the SMPP connection to `10.90.250.186:2775` and binds as `system_id=smpp_router`. That means PDUs sent back from the router to this session are handled by the SMPP client read loop, not the SMPP server listener.

## Root Cause

The SMPP client read loop handled:

- `enquire_link`
- `unbind`
- `deliver_sm`

but it did not handle incoming `submit_sm`.

As a result, when the router forwarded a message to us as `submit_sm`, our client session treated it as an unhandled PDU and never returned `submit_sm_resp`.

This exactly matches the PCAP:

- bind succeeds
- router sends `submit_sm`
- our host TCP-ACKs the packet
- no `submit_sm_resp` is emitted
- the session remains alive and still exchanges `enquire_link`

## Code Evidence

Before the fix, the client read loop in [internal/smpp/client/session.go](/usr/src/vectorcore-smsc/internal/smpp/client/session.go#L275) only dispatched `deliver_sm` and did not have a `submit_sm` case.

The bug was in the outbound client session path because the connection in the capture is client-initiated from our side.

The fix adds:

- explicit handling of incoming `submit_sm` in the client read loop at [internal/smpp/client/session.go](/usr/src/vectorcore-smsc/internal/smpp/client/session.go#L303)
- immediate `submit_sm_resp` generation in [internal/smpp/client/session.go](/usr/src/vectorcore-smsc/internal/smpp/client/session.go#L394)
- the same handling during graceful unbind at [internal/smpp/client/session.go](/usr/src/vectorcore-smsc/internal/smpp/client/session.go#L366)

## Why The Earlier Server Review Was Insufficient

The inbound SMPP server code does respond immediately to `submit_sm`, but that path was not the one used in this incident.

The PCAP shows our host connected outbound to the router. Therefore the relevant handler is the outbound SMPP client session, where the missing `submit_sm` case was the defect.

## Validation

Added regression test:

- [internal/smpp/client/session_test.go](/usr/src/vectorcore-smsc/internal/smpp/client/session_test.go#L49)

That test sends an inbound `submit_sm` into the client-side read loop and verifies that:

- we return `submit_sm_resp`
- the sequence number is preserved
- a message ID is included
- the decoded message is dispatched internally

Verified locally:

```text
env GOCACHE=/tmp/vectorcore-smsc-gocache GOMODCACHE=/tmp/vectorcore-smsc-gomodcache go test ./internal/smpp/client/...
ok  	github.com/svinson1121/vectorcore-smsc/internal/smpp/client	0.033s
```

## Suggested Reply To SMPP Router

We reviewed the router PCAP and confirmed the fault was on our side. Root cause was a bug in our outbound SMPP client transceiver path: it handled inbound `deliver_sm` but not inbound `submit_sm`, so when the router forwarded a message to our bound client session as `submit_sm`, our system did not emit the required `submit_sm_resp`. The session stayed alive, which is why `enquire_link` continued to work. We have identified the defect, implemented the fix, and added a regression test covering this case.
