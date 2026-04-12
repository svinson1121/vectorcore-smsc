package codec

// S6c AVP codes per 3GPP TS 29.338 (Vendor-ID 10415 unless noted).
const (
	CodeServingNode               uint32 = 2401 // Grouped
	CodeMMEName                   uint32 = 2402 // DiameterIdentity
	CodeMMENumberForMTSMSS6c      uint32 = 2403 // OctetString
	CodeMMENumberForMTSMSServing  uint32 = 1645 // OctetString
	CodeAdditionalServingNode     uint32 = 2406 // Grouped
	CodeMMERealm                  uint32 = 2408 // DiameterIdentity
	CodeSGSNName                  uint32 = 2409 // DiameterIdentity
	CodeSGSNRealm                 uint32 = 2410 // DiameterIdentity
	CodeSCAddressS6c              uint32 = 3300 // OctetString
	CodeSMRPMTIS6c                uint32 = 3308 // Enumerated
	CodeMWDStatusS6c              uint32 = 3312 // Unsigned32
	CodeMMEAbsentUserDiagnosticSM uint32 = 3313 // Unsigned32
	CodeSGSNAbsentUserDiagnostic  uint32 = 3314 // Unsigned32
	CodeMSCAbsentUserDiagnostic   uint32 = 3315 // Unsigned32
	CodeSMDeliveryOutcomeS6c      uint32 = 3316 // Grouped
	CodeMMEDeliveryOutcome        uint32 = 3317 // Grouped
	CodeSGSNDeliveryOutcome       uint32 = 3318 // Grouped
	CodeMSCDeliveryOutcome        uint32 = 3319 // Grouped
	CodeIPSMGWDeliveryOutcome     uint32 = 3320 // Grouped
	CodeSMDeliveryCause           uint32 = 3321 // Enumerated
	CodeAbsentUserDiagnosticSM    uint32 = 3322 // Unsigned32
	CodeIPSMGWNumber              uint32 = 3327 // OctetString
	CodeIPSMGWName                uint32 = 3328 // DiameterIdentity
	CodeIPSMGWRealm               uint32 = 3329 // DiameterIdentity
)

const (
	SMRPMTIS6cDeliver      uint32 = 0
	SMRPMTIS6cSubmitReport uint32 = 1
)

const (
	MWDStatusMNRF uint32 = 0x02
	MWDStatusMCEF uint32 = 0x04
	MWDStatusMNRG uint32 = 0x08
)

const (
	SMDeliveryCauseMemoryCapacityExceeded uint32 = 0
	SMDeliveryCauseAbsentUser             uint32 = 1
	SMDeliveryCauseSuccessfulTransfer     uint32 = 2
)
