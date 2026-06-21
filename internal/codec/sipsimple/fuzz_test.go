package sipsimple

import "testing"

func FuzzDecodeCPIM(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("From: <sip:+15551230001@example.com>\r\nTo: <sip:+15551230002@example.com>\r\n\r\nContent-Type: text/plain\r\n\r\nhello"))
	f.Add([]byte("From: sip:alice@example.com\n\nhello"))

	f.Fuzz(func(t *testing.T, body []byte) {
		_, _ = Decode(body, ContentTypeCPIM, "15550000001", "15550000002")
	})
}
