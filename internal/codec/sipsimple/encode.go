package sipsimple

import (
	"fmt"
	"strings"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/codec"
)

// EncodePlain returns a text/plain body for the message.
func EncodePlain(msg *codec.Message) []byte {
	return []byte(msg.Text)
}

// EncodeCPIM returns a message/cpim body per RFC 3862.
// fromURI and toURI are the SIP URIs of sender and recipient.
func EncodeCPIM(msg *codec.Message, fromURI, toURI string) []byte {
	if fromURI == "" && msg.Source.SIPURI != "" {
		fromURI = msg.Source.SIPURI
	}
	if fromURI == "" && msg.Source.MSISDN != "" {
		fromURI = "sip:" + msg.Source.MSISDN + "@unknown"
	}
	if toURI == "" && msg.Destination.SIPURI != "" {
		toURI = msg.Destination.SIPURI
	}
	if toURI == "" && msg.Destination.MSISDN != "" {
		toURI = "sip:" + msg.Destination.MSISDN + "@unknown"
	}

	ts := msg.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	var b strings.Builder
	// CPIM headers
	fmt.Fprintf(&b, "From: <%s>\r\n", fromURI)
	fmt.Fprintf(&b, "To: <%s>\r\n", toURI)
	fmt.Fprintf(&b, "DateTime: %s\r\n", ts.UTC().Format(time.RFC3339))
	b.WriteString("\r\n")
	// Inner MIME header
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	// Payload
	b.WriteString(msg.Text)

	return []byte(b.String())
}
