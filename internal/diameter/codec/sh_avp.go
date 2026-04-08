package codec

// Sh AVP codes per 3GPP TS 29.328 (Vendor-ID 10415 unless noted)
const (
	// Sh command code
	CmdUserData uint32 = 306 // UDR/UDA

	// Sh AVP codes (all vendor 10415)
	CodeUserIdentity   uint32 = 700 // Grouped
	CodePublicIdentity uint32 = 601 // UTF8String — SIP URI or tel URI
	CodeUserData       uint32 = 702 // OctetString — Sh-Data XML
	CodeDataReference  uint32 = 703 // Enumerated

	// Data-Reference enumerated values (TS 29.328 §6.3.1)
	DataRefRepositoryData        = 0
	DataRefIMSPublicIdentity     = 10
	DataRefIMSUserState          = 11
	DataRefSCSCFName             = 12
	DataRefInitialFilterCriteria = 13
	DataRefLocationInformation   = 14
	DataRefUserState             = 15
	DataRefMSISDN                = 17
)
