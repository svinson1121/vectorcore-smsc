// Package codec provides Diameter AVP and message encode/decode for the SMSC.
// The AVP codec is adapted from the VectorCore DRA diameter/avp package.
package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

// AVP flag bits
const (
	FlagVendorSpecific = 0x80
	FlagMandatory      = 0x40
	FlagProtected      = 0x20
)

// Well-known base AVP codes (no vendor ID)
const (
	CodeSessionID                   uint32 = 263
	CodeHostIPAddress               uint32 = 257
	CodeAuthApplicationID           uint32 = 258
	CodeAcctApplicationID           uint32 = 259
	CodeVendorSpecificApplicationID uint32 = 260
	CodeAuthSessionState            uint32 = 277
	CodeOriginHost                  uint32 = 264
	CodeSupportedVendorID           uint32 = 265
	CodeVendorID                    uint32 = 266
	CodeFirmwareRevision            uint32 = 267
	CodeResultCode                  uint32 = 268
	CodeProductName                 uint32 = 269
	CodeDisconnectCause             uint32 = 273
	CodeOriginRealm                 uint32 = 296
	CodeExperimentalResult          uint32 = 297
	CodeExperimentalResultCode      uint32 = 298
	CodeInbandSecurityID            uint32 = 299
	CodeDestinationHost             uint32 = 293
	CodeDestinationRealm            uint32 = 283
	CodeOriginStateID               uint32 = 278
	CodeUserName                    uint32 = 1
)

const (
	AuthSessionStateNoStateMaintained uint32 = 1
)

// Address family values
const (
	AddressFamilyIPv4 = 1
	AddressFamilyIPv6 = 2
)

// Inband-Security-Id values
const (
	InbandSecurityNoSec = 0
)

// AVP represents a single Diameter AVP.
type AVP struct {
	Code     uint32
	VendorID uint32
	Flags    byte
	Data     []byte
	Children []*AVP
}

// Encode encodes the AVP to wire format (with padding).
func Encode(a *AVP) ([]byte, error) {
	hdrSize := 8
	if a.VendorID != 0 {
		hdrSize = 12
	}
	dataLen := len(a.Data)
	totalLen := hdrSize + dataLen
	padded := (totalLen + 3) &^ 3
	buf := make([]byte, padded)

	binary.BigEndian.PutUint32(buf[0:4], a.Code)

	flags := a.Flags
	if a.VendorID != 0 {
		flags |= FlagVendorSpecific
	} else {
		flags &^= FlagVendorSpecific
	}
	buf[4] = flags
	length := uint32(totalLen)
	buf[5] = byte(length >> 16)
	buf[6] = byte(length >> 8)
	buf[7] = byte(length)

	if a.VendorID != 0 {
		binary.BigEndian.PutUint32(buf[8:12], a.VendorID)
		copy(buf[12:], a.Data)
	} else {
		copy(buf[8:], a.Data)
	}
	return buf, nil
}

// Decode decodes a single AVP from b, returning the AVP and bytes consumed.
func Decode(b []byte) (*AVP, int, error) {
	if len(b) < 8 {
		return nil, 0, errors.New("avp: buffer too short")
	}
	a := &AVP{}
	a.Code = binary.BigEndian.Uint32(b[0:4])
	a.Flags = b[4]
	length := uint32(b[5])<<16 | uint32(b[6])<<8 | uint32(b[7])
	if length < 8 {
		return nil, 0, fmt.Errorf("avp: invalid length %d", length)
	}
	hdrSize := 8
	if a.Flags&FlagVendorSpecific != 0 {
		if len(b) < 12 {
			return nil, 0, errors.New("avp: buffer too short for vendor header")
		}
		a.VendorID = binary.BigEndian.Uint32(b[8:12])
		hdrSize = 12
	}
	dataLen := int(length) - hdrSize
	if dataLen < 0 || len(b) < int(length) {
		return nil, 0, fmt.Errorf("avp: buffer too short for data: code=%d", a.Code)
	}
	a.Data = make([]byte, dataLen)
	copy(a.Data, b[hdrSize:int(length)])
	consumed := (int(length) + 3) &^ 3
	if consumed > len(b) {
		consumed = len(b)
	}
	return a, consumed, nil
}

// DecodeAll decodes all AVPs from b.
func DecodeAll(b []byte) ([]*AVP, error) {
	var avps []*AVP
	for len(b) > 0 {
		if len(b) < 8 {
			return nil, fmt.Errorf("avp: trailing bytes too short: %d", len(b))
		}
		a, n, err := Decode(b)
		if err != nil {
			return nil, err
		}
		avps = append(avps, a)
		b = b[n:]
	}
	return avps, nil
}

// DecodeGrouped decodes the children of a Grouped AVP.
func DecodeGrouped(a *AVP) ([]*AVP, error) {
	children, err := DecodeAll(a.Data)
	if err != nil {
		return nil, fmt.Errorf("avp: grouped %d: %w", a.Code, err)
	}
	a.Children = children
	return children, nil
}

// NewUint32 creates a Uint32 AVP.
func NewUint32(code, vendorID uint32, flags byte, val uint32) *AVP {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, val)
	a := &AVP{Code: code, VendorID: vendorID, Flags: flags, Data: data}
	if vendorID != 0 {
		a.Flags |= FlagVendorSpecific
	}
	return a
}

// NewString creates a string AVP.
func NewString(code, vendorID uint32, flags byte, val string) *AVP {
	a := &AVP{Code: code, VendorID: vendorID, Flags: flags, Data: []byte(val)}
	if vendorID != 0 {
		a.Flags |= FlagVendorSpecific
	}
	return a
}

// NewOctetString creates a raw bytes AVP.
func NewOctetString(code, vendorID uint32, flags byte, val []byte) *AVP {
	data := make([]byte, len(val))
	copy(data, val)
	a := &AVP{Code: code, VendorID: vendorID, Flags: flags, Data: data}
	if vendorID != 0 {
		a.Flags |= FlagVendorSpecific
	}
	return a
}

// NewAddress creates an Address-type AVP.
func NewAddress(code, vendorID uint32, flags byte, ip net.IP) *AVP {
	var data []byte
	if ip4 := ip.To4(); ip4 != nil {
		data = make([]byte, 6)
		binary.BigEndian.PutUint16(data[0:2], AddressFamilyIPv4)
		copy(data[2:], ip4)
	} else {
		ip6 := ip.To16()
		data = make([]byte, 18)
		binary.BigEndian.PutUint16(data[0:2], AddressFamilyIPv6)
		copy(data[2:], ip6)
	}
	a := &AVP{Code: code, VendorID: vendorID, Flags: flags, Data: data}
	if vendorID != 0 {
		a.Flags |= FlagVendorSpecific
	}
	return a
}

// NewGrouped creates a Grouped AVP from child AVPs.
func NewGrouped(code, vendorID uint32, flags byte, children []*AVP) (*AVP, error) {
	var data []byte
	for _, child := range children {
		enc, err := Encode(child)
		if err != nil {
			return nil, fmt.Errorf("avp: grouped child %d: %w", child.Code, err)
		}
		data = append(data, enc...)
	}
	a := &AVP{Code: code, VendorID: vendorID, Flags: flags, Data: data, Children: children}
	if vendorID != 0 {
		a.Flags |= FlagVendorSpecific
	}
	return a, nil
}

// Uint32 returns the AVP value as uint32.
func (a *AVP) Uint32() (uint32, error) {
	if len(a.Data) != 4 {
		return 0, fmt.Errorf("avp: expected 4 bytes, got %d (code=%d)", len(a.Data), a.Code)
	}
	return binary.BigEndian.Uint32(a.Data), nil
}

// String returns the AVP value as string.
func (a *AVP) String() (string, error) {
	return string(a.Data), nil
}
