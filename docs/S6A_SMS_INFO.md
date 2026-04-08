# S6a SMS Information

This note explains where SMS-related serving node information is sent on `S6a`, what the important AVPs are, and how that data is later used by `S6c` and `SGd`.

## Summary

For SMS over LTE with SMS in MME:

1. The UE attaches to the MME.
2. The MME sends an `Update-Location-Request (ULR)` to the HSS on `S6a`.
3. The `ULR` carries the MME's SMS-related registration information.
4. The HSS stores that serving-node data for later MT-SMS routing.
5. The SMSC queries the HSS on `S6c` with `SRI-SM`.
6. The HSS returns serving node information for the subscriber.
7. The SMSC uses that returned MME identity to address the MT message on `SGd`.

## Where The SMS Info Is Sent

The MME sends the SMS-related information to the HSS in the `Update-Location-Request (ULR)` on the `S6a` interface.

This is the standards-based place where the MME registers itself for SMS in MME.

## Relevant S6a AVPs

The important `ULR` content for SMS in MME is:

- `Origin-Host`
- `Origin-Realm`
- `MME-Name`
- `MME-Realm`
- `MME-Number-for-MT-SMS`
- `SMS-Register-Request`
- `SGs-MME-Identity` when used for SGs-related behavior

Operationally:

- `Origin-Host` is the Diameter host identity of the `S6a` endpoint sending the `ULR`
- `MME-Name` is the MME identity stored by the HSS as part of serving node information
- `MME-Realm` is the Diameter realm for that MME identity
- `MME-Number-for-MT-SMS` is the E.164 number associated with routing MT SMS through the MME
- `SMS-Register-Request` indicates the MME is requesting SMS registration with the HSS

## What The HSS Does With It

If SMS in MME is accepted, the HSS registers the MME for SMS and stores the SMS-serving-node information for the subscriber.

Later, when the SMSC sends `SRI-SM` on `S6c`, the HSS returns serving node information that can include:

- `MME-Name`
- `MME-Realm`
- `MME-Number-for-MT-SMS`

The SMSC then uses the returned Diameter address/name of the MME for MT delivery on `SGd`.

## Important Identity Rule

For standards-based MT delivery on `SGd`, the SMSC expects the MME identity returned by the HSS on `S6c` to be the Diameter host identity usable for `SGd` routing.

In practice, this means the `MME-Name` returned by `S6c` needs to line up with the Diameter identity used for the MME on `SGd`, or the Diameter network needs to route that identity correctly.

## Cisco QVPC-SI Note

In the observed Cisco QVPC-SI deployment, different Diameter identities are used on different interfaces for the same logical MME:

- `S6a` host example: `s6a-vpc-si-01.epc.mnc435.mcc311.3gppnetwork.org`
- `SGd` host example: `sgd-mme-vpc-si-01.epc.mnc435.mcc311.3gppnetwork.org`

That means:

- the HSS may return the `S6a`-side MME identity on `S6c`
- the active `SGd` peer may present a different `Origin-Host`

If those identities differ, an SMSC that assumes exact host equality may fail to select the live `SGd` peer even though the MME is attached and `SGd` is up.

## Practical Troubleshooting

If MT SMS is not going over `SGd`, verify:

1. the MME sent SMS registration data in `S6a ULR`
2. the HSS accepted SMS in MME registration
3. `S6c SRI-SM` returns an attached serving MME
4. the returned `MME-Name` is actually routable on `SGd`
5. the live `SGd` peer identity matches the identity the SMSC uses as `Destination-Host`

## Standards Pointers

- `3GPP TS 29.272`: `S6a` procedures and `ULR/ULA`
- `3GPP TS 29.338`: `S6c` and `SGd` procedures for SMS in MME
- `3GPP TS 23.272`: architecture and behavior for SMS over SGs and SMS over `SGd`
