package forwarder

import (
	"context"
	"testing"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/codec"
	"github.com/svinson1121/vectorcore-smsc/internal/diameter/s6c"
	"github.com/svinson1121/vectorcore-smsc/internal/registry"
	"github.com/svinson1121/vectorcore-smsc/internal/routing"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

type forwarderTestStore struct {
	subscriber    *store.Subscriber
	sipPeers      map[string]store.SIPPeer
	saved         *store.Message
	routeIface    string
	routePeer     string
	routingCalls  int
	statusUpdates []string
	retryCount    int
	routeCursor   int
	nextRetryAt   *time.Time
}

func (s *forwarderTestStore) GetIMSRegistration(context.Context, string) (*store.IMSRegistration, error) {
	return nil, nil
}
func (s *forwarderTestStore) UpsertIMSRegistration(context.Context, store.IMSRegistration) error {
	return nil
}
func (s *forwarderTestStore) DeleteIMSRegistration(context.Context, string) error { return nil }
func (s *forwarderTestStore) ListIMSRegistrations(context.Context) ([]store.IMSRegistration, error) {
	return nil, nil
}
func (s *forwarderTestStore) GetSubscriber(context.Context, string) (*store.Subscriber, error) {
	return s.subscriber, nil
}
func (s *forwarderTestStore) UpsertSubscriber(_ context.Context, sub store.Subscriber) error {
	cp := sub
	s.subscriber = &cp
	return nil
}
func (s *forwarderTestStore) GetSMPPServerAccount(context.Context, string) (*store.SMPPServerAccount, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) ListSMPPServerAccounts(context.Context) ([]store.SMPPServerAccount, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) ListSMPPClients(context.Context) ([]store.SMPPClient, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) ListSIPPeers(context.Context) ([]store.SIPPeer, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) GetSIPPeer(_ context.Context, name string) (*store.SIPPeer, error) {
	if peer, ok := s.sipPeers[name]; ok {
		cp := peer
		return &cp, nil
	}
	return nil, nil
}
func (s *forwarderTestStore) ListDiameterPeers(context.Context) ([]store.DiameterPeer, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) GetDiameterPeer(context.Context, string) (*store.DiameterPeer, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) ListRoutingRules(context.Context) ([]store.RoutingRule, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) GetSFPolicy(context.Context, string) (*store.SFPolicy, error) {
	return nil, nil
}
func (s *forwarderTestStore) SaveMessage(_ context.Context, msg store.Message) error {
	cp := msg
	s.saved = &cp
	return nil
}
func (s *forwarderTestStore) UpdateMessageRouting(_ context.Context, _ string, egressIface, egressPeer string) error {
	s.routeIface = egressIface
	s.routePeer = egressPeer
	s.routingCalls++
	if s.saved != nil {
		s.saved.EgressIface = egressIface
		s.saved.EgressPeer = egressPeer
	}
	return nil
}
func (s *forwarderTestStore) UpdateMessageStatus(_ context.Context, _ string, status string) error {
	s.statusUpdates = append(s.statusUpdates, status)
	return nil
}
func (s *forwarderTestStore) UpdateMessageRetry(_ context.Context, _ string, retryCount int, nextRetryAt time.Time, routeCursor int) error {
	s.retryCount = retryCount
	s.routeCursor = routeCursor
	s.nextRetryAt = &nextRetryAt
	return nil
}
func (s *forwarderTestStore) ListRetryableMessages(context.Context) ([]store.Message, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) ListExpiredMessages(context.Context) ([]store.Message, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) GetMessage(_ context.Context, id string) (*store.Message, error) {
	if s.saved != nil && s.saved.ID == id {
		cp := *s.saved
		return &cp, nil
	}
	return nil, nil
}
func (s *forwarderTestStore) DeleteMessage(context.Context, string) error { panic("unexpected call") }
func (s *forwarderTestStore) SaveDeliveryReport(context.Context, store.DeliveryReport) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) GetSMPPServerAccountByID(context.Context, string) (*store.SMPPServerAccount, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) CreateSMPPServerAccount(context.Context, store.SMPPServerAccount) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) UpdateSMPPServerAccount(context.Context, store.SMPPServerAccount) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) DeleteSMPPServerAccount(context.Context, string) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) GetSMPPClient(context.Context, string) (*store.SMPPClient, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) CreateSMPPClient(context.Context, store.SMPPClient) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) UpdateSMPPClient(context.Context, store.SMPPClient) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) DeleteSMPPClient(context.Context, string) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) ListAllSIPPeers(context.Context) ([]store.SIPPeer, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) GetSIPPeerByID(context.Context, string) (*store.SIPPeer, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) CreateSIPPeer(context.Context, store.SIPPeer) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) UpdateSIPPeer(context.Context, store.SIPPeer) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) DeleteSIPPeer(context.Context, string) error { panic("unexpected call") }
func (s *forwarderTestStore) ListAllDiameterPeers(context.Context) ([]store.DiameterPeer, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) GetDiameterPeerByID(context.Context, string) (*store.DiameterPeer, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) CreateDiameterPeer(context.Context, store.DiameterPeer) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) UpdateDiameterPeer(context.Context, store.DiameterPeer) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) DeleteDiameterPeer(context.Context, string) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) ListAllRoutingRules(context.Context) ([]store.RoutingRule, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) GetRoutingRule(context.Context, string) (*store.RoutingRule, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) CreateRoutingRule(context.Context, store.RoutingRule) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) UpdateRoutingRule(context.Context, store.RoutingRule) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) DeleteRoutingRule(context.Context, string) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) ListSFPolicies(context.Context) ([]store.SFPolicy, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) CreateSFPolicy(context.Context, store.SFPolicy) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) UpdateSFPolicy(context.Context, store.SFPolicy) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) DeleteSFPolicy(context.Context, string) error { panic("unexpected call") }
func (s *forwarderTestStore) ListSubscribers(context.Context) ([]store.Subscriber, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) GetSubscriberByID(context.Context, string) (*store.Subscriber, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) DeleteSubscriber(context.Context, string) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) ListMessages(context.Context, int) ([]store.Message, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) ListFilteredMessages(context.Context, store.MessageFilter) ([]store.Message, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) CountMessagesByStatus(context.Context) (map[string]int64, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) ListDeliveryReports(context.Context, int) ([]store.DeliveryReport, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) GetDeliveryReport(context.Context, string) (*store.DeliveryReport, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) Subscribe(context.Context, string, chan<- store.ChangeEvent) error {
	return nil
}
func (s *forwarderTestStore) Close() error { return nil }
func (s *forwarderTestStore) ListSGDMMEMappings(context.Context) ([]store.SGDMMEMapping, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) GetSGDMMEMappingByID(context.Context, string) (*store.SGDMMEMapping, error) {
	panic("unexpected call")
}
func (s *forwarderTestStore) CreateSGDMMEMapping(context.Context, store.SGDMMEMapping) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) UpdateSGDMMEMapping(context.Context, store.SGDMMEMapping) error {
	panic("unexpected call")
}
func (s *forwarderTestStore) DeleteSGDMMEMapping(context.Context, string) error {
	panic("unexpected call")
}

type fakeISCSender struct {
	calls int
	err   error
	msgs  []*codec.Message
}

func (f *fakeISCSender) Send(_ context.Context, msg *codec.Message, _ *registry.Registration) error {
	f.calls++
	if msg != nil {
		cp := *msg
		if msg.Binary != nil {
			cp.Binary = append([]byte(nil), msg.Binary...)
		}
		if msg.UDH != nil {
			cp.UDH = &codec.UDH{Raw: append([]byte(nil), msg.UDH.Raw...)}
		}
		f.msgs = append(f.msgs, &cp)
	}
	return f.err
}

type fakeSimpleSender struct {
	calls int
	err   error
	errs  []error
}

func (f *fakeSimpleSender) Send(context.Context, *codec.Message, store.SIPPeer) error {
	f.calls++
	if idx := f.calls - 1; idx >= 0 && idx < len(f.errs) && f.errs[idx] != nil {
		return f.errs[idx]
	}
	return f.err
}

type fakeS6cClient struct {
	info  *s6c.RoutingInfo
	err   error
	calls int
}

func (f *fakeS6cClient) LookupRouting(msisdn string) (*s6c.RoutingInfo, error) {
	f.calls++
	return f.info, f.err
}

type fakeSGdSender struct {
	lastSCAddr string
	lastMME    string
}

func (f *fakeSGdSender) SendOFR(_ context.Context, _ *codec.Message, mmeHost, scAddr string) error {
	f.lastMME = mmeHost
	f.lastSCAddr = scAddr
	return nil
}
func (f *fakeSGdSender) HasPeerForMME(string) bool { return true }

func newTestRegistry(t *testing.T, st store.Store) *registry.Registry {
	t.Helper()
	reg, err := registry.New(context.Background(), st)
	if err != nil {
		t.Fatalf("registry.New() error = %v", err)
	}
	return reg
}

func TestSelectRouteUsesS6cOnlyForSGdCandidates(t *testing.T) {
	st := &forwarderTestStore{
		sipPeers: map[string]store.SIPPeer{
			"simple1": {Name: "simple1"},
		},
	}
	reg := newTestRegistry(t, st)
	s6cClient := &fakeS6cClient{}
	reg.SetS6cClient(s6cClient)

	engine := routing.NewEngine()
	engine.Reload([]store.RoutingRule{
		{Name: "simple", Priority: 10, MatchSrcIface: "smpp", MatchDstPrefix: "334", EgressIface: "sipsimple", EgressPeer: "simple1"},
	})

	f := New(Config{
		Registry:     reg,
		Engine:       engine,
		Store:        st,
		SimpleSender: &fakeSimpleSender{},
		SGdSender:    &fakeSGdSender{},
	})

	route, err := f.selectRoute(context.Background(), &codec.Message{
		IngressInterface: codec.InterfaceSMPP,
		Destination:      codec.Address{MSISDN: "3342012860"},
	}, 0)
	if err != nil {
		t.Fatalf("selectRoute() error = %v", err)
	}
	if got, want := route.egressIface, "sipsimple"; got != want {
		t.Fatalf("egressIface = %q, want %q", got, want)
	}
	if got := s6cClient.calls; got != 0 {
		t.Fatalf("S6c calls = %d, want 0", got)
	}
}

func TestPersistMessageStoresBinaryMetadataForSMPPIngress(t *testing.T) {
	st := &forwarderTestStore{}
	f := &Forwarder{st: st}
	now := time.Now().UTC()

	msg := &codec.Message{
		ID:               "msg-1",
		IngressInterface: codec.InterfaceSMPP,
		Encoding:         codec.EncodingBinary,
		DCS:              0x04,
		Source:           codec.Address{MSISDN: "3342012832"},
		Destination:      codec.Address{MSISDN: "3342012860"},
		UDH:              &codec.UDH{Raw: []byte{0x0b, 0x05, 0x04, 0x0b, 0x84, 0x23, 0xf0, 0x00, 0x03, 0x42, 0x02, 0x01}},
		Binary:           []byte{0x01, 0x02, 0x03},
	}

	f.persistMessage(context.Background(), msg, "sip3gpp", "peer1", store.MessageStatusQueued, now)

	if st.saved == nil {
		t.Fatal("message was not saved")
	}
	if got, want := st.saved.Encoding, int(codec.EncodingBinary); got != want {
		t.Fatalf("saved encoding = %d, want %d", got, want)
	}
	if got, want := st.saved.DCS, 4; got != want {
		t.Fatalf("saved dcs = %d, want %d", got, want)
	}
	if string(st.saved.Payload) != string(msg.Binary) {
		t.Fatalf("saved payload = %x, want %x", st.saved.Payload, msg.Binary)
	}
	if string(st.saved.UDH) != string(msg.UDH.Raw) {
		t.Fatalf("saved UDH = %x, want %x", st.saved.UDH, msg.UDH.Raw)
	}
}

func TestPersistMessageStoresTextPayloadForRetry(t *testing.T) {
	st := &forwarderTestStore{}
	f := &Forwarder{st: st}
	now := time.Now().UTC()

	msg := &codec.Message{
		ID:               "msg-text-1",
		IngressInterface: codec.InterfaceSMPP,
		Encoding:         codec.EncodingUCS2,
		DCS:              0x08,
		Source:           codec.Address{MSISDN: "3342012832"},
		Destination:      codec.Address{MSISDN: "3342012860"},
		Text:             "hello over retry",
	}

	f.persistMessage(context.Background(), msg, "sip3gpp", "peer1", store.MessageStatusQueued, now)

	if st.saved == nil {
		t.Fatal("message was not saved")
	}
	if got, want := string(st.saved.Payload), msg.Text; got != want {
		t.Fatalf("saved payload text = %q, want %q", got, want)
	}
}

func TestStoreToCodecMessageRestoresBinaryEncodingAndUDH(t *testing.T) {
	m := store.Message{
		ID:          "msg-2",
		OriginIface: string(codec.InterfaceSMPP),
		EgressIface: string(codec.InterfaceSIP3GPP),
		SrcMSISDN:   "3342012832",
		DstMSISDN:   "3342012860",
		Payload:     []byte{0xde, 0xad},
		UDH:         []byte{0x06, 0x08, 0x04, 0x12, 0x34, 0x02, 0x01},
		Encoding:    int(codec.EncodingBinary),
		DCS:         0x04,
	}

	msg := storeToCodecMessage(m)
	if got, want := msg.Encoding, codec.EncodingBinary; got != want {
		t.Fatalf("encoding = %v, want %v", got, want)
	}
	if string(msg.Binary) != string(m.Payload) {
		t.Fatalf("binary payload = %x, want %x", msg.Binary, m.Payload)
	}
	if msg.UDH == nil || string(msg.UDH.Raw) != string(m.UDH) {
		t.Fatalf("UDH = %x, want %x", msg.UDH.Raw, m.UDH)
	}
}

func TestStoreToCodecMessageRestoresTextPayload(t *testing.T) {
	m := store.Message{
		ID:          "msg-text-2",
		OriginIface: string(codec.InterfaceSMPP),
		EgressIface: string(codec.InterfaceSIP3GPP),
		SrcMSISDN:   "3342012832",
		DstMSISDN:   "3342012860",
		Payload:     []byte("hello over retry"),
		Encoding:    int(codec.EncodingUCS2),
		DCS:         0x08,
	}

	msg := storeToCodecMessage(m)
	if got, want := msg.Encoding, codec.EncodingUCS2; got != want {
		t.Fatalf("encoding = %v, want %v", got, want)
	}
	if got, want := msg.Text, "hello over retry"; got != want {
		t.Fatalf("text = %q, want %q", got, want)
	}
	if len(msg.Binary) != 0 {
		t.Fatalf("binary payload = %x, want empty", msg.Binary)
	}
}

func TestSelectRouteFallsThroughFromSGdToNextRule(t *testing.T) {
	st := &forwarderTestStore{
		sipPeers: map[string]store.SIPPeer{
			"simple1": {Name: "simple1"},
		},
	}
	reg := newTestRegistry(t, st)
	s6cClient := &fakeS6cClient{info: &s6c.RoutingInfo{Attached: false}}
	reg.SetS6cClient(s6cClient)

	engine := routing.NewEngine()
	engine.Reload([]store.RoutingRule{
		{Name: "sgd-first", Priority: 10, MatchSrcIface: "smpp", MatchDstPrefix: "334", EgressIface: "sgd"},
		{Name: "simple-second", Priority: 20, MatchSrcIface: "smpp", MatchDstPrefix: "334", EgressIface: "sipsimple", EgressPeer: "simple1"},
	})

	f := New(Config{
		Registry:     reg,
		Engine:       engine,
		Store:        st,
		SimpleSender: &fakeSimpleSender{},
		SGdSender:    &fakeSGdSender{},
	})

	route, err := f.selectRoute(context.Background(), &codec.Message{
		IngressInterface: codec.InterfaceSMPP,
		Destination:      codec.Address{MSISDN: "3342012860"},
	}, 0)
	if err != nil {
		t.Fatalf("selectRoute() error = %v", err)
	}
	if got, want := route.egressIface, "sipsimple"; got != want {
		t.Fatalf("egressIface = %q, want %q", got, want)
	}
	if got, want := route.egressPeer, "simple1"; got != want {
		t.Fatalf("egressPeer = %q, want %q", got, want)
	}
	if got := s6cClient.calls; got != 1 {
		t.Fatalf("S6c calls = %d, want 1", got)
	}
}

func TestSelectRouteRefreshesS6cBeforeSGdWhenSubscriberCacheClaimsAttached(t *testing.T) {
	st := &forwarderTestStore{
		subscriber: &store.Subscriber{
			MSISDN:        "3342012832",
			IMSI:          "311435000070570",
			LTEAttached:   true,
			MMENumber:     "15550000001",
			MMEHost:       "stale-mme.epc.mnc435.mcc311.3gppnetwork.org",
			IMSRegistered: false,
			UpdatedAt:     time.Now().Add(-301 * time.Second),
		},
		sipPeers: map[string]store.SIPPeer{
			"simple1": {Name: "simple1"},
		},
	}
	reg := newTestRegistry(t, st)
	s6cClient := &fakeS6cClient{info: &s6c.RoutingInfo{Attached: false}}
	reg.SetS6cClient(s6cClient)

	engine := routing.NewEngine()
	engine.Reload([]store.RoutingRule{
		{Name: "sgd-first", Priority: 10, MatchSrcIface: "smpp", MatchDstPrefix: "334", EgressIface: "sgd"},
		{Name: "simple-second", Priority: 20, MatchSrcIface: "smpp", MatchDstPrefix: "334", EgressIface: "sipsimple", EgressPeer: "simple1"},
	})

	f := New(Config{
		Registry:     reg,
		Engine:       engine,
		Store:        st,
		SimpleSender: &fakeSimpleSender{},
		SGdSender:    &fakeSGdSender{},
	})

	route, err := f.selectRoute(context.Background(), &codec.Message{
		IngressInterface: codec.InterfaceSMPP,
		Destination:      codec.Address{MSISDN: "3342012832"},
	}, 0)
	if err != nil {
		t.Fatalf("selectRoute() error = %v", err)
	}
	if got, want := route.egressIface, "sipsimple"; got != want {
		t.Fatalf("egressIface = %q, want %q", got, want)
	}
	if got := s6cClient.calls; got != 1 {
		t.Fatalf("S6c calls = %d, want 1", got)
	}
	if st.subscriber == nil {
		t.Fatal("subscriber cache was not updated")
	}
	if st.subscriber.LTEAttached {
		t.Fatalf("LTEAttached = %v, want false after S6c refresh", st.subscriber.LTEAttached)
	}
}

func TestSelectRouteUsesFreshS6cCacheForSGdCandidate(t *testing.T) {
	st := &forwarderTestStore{
		subscriber: &store.Subscriber{
			MSISDN:        "3342012832",
			IMSI:          "311435000070570",
			LTEAttached:   true,
			MMENumber:     "15550000001",
			MMEHost:       "fresh-mme.epc.mnc435.mcc311.3gppnetwork.org",
			IMSRegistered: false,
			UpdatedAt:     time.Now().Add(-299 * time.Second),
		},
	}
	reg := newTestRegistry(t, st)
	s6cClient := &fakeS6cClient{info: &s6c.RoutingInfo{Attached: false}}
	reg.SetS6cClient(s6cClient)

	engine := routing.NewEngine()
	engine.Reload([]store.RoutingRule{
		{Name: "sgd-first", Priority: 10, MatchSrcIface: "smpp", MatchDstPrefix: "334", EgressIface: "sgd"},
	})

	f := New(Config{
		Registry:  reg,
		Engine:    engine,
		Store:     st,
		SGdSender: &fakeSGdSender{},
	})

	route, err := f.selectRoute(context.Background(), &codec.Message{
		IngressInterface: codec.InterfaceSMPP,
		Destination:      codec.Address{MSISDN: "3342012832"},
	}, 0)
	if err != nil {
		t.Fatalf("selectRoute() error = %v", err)
	}
	if got, want := route.egressIface, "sgd"; got != want {
		t.Fatalf("egressIface = %q, want %q", got, want)
	}
	if got, want := route.egressPeer, "fresh-mme.epc.mnc435.mcc311.3gppnetwork.org"; got != want {
		t.Fatalf("egressPeer = %q, want %q", got, want)
	}
	if got, want := route.sgdMMENum, "15550000001"; got != want {
		t.Fatalf("sgdMMENum = %q, want %q", got, want)
	}
	if got := s6cClient.calls; got != 0 {
		t.Fatalf("S6c calls = %d, want 0", got)
	}
}

func TestSelectRouteUsesFreshS6cCacheHexMMENumberForSGdCandidate(t *testing.T) {
	st := &forwarderTestStore{
		subscriber: &store.Subscriber{
			MSISDN:        "3342012832",
			IMSI:          "311435000070570",
			LTEAttached:   true,
			MMENumber:     "5155000000f1",
			MMEHost:       "fresh-mme.epc.mnc435.mcc311.3gppnetwork.org",
			IMSRegistered: false,
			UpdatedAt:     time.Now().Add(-299 * time.Second),
		},
	}
	reg := newTestRegistry(t, st)
	s6cClient := &fakeS6cClient{info: &s6c.RoutingInfo{Attached: false}}
	reg.SetS6cClient(s6cClient)

	engine := routing.NewEngine()
	engine.Reload([]store.RoutingRule{
		{Name: "sgd-first", Priority: 10, MatchSrcIface: "smpp", MatchDstPrefix: "334", EgressIface: "sgd"},
	})

	sgdSender := &fakeSGdSender{}
	f := New(Config{
		Registry:  reg,
		Engine:    engine,
		Store:     st,
		SGdSender: sgdSender,
		SCAddr:    "15550000000",
	})

	msg := &codec.Message{
		IngressInterface: codec.InterfaceSMPP,
		Destination:      codec.Address{MSISDN: "3342012832"},
	}
	route, err := f.selectRoute(context.Background(), msg, 0)
	if err != nil {
		t.Fatalf("selectRoute() error = %v", err)
	}
	if got, want := route.sgdMMENum, "15550000001"; got != want {
		t.Fatalf("sgdMMENum = %q, want %q", got, want)
	}
	if got := s6cClient.calls; got != 0 {
		t.Fatalf("S6c calls = %d, want 0", got)
	}

	if err := f.deliverSelectedRoute(context.Background(), msg, route); err != nil {
		t.Fatalf("deliverSelectedRoute() error = %v", err)
	}
	if got, want := sgdSender.lastSCAddr, "15550000000"; got != want {
		t.Fatalf("SCAddress = %q, want %q", got, want)
	}
}

func TestDeliverSGdPrefersDestinationMMENumberAsSCAddress(t *testing.T) {
	sgdSender := &fakeSGdSender{}
	f := &Forwarder{
		scAddr:    "15550000000",
		sgdSender: sgdSender,
	}

	msg := &codec.Message{
		Destination: codec.Address{
			MSISDN:    "3342012832",
			IMSI:      "311435000070570",
			MMENumber: "15550000001",
		},
	}

	if err := f.deliverSGd(context.Background(), msg, "mme01.example.net"); err != nil {
		t.Fatalf("deliverSGd() error = %v", err)
	}
	if got, want := sgdSender.lastSCAddr, "15550000000"; got != want {
		t.Fatalf("SCAddress = %q, want %q", got, want)
	}
	if got, want := sgdSender.lastMME, "mme01.example.net"; got != want {
		t.Fatalf("MME = %q, want %q", got, want)
	}
}

func TestDeliverSGdUsesConfiguredSCAddress(t *testing.T) {
	sgdSender := &fakeSGdSender{}
	f := &Forwarder{
		scAddr:    "15550000000",
		sgdSender: sgdSender,
	}

	msg := &codec.Message{
		Destination: codec.Address{
			MSISDN:    "3342012832",
			IMSI:      "311435000070570",
			MMENumber: "5155000000f1",
		},
	}

	if err := f.deliverSGd(context.Background(), msg, "mme01.example.net"); err != nil {
		t.Fatalf("deliverSGd() error = %v", err)
	}
	if got, want := sgdSender.lastSCAddr, "15550000000"; got != want {
		t.Fatalf("SCAddress = %q, want %q", got, want)
	}
}

func TestRetryDispatchTriesNextRuleWithinSameRetryPass(t *testing.T) {
	st := &forwarderTestStore{
		sipPeers: map[string]store.SIPPeer{
			"simple1": {Name: "simple1"},
			"simple2": {Name: "simple2"},
		},
	}
	reg := newTestRegistry(t, st)

	engine := routing.NewEngine()
	engine.Reload([]store.RoutingRule{
		{Name: "rule-1", Priority: 10, MatchSrcIface: "smpp", MatchDstPrefix: "334", EgressIface: "sipsimple", EgressPeer: "simple1"},
		{Name: "rule-2", Priority: 20, MatchSrcIface: "smpp", MatchDstPrefix: "334", EgressIface: "sipsimple", EgressPeer: "simple2"},
	})

	simpleSender := &fakeSimpleSender{errs: []error{context.DeadlineExceeded, nil}}
	f := New(Config{
		Registry:     reg,
		Engine:       engine,
		Store:        st,
		SimpleSender: simpleSender,
	})
	retries := NewRetryScheduler(f, time.Second)

	retries.dispatch(context.Background(), store.Message{
		ID:          "msg-1",
		OriginIface: string(codec.InterfaceSMPP),
		DstMSISDN:   "3342012860",
		SrcMSISDN:   "3342012832",
	})

	if got := st.retryCount; got != 0 {
		t.Fatalf("retryCount = %d, want 0", got)
	}
	if got, want := st.routeIface, "sipsimple"; got != want {
		t.Fatalf("routeIface = %q, want %q", got, want)
	}
	if got, want := st.routePeer, "simple2"; got != want {
		t.Fatalf("routePeer = %q, want %q", got, want)
	}
	if got, want := st.routingCalls, 1; got != want {
		t.Fatalf("routingCalls = %d, want %d", got, want)
	}
	if got, want := simpleSender.calls, 2; got != want {
		t.Fatalf("simple sender calls = %d, want %d", got, want)
	}
	if len(st.statusUpdates) == 0 || st.statusUpdates[len(st.statusUpdates)-1] != store.MessageStatusDelivered {
		t.Fatalf("final status = %v, want delivered", st.statusUpdates)
	}
}

func TestRetryDispatchPreservesTextPayloadForISC(t *testing.T) {
	st := &forwarderTestStore{}
	reg := newTestRegistry(t, st)
	if err := reg.Upsert(context.Background(), registry.Registration{
		MSISDN:     "3342012860",
		SIPAOR:     "sip:3342012860@example.com",
		SCSCF:      "scscf.example.com",
		Registered: true,
		Expiry:     time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("registry upsert: %v", err)
	}

	iscSender := &fakeISCSender{}
	f := New(Config{
		Registry:  reg,
		Engine:    routing.NewEngine(),
		Store:     st,
		ISCSender: iscSender,
	})
	retries := NewRetryScheduler(f, time.Second)

	retries.dispatch(context.Background(), store.Message{
		ID:          "msg-isc-1",
		OriginIface: string(codec.InterfaceSMPP),
		SrcMSISDN:   "3342012832",
		DstMSISDN:   "3342012860",
		Payload:     []byte("hello over retry"),
		Encoding:    int(codec.EncodingUCS2),
		DCS:         0x08,
	})

	if got, want := iscSender.calls, 1; got != want {
		t.Fatalf("ISC sender calls = %d, want %d", got, want)
	}
	if len(iscSender.msgs) != 1 {
		t.Fatalf("captured messages = %d, want 1", len(iscSender.msgs))
	}
	if got, want := iscSender.msgs[0].Text, "hello over retry"; got != want {
		t.Fatalf("retried text = %q, want %q", got, want)
	}
	if len(iscSender.msgs[0].Binary) != 0 {
		t.Fatalf("retried binary = %x, want empty", iscSender.msgs[0].Binary)
	}
	if len(st.statusUpdates) == 0 || st.statusUpdates[len(st.statusUpdates)-1] != store.MessageStatusDelivered {
		t.Fatalf("final status = %v, want delivered", st.statusUpdates)
	}
}
