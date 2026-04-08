// Package sipsimple decodes and encodes SIP SIMPLE (RFC 3428) message bodies
// to and from the canonical codec.Message type.
//
// Two content types are supported:
//   - text/plain — raw UTF-8 text, minimal metadata available from SIP headers
//   - message/cpim — RFC 3862 CPIM envelope with From/To/DateTime/Subject headers
package sipsimple

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/codec"
)

const (
	ContentTypePlain = "text/plain"
	ContentTypeCPIM  = "message/cpim"
)

// Decode parses a SIP SIMPLE message body (text/plain or message/cpim) into
// a codec.Message.  The contentType parameter is the value of the SIP
// Content-Type header (may include parameters, e.g. "text/plain;charset=UTF-8").
// srcMSISDN and dstMSISDN are extracted from the SIP layer by the caller and
// supplied as hints; CPIM From/To headers override them when present.
func Decode(body []byte, contentType, srcMSISDN, dstMSISDN string) (*codec.Message, error) {
	ct := strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))

	switch ct {
	case ContentTypePlain:
		return decodePlain(body, srcMSISDN, dstMSISDN)
	case ContentTypeCPIM:
		return decodeCPIM(body, srcMSISDN, dstMSISDN)
	default:
		return nil, fmt.Errorf("sipsimple: unsupported content-type %q", ct)
	}
}

func decodePlain(body []byte, srcMSISDN, dstMSISDN string) (*codec.Message, error) {
	msg := &codec.Message{
		Text:      strings.TrimRight(string(body), "\r\n"),
		Encoding:  codec.EncodingUTF8,
		Timestamp: time.Now(),
	}
	msg.Source.MSISDN = srcMSISDN
	msg.Destination.MSISDN = dstMSISDN
	return msg, nil
}

// decodeCPIM parses an RFC 3862 CPIM body.
//
// Structure:
//
//	CPIM headers (From, To, DateTime, Subject, …)
//	blank line
//	encapsulated content-type (text/plain or similar)
//	blank line
//	payload
func decodeCPIM(body []byte, srcMSISDN, dstMSISDN string) (*codec.Message, error) {
	msg := &codec.Message{
		Encoding:  codec.EncodingUTF8,
		Timestamp: time.Now(),
	}
	msg.Source.MSISDN = srcMSISDN
	msg.Destination.MSISDN = dstMSISDN

	// Split header block from inner content
	headerBlock, rest, found := bytes.Cut(body, []byte("\r\n\r\n"))
	if !found {
		// Try LF-only line endings
		headerBlock, rest, found = bytes.Cut(body, []byte("\n\n"))
		if !found {
			return nil, fmt.Errorf("sipsimple: malformed CPIM body — no header/body separator")
		}
	}

	// Parse CPIM headers
	scanner := bufio.NewScanner(bytes.NewReader(headerBlock))
	for scanner.Scan() {
		line := scanner.Text()
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(v)
		switch strings.ToLower(key) {
		case "from":
			// "From: <sip:+14155551234@example.com>" or bare URI
			if msisdn := msisdnFromCPIMAddr(val); msisdn != "" {
				msg.Source.MSISDN = msisdn
			}
			msg.Source.SIPURI = extractURI(val)
		case "to":
			if msisdn := msisdnFromCPIMAddr(val); msisdn != "" {
				msg.Destination.MSISDN = msisdn
			}
			msg.Destination.SIPURI = extractURI(val)
		case "datetime":
			if t, err := time.Parse(time.RFC3339, val); err == nil {
				msg.Timestamp = t
			}
		case "subject":
			// Subject is not a standard SMS field; ignore for now
		}
	}

	// The rest contains the inner Content-Type header(s) + blank line + payload.
	// Skip the inner MIME headers to reach the payload.
	innerHeaders, payload, found := bytes.Cut(rest, []byte("\r\n\r\n"))
	if !found {
		innerHeaders, payload, found = bytes.Cut(rest, []byte("\n\n"))
		if !found {
			// No inner headers — treat rest as payload directly
			payload = rest
			innerHeaders = nil
		}
	}
	_ = innerHeaders // inner content-type not examined; we always expect text

	msg.Text = strings.TrimRight(string(payload), "\r\n")
	return msg, nil
}

// msisdnFromCPIMAddr extracts an E.164 MSISDN from a CPIM address like
// "<sip:+14155551234@example.com>" or "sip:+14155551234@example.com".
func msisdnFromCPIMAddr(addr string) string {
	// Strip angle brackets
	addr = strings.TrimSpace(addr)
	if i := strings.Index(addr, "<"); i >= 0 {
		if j := strings.Index(addr[i:], ">"); j >= 0 {
			addr = addr[i+1 : i+j]
		}
	}
	// Strip scheme
	if idx := strings.Index(addr, ":"); idx >= 0 {
		addr = addr[idx+1:]
	}
	// Take user part (before @)
	if idx := strings.Index(addr, "@"); idx >= 0 {
		addr = addr[:idx]
	}
	// Strip leading +
	addr = strings.TrimPrefix(addr, "+")
	if addr == "" {
		return ""
	}
	// Must be all digits to be a MSISDN
	for _, r := range addr {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return "+" + addr
}

// extractURI returns the bare URI from a CPIM address field.
func extractURI(addr string) string {
	addr = strings.TrimSpace(addr)
	if i := strings.Index(addr, "<"); i >= 0 {
		if j := strings.Index(addr[i:], ">"); j >= 0 {
			return addr[i+1 : i+j]
		}
	}
	return addr
}
