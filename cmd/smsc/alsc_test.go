package main

import (
	"context"
	"testing"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/codec"
	"github.com/svinson1121/vectorcore-smsc/internal/diameter/s6c"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

type alertTestStore struct {
	messages []store.Message
	updated  map[string]time.Time
}

func (s *alertTestStore) GetIMSRegistration(context.Context, string) (*store.IMSRegistration, error) {
	panic("unexpected call")
}
func (s *alertTestStore) UpsertIMSRegistration(context.Context, store.IMSRegistration) error {
	panic("unexpected call")
}
func (s *alertTestStore) DeleteIMSRegistration(context.Context, string) error {
	panic("unexpected call")
}
func (s *alertTestStore) ListIMSRegistrations(context.Context) ([]store.IMSRegistration, error) {
	panic("unexpected call")
}
func (s *alertTestStore) GetSubscriber(context.Context, string) (*store.Subscriber, error) {
	panic("unexpected call")
}
func (s *alertTestStore) UpsertSubscriber(context.Context, store.Subscriber) error {
	panic("unexpected call")
}
func (s *alertTestStore) GetSMPPServerAccount(context.Context, string) (*store.SMPPServerAccount, error) {
	panic("unexpected call")
}
func (s *alertTestStore) ListSMPPServerAccounts(context.Context) ([]store.SMPPServerAccount, error) {
	panic("unexpected call")
}
func (s *alertTestStore) ListSMPPClients(context.Context) ([]store.SMPPClient, error) {
	panic("unexpected call")
}
func (s *alertTestStore) ListSIPPeers(context.Context) ([]store.SIPPeer, error) {
	panic("unexpected call")
}
func (s *alertTestStore) GetSIPPeer(context.Context, string) (*store.SIPPeer, error) {
	panic("unexpected call")
}
func (s *alertTestStore) ListDiameterPeers(context.Context) ([]store.DiameterPeer, error) {
	panic("unexpected call")
}
func (s *alertTestStore) GetDiameterPeer(context.Context, string) (*store.DiameterPeer, error) {
	panic("unexpected call")
}
func (s *alertTestStore) ListRoutingRules(context.Context) ([]store.RoutingRule, error) {
	panic("unexpected call")
}
func (s *alertTestStore) GetSFPolicy(context.Context, string) (*store.SFPolicy, error) {
	panic("unexpected call")
}
func (s *alertTestStore) SaveMessage(context.Context, store.Message) error { panic("unexpected call") }
func (s *alertTestStore) UpdateMessageRouting(context.Context, string, string, string) error {
	panic("unexpected call")
}
func (s *alertTestStore) UpdateMessageStatus(context.Context, string, string) error {
	panic("unexpected call")
}
func (s *alertTestStore) UpdateMessageRetry(_ context.Context, id string, _ int, nextRetryAt time.Time) error {
	if s.updated == nil {
		s.updated = map[string]time.Time{}
	}
	s.updated[id] = nextRetryAt
	return nil
}
func (s *alertTestStore) ListRetryableMessages(context.Context) ([]store.Message, error) {
	panic("unexpected call")
}
func (s *alertTestStore) ListExpiredMessages(context.Context) ([]store.Message, error) {
	panic("unexpected call")
}
func (s *alertTestStore) GetMessage(context.Context, string) (*store.Message, error) {
	panic("unexpected call")
}
func (s *alertTestStore) DeleteMessage(context.Context, string) error { panic("unexpected call") }
func (s *alertTestStore) SaveDeliveryReport(context.Context, store.DeliveryReport) error {
	panic("unexpected call")
}
func (s *alertTestStore) GetSMPPServerAccountByID(context.Context, string) (*store.SMPPServerAccount, error) {
	panic("unexpected call")
}
func (s *alertTestStore) CreateSMPPServerAccount(context.Context, store.SMPPServerAccount) error {
	panic("unexpected call")
}
func (s *alertTestStore) UpdateSMPPServerAccount(context.Context, store.SMPPServerAccount) error {
	panic("unexpected call")
}
func (s *alertTestStore) DeleteSMPPServerAccount(context.Context, string) error {
	panic("unexpected call")
}
func (s *alertTestStore) GetSMPPClient(context.Context, string) (*store.SMPPClient, error) {
	panic("unexpected call")
}
func (s *alertTestStore) CreateSMPPClient(context.Context, store.SMPPClient) error {
	panic("unexpected call")
}
func (s *alertTestStore) UpdateSMPPClient(context.Context, store.SMPPClient) error {
	panic("unexpected call")
}
func (s *alertTestStore) DeleteSMPPClient(context.Context, string) error { panic("unexpected call") }
func (s *alertTestStore) ListAllSIPPeers(context.Context) ([]store.SIPPeer, error) {
	panic("unexpected call")
}
func (s *alertTestStore) GetSIPPeerByID(context.Context, string) (*store.SIPPeer, error) {
	panic("unexpected call")
}
func (s *alertTestStore) CreateSIPPeer(context.Context, store.SIPPeer) error {
	panic("unexpected call")
}
func (s *alertTestStore) UpdateSIPPeer(context.Context, store.SIPPeer) error {
	panic("unexpected call")
}
func (s *alertTestStore) DeleteSIPPeer(context.Context, string) error { panic("unexpected call") }
func (s *alertTestStore) ListAllDiameterPeers(context.Context) ([]store.DiameterPeer, error) {
	panic("unexpected call")
}
func (s *alertTestStore) GetDiameterPeerByID(context.Context, string) (*store.DiameterPeer, error) {
	panic("unexpected call")
}
func (s *alertTestStore) CreateDiameterPeer(context.Context, store.DiameterPeer) error {
	panic("unexpected call")
}
func (s *alertTestStore) UpdateDiameterPeer(context.Context, store.DiameterPeer) error {
	panic("unexpected call")
}
func (s *alertTestStore) DeleteDiameterPeer(context.Context, string) error { panic("unexpected call") }
func (s *alertTestStore) ListAllRoutingRules(context.Context) ([]store.RoutingRule, error) {
	panic("unexpected call")
}
func (s *alertTestStore) GetRoutingRule(context.Context, string) (*store.RoutingRule, error) {
	panic("unexpected call")
}
func (s *alertTestStore) CreateRoutingRule(context.Context, store.RoutingRule) error {
	panic("unexpected call")
}
func (s *alertTestStore) UpdateRoutingRule(context.Context, store.RoutingRule) error {
	panic("unexpected call")
}
func (s *alertTestStore) DeleteRoutingRule(context.Context, string) error { panic("unexpected call") }
func (s *alertTestStore) ListSFPolicies(context.Context) ([]store.SFPolicy, error) {
	panic("unexpected call")
}
func (s *alertTestStore) CreateSFPolicy(context.Context, store.SFPolicy) error {
	panic("unexpected call")
}
func (s *alertTestStore) UpdateSFPolicy(context.Context, store.SFPolicy) error {
	panic("unexpected call")
}
func (s *alertTestStore) DeleteSFPolicy(context.Context, string) error { panic("unexpected call") }
func (s *alertTestStore) ListSubscribers(context.Context) ([]store.Subscriber, error) {
	panic("unexpected call")
}
func (s *alertTestStore) GetSubscriberByID(context.Context, string) (*store.Subscriber, error) {
	panic("unexpected call")
}
func (s *alertTestStore) DeleteSubscriber(context.Context, string) error { panic("unexpected call") }
func (s *alertTestStore) ListMessages(context.Context, int) ([]store.Message, error) {
	panic("unexpected call")
}
func (s *alertTestStore) ListFilteredMessages(_ context.Context, filter store.MessageFilter) ([]store.Message, error) {
	var out []store.Message
	for _, msg := range s.messages {
		if filter.DstMSISDN != "" && msg.DstMSISDN != filter.DstMSISDN {
			continue
		}
		matchStatus := len(filter.Statuses) == 0
		for _, status := range filter.Statuses {
			if msg.Status == status {
				matchStatus = true
			}
		}
		if !matchStatus {
			continue
		}
		out = append(out, msg)
	}
	return out, nil
}
func (s *alertTestStore) CountMessagesByStatus(context.Context) (map[string]int64, error) {
	panic("unexpected call")
}
func (s *alertTestStore) ListDeliveryReports(context.Context, int) ([]store.DeliveryReport, error) {
	panic("unexpected call")
}
func (s *alertTestStore) GetDeliveryReport(context.Context, string) (*store.DeliveryReport, error) {
	panic("unexpected call")
}
func (s *alertTestStore) Subscribe(context.Context, string, chan<- store.ChangeEvent) error {
	panic("unexpected call")
}
func (s *alertTestStore) Close() error { return nil }

func makeALSCRequeueHandler(st store.Store) func(s6c.AlertServiceCentreRequest) error {
	return func(req s6c.AlertServiceCentreRequest) error {
		msisdn := req.MSISDN
		if msisdn == "" {
			return nil
		}
		msgs, err := st.ListFilteredMessages(context.Background(), store.MessageFilter{
			Statuses:  []string{store.MessageStatusQueued},
			DstMSISDN: msisdn,
			Limit:     1000,
		})
		if err != nil {
			return err
		}
		now := time.Now()
		for _, msg := range msgs {
			if msg.EgressIface != string(codec.InterfaceSGd) {
				continue
			}
			if err := st.UpdateMessageRetry(context.Background(), msg.ID, msg.RetryCount, now); err != nil {
				return err
			}
		}
		return nil
	}
}

func TestALSCRequeueHandlerOnlyRequeuesQueuedSGdMessages(t *testing.T) {
	st := &alertTestStore{
		messages: []store.Message{
			{ID: "m1", DstMSISDN: "3342012832", Status: store.MessageStatusQueued, EgressIface: string(codec.InterfaceSGd)},
			{ID: "m2", DstMSISDN: "3342012832", Status: store.MessageStatusQueued, EgressIface: string(codec.InterfaceSMPP)},
			{ID: "m3", DstMSISDN: "1111111111", Status: store.MessageStatusQueued, EgressIface: string(codec.InterfaceSGd)},
		},
	}

	handler := makeALSCRequeueHandler(st)
	if err := handler(s6c.AlertServiceCentreRequest{MSISDN: "3342012832"}); err != nil {
		t.Fatalf("handler() error = %v", err)
	}

	if got, want := len(st.updated), 1; got != want {
		t.Fatalf("updated messages = %d, want %d", got, want)
	}
	if _, ok := st.updated["m1"]; !ok {
		t.Fatal("expected m1 to be requeued")
	}
}
