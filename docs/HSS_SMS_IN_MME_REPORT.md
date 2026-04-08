# HSS SMS-in-MME Investigation Report

This report summarizes the MT SMS over `SGd` investigation from the `SMSC` side and identifies the remaining work that appears to belong on the `HSS` and `MME` registration path.

## Executive Summary

The `SMSC -> DRA -> MME` `SGd` message shape was corrected and is now accepted by the MME's SMS service parser. The remaining failure is a downstream `DIAMETER_ERROR_USER_UNKNOWN (5001)` returned by the MME-side `SGd` service after it receives and parses the MT request.

The evidence now points away from malformed `SGd` encoding on the `SMSC` and toward missing or incomplete `SMS-in-MME` registration/subscription state on the `HSS/MME` side.

## Environment Summary

- Subscriber under test:
  - `MSISDN = 3342012832`
  - `IMSI = 311435000070570`
- Serving MME returned by `S6c` / cached in subscriber state:
  - `s6a-vpc-si-01.epc.mnc435.mcc311.3gppnetwork.org`
- Connected Diameter proxy peer from the SMSC:
  - `dra01.epc.mnc435.mcc311.3gppnetwork.org`

## Original SMSC-Side Problems Found

The following `SMSC` issues were found and corrected during the investigation:

1. `SGd` peer selection assumed a direct MME peer.
   - In this deployment the SMSC uses a `DRA/proxy`, not a direct MME `SGd` session.
   - The SMSC previously rejected valid `SGd` routing when the returned `MME-Host` did not exactly match a directly connected `SGd` peer.

2. The SMSC did not wait for the `OFA/TFA` answer before treating delivery as successful.
   - Failed `SGd` attempts could be misreported as delivered.

3. `SGd` answer parsing did not honor `Experimental-Result-Code`.
   - This caused incorrect local logging of downstream errors.

4. MT `SGd` command and AVP usage were not aligned with `3GPP TS 29.338`.
   - MT delivery was corrected to use `8388646` and `User-Name = IMSI`.

5. Several `SGd` AVP numeric codes were wrong.
   - Earlier captures showed `Failed-AVP` for AVPs incorrectly interpreted downstream as `IP-SM-GW-Name` and `AESE-Communication-Pattern`.

6. `SM-RP-MTI` was being sent on normal MT `SGd` delivery where it was not accepted by the MME.

## Current SMSC Behavior

After the SMSC fixes, the outbound MT request now has the expected high-level shape:

- Diameter application:
  - `SGd`
- Command:
  - `8388646` (`MT-Forward-Short-Message`)
- Destination:
  - routed through the connected `DRA`
  - `Destination-Host` set to the serving MME returned by `S6c`
- Subscriber identity:
  - `User-Name = IMSI`
- Payload:
  - `SM-RP-UI` containing a GSM `SMS-DELIVER` TPDU
- Service center address:
  - `SC-Address` encoded in `TBCD`

## Evidence Collected

### 1. SMSC Logs

The SMSC now consistently shows:

- `forwarder: SGd route resolved ... via_proxy=true`
- `diameter message ... command=TFR/TFA command_code=8388646 request=true`
- `sgd OFA received ... result_code=5001`

This confirms:

- the SMSC selects the `SGd` route correctly through the `DRA`
- the corrected MT request is being sent on `SGd`
- the reject is coming back from downstream, not produced locally by route selection

### 2. DRA PCAPs

The `DRA` captures showed a clear progression:

1. Early captures:
   - downstream rejected malformed or wrong AVP usage
   - `Failed-AVP` indicated incorrect AVP numbers and later `SM-RP-MTI`

2. Latest capture:
   - the MME receives the corrected MT `SGd` request
   - no `Failed-AVP` is present
   - answer contains `Experimental-Result-Code = 5001`
   - Wireshark decodes this as `DIAMETER_ERROR_USER_UNKNOWN`

This is the strongest signal that message syntax/encoding is no longer the main problem.

### 3. MME SGd Service Logs

The MME logs show that it receives the request:

- `Application request message app 35 code 8388646 from peer dra01...`

The MME then reports:

- `Inbound message with no Session`

This should be interpreted carefully:

- the Diameter `Session-Id` AVP is present on the wire
- the warning appears to mean the MME's internal SMS/session logic did not find the subscriber/session context it expected for MT `SGd`

### 4. MME SMSC Service Statistics

The MME `show smsc-service statistics all` output is decisive:

- `TF Request: 6`
- `TF Answer: 6`
- `User Unknown: 6`
- `Parse-Message-Errors: 0`
- `Parse-Misc-Errors: 0`
- `Unable To Comply: 0`

Interpretation:

- the MME SMSC/`SGd` service is parsing the request successfully
- the MME is classifying the request as `User Unknown`
- this is not a Diameter parser or `SGd` AVP formatting error anymore

### 5. MME Subscriber State

The MME active-subscriber output shows the UE is attached:

- `IMSI = 311435000070570`
- `MSISDN = 3342012832`

This means the subscriber exists on the MME, but that alone does not prove that the UE is registered and eligible for `SMS-in-MME`.

### 6. MME Service Statistics

The MME-wide counters include:

- `Paging Initiation for PS SMS Events: Attempted: 0 Success: 0 Failures: 0`

Interpretation:

- the MT `SGd` request is not progressing to actual PS SMS paging
- the MME is rejecting the request before starting SMS delivery handling
- this is consistent with missing `SMS-in-MME` registration or authorization state

## 3GPP Standards Interpretation

The standards review supports the need for explicit `HSS` support for `SMS-in-MME`.

Key points from `3GPP TS 23.272`, `TS 29.272`, and `TS 29.338`:

1. The `MME` registers for SMS via `S6a Update-Location`.
2. The `HSS` must support `SMS-in-MME` and return SMS-related subscription and registration information to the `MME`.
3. The `HSS` later returns serving-node data on `S6c SRI-SM` for MT SMS routing.
4. The `SMSC` uses that returned serving-node information to address MT SMS over `SGd`.

Operational conclusion:

- if the `HSS` does not correctly support `SMS-in-MME`, or
- if the `MME` does not become registered for SMS for the subscriber,

then MT SMS over `SGd` can fail even when:

- the UE is attached,
- `S6c` returns an MME,
- and the `SGd` request is well-formed.

## Most Likely Remaining Fault Domain

The leading hypothesis is now:

- the UE is attached on LTE, but not fully registered/enabled for `SMS-in-MME`, or
- the `HSS` is not returning the required SMS-related subscription and registration state in `S6a/ULA`, or
- the `MME` is not persisting/using that state to enable MT SMS handling for the subscriber

In short:

- `SMSC` encoding appears good enough
- remaining work is likely on the `HSS/MME` registration and subscriber-data path

## Recommended HSS-Side Checks

The next investigation should focus on the `S6a ULR/ULA` exchange and the subscriber data stored by the `HSS`.

### A. Verify MME SMS Registration Inputs In ULR

Confirm the `MME` sends the expected SMS-related information in `S6a ULR`, including:

- `MME-Name`
- `MME-Realm`
- `MME-Number-for-MT-SMS`
- `SMS-Register-Request`

### B. Verify HSS Accepts SMS-in-MME

Confirm the `HSS`:

- recognizes `SMS-Register-Request`
- stores the MME as registered for SMS
- stores the `MME-Number-for-MT-SMS`
- retains the SMS-serving-node information for later `S6c` routing

### C. Verify ULA Subscription Data

Check whether the `HSS` returns the subscriber data the `MME` expects to enable `SMS-in-MME`, including any SMS-related subscription or flags used by the `MME` implementation.

The exact vendor-specific mapping may differ, but the practical test is simple:

- compare the `ULA` for this failing subscriber against a known-good `SMS-in-MME` subscriber, if available

### D. Verify S6c Consistency

Confirm the `HSS` returns consistent serving node data on `S6c`:

- correct `MME-Name`
- correct `MME-Realm`
- correct subscriber `IMSI`
- attached state that matches the MME's live view

### E. Compare Working vs Failing Subscriber

If any working `SGd` MT case exists, compare:

- `ULR/ULA`
- `S6c SRI-SM / SRI-SM-Answer`
- HSS subscriber flags
- MME per-subscriber SMS-related state

This is likely the fastest path to root cause.

## Recommended MME-Side Checks

The following checks should be performed in parallel with `HSS` work:

1. Verify whether the subscriber is marked `MME registered for SMS`.
2. Verify whether the MME has an explicit `SMS-in-MME` enabled flag for the subscriber.
3. Verify whether the MME's SMSC/`SGd` service resolves users from a different internal context than the generic attached-subscriber table.
4. Check why `Paging Initiation for PS SMS Events` remains `0`.

## Conclusion

The SMSC-side transport, routing, command code, subscriber identity, and `SGd` AVP formatting issues have been corrected. The MME now accepts and parses the MT `SGd` request, but returns `DIAMETER_ERROR_USER_UNKNOWN` and does not initiate PS SMS paging.

The remaining work should focus on `HSS/MME` support for `SMS-in-MME`, especially:

- `S6a` SMS registration handling
- `ULA` SMS-related subscription data
- MME subscriber SMS registration state
- consistency between HSS serving-node data and MME SMS service state
