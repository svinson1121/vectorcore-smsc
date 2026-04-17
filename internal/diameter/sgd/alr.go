package sgd

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/diameter"
	dcodec "github.com/svinson1121/vectorcore-smsc/internal/diameter/codec"
)

// AlertServiceCentreRequest is the decoded inbound SGd ALR payload.
type AlertServiceCentreRequest struct {
	SessionID                 string
	OriginHost                string
	OriginRealm               string
	MSISDN                    string
	IMSI                      string
	SCAddress                 string
	AlertCorrelationID        string
	AlertReason               uint32
	MaximumUEAvailabilityTime *time.Time
	SMSGMSCAlertEvent         uint32
}

func parseAlertServiceCentre(msg *dcodec.Message) (AlertServiceCentreRequest, error) {
	req := AlertServiceCentreRequest{
		SessionID:   parseStringAVP(msg, dcodec.CodeSessionID, 0),
		OriginHost:  parseStringAVP(msg, dcodec.CodeOriginHost, 0),
		OriginRealm: parseStringAVP(msg, dcodec.CodeOriginRealm, 0),
	}
	if req.SessionID == "" {
		return AlertServiceCentreRequest{}, fmt.Errorf("missing Session-Id")
	}
	req.IMSI, req.MSISDN = parseUserIdentifier(msg)
	if req.IMSI == "" {
		req.IMSI = parseStringAVP(msg, dcodec.CodeUserName, 0)
	}
	if req.MSISDN == "" {
		if msisdn := msg.FindAVP(dcodec.CodeMSISDN, dcodec.Vendor3GPP); msisdn != nil {
			req.MSISDN = decodeBCDMSISDN(msisdn.Data)
		}
	}
	if scAddr := msg.FindAVP(dcodec.CodeSCAddress, dcodec.Vendor3GPP); scAddr != nil {
		req.SCAddress = decodeSCAddress(scAddr.Data)
	}
	if corr := msg.FindAVP(dcodec.CodeSMSMICorrelationID, dcodec.Vendor3GPP); corr != nil && len(corr.Data) > 0 {
		req.AlertCorrelationID = base64.StdEncoding.EncodeToString(corr.Data)
	}
	if alertReason := msg.FindAVP(dcodec.CodeAlertReason, dcodec.Vendor3GPP); alertReason != nil {
		req.AlertReason, _ = alertReason.Uint32()
	}
	if alertEvent := msg.FindAVP(dcodec.CodeSMSGMSCAlertEvent, dcodec.Vendor3GPP); alertEvent != nil {
		req.SMSGMSCAlertEvent, _ = alertEvent.Uint32()
	}
	if maxAvail := msg.FindAVP(dcodec.CodeMaximumUEAvailability, dcodec.Vendor3GPP); maxAvail != nil && len(maxAvail.Data) == 4 {
		if secs, err := maxAvail.Uint32(); err == nil {
			t := decodeDiameterTime(secs)
			req.MaximumUEAvailabilityTime = &t
		}
	}
	return req, nil
}

func parseStringAVP(msg *dcodec.Message, code, vendorID uint32) string {
	avp := msg.FindAVP(code, vendorID)
	if avp == nil {
		return ""
	}
	if val, err := avp.String(); err == nil {
		return val
	}
	return string(avp.Data)
}

func parseUserIdentifier(msg *dcodec.Message) (imsi, msisdn string) {
	uid := msg.FindAVP(dcodec.CodeUserIdentifier, dcodec.Vendor3GPP)
	if uid == nil {
		return "", ""
	}
	children, err := dcodec.DecodeGrouped(uid)
	if err != nil {
		return "", ""
	}
	for _, child := range children {
		switch {
		case child.Code == dcodec.CodeUserName && child.VendorID == 0:
			if val, err := child.String(); err == nil {
				imsi = val
			}
		case child.Code == dcodec.CodeMSISDN && child.VendorID == dcodec.Vendor3GPP:
			msisdn = decodeBCDMSISDN(child.Data)
		}
	}
	return imsi, msisdn
}

func decodeBCDMSISDN(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	offset := 0
	international := false
	if data[0] < byte(len(data)) {
		offset = 1
	}
	if offset < len(data) && data[offset]&0x80 != 0 {
		noa := data[offset]
		offset++
		international = (noa>>4)&0x7 == 1
	}
	digits := decodeBCDDigits(data[offset:])
	if international {
		return "+" + digits
	}
	return digits
}

func decodeSCAddress(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	if isASCIIDigits(data) {
		return string(data)
	}
	if data[0] < 0x80 {
		return decodeBCDDigits(data)
	}
	return decodeBCDMSISDN(data)
}

func isASCIIDigits(data []byte) bool {
	for _, b := range data {
		if b < '0' || b > '9' {
			return false
		}
	}
	return len(data) > 0
}

func decodeBCDDigits(data []byte) string {
	buf := make([]byte, 0, len(data)*2)
	for _, b := range data {
		lo := b & 0x0F
		hi := (b >> 4) & 0x0F
		if lo <= 9 {
			buf = append(buf, '0'+lo)
		}
		if hi <= 9 {
			buf = append(buf, '0'+hi)
		}
	}
	return string(buf)
}

func decodeDiameterTime(v uint32) time.Time {
	const diameterTimeOffset = 2208988800
	if v <= diameterTimeOffset {
		return time.Unix(0, 0).UTC()
	}
	return time.Unix(int64(v-diameterTimeOffset), 0).UTC()
}

func (s *Server) handleAlertServiceCentreRequest(p *diameter.Peer, msg *dcodec.Message) {
	req, err := parseAlertServiceCentre(msg)
	if err != nil {
		sendAnswer(p, msg, dcodec.DiameterUnableToComply)
		return
	}

	s.mu.RLock()
	fn := s.onAlertSC
	s.mu.RUnlock()

	resultCode := dcodec.DiameterSuccess
	if fn != nil {
		if err := fn(req); err != nil {
			resultCode = dcodec.DiameterUnableToComply
		}
	}
	sendAnswer(p, msg, resultCode)
}
