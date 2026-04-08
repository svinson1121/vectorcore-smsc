package s6c

import (
	"fmt"
	"strings"

	"github.com/svinson1121/vectorcore-smsc/internal/diameter"
	dcodec "github.com/svinson1121/vectorcore-smsc/internal/diameter/codec"
)

type ErrUnknownUser struct {
	ResultCode uint32
}

func (e *ErrUnknownUser) Error() string {
	return fmt.Sprintf("s6c: HSS returned result-code %d (subscriber unknown or error)", e.ResultCode)
}

func parseSRIAnswer(msg *dcodec.Message) (*RoutingInfo, error) {
	resultCode := parseResultCode(msg)
	if resultCode != dcodec.DiameterSuccess {
		return nil, &ErrUnknownUser{ResultCode: resultCode}
	}

	info := &RoutingInfo{
		SessionID:  parseStringAVP(msg, dcodec.CodeSessionID, 0),
		ResultCode: resultCode,
		IMSI:       parseStringAVP(msg, dcodec.CodeUserName, 0),
	}
	if msisdn := msg.FindAVP(dcodec.CodeMSISDN, dcodec.Vendor3GPP); msisdn != nil {
		info.MSISDN = decodeTBCD(msisdn.Data)
	}
	if mwd := msg.FindAVP(dcodec.CodeMWDStatusS6c, dcodec.Vendor3GPP); mwd != nil {
		info.MWDStatus, _ = mwd.Uint32()
	}
	if servingNode := msg.FindAVP(dcodec.CodeServingNode, dcodec.Vendor3GPP); servingNode != nil {
		children, err := dcodec.DecodeGrouped(servingNode)
		if err != nil {
			return nil, fmt.Errorf("s6c: decode Serving-Node: %w", err)
		}
		for _, child := range children {
			switch {
			case child.Code == dcodec.CodeMMEName && child.VendorID == dcodec.Vendor3GPP:
				info.MMEName = string(child.Data)
			case child.Code == dcodec.CodeMMERealm && child.VendorID == dcodec.Vendor3GPP:
				info.MMERealm = string(child.Data)
			}
		}
		info.Attached = info.MMEName != ""
	}
	return info, nil
}

func parseRSDSAnswer(msg *dcodec.Message) (*ReportDeliveryResult, error) {
	result := &ReportDeliveryResult{
		SessionID:  parseStringAVP(msg, dcodec.CodeSessionID, 0),
		ResultCode: parseResultCode(msg),
	}
	if result.ResultCode != dcodec.DiameterSuccess {
		return nil, &ErrUnknownUser{ResultCode: result.ResultCode}
	}
	if mwd := msg.FindAVP(dcodec.CodeMWDStatusS6c, dcodec.Vendor3GPP); mwd != nil {
		result.MWDStatus, _ = mwd.Uint32()
	}
	return result, nil
}

func parseAlertServiceCentre(msg *dcodec.Message) (AlertServiceCentreRequest, error) {
	req := AlertServiceCentreRequest{
		SessionID:   parseStringAVP(msg, dcodec.CodeSessionID, 0),
		OriginHost:  parseStringAVP(msg, dcodec.CodeOriginHost, 0),
		OriginRealm: parseStringAVP(msg, dcodec.CodeOriginRealm, 0),
		IMSI:        parseStringAVP(msg, dcodec.CodeUserName, 0),
	}
	if req.SessionID == "" {
		return AlertServiceCentreRequest{}, fmt.Errorf("missing Session-Id")
	}
	if scAddr := msg.FindAVP(dcodec.CodeSCAddressS6c, dcodec.Vendor3GPP); scAddr != nil {
		req.SCAddress = decodeTBCD(scAddr.Data)
	}
	if msisdn := msg.FindAVP(dcodec.CodeMSISDN, dcodec.Vendor3GPP); msisdn != nil {
		req.MSISDN = decodeTBCD(msisdn.Data)
	}
	if diag := msg.FindAVP(dcodec.CodeAbsentUserDiagnosticSM, dcodec.Vendor3GPP); diag != nil {
		req.AbsentUserDiagnosticSM, _ = diag.Uint32()
	}
	return req, nil
}

func sendAlertServiceCentreAnswer(peer *diameter.Peer, req *dcodec.Message, resultCode uint32) error {
	cfg := peer.Cfg()
	b := dcodec.NewAnswer(req)
	b.Add(
		dcodec.NewString(dcodec.CodeSessionID, 0, dcodec.FlagMandatory, parseStringAVP(req, dcodec.CodeSessionID, 0)),
		dcodec.NewUint32(dcodec.CodeAuthSessionState, 0, dcodec.FlagMandatory, dcodec.AuthSessionStateNoStateMaintained),
		dcodec.NewUint32(dcodec.CodeResultCode, 0, dcodec.FlagMandatory, resultCode),
		dcodec.NewString(dcodec.CodeOriginHost, 0, dcodec.FlagMandatory, cfg.LocalFQDN),
		dcodec.NewString(dcodec.CodeOriginRealm, 0, dcodec.FlagMandatory, cfg.LocalRealm),
	)
	return peer.Send(b.Build())
}

func parseResultCode(msg *dcodec.Message) uint32 {
	rc := msg.FindAVP(dcodec.CodeResultCode, 0)
	if rc == nil {
		return dcodec.DiameterUnableToComply
	}
	resultCode, err := rc.Uint32()
	if err != nil {
		return dcodec.DiameterUnableToComply
	}
	return resultCode
}

func parseStringAVP(msg *dcodec.Message, code, vendorID uint32) string {
	avp := msg.FindAVP(code, vendorID)
	if avp == nil {
		return ""
	}
	return string(avp.Data)
}

func encodeTBCD(msisdn string) []byte {
	msisdn = strings.TrimPrefix(msisdn, "+")
	if len(msisdn)%2 != 0 {
		msisdn += "F"
	}
	out := make([]byte, len(msisdn)/2)
	for i := 0; i < len(msisdn); i += 2 {
		out[i/2] = (digitNibble(msisdn[i+1]) << 4) | digitNibble(msisdn[i])
	}
	return out
}

func decodeTBCD(data []byte) string {
	var out []byte
	for _, octet := range data {
		lo := octet & 0x0F
		hi := (octet >> 4) & 0x0F
		out = appendDigit(out, lo)
		out = appendDigit(out, hi)
	}
	return string(out)
}

func digitNibble(d byte) byte {
	if d >= '0' && d <= '9' {
		return d - '0'
	}
	return 0x0F
}

func appendDigit(out []byte, nibble byte) []byte {
	if nibble <= 9 {
		return append(out, '0'+nibble)
	}
	return out
}
