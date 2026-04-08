package sgd

import (
	"log/slog"

	"github.com/svinson1121/vectorcore-smsc/internal/diameter"
	dcodec "github.com/svinson1121/vectorcore-smsc/internal/diameter/codec"
)

// newSessionID generates a unique Diameter Session-Id string.
func newSessionID() string { return dcodec.NewSessionID("vectorcore-smsc") }

// sendAnswer sends a simple answer with just Result-Code, Origin-Host, Origin-Realm.
func sendAnswer(p *diameter.Peer, req *dcodec.Message, resultCode uint32) {
	b := dcodec.NewAnswer(req)
	b.Add(
		dcodec.NewUint32(dcodec.CodeResultCode, 0, dcodec.FlagMandatory, resultCode),
		dcodec.NewString(dcodec.CodeOriginHost, 0, dcodec.FlagMandatory, p.Cfg().LocalFQDN),
		dcodec.NewString(dcodec.CodeOriginRealm, 0, dcodec.FlagMandatory, p.Cfg().LocalRealm),
	)
	if err := p.Send(b.Build()); err != nil {
		slog.Warn("sgd answer send failed",
			"peer", p.RemoteFQDN,
			"command_code", req.Header.CommandCode,
			"hop_by_hop", req.Header.HopByHop,
			"result_code", resultCode,
			"err", err,
		)
		return
	}
	slog.Debug("sgd answer sent",
		"peer", p.RemoteFQDN,
		"command_code", req.Header.CommandCode,
		"hop_by_hop", req.Header.HopByHop,
		"result_code", resultCode,
	)
}

// encodeBCDMSISDN and encodeSCAddress are re-exported from codec/sgd for use
// within this package without a cross-import.

func encodeBCDMSISDN(msisdn string) []byte {
	international := false
	if len(msisdn) > 0 && msisdn[0] == '+' {
		international = true
		msisdn = msisdn[1:]
	}
	noa := byte(0x81)
	if international {
		noa = 0x91
	}
	digits := encodeBCDDigits(msisdn)
	result := make([]byte, 2+len(digits))
	result[0] = byte(1 + len(digits))
	result[1] = noa
	copy(result[2:], digits)
	return result
}

func encodeSCAddress(sc string) []byte {
	if sc == "" {
		return nil
	}
	if sc[0] == '+' {
		sc = sc[1:]
	}
	return encodeBCDDigits(sc)
}

func encodeBCDDigits(digits string) []byte {
	size := (len(digits) + 1) / 2
	buf := make([]byte, size)
	for i := 0; i < len(digits); i++ {
		d := digits[i] - '0'
		if i%2 == 0 {
			buf[i/2] = d
		} else {
			buf[i/2] |= d << 4
		}
	}
	if len(digits)%2 != 0 {
		buf[size-1] |= 0xF0
	}
	return buf
}
