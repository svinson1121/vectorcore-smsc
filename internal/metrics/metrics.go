// Package metrics registers VectorCore SMSC Prometheus collectors.
package metrics

import "github.com/prometheus/client_golang/prometheus"

// M holds all SMSC Prometheus metrics.
type M struct {
	// Message counters per interface
	MessagesIn  *prometheus.CounterVec
	MessagesOut *prometheus.CounterVec

	// Delivery report counters
	DeliveryReports *prometheus.CounterVec

	// Store-and-forward
	SFQueued  prometheus.Gauge
	SFRetried prometheus.Counter
	SFExpired prometheus.Gauge

	// Peer connection gauges
	SMPPSessions  prometheus.Gauge
	SIPPeers      prometheus.Gauge
	DiameterPeers prometheus.Gauge
}

// New creates and registers all metrics with the default Prometheus registry.
func New() *M {
	m := &M{
		MessagesIn: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "smsc",
			Name:      "messages_in_total",
			Help:      "Total inbound messages by ingress interface.",
		}, []string{"interface"}),

		MessagesOut: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "smsc",
			Name:      "messages_out_total",
			Help:      "Total outbound messages by egress interface.",
		}, []string{"interface"}),

		DeliveryReports: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "smsc",
			Name:      "delivery_reports_total",
			Help:      "Total delivery reports by interface and status.",
		}, []string{"interface", "status"}),

		SFQueued: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "smsc",
			Name:      "store_forward_queued",
			Help:      "Current number of messages in the store-and-forward queue.",
		}),

		SFRetried: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "smsc",
			Name:      "store_forward_retried_total",
			Help:      "Total store-and-forward retry attempts.",
		}),

		SFExpired: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "smsc",
			Name:      "store_forward_expired",
			Help:      "Current count of messages that have expired in the store-and-forward queue.",
		}),

		SMPPSessions: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "smsc",
			Name:      "smpp_sessions_connected",
			Help:      "Number of currently connected SMPP server sessions.",
		}),

		SIPPeers: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "smsc",
			Name:      "sip_peers_connected",
			Help:      "Number of SIP SIMPLE peers currently reachable.",
		}),

		DiameterPeers: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "smsc",
			Name:      "diameter_peers_connected",
			Help:      "Number of Diameter peers in OPEN state.",
		}),
	}

	prometheus.MustRegister(
		m.MessagesIn,
		m.MessagesOut,
		m.DeliveryReports,
		m.SFQueued,
		m.SFRetried,
		m.SFExpired,
		m.SMPPSessions,
		m.SIPPeers,
		m.DiameterPeers,
	)

	return m
}
