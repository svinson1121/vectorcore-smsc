package main

import (
	"context"
	"testing"
	"time"

	"github.com/svinson1121/vectorcore-smsc/internal/diameter/s6c"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

type alertTestStore struct {
	messages []store.Message
	updated  map[string]time.Time
	reasons  map[string]string
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
func (s *alertTestStore) ClaimMessageForDispatch(context.Context, string, []string) (bool, error) {
	panic("unexpected call")
}
func (s *alertTestStore) UpdateMessageRetry(_ context.Context, id string, _ int, nextRetryAt time.Time, _ int) error {
	if s.updated == nil {
		s.updated = map[string]time.Time{}
	}
	s.updated[id] = nextRetryAt
	return nil
}
func (s *alertTestStore) UpdateMessageExpiryCap(context.Context, string, time.Time) error {
	panic("unexpected call")
}
func (s *alertTestStore) UpdateMessageDeferred(_ context.Context, id, deferredReason, _, _ string, _ int) error {
	if s.reasons == nil {
		s.reasons = map[string]string{}
	}
	s.reasons[id] = deferredReason
	return nil
}
func (s *alertTestStore) RequeueMessageForAlert(_ context.Context, id string, nextRetryAt time.Time, _ int, deferredReason string, allowedStatuses []string) (bool, error) {
	allowed := map[string]struct{}{}
	for _, status := range allowedStatuses {
		allowed[status] = struct{}{}
	}
	for i := range s.messages {
		if s.messages[i].ID != id {
			continue
		}
		if _, ok := allowed[s.messages[i].Status]; !ok {
			return false, nil
		}
		s.messages[i].Status = store.MessageStatusWaitTimer
		if s.updated == nil {
			s.updated = map[string]time.Time{}
		}
		s.updated[id] = nextRetryAt
		if deferredReason != "" {
			if s.reasons == nil {
				s.reasons = map[string]string{}
			}
			s.reasons[id] = deferredReason
		}
		return true, nil
	}
	return false, nil
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
		if filter.AlertCorrelationID != "" && msg.AlertCorrelationID != filter.AlertCorrelationID {
			continue
		}
		if filter.DeferredInterface != "" && msg.DeferredInterface != filter.DeferredInterface {
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
func (s *alertTestStore) ListSGDMMEMappings(context.Context) ([]store.SGDMMEMapping, error) {
	panic("unexpected call")
}
func (s *alertTestStore) GetSGDMMEMappingByID(context.Context, string) (*store.SGDMMEMapping, error) {
	panic("unexpected call")
}
func (s *alertTestStore) CreateSGDMMEMapping(context.Context, store.SGDMMEMapping) error {
	panic("unexpected call")
}
func (s *alertTestStore) UpdateSGDMMEMapping(context.Context, store.SGDMMEMapping) error {
	panic("unexpected call")
}
func (s *alertTestStore) DeleteSGDMMEMapping(context.Context, string) error {
	panic("unexpected call")
}

func makeALSCRequeueHandler(st store.Store) func(s6c.AlertServiceCentreRequest) error {
	return func(req s6c.AlertServiceCentreRequest) error {
		if req.AlertCorrelationID == "" && req.MSISDN == "" {
			return nil
		}
		filter := store.MessageFilter{
			Statuses: []string{
				store.MessageStatusQueued,
				store.MessageStatusWaitTimer,
				store.MessageStatusWaitEvent,
				store.MessageStatusWaitTimerEvent,
			},
			DstMSISDN:          req.MSISDN,
			AlertCorrelationID: req.AlertCorrelationID,
			Limit:              1000,
		}
		if req.AlertCorrelationID != "" {
			filter.DstMSISDN = ""
		}
		msgs, err := st.ListFilteredMessages(context.Background(), filter)
		if err != nil {
			return err
		}
		now := time.Now()
		for _, msg := range msgs {
			ok, err := st.RequeueMessageForAlert(context.Background(), msg.ID, now, msg.RouteCursor, "", []string{
				store.MessageStatusQueued,
				store.MessageStatusWaitTimer,
				store.MessageStatusWaitEvent,
				store.MessageStatusWaitTimerEvent,
			})
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
		}
		return nil
	}
}

func TestALSCRequeueHandlerRequeuesQueuedMessagesByCorrelationFirst(t *testing.T) {
	st := &alertTestStore{
		messages: []store.Message{
			{ID: "m1", DstMSISDN: "3342012832", Status: store.MessageStatusQueued, AlertCorrelationID: "corr-1"},
			{ID: "m2", DstMSISDN: "3342012832", Status: store.MessageStatusQueued, AlertCorrelationID: "corr-2"},
			{ID: "m3", DstMSISDN: "1111111111", Status: store.MessageStatusQueued, AlertCorrelationID: "corr-1"},
		},
	}

	handler := makeALSCRequeueHandler(st)
	if err := handler(s6c.AlertServiceCentreRequest{AlertCorrelationID: "corr-1"}); err != nil {
		t.Fatalf("handler() error = %v", err)
	}

	if got, want := len(st.updated), 2; got != want {
		t.Fatalf("updated messages = %d, want %d", got, want)
	}
	if _, ok := st.updated["m1"]; !ok {
		t.Fatal("expected m1 to be requeued")
	}
	if _, ok := st.updated["m3"]; !ok {
		t.Fatal("expected m3 to be requeued")
	}
}

func TestAlertStoreDeferredMarkerCanBeUpdated(t *testing.T) {
	st := &alertTestStore{}
	if err := st.UpdateMessageDeferred(context.Background(), "m1", "sgd_alert_retry", "sgd", "mme01.example.net", 2); err != nil {
		t.Fatalf("UpdateMessageDeferred() error = %v", err)
	}
	if got, want := st.reasons["m1"], "sgd_alert_retry"; got != want {
		t.Fatalf("deferred reason = %q, want %q", got, want)
	}
}

func TestAlertStoreRequeueSkipsDeliveredMessages(t *testing.T) {
	st := &alertTestStore{
		messages: []store.Message{
			{ID: "m1", Status: store.MessageStatusDelivered},
		},
	}
	ok, err := st.RequeueMessageForAlert(context.Background(), "m1", time.Now(), 0, "", []string{
		store.MessageStatusQueued,
		store.MessageStatusWaitTimer,
		store.MessageStatusWaitEvent,
		store.MessageStatusWaitTimerEvent,
	})
	if err != nil {
		t.Fatalf("RequeueMessageForAlert() error = %v", err)
	}
	if ok {
		t.Fatal("expected delivered message requeue to be skipped")
	}
}
