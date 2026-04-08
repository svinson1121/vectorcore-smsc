package isc

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/emiago/sipgo/sip"
	"github.com/svinson1121/vectorcore-smsc/internal/registry"
)

// RegisterHandler handles 3rd-party REGISTER and NOTIFY requests from the
// S-CSCF to maintain the local IMS registration cache.
type RegisterHandler struct {
	Registry *registry.Registry
}

// HandleRegister processes a 3rd-party REGISTER from the S-CSCF.
// On registration: updates ims_registrations.
// On de-registration (Expires: 0 or Contact: *): removes the entry.
func (h *RegisterHandler) HandleRegister(req *sip.Request, tx sip.ServerTransaction) {
	to := req.To()
	if to == nil {
		respond(tx, req, 400, "Missing To")
		return
	}

	expires := expiresValue(req)
	contact := req.GetHeader("Contact")
	deregister := expires == 0 || (contact != nil && strings.Contains(contact.Value(), "*"))

	// Collect all IMPUs: To URI + P-Associated-URI entries.
	impus := []string{to.Address.String()}
	if pau := req.GetHeader("P-Associated-URI"); pau != nil {
		for _, part := range strings.Split(pau.Value(), ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				impus = append(impus, part)
			}
		}
	}

	// Classify each IMPU as MSISDN or IMSI-derived.
	// IMSI-derived: exactly 15 all-digit user part (MCC+MNC+MSIN).
	// MSISDN: shorter all-digit user part (dialable number).
	msisdn, imsi := "", ""
	var sipAOR string
	for _, raw := range impus {
		user := msisdnFromURI(raw)
		if user == "" {
			continue
		}
		if isIMSI(user) {
			if imsi == "" {
				imsi = user
			}
		} else {
			if msisdn == "" {
				msisdn = user
				sipAOR = strings.Trim(strings.TrimSpace(raw), "<>")
				if semi := strings.Index(sipAOR, ";"); semi >= 0 {
					sipAOR = sipAOR[:semi]
				}
			}
		}
	}

	// Fall back to the To URI user part if no MSISDN IMPU was found.
	if msisdn == "" {
		msisdn = msisdnFromURI(to.Address.String())
		sipAOR = to.Address.String()
	}
	if msisdn == "" {
		respond(tx, req, 200, "OK")
		return
	}

	if deregister {
		slog.Info("ISC de-register", "msisdn", msisdn, "imsi", imsi)
		if err := h.Registry.Delete(context.Background(), msisdn); err != nil {
			slog.Error("registry delete failed", "msisdn", msisdn, "err", err)
		}
		respond(tx, req, 200, "OK")
		return
	}

	// Extract UE contact from the inner sipfrag body (message/sip).
	// The outer Contact is the S-CSCF address, not the UE.
	contactURI := sipfragContact(req.Body())
	slog.Debug("ISC register sipfrag", "body_len", len(req.Body()), "contact_found", contactURI)
	if contactURI == "" && contact != nil {
		// Fallback: use outer Contact if no sipfrag body.
		contactURI = strings.Trim(strings.TrimSpace(contact.Value()), "<>")
		if semi := strings.Index(contactURI, ";"); semi >= 0 {
			contactURI = contactURI[:semi]
		}
	}

	scscf := ""
	if via := req.Via(); via != nil {
		scscf = via.Host
	}

	expiry := time.Now().Add(time.Duration(expires) * time.Second)
	reg := registry.Registration{
		MSISDN:     msisdn,
		IMSI:       imsi,
		SIPAOR:     sipAOR,
		ContactURI: contactURI,
		SCSCF:      scscf,
		Registered: true,
		Expiry:     expiry,
	}

	slog.Info("ISC register", "msisdn", msisdn, "imsi", imsi, "contact", contactURI, "expires", expires)
	if err := h.Registry.Upsert(context.Background(), reg); err != nil {
		slog.Error("registry upsert failed", "msisdn", msisdn, "err", err)
		respond(tx, req, 500, "Internal Server Error")
		return
	}

	respond(tx, req, 200, "OK")
}

// isIMSI returns true if s looks like an IMSI-derived user part:
// exactly 15 digits (MCC 3 + MNC 2-3 + MSIN).
func isIMSI(s string) bool {
	if len(s) != 15 {
		return false
	}
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

// sipfragContact extracts the Contact URI from an inner message/sip body
// (3rd-party REGISTER sipfrag). Returns the bare SIP URI without parameters.
func sipfragContact(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimRight(line, "\r")
		lower := strings.ToLower(line)
		if !strings.HasPrefix(lower, "contact:") && !strings.HasPrefix(lower, "m:") {
			continue
		}
		val := strings.TrimSpace(line[strings.Index(line, ":")+1:])
		// Strip angle brackets and parameters.
		val = strings.Trim(val, "<>")
		if lt := strings.Index(val, "<"); lt >= 0 {
			val = val[lt+1:]
			if gt := strings.Index(val, ">"); gt >= 0 {
				val = val[:gt]
			}
		}
		if semi := strings.Index(val, ";"); semi >= 0 {
			val = val[:semi]
		}
		val = strings.TrimSpace(val)
		if strings.HasPrefix(strings.ToLower(val), "sip:") || strings.HasPrefix(strings.ToLower(val), "sips:") {
			return val
		}
	}
	return ""
}

// HandleNotify processes a NOTIFY for the reg event package from the S-CSCF.
// The body carries an application/reginfo+xml or similar document.
// For Phase 1 we handle the simple case: extract registration state from
// the Subscription-State and P-Asserted-Identity headers.
func (h *RegisterHandler) HandleNotify(req *sip.Request, tx sip.ServerTransaction) {
	subState := req.GetHeader("Subscription-State")
	if subState == nil {
		respond(tx, req, 200, "OK")
		return
	}

	state := strings.ToLower(strings.Fields(subState.Value())[0])
	if state == "terminated" {
		// De-registration indicated via NOTIFY termination
		if pai := req.GetHeader("P-Asserted-Identity"); pai != nil {
			msisdn := msisdnFromURI(pai.Value())
			if msisdn != "" {
				slog.Info("ISC NOTIFY de-register", "msisdn", msisdn)
				if err := h.Registry.Delete(context.Background(), msisdn); err != nil {
					slog.Error("registry delete failed (notify)", "msisdn", msisdn, "err", err)
				}
			}
		}
	}

	respond(tx, req, 200, "OK")
}

// expiresValue returns the Expires header value, defaulting to 3600.
func expiresValue(req *sip.Request) int {
	exp := req.GetHeader("Expires")
	if exp == nil {
		return 3600
	}
	v, err := strconv.Atoi(strings.TrimSpace(exp.Value()))
	if err != nil {
		return 3600
	}
	return v
}
