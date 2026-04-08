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
	statusUpdates []string
	retryCount    int
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
	return nil
}
func (s *forwarderTestStore) UpdateMessageStatus(_ context.Context, _ string, status string) error {
	s.statusUpdates = append(s.statusUpdates, status)
	return nil
}
func (s *forwarderTestStore) UpdateMessageRetry(_ context.Context, _ string, retryCount int, nextRetryAt time.Time) error {
	s.retryCount = retryCount
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

type fakeSimpleSender struct{ calls int }

func (f *fakeSimpleSender) Send(context.Context, *codec.Message, store.SIPPeer) error {
	f.calls++
	return nil
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

type fakeSGdSender struct{}

func (f *fakeSGdSender) SendOFR(context.Context, *codec.Message, string, string) error { return nil }
func (f *fakeSGdSender) HasPeerForMME(string) bool                                     { return true }

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
	})
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
	})
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
