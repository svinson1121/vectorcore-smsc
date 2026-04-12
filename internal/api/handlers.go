package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/crypto/bcrypt"

	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func dbErr(err error) error {
	if err == nil {
		return nil
	}
	slog.Error("database error", "err", err)
	return huma.Error500InternalServerError(err.Error(), err)
}
func notFound(msg string) error { return huma.Error404NotFound(msg, nil) }

func routingRulePeerReference(ctx context.Context, st store.Store, peerID string) (*store.RoutingRule, error) {
	rules, err := st.ListAllRoutingRules(ctx)
	if err != nil {
		return nil, err
	}
	for _, rule := range rules {
		if rule.MatchSrcPeer == peerID || rule.EgressPeer == peerID {
			return &rule, nil
		}
	}
	return nil, nil
}

func routingRulePolicyReference(ctx context.Context, st store.Store, policyID string) (*store.RoutingRule, error) {
	rules, err := st.ListAllRoutingRules(ctx)
	if err != nil {
		return nil, err
	}
	for _, rule := range rules {
		if rule.SFPolicyID == policyID {
			return &rule, nil
		}
	}
	return nil, nil
}

func routingRuleLabel(rule store.RoutingRule) string {
	if rule.Name != "" {
		return rule.Name
	}
	return fmt.Sprintf("priority %d", rule.Priority)
}

// ── SMPP Server Accounts ─────────────────────────────────────────────────────

type smppAccInput struct {
	Name            string `json:"name"`
	SystemID        string `json:"system_id"`
	Password        string `json:"password,omitempty" doc:"Plaintext — hashed on save; omit to keep existing"`
	AllowedIP       string `json:"allowed_ip,omitempty"`
	BindType        string `json:"bind_type" enum:"transmitter,receiver,transceiver"`
	ThroughputLimit int    `json:"throughput_limit" doc:"msg/sec; 0=unlimited"`
	Enabled         bool   `json:"enabled"`
}

func registerSMPPAccounts(api huma.API, st store.Store) {
	huma.Register(api, huma.Operation{OperationID: "list-smpp-accounts", Method: http.MethodGet,
		Path: "/api/v1/smpp/server/accounts", Summary: "List SMPP server accounts", Tags: []string{"SMPP"},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []store.SMPPServerAccount }, error) {
		v, err := st.ListSMPPServerAccounts(ctx)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			v = []store.SMPPServerAccount{}
		}
		return &struct{ Body []store.SMPPServerAccount }{v}, nil
	})

	huma.Register(api, huma.Operation{OperationID: "get-smpp-account", Method: http.MethodGet,
		Path: "/api/v1/smpp/server/accounts/{id}", Summary: "Get SMPP server account", Tags: []string{"SMPP"},
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{ Body store.SMPPServerAccount }, error) {
		v, err := st.GetSMPPServerAccountByID(ctx, input.ID)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			return nil, notFound("account not found")
		}
		return &struct{ Body store.SMPPServerAccount }{*v}, nil
	})

	huma.Register(api, huma.Operation{OperationID: "create-smpp-account", Method: http.MethodPost,
		Path: "/api/v1/smpp/server/accounts", Summary: "Create SMPP server account",
		Tags: []string{"SMPP"}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *struct{ Body smppAccInput }) (*struct{}, error) {
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Body.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid password", err)
		}
		return nil, dbErr(st.CreateSMPPServerAccount(ctx, store.SMPPServerAccount{
			Name: input.Body.Name, SystemID: input.Body.SystemID, PasswordHash: string(hash),
			AllowedIP: input.Body.AllowedIP, BindType: input.Body.BindType,
			ThroughputLimit: input.Body.ThroughputLimit, Enabled: input.Body.Enabled,
		}))
	})

	huma.Register(api, huma.Operation{OperationID: "update-smpp-account", Method: http.MethodPut,
		Path: "/api/v1/smpp/server/accounts/{id}", Summary: "Update SMPP server account", Tags: []string{"SMPP"},
	}, func(ctx context.Context, input *struct {
		ID   string `path:"id"`
		Body smppAccInput
	}) (*struct{}, error) {
		acc, err := st.GetSMPPServerAccountByID(ctx, input.ID)
		if err != nil {
			return nil, dbErr(err)
		}
		if acc == nil {
			return nil, notFound("account not found")
		}
		if input.Body.Password != "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(input.Body.Password), bcrypt.DefaultCost)
			if err != nil {
				return nil, huma.Error400BadRequest("invalid password", err)
			}
			acc.PasswordHash = string(hash)
		}
		acc.Name = input.Body.Name
		acc.SystemID = input.Body.SystemID
		acc.AllowedIP = input.Body.AllowedIP
		acc.BindType = input.Body.BindType
		acc.ThroughputLimit = input.Body.ThroughputLimit
		acc.Enabled = input.Body.Enabled
		return nil, dbErr(st.UpdateSMPPServerAccount(ctx, *acc))
	})

	huma.Register(api, huma.Operation{OperationID: "delete-smpp-account", Method: http.MethodDelete,
		Path: "/api/v1/smpp/server/accounts/{id}", Summary: "Delete SMPP server account",
		Tags: []string{"SMPP"}, DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{}, error) {
		acc, err := st.GetSMPPServerAccountByID(ctx, input.ID)
		if err != nil {
			return nil, dbErr(err)
		}
		if acc == nil {
			return nil, notFound("account not found")
		}
		if rule, err := routingRulePeerReference(ctx, st, acc.SystemID); err != nil {
			return nil, dbErr(err)
		} else if rule != nil {
			return nil, huma.Error409Conflict(fmt.Sprintf("peer is still used by routing rule %q", routingRuleLabel(*rule)), nil)
		}
		return nil, dbErr(st.DeleteSMPPServerAccount(ctx, input.ID))
	})
}

// ── SMPP Clients ─────────────────────────────────────────────────────────────

type smppClientInput struct {
	Name              string `json:"name"`
	Host              string `json:"host"`
	Port              int    `json:"port"`
	Transport         string `json:"transport" enum:"tcp,tls"`
	VerifyServerCert  bool   `json:"verify_server_cert"`
	SystemID          string `json:"system_id"`
	Password          string `json:"password"`
	BindType          string `json:"bind_type" enum:"transmitter,receiver,transceiver"`
	ReconnectInterval string `json:"reconnect_interval" doc:"Go duration e.g. 10s"`
	ThroughputLimit   int    `json:"throughput_limit"`
	Enabled           bool   `json:"enabled"`
}

type smppClientUpdateInput struct {
	Name              string `json:"name"`
	Host              string `json:"host"`
	Port              int    `json:"port"`
	Transport         string `json:"transport" enum:"tcp,tls"`
	VerifyServerCert  bool   `json:"verify_server_cert"`
	SystemID          string `json:"system_id"`
	Password          string `json:"password,omitempty"`
	BindType          string `json:"bind_type" enum:"transmitter,receiver,transceiver"`
	ReconnectInterval string `json:"reconnect_interval" doc:"Go duration e.g. 10s"`
	ThroughputLimit   int    `json:"throughput_limit"`
	Enabled           bool   `json:"enabled"`
}

func registerSMPPClients(api huma.API, st store.Store) {
	huma.Register(api, huma.Operation{OperationID: "list-smpp-clients", Method: http.MethodGet,
		Path: "/api/v1/smpp/clients", Summary: "List SMPP outbound clients", Tags: []string{"SMPP"},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []store.SMPPClient }, error) {
		v, err := st.ListSMPPClients(ctx)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			v = []store.SMPPClient{}
		}
		return &struct{ Body []store.SMPPClient }{v}, nil
	})

	huma.Register(api, huma.Operation{OperationID: "get-smpp-client", Method: http.MethodGet,
		Path: "/api/v1/smpp/clients/{id}", Summary: "Get SMPP outbound client", Tags: []string{"SMPP"},
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{ Body store.SMPPClient }, error) {
		v, err := st.GetSMPPClient(ctx, input.ID)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			return nil, notFound("client not found")
		}
		return &struct{ Body store.SMPPClient }{*v}, nil
	})

	huma.Register(api, huma.Operation{OperationID: "create-smpp-client", Method: http.MethodPost,
		Path: "/api/v1/smpp/clients", Summary: "Create SMPP outbound client",
		Tags: []string{"SMPP"}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *struct{ Body smppClientInput }) (*struct{}, error) {
		c, err := smppClientFromInput(input.Body)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid input", err)
		}
		return nil, dbErr(st.CreateSMPPClient(ctx, c))
	})

	huma.Register(api, huma.Operation{OperationID: "update-smpp-client", Method: http.MethodPut,
		Path: "/api/v1/smpp/clients/{id}", Summary: "Update SMPP outbound client", Tags: []string{"SMPP"},
	}, func(ctx context.Context, input *struct {
		ID   string `path:"id"`
		Body smppClientUpdateInput
	}) (*struct{}, error) {
		existing, err := st.GetSMPPClient(ctx, input.ID)
		if err != nil {
			return nil, dbErr(err)
		}
		if existing == nil {
			return nil, notFound("client not found")
		}
		c, err := smppClientFromUpdateInput(input.Body)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid input", err)
		}
		c.ID = input.ID
		if input.Body.Password == "" {
			c.Password = existing.Password
		}
		return nil, dbErr(st.UpdateSMPPClient(ctx, c))
	})

	huma.Register(api, huma.Operation{OperationID: "delete-smpp-client", Method: http.MethodDelete,
		Path: "/api/v1/smpp/clients/{id}", Summary: "Delete SMPP outbound client",
		Tags: []string{"SMPP"}, DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{}, error) {
		client, err := st.GetSMPPClient(ctx, input.ID)
		if err != nil {
			return nil, dbErr(err)
		}
		if client == nil {
			return nil, notFound("client not found")
		}
		if rule, err := routingRulePeerReference(ctx, st, client.Name); err != nil {
			return nil, dbErr(err)
		} else if rule != nil {
			return nil, huma.Error409Conflict(fmt.Sprintf("peer is still used by routing rule %q", routingRuleLabel(*rule)), nil)
		}
		return nil, dbErr(st.DeleteSMPPClient(ctx, input.ID))
	})
}

func smppClientFromInput(b smppClientInput) (store.SMPPClient, error) {
	d, err := time.ParseDuration(b.ReconnectInterval)
	if err != nil {
		d = 10 * time.Second
	}
	transport := b.Transport
	if transport == "" {
		transport = "tcp"
	}
	return store.SMPPClient{
		Name: b.Name, Host: b.Host, Port: b.Port, Transport: transport, VerifyServerCert: b.VerifyServerCert,
		SystemID: b.SystemID, Password: b.Password, BindType: b.BindType,
		ReconnectInterval: d, ThroughputLimit: b.ThroughputLimit, Enabled: b.Enabled,
	}, nil
}

func smppClientFromUpdateInput(b smppClientUpdateInput) (store.SMPPClient, error) {
	d, err := time.ParseDuration(b.ReconnectInterval)
	if err != nil {
		d = 10 * time.Second
	}
	transport := b.Transport
	if transport == "" {
		transport = "tcp"
	}
	return store.SMPPClient{
		Name: b.Name, Host: b.Host, Port: b.Port, Transport: transport, VerifyServerCert: b.VerifyServerCert,
		SystemID: b.SystemID, Password: b.Password, BindType: b.BindType,
		ReconnectInterval: d, ThroughputLimit: b.ThroughputLimit, Enabled: b.Enabled,
	}, nil
}

// ── SIP Peers ─────────────────────────────────────────────────────────────────

type sipPeerInput struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	Port      int    `json:"port"`
	Transport string `json:"transport" enum:"udp,tcp,tls"`
	Domain    string `json:"domain"`
	AuthUser  string `json:"auth_user,omitempty"`
	AuthPass  string `json:"auth_pass,omitempty"`
	Enabled   bool   `json:"enabled"`
}

func registerSIPPeers(api huma.API, st store.Store) {
	huma.Register(api, huma.Operation{OperationID: "list-sip-peers", Method: http.MethodGet,
		Path: "/api/v1/sip/peers", Summary: "List SIP SIMPLE peers", Tags: []string{"SIP"},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []store.SIPPeer }, error) {
		v, err := st.ListAllSIPPeers(ctx)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			v = []store.SIPPeer{}
		}
		return &struct{ Body []store.SIPPeer }{v}, nil
	})

	huma.Register(api, huma.Operation{OperationID: "get-sip-peer", Method: http.MethodGet,
		Path: "/api/v1/sip/peers/{id}", Summary: "Get SIP SIMPLE peer", Tags: []string{"SIP"},
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{ Body store.SIPPeer }, error) {
		v, err := st.GetSIPPeerByID(ctx, input.ID)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			return nil, notFound("peer not found")
		}
		return &struct{ Body store.SIPPeer }{*v}, nil
	})

	huma.Register(api, huma.Operation{OperationID: "create-sip-peer", Method: http.MethodPost,
		Path: "/api/v1/sip/peers", Summary: "Create SIP SIMPLE peer",
		Tags: []string{"SIP"}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *struct{ Body sipPeerInput }) (*struct{}, error) {
		return nil, dbErr(st.CreateSIPPeer(ctx, store.SIPPeer{
			Name: input.Body.Name, Address: input.Body.Address, Port: input.Body.Port,
			Transport: input.Body.Transport, Domain: input.Body.Domain,
			AuthUser: input.Body.AuthUser, AuthPass: input.Body.AuthPass, Enabled: input.Body.Enabled,
		}))
	})

	huma.Register(api, huma.Operation{OperationID: "update-sip-peer", Method: http.MethodPut,
		Path: "/api/v1/sip/peers/{id}", Summary: "Update SIP SIMPLE peer", Tags: []string{"SIP"},
	}, func(ctx context.Context, input *struct {
		ID   string `path:"id"`
		Body sipPeerInput
	}) (*struct{}, error) {
		return nil, dbErr(st.UpdateSIPPeer(ctx, store.SIPPeer{
			ID: input.ID, Name: input.Body.Name, Address: input.Body.Address, Port: input.Body.Port,
			Transport: input.Body.Transport, Domain: input.Body.Domain,
			AuthUser: input.Body.AuthUser, AuthPass: input.Body.AuthPass, Enabled: input.Body.Enabled,
		}))
	})

	huma.Register(api, huma.Operation{OperationID: "delete-sip-peer", Method: http.MethodDelete,
		Path: "/api/v1/sip/peers/{id}", Summary: "Delete SIP SIMPLE peer",
		Tags: []string{"SIP"}, DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{}, error) {
		peer, err := st.GetSIPPeerByID(ctx, input.ID)
		if err != nil {
			return nil, dbErr(err)
		}
		if peer == nil {
			return nil, notFound("peer not found")
		}
		if rule, err := routingRulePeerReference(ctx, st, peer.Name); err != nil {
			return nil, dbErr(err)
		} else if rule != nil {
			return nil, huma.Error409Conflict(fmt.Sprintf("peer is still used by routing rule %q", routingRuleLabel(*rule)), nil)
		}
		return nil, dbErr(st.DeleteSIPPeer(ctx, input.ID))
	})
}

// ── Diameter Peers ────────────────────────────────────────────────────────────

type diamPeerInput struct {
	Name         string   `json:"name"`
	Host         string   `json:"host"`
	Realm        string   `json:"realm"`
	Port         int      `json:"port"`
	Transport    string   `json:"transport" enum:"tcp,sctp"`
	Applications []string `json:"applications" doc:"One or more of: sgd, sh, s6c"`
	Enabled      bool     `json:"enabled"`
}

func registerDiameterPeers(api huma.API, st store.Store) {
	huma.Register(api, huma.Operation{OperationID: "list-diameter-peers", Method: http.MethodGet,
		Path: "/api/v1/diameter/peers", Summary: "List Diameter peers", Tags: []string{"Diameter"},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []store.DiameterPeer }, error) {
		v, err := st.ListAllDiameterPeers(ctx)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			v = []store.DiameterPeer{}
		}
		return &struct{ Body []store.DiameterPeer }{v}, nil
	})

	huma.Register(api, huma.Operation{OperationID: "get-diameter-peer", Method: http.MethodGet,
		Path: "/api/v1/diameter/peers/{id}", Summary: "Get Diameter peer", Tags: []string{"Diameter"},
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{ Body store.DiameterPeer }, error) {
		v, err := st.GetDiameterPeerByID(ctx, input.ID)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			return nil, notFound("peer not found")
		}
		return &struct{ Body store.DiameterPeer }{*v}, nil
	})

	huma.Register(api, huma.Operation{OperationID: "create-diameter-peer", Method: http.MethodPost,
		Path: "/api/v1/diameter/peers", Summary: "Create Diameter peer",
		Tags: []string{"Diameter"}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *struct{ Body diamPeerInput }) (*struct{}, error) {
		return nil, dbErr(st.CreateDiameterPeer(ctx, store.DiameterPeer{
			Name: input.Body.Name, Host: input.Body.Host, Realm: input.Body.Realm,
			Port: input.Body.Port, Transport: input.Body.Transport,
			Applications: input.Body.Applications, Enabled: input.Body.Enabled,
		}))
	})

	huma.Register(api, huma.Operation{OperationID: "update-diameter-peer", Method: http.MethodPut,
		Path: "/api/v1/diameter/peers/{id}", Summary: "Update Diameter peer", Tags: []string{"Diameter"},
	}, func(ctx context.Context, input *struct {
		ID   string `path:"id"`
		Body diamPeerInput
	}) (*struct{}, error) {
		return nil, dbErr(st.UpdateDiameterPeer(ctx, store.DiameterPeer{
			ID: input.ID, Name: input.Body.Name, Host: input.Body.Host, Realm: input.Body.Realm,
			Port: input.Body.Port, Transport: input.Body.Transport,
			Applications: input.Body.Applications, Enabled: input.Body.Enabled,
		}))
	})

	huma.Register(api, huma.Operation{OperationID: "delete-diameter-peer", Method: http.MethodDelete,
		Path: "/api/v1/diameter/peers/{id}", Summary: "Delete Diameter peer",
		Tags: []string{"Diameter"}, DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{}, error) {
		peer, err := st.GetDiameterPeerByID(ctx, input.ID)
		if err != nil {
			return nil, dbErr(err)
		}
		if peer == nil {
			return nil, notFound("peer not found")
		}
		if rule, err := routingRulePeerReference(ctx, st, peer.Name); err != nil {
			return nil, dbErr(err)
		} else if rule != nil {
			return nil, huma.Error409Conflict(fmt.Sprintf("peer is still used by routing rule %q", routingRuleLabel(*rule)), nil)
		}
		return nil, dbErr(st.DeleteDiameterPeer(ctx, input.ID))
	})
}

// ── Routing Rules ─────────────────────────────────────────────────────────────

type routingRuleInput struct {
	Name           string `json:"name"`
	Priority       int    `json:"priority"`
	MatchSrcIface  string `json:"match_src_iface,omitempty"`
	MatchSrcPeer   string `json:"match_src_peer,omitempty"`
	MatchDstPrefix string `json:"match_dst_prefix,omitempty"`
	MatchMSISDNMin string `json:"match_msisdn_min,omitempty"`
	MatchMSISDNMax string `json:"match_msisdn_max,omitempty"`
	EgressIface    string `json:"egress_iface" enum:"sip3gpp,sipsimple,smpp,sgd"`
	EgressPeer     string `json:"egress_peer,omitempty"`
	SFPolicyID     string `json:"sf_policy_id,omitempty"`
	Enabled        bool   `json:"enabled"`
}

func registerRoutingRules(api huma.API, st store.Store) {
	huma.Register(api, huma.Operation{OperationID: "list-routing-rules", Method: http.MethodGet,
		Path: "/api/v1/routing/rules", Summary: "List routing rules", Tags: []string{"Routing"},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []store.RoutingRule }, error) {
		v, err := st.ListAllRoutingRules(ctx)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			v = []store.RoutingRule{}
		}
		return &struct{ Body []store.RoutingRule }{v}, nil
	})

	huma.Register(api, huma.Operation{OperationID: "get-routing-rule", Method: http.MethodGet,
		Path: "/api/v1/routing/rules/{id}", Summary: "Get routing rule", Tags: []string{"Routing"},
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{ Body store.RoutingRule }, error) {
		v, err := st.GetRoutingRule(ctx, input.ID)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			return nil, notFound("rule not found")
		}
		return &struct{ Body store.RoutingRule }{*v}, nil
	})

	huma.Register(api, huma.Operation{OperationID: "create-routing-rule", Method: http.MethodPost,
		Path: "/api/v1/routing/rules", Summary: "Create routing rule",
		Tags: []string{"Routing"}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *struct{ Body routingRuleInput }) (*struct{}, error) {
		return nil, dbErr(st.CreateRoutingRule(ctx, routingRuleFromInput(input.Body)))
	})

	huma.Register(api, huma.Operation{OperationID: "update-routing-rule", Method: http.MethodPut,
		Path: "/api/v1/routing/rules/{id}", Summary: "Update routing rule", Tags: []string{"Routing"},
	}, func(ctx context.Context, input *struct {
		ID   string `path:"id"`
		Body routingRuleInput
	}) (*struct{}, error) {
		r := routingRuleFromInput(input.Body)
		r.ID = input.ID
		return nil, dbErr(st.UpdateRoutingRule(ctx, r))
	})

	huma.Register(api, huma.Operation{OperationID: "delete-routing-rule", Method: http.MethodDelete,
		Path: "/api/v1/routing/rules/{id}", Summary: "Delete routing rule",
		Tags: []string{"Routing"}, DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{}, error) {
		return nil, dbErr(st.DeleteRoutingRule(ctx, input.ID))
	})
}

func routingRuleFromInput(b routingRuleInput) store.RoutingRule {
	return store.RoutingRule{
		Name: b.Name, Priority: b.Priority,
		MatchSrcIface: b.MatchSrcIface, MatchSrcPeer: b.MatchSrcPeer,
		MatchDstPrefix: b.MatchDstPrefix, MatchMSISDNMin: b.MatchMSISDNMin, MatchMSISDNMax: b.MatchMSISDNMax,
		EgressIface: b.EgressIface, EgressPeer: b.EgressPeer, SFPolicyID: b.SFPolicyID,
		Enabled: b.Enabled,
	}
}

// ── SF Policies ───────────────────────────────────────────────────────────────

type sfPolicyInput struct {
	Name          string `json:"name"`
	MaxRetries    int    `json:"max_retries"`
	RetrySchedule []int  `json:"retry_schedule" doc:"Seconds between retry attempts"`
	MaxTTL        string `json:"max_ttl" doc:"Go duration e.g. 48h"`
	VPOverride    string `json:"vp_override,omitempty" doc:"Go duration; empty = honour TP-VP"`
}

func registerSFPolicies(api huma.API, st store.Store) {
	huma.Register(api, huma.Operation{OperationID: "list-sf-policies", Method: http.MethodGet,
		Path: "/api/v1/routing/policies", Summary: "List SF policies", Tags: []string{"Routing"},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []store.SFPolicy }, error) {
		v, err := st.ListSFPolicies(ctx)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			v = []store.SFPolicy{}
		}
		return &struct{ Body []store.SFPolicy }{v}, nil
	})

	huma.Register(api, huma.Operation{OperationID: "get-sf-policy", Method: http.MethodGet,
		Path: "/api/v1/routing/policies/{id}", Summary: "Get SF policy", Tags: []string{"Routing"},
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{ Body store.SFPolicy }, error) {
		v, err := st.GetSFPolicy(ctx, input.ID)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			return nil, notFound("policy not found")
		}
		return &struct{ Body store.SFPolicy }{*v}, nil
	})

	huma.Register(api, huma.Operation{OperationID: "create-sf-policy", Method: http.MethodPost,
		Path: "/api/v1/routing/policies", Summary: "Create SF policy",
		Tags: []string{"Routing"}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *struct{ Body sfPolicyInput }) (*struct{}, error) {
		p, err := sfPolicyFromInput(input.Body)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid input", err)
		}
		return nil, dbErr(st.CreateSFPolicy(ctx, p))
	})

	huma.Register(api, huma.Operation{OperationID: "update-sf-policy", Method: http.MethodPut,
		Path: "/api/v1/routing/policies/{id}", Summary: "Update SF policy", Tags: []string{"Routing"},
	}, func(ctx context.Context, input *struct {
		ID   string `path:"id"`
		Body sfPolicyInput
	}) (*struct{}, error) {
		p, err := sfPolicyFromInput(input.Body)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid input", err)
		}
		p.ID = input.ID
		return nil, dbErr(st.UpdateSFPolicy(ctx, p))
	})

	huma.Register(api, huma.Operation{OperationID: "delete-sf-policy", Method: http.MethodDelete,
		Path: "/api/v1/routing/policies/{id}", Summary: "Delete SF policy",
		Tags: []string{"Routing"}, DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{}, error) {
		if rule, err := routingRulePolicyReference(ctx, st, input.ID); err != nil {
			return nil, dbErr(err)
		} else if rule != nil {
			return nil, huma.Error409Conflict(fmt.Sprintf("policy is still used by routing rule %q", routingRuleLabel(*rule)), nil)
		}
		return nil, dbErr(st.DeleteSFPolicy(ctx, input.ID))
	})
}

func sfPolicyFromInput(b sfPolicyInput) (store.SFPolicy, error) {
	maxTTL, err := time.ParseDuration(b.MaxTTL)
	if err != nil {
		return store.SFPolicy{}, err
	}
	p := store.SFPolicy{
		Name: b.Name, MaxRetries: b.MaxRetries,
		RetrySchedule: b.RetrySchedule, MaxTTL: maxTTL,
	}
	if b.VPOverride != "" {
		d, err := time.ParseDuration(b.VPOverride)
		if err != nil {
			return store.SFPolicy{}, err
		}
		p.VPOverride = &d
	}
	return p, nil
}

// ── Subscribers ───────────────────────────────────────────────────────────────

type subscriberInput struct {
	MSISDN        string `json:"msisdn"`
	IMSI          string `json:"imsi,omitempty"`
	IMSRegistered bool   `json:"ims_registered"`
	LTEAttached   bool   `json:"lte_attached"`
	MMENumber     string `json:"mme_number_for_mt_sms,omitempty"`
	MMEHost       string `json:"mme_host,omitempty"`
	MWDSet        bool   `json:"mwd_set"`
}

func registerSubscribers(api huma.API, st store.Store) {
	huma.Register(api, huma.Operation{OperationID: "list-subscribers", Method: http.MethodGet,
		Path: "/api/v1/subscribers", Summary: "List subscribers", Tags: []string{"Subscribers"},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []store.Subscriber }, error) {
		v, err := st.ListSubscribers(ctx)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			v = []store.Subscriber{}
		}
		return &struct{ Body []store.Subscriber }{v}, nil
	})

	huma.Register(api, huma.Operation{OperationID: "get-subscriber", Method: http.MethodGet,
		Path: "/api/v1/subscribers/{id}", Summary: "Get subscriber", Tags: []string{"Subscribers"},
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{ Body store.Subscriber }, error) {
		v, err := st.GetSubscriberByID(ctx, input.ID)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			return nil, notFound("subscriber not found")
		}
		return &struct{ Body store.Subscriber }{*v}, nil
	})

	huma.Register(api, huma.Operation{OperationID: "upsert-subscriber", Method: http.MethodPost,
		Path: "/api/v1/subscribers", Summary: "Create or update subscriber",
		Tags: []string{"Subscribers"}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *struct{ Body subscriberInput }) (*struct{}, error) {
		return nil, dbErr(st.UpsertSubscriber(ctx, store.Subscriber{
			MSISDN: input.Body.MSISDN, IMSI: input.Body.IMSI,
			IMSRegistered: input.Body.IMSRegistered, LTEAttached: input.Body.LTEAttached,
			MMENumber: input.Body.MMENumber, MMEHost: input.Body.MMEHost, MWDSet: input.Body.MWDSet,
		}))
	})

	huma.Register(api, huma.Operation{OperationID: "update-subscriber", Method: http.MethodPut,
		Path: "/api/v1/subscribers/{id}", Summary: "Update subscriber", Tags: []string{"Subscribers"},
	}, func(ctx context.Context, input *struct {
		ID   string `path:"id"`
		Body subscriberInput
	}) (*struct{}, error) {
		return nil, dbErr(st.UpsertSubscriber(ctx, store.Subscriber{
			ID: input.ID, MSISDN: input.Body.MSISDN, IMSI: input.Body.IMSI,
			IMSRegistered: input.Body.IMSRegistered, LTEAttached: input.Body.LTEAttached,
			MMENumber: input.Body.MMENumber, MMEHost: input.Body.MMEHost, MWDSet: input.Body.MWDSet,
		}))
	})

	huma.Register(api, huma.Operation{OperationID: "delete-subscriber", Method: http.MethodDelete,
		Path: "/api/v1/subscribers/{id}", Summary: "Delete subscriber",
		Tags: []string{"Subscribers"}, DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{}, error) {
		return nil, dbErr(st.DeleteSubscriber(ctx, input.ID))
	})
}

// ── Messages & Delivery Reports (read-only) ───────────────────────────────────

func registerMessages(api huma.API, st store.Store) {
	huma.Register(api, huma.Operation{OperationID: "list-messages", Method: http.MethodGet,
		Path: "/api/v1/messages", Summary: "List recent messages", Tags: []string{"Messages"},
	}, func(ctx context.Context, input *struct {
		Limit int `query:"limit" default:"100"`
	}) (*struct{ Body []store.Message }, error) {
		v, err := st.ListMessages(ctx, input.Limit)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			v = []store.Message{}
		}
		return &struct{ Body []store.Message }{v}, nil
	})

	huma.Register(api, huma.Operation{OperationID: "get-message", Method: http.MethodGet,
		Path: "/api/v1/messages/{id}", Summary: "Get message", Tags: []string{"Messages"},
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{ Body store.Message }, error) {
		v, err := st.GetMessage(ctx, input.ID)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			return nil, notFound("message not found")
		}
		return &struct{ Body store.Message }{*v}, nil
	})

	huma.Register(api, huma.Operation{OperationID: "list-queue-messages", Method: http.MethodGet,
		Path: "/api/v1/messages/queue", Summary: "List queued or dispatched messages", Tags: []string{"Messages"},
	}, func(ctx context.Context, input *struct {
		Limit      int    `query:"limit" default:"100"`
		SrcMSISDN  string `query:"src_msisdn"`
		DstMSISDN  string `query:"dst_msisdn"`
		OriginPeer string `query:"origin_peer"`
	}) (*struct{ Body []store.Message }, error) {
		v, err := st.ListFilteredMessages(ctx, store.MessageFilter{
			Statuses:   []string{store.MessageStatusQueued, store.MessageStatusDispatched},
			SrcMSISDN:  input.SrcMSISDN,
			DstMSISDN:  input.DstMSISDN,
			OriginPeer: input.OriginPeer,
			Limit:      input.Limit,
		})
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			v = []store.Message{}
		}
		return &struct{ Body []store.Message }{v}, nil
	})

	huma.Register(api, huma.Operation{OperationID: "delete-queue-message", Method: http.MethodDelete,
		Path: "/api/v1/messages/queue/{id}", Summary: "Delete a queued or dispatched message",
		Tags: []string{"Messages"}, DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{}, error) {
		msg, err := st.GetMessage(ctx, input.ID)
		if err != nil {
			return nil, dbErr(err)
		}
		if msg == nil {
			return nil, notFound("message not found")
		}
		if msg.Status != store.MessageStatusQueued && msg.Status != store.MessageStatusDispatched {
			return nil, huma.Error409Conflict("only queued or dispatched messages can be deleted from the queue view", nil)
		}
		return nil, dbErr(st.DeleteMessage(ctx, input.ID))
	})

	huma.Register(api, huma.Operation{OperationID: "list-delivery-reports", Method: http.MethodGet,
		Path: "/api/v1/delivery-reports", Summary: "List recent delivery reports", Tags: []string{"Messages"},
	}, func(ctx context.Context, input *struct {
		Limit int `query:"limit" default:"100"`
	}) (*struct{ Body []store.DeliveryReport }, error) {
		v, err := st.ListDeliveryReports(ctx, input.Limit)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			v = []store.DeliveryReport{}
		}
		return &struct{ Body []store.DeliveryReport }{v}, nil
	})

	huma.Register(api, huma.Operation{OperationID: "get-delivery-report", Method: http.MethodGet,
		Path: "/api/v1/delivery-reports/{id}", Summary: "Get delivery report", Tags: []string{"Messages"},
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{ Body store.DeliveryReport }, error) {
		v, err := st.GetDeliveryReport(ctx, input.ID)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			return nil, notFound("delivery report not found")
		}
		return &struct{ Body store.DeliveryReport }{*v}, nil
	})
}

// ── SGd MME Mappings ──────────────────────────────────────────────────────────

type sgdMMEMappingInput struct {
	S6CResult string `json:"s6c_result" doc:"MME hostname as returned by S6c (S6a FQDN)"`
	SGDHost   string `json:"sgd_host" doc:"MME SGd FQDN to use for Diameter SGd delivery"`
	Enabled   bool   `json:"enabled"`
}

func registerSGDMMEMappings(api huma.API, st store.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-sgd-mme-mappings", Method: http.MethodGet,
		Path: "/api/v1/sgd/mme-mappings", Summary: "List S6c to SGd MME mappings", Tags: []string{"Diameter"},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []store.SGDMMEMapping }, error) {
		v, err := st.ListSGDMMEMappings(ctx)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			v = []store.SGDMMEMapping{}
		}
		return &struct{ Body []store.SGDMMEMapping }{v}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-sgd-mme-mapping", Method: http.MethodGet,
		Path: "/api/v1/sgd/mme-mappings/{id}", Summary: "Get S6c to SGd MME mapping", Tags: []string{"Diameter"},
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{ Body store.SGDMMEMapping }, error) {
		v, err := st.GetSGDMMEMappingByID(ctx, input.ID)
		if err != nil {
			return nil, dbErr(err)
		}
		if v == nil {
			return nil, notFound("mapping not found")
		}
		return &struct{ Body store.SGDMMEMapping }{*v}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-sgd-mme-mapping", Method: http.MethodPost,
		Path: "/api/v1/sgd/mme-mappings", Summary: "Create S6c to SGd MME mapping",
		Tags: []string{"Diameter"}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *struct{ Body sgdMMEMappingInput }) (*struct{}, error) {
		return nil, dbErr(st.CreateSGDMMEMapping(ctx, store.SGDMMEMapping{
			S6CResult: input.Body.S6CResult,
			SGDHost:   input.Body.SGDHost,
			Enabled:   input.Body.Enabled,
		}))
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-sgd-mme-mapping", Method: http.MethodPut,
		Path: "/api/v1/sgd/mme-mappings/{id}", Summary: "Update S6c to SGd MME mapping", Tags: []string{"Diameter"},
	}, func(ctx context.Context, input *struct {
		ID   string `path:"id"`
		Body sgdMMEMappingInput
	}) (*struct{}, error) {
		existing, err := st.GetSGDMMEMappingByID(ctx, input.ID)
		if err != nil {
			return nil, dbErr(err)
		}
		if existing == nil {
			return nil, notFound("mapping not found")
		}
		return nil, dbErr(st.UpdateSGDMMEMapping(ctx, store.SGDMMEMapping{
			ID:        input.ID,
			S6CResult: input.Body.S6CResult,
			SGDHost:   input.Body.SGDHost,
			Enabled:   input.Body.Enabled,
		}))
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-sgd-mme-mapping", Method: http.MethodDelete,
		Path: "/api/v1/sgd/mme-mappings/{id}", Summary: "Delete S6c to SGd MME mapping",
		Tags: []string{"Diameter"}, DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *struct {
		ID string `path:"id"`
	}) (*struct{}, error) {
		existing, err := st.GetSGDMMEMappingByID(ctx, input.ID)
		if err != nil {
			return nil, dbErr(err)
		}
		if existing == nil {
			return nil, notFound("mapping not found")
		}
		return nil, dbErr(st.DeleteSGDMMEMapping(ctx, input.ID))
	})
}
