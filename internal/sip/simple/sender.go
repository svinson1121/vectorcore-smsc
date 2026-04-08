package simple

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/svinson1121/vectorcore-smsc/internal/codec"
	"github.com/svinson1121/vectorcore-smsc/internal/codec/sipsimple"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

// Sender delivers MT SMS to inter-site peers over SIP SIMPLE.
type Sender struct {
	client      *sipgo.Client
	localDomain string
}

// NewSender creates a Sender using the given sipgo client.
// localDomain is placed in the From header URI.
func NewSender(client *sipgo.Client, localDomain string) *Sender {
	return &Sender{client: client, localDomain: localDomain}
}

// Send delivers msg to the named peer.
func (s *Sender) Send(ctx context.Context, msg *codec.Message, peer store.SIPPeer) error {
	// Build target URI: sip:<dst>@<peer.Address>
	targetStr := sipPeerURI(fmt.Sprintf("+%s", msg.Destination.MSISDN), peer)
	var targetURI sip.Uri
	if err := sip.ParseUri(targetStr, &targetURI); err != nil {
		return fmt.Errorf("parse target URI: %w", err)
	}

	req := sip.NewRequest(sip.MESSAGE, targetURI)

	// From: local MSISDN at our domain (always override sipgo default).
	fromDomain := strings.TrimSpace(s.localDomain)
	if fromDomain == "" {
		fromDomain = strings.TrimSpace(peer.Address)
	}
	if fromDomain == "" {
		fromDomain = "localhost"
	}
	fromStr := fmt.Sprintf("sip:+%s@%s", msg.Source.MSISDN, fromDomain)
	var fromURI sip.Uri
	if err := sip.ParseUri(fromStr, &fromURI); err != nil {
		return fmt.Errorf("parse from URI: %w", err)
	}
	req.AppendHeader(&sip.FromHeader{Address: fromURI})

	// Build body — prefer CPIM for rich metadata
	body := sipsimple.EncodeCPIM(msg, fromStr, targetStr)
	req.AppendHeader(sip.NewHeader("Content-Type", sipsimple.ContentTypeCPIM))
	req.SetBody(body)

	tx, err := s.client.TransactionRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("send SIP SIMPLE MESSAGE: %w", err)
	}

	select {
	case resp := <-tx.Responses():
		if resp == nil {
			return fmt.Errorf("no response from peer %s", peer.Name)
		}
		code := resp.StatusCode
		slog.Info("SIMPLE MT delivered",
			"peer", peer.Name,
			"dst", msg.Destination.MSISDN,
			"status", code,
		)
		if code >= 200 && code < 300 {
			return nil
		}
		return fmt.Errorf("peer %s returned %d %s", peer.Name, code, resp.Reason)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SendIMDN sends an IMDN delivery notification back to the originating peer.
// messageID is the original Call-ID (used as IMDN message-id).
func (s *Sender) SendIMDN(ctx context.Context, messageID string, peer store.SIPPeer, status string) error {
	targetStr := sipPeerURI("", peer)
	var targetURI sip.Uri
	if err := sip.ParseUri(targetStr, &targetURI); err != nil {
		return fmt.Errorf("parse IMDN target: %w", err)
	}

	req := sip.NewRequest(sip.MESSAGE, targetURI)
	fromDomain := strings.TrimSpace(s.localDomain)
	if fromDomain == "" {
		fromDomain = strings.TrimSpace(peer.Address)
	}
	if fromDomain == "" {
		fromDomain = "localhost"
	}
	fromStr := fmt.Sprintf("sip:smsc@%s", fromDomain)
	var fromURI sip.Uri
	if err := sip.ParseUri(fromStr, &fromURI); err == nil {
		req.AppendHeader(&sip.FromHeader{Address: fromURI})
	}
	req.AppendHeader(sip.NewHeader("Content-Type", IMDNContentType))
	req.SetBody(IMDNBody(messageID, status))

	tx, err := s.client.TransactionRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("send IMDN: %w", err)
	}

	select {
	case resp := <-tx.Responses():
		if resp != nil && resp.StatusCode >= 300 {
			return fmt.Errorf("IMDN rejected: %d %s", resp.StatusCode, resp.Reason)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func sipPeerURI(user string, peer store.SIPPeer) string {
	target := "sip:"
	if user != "" {
		target += user + "@"
	}
	target += fmt.Sprintf("%s:%d", peer.Address, peer.Port)
	if transport := strings.ToLower(strings.TrimSpace(peer.Transport)); transport != "" {
		target += ";transport=" + transport
	}
	return target
}
