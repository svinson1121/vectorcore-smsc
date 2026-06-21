package smpp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	HeaderLen = 16
	// MaxPDULen bounds memory allocated from the untrusted command_length
	// field. SMPP uses a 32-bit length but normal SMS PDUs are far smaller.
	MaxPDULen = 16 * 1024 * 1024
)

// Encode serialises a PDU to its wire representation.
// CommandLength is set automatically.
func Encode(pdu *PDU) ([]byte, error) {
	body, err := encodeBody(pdu)
	if err != nil {
		return nil, err
	}

	pdu.CommandLength = uint32(HeaderLen + len(body))
	if pdu.CommandLength > MaxPDULen {
		return nil, fmt.Errorf("smpp: command_length %d exceeds maximum %d", pdu.CommandLength, MaxPDULen)
	}

	buf := make([]byte, HeaderLen+len(body))
	binary.BigEndian.PutUint32(buf[0:4], pdu.CommandLength)
	binary.BigEndian.PutUint32(buf[4:8], pdu.CommandID)
	binary.BigEndian.PutUint32(buf[8:12], pdu.CommandStatus)
	binary.BigEndian.PutUint32(buf[12:16], pdu.SequenceNumber)
	copy(buf[16:], body)
	return buf, nil
}

// Decode reads exactly one PDU from r.
func Decode(r io.Reader) (*PDU, error) {
	hdr := make([]byte, HeaderLen)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return nil, err
	}

	pdu := &PDU{}
	pdu.CommandLength = binary.BigEndian.Uint32(hdr[0:4])
	pdu.CommandID = binary.BigEndian.Uint32(hdr[4:8])
	pdu.CommandStatus = binary.BigEndian.Uint32(hdr[8:12])
	pdu.SequenceNumber = binary.BigEndian.Uint32(hdr[12:16])

	if pdu.CommandLength < HeaderLen {
		return nil, fmt.Errorf("smpp: invalid command_length %d", pdu.CommandLength)
	}
	if pdu.CommandLength > MaxPDULen {
		return nil, fmt.Errorf("smpp: command_length %d exceeds maximum %d", pdu.CommandLength, MaxPDULen)
	}

	bodyLen := int(pdu.CommandLength) - HeaderLen
	if bodyLen > 0 {
		body := make([]byte, bodyLen)
		if _, err := io.ReadFull(r, body); err != nil {
			return nil, err
		}
		if err := decodeBody(pdu, body); err != nil {
			return nil, err
		}
	}

	return pdu, nil
}

// ---- body encode ----

func encodeBody(pdu *PDU) ([]byte, error) {
	var buf bytes.Buffer

	switch pdu.CommandID {
	case CmdBindReceiver, CmdBindTransmitter, CmdBindTransceiver:
		writeCStr(&buf, pdu.SystemID)
		writeCStr(&buf, pdu.Password)
		writeCStr(&buf, pdu.SystemType)
		buf.WriteByte(pdu.InterfaceVersion)
		buf.WriteByte(pdu.AddrTON)
		buf.WriteByte(pdu.AddrNPI)
		writeCStr(&buf, pdu.AddressRange)

	case CmdBindReceiverResp, CmdBindTransmitterResp, CmdBindTransceiverResp:
		writeCStr(&buf, pdu.SystemID)

	case CmdSubmitSM, CmdDeliverSM:
		if len(pdu.ShortMessage) > 255 {
			return nil, fmt.Errorf("smpp: short_message length %d exceeds 255", len(pdu.ShortMessage))
		}
		writeCStr(&buf, pdu.ServiceType)
		buf.WriteByte(pdu.SourceAddrTON)
		buf.WriteByte(pdu.SourceAddrNPI)
		writeCStr(&buf, pdu.SourceAddr)
		buf.WriteByte(pdu.DestAddrTON)
		buf.WriteByte(pdu.DestAddrNPI)
		writeCStr(&buf, pdu.DestinationAddr)
		buf.WriteByte(pdu.ESMClass)
		buf.WriteByte(pdu.ProtocolID)
		buf.WriteByte(pdu.PriorityFlag)
		writeCStr(&buf, pdu.ScheduleDeliveryTime)
		writeCStr(&buf, pdu.ValidityPeriod)
		buf.WriteByte(pdu.RegisteredDelivery)
		buf.WriteByte(pdu.ReplaceIfPresentFlag)
		buf.WriteByte(pdu.DataCoding)
		buf.WriteByte(pdu.SMDefaultMsgID)
		pdu.SMLength = byte(len(pdu.ShortMessage))
		buf.WriteByte(pdu.SMLength)
		buf.Write(pdu.ShortMessage)
		// Append optional TLV parameters
		if err := encodeTLVs(&buf, pdu.TLVs); err != nil {
			return nil, err
		}

	case CmdSubmitSMResp, CmdDeliverSMResp:
		writeCStr(&buf, pdu.MessageID)

	case CmdEnquireLink, CmdEnquireLinkResp,
		CmdUnbind, CmdUnbindResp,
		CmdGenericNack:
		// header-only PDUs — no body

	default:
		// unknown: no body
	}

	return buf.Bytes(), nil
}

// ---- body decode ----

func decodeBody(pdu *PDU, body []byte) error {
	r := bytes.NewReader(body)
	var err error

	switch pdu.CommandID {
	case CmdBindReceiver, CmdBindTransmitter, CmdBindTransceiver:
		if pdu.SystemID, err = readCStr(r); err != nil {
			return err
		}
		if pdu.Password, err = readCStr(r); err != nil {
			return err
		}
		if pdu.SystemType, err = readCStr(r); err != nil {
			return err
		}
		if pdu.InterfaceVersion, err = r.ReadByte(); err != nil {
			return err
		}
		if pdu.AddrTON, err = r.ReadByte(); err != nil {
			return err
		}
		if pdu.AddrNPI, err = r.ReadByte(); err != nil {
			return err
		}
		if pdu.AddressRange, err = readCStr(r); err != nil {
			return err
		}

	case CmdBindReceiverResp, CmdBindTransmitterResp, CmdBindTransceiverResp:
		if pdu.SystemID, err = readCStr(r); err != nil {
			return err
		}

	case CmdSubmitSM, CmdDeliverSM:
		if pdu.ServiceType, err = readCStr(r); err != nil {
			return err
		}
		if pdu.SourceAddrTON, err = r.ReadByte(); err != nil {
			return err
		}
		if pdu.SourceAddrNPI, err = r.ReadByte(); err != nil {
			return err
		}
		if pdu.SourceAddr, err = readCStr(r); err != nil {
			return err
		}
		if pdu.DestAddrTON, err = r.ReadByte(); err != nil {
			return err
		}
		if pdu.DestAddrNPI, err = r.ReadByte(); err != nil {
			return err
		}
		if pdu.DestinationAddr, err = readCStr(r); err != nil {
			return err
		}
		if pdu.ESMClass, err = r.ReadByte(); err != nil {
			return err
		}
		if pdu.ProtocolID, err = r.ReadByte(); err != nil {
			return err
		}
		if pdu.PriorityFlag, err = r.ReadByte(); err != nil {
			return err
		}
		if pdu.ScheduleDeliveryTime, err = readCStr(r); err != nil {
			return err
		}
		if pdu.ValidityPeriod, err = readCStr(r); err != nil {
			return err
		}
		if pdu.RegisteredDelivery, err = r.ReadByte(); err != nil {
			return err
		}
		if pdu.ReplaceIfPresentFlag, err = r.ReadByte(); err != nil {
			return err
		}
		if pdu.DataCoding, err = r.ReadByte(); err != nil {
			return err
		}
		if pdu.SMDefaultMsgID, err = r.ReadByte(); err != nil {
			return err
		}
		if pdu.SMLength, err = r.ReadByte(); err != nil {
			return err
		}
		pdu.ShortMessage = make([]byte, pdu.SMLength)
		if _, err = io.ReadFull(r, pdu.ShortMessage); err != nil {
			return err
		}
		// Decode any trailing TLV parameters
		if err = decodeTLVs(pdu, r); err != nil {
			return err
		}

	case CmdSubmitSMResp, CmdDeliverSMResp:
		if pdu.MessageID, err = readCStr(r); err != nil {
			// deliver_sm_resp may have empty body — tolerate EOF
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				pdu.MessageID = ""
				return nil
			}
			return err
		}
	}

	return nil
}

// ---- TLV encode/decode ----

func encodeTLVs(buf *bytes.Buffer, tlvs map[uint16][]byte) error {
	for tag, val := range tlvs {
		if len(val) > 65535 {
			return fmt.Errorf("smpp: TLV 0x%04x length %d exceeds 65535", tag, len(val))
		}
		if err := binary.Write(buf, binary.BigEndian, tag); err != nil {
			return err
		}
		if err := binary.Write(buf, binary.BigEndian, uint16(len(val))); err != nil {
			return err
		}
		buf.Write(val)
	}
	return nil
}

func decodeTLVs(pdu *PDU, r *bytes.Reader) error {
	for r.Len() >= 4 {
		var tag, length uint16
		if err := binary.Read(r, binary.BigEndian, &tag); err != nil {
			return err
		}
		if err := binary.Read(r, binary.BigEndian, &length); err != nil {
			return err
		}
		val := make([]byte, length)
		if _, err := io.ReadFull(r, val); err != nil {
			return err
		}
		if pdu.TLVs == nil {
			pdu.TLVs = make(map[uint16][]byte)
		}
		pdu.TLVs[tag] = val
	}
	if r.Len() != 0 {
		return fmt.Errorf("smpp: trailing %d byte(s) after TLVs", r.Len())
	}
	return nil
}

// ---- helpers ----

func writeCStr(w *bytes.Buffer, s string) {
	w.WriteString(s)
	w.WriteByte(0x00)
}

func readCStr(r *bytes.Reader) (string, error) {
	var buf []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			return "", fmt.Errorf("smpp: unterminated C-octet string: %w", err)
		}
		if b == 0x00 {
			break
		}
		buf = append(buf, b)
	}
	return string(buf), nil
}
