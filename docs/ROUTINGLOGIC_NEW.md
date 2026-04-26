# SMSC Routing Logic Refactor (Targeted Change Only)

## Overview

This task updates the **MT SMS routing logic** in the existing SMSC / IP-SM-GW.

The goal is to fix routing behavior so that:
- A message is treated as **one logical entity with multiple delivery attempts**
- Routing becomes **trigger-driven**, not a continuous loop
- All protocol implementations remain unchanged

---

## Scope

### In Scope
- MT routing decision logic
- Retry behavior (timer + event driven)
- Handling of deferred delivery (e.g., SMS-in-MME / ALR)
- Preventing duplicate or stale delivery attempts

### Out of Scope (DO NOT CHANGE)
- SIP / IMS protocol implementation
- Sh interface implementation
- SMS-in-MME (SGd/S6c) protocol logic
- SMPP implementation
- SIP SIMPLE implementation
- Peer/session handling
- Message encoding/decoding
- External APIs

---

## Current Behavior (Problem)

Current routing logic behaves like:

IMS -> Sh -> SMS-in-MME -> SMPP/SIP -> repeat loop until delivered or timeout

Issues:
- Continuous loop wastes resources
- No distinction between temporary vs deferred failures
- SMS-in-MME (ALR) behavior conflicts with looping model
- No clean retry model for IMS/Sh
- Messages can effectively "bounce" forever

---

## Target Behavior

Routing must become **trigger-based** and **single-pass per trigger**.

### Routing Pass

Each routing pass:
1. Runs once through all enabled routes (in order)
2. Stops immediately when:
   - message is delivered
   - message is deferred (waiting condition)
   - all routes are exhausted

No internal looping.

---

## Routing Order (Default)

1. Local IMS registration
2. IMS via Sh lookup (if enabled/configured)
3. SMS-in-MME (if enabled/configured)
4. Fallback routes (SMPP, SIP SIMPLE, etc.)

Must not assume any route is enabled.

---

## Route Outcomes

Each route attempt must resolve to one of:

| Outcome        | Meaning |
|----------------|--------|
| DELIVERED      | Message successfully delivered |
| TRY_NEXT       | Route not usable, continue |
| WAIT_EVENT     | Wait for external trigger (e.g., ALR) |
| WAIT_TIMER     | Retry later via timer |
| FAIL_PERMANENT | Permanent failure |

---

## Core Routing Rules

### Rule 1: No Looping
Routing must **not** loop continuously inside one execution.

### Rule 2: Trigger-Based Execution
Routing runs only on:
- Message arrival
- Timer retry
- ALR (SMS-in-MME alert)
- Optional IMS registration event (if available)

---

## SMS-in-MME Behavior (Critical)

If SMS-in-MME returns a **deferred / alert condition**:

- Store message as waiting
- Do NOT continue looping
- Resume routing only when:
  - ALR arrives
  - or timer fires

---

## IMS / Sh Retry Requirement

If IMS is not available:
- Do NOT assume permanent failure
- Must retry later via timer or event

Example:
- UE offline now → may be online later

---

## Global Message Ownership

A message must NOT belong to a protocol.

Instead:
- IMS, Sh, SGd, SMPP, SIP SIMPLE are **attempts**
- Only one **final delivery outcome** exists

---

## Cross-Interface Completion

Scenario:

1. Message deferred by SMS-in-MME  
2. UE comes online in IMS  
3. Message delivered via IMS  

Required behavior:

- Mark message **DELIVERED globally**
- Cancel all pending retries and waits
- Ignore future ALR or retry events
- Trigger final delivery reporting once

---

## Late Event Handling

If message is already delivered:

Ignore:
- ALR events
- Retry timers
- IMS registration triggers

Must be safe and idempotent.

---

## Retry Model

### Timer Retry
Used for:
- IMS unavailable
- Sh unavailable
- temporary failures
- fallback routes

### Event Retry
Used for:
- SMS-in-MME (ALR)

### Important

A message may have:
- both timer and event triggers active

Whichever fires first should re-run routing.

---

## Minimal Implementation Strategy

1. Locate existing routing loop  
2. Replace with single-pass routing execution  
3. Add lightweight state tracking:
   - waiting for timer
   - waiting for event
   - delivered state  
4. Map existing route results to routing outcomes  
5. Ensure:
   - retries are scheduled properly  
   - no tight loops remain  
   - final success cancels pending work  

---

## HSS / Delivery Reporting

- Final delivery must be reported **once**
- Do not report while message is deferred
- Use existing reporting logic
- Ensure no duplicate reporting

---

## Test Cases

### 1. IMS Immediate Delivery
- Delivered on first pass  
- No retry scheduled  

### 2. IMS Unavailable → Timer Retry → Success
- First pass fails  
- Retry later succeeds  

### 3. SMS-in-MME Deferred → IMS Later Success
- First pass waits for ALR  
- Later routing delivers via IMS  
- ALR ignored after success  

### 4. No SMS-in-MME Deployment
- Routing still works  
- No dependency on SGd  

### 5. No Sh Deployment
- Routing still works  
- No dependency on Sh  

### 6. Late ALR After Delivery
- Ignored safely  

---

## Key Design Principle

Routing is **trigger-driven**, not loop-driven.  
A message has **one lifecycle**, not per-protocol ownership.

---

## Deliverables

- Minimal changes to existing routing logic  
- No protocol changes  
- Tests covering new behavior  
- Short note describing new routing behavior  

