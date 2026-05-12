package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "yunt",
		Name:      "http_requests_total",
		Help:      "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "yunt",
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request duration in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "path"})

	HTTPActiveRequests = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "yunt",
		Name:      "http_active_requests",
		Help:      "Number of active HTTP requests.",
	})

	DBQueriesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "yunt",
		Name:      "db_queries_total",
		Help:      "Total number of database queries.",
	}, []string{"operation"})

	DBQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "yunt",
		Name:      "db_query_duration_seconds",
		Help:      "Database query duration in seconds.",
		Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	}, []string{"operation"})

	SMTPMessagesReceived = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "yunt",
		Name:      "smtp_messages_received_total",
		Help:      "Total number of SMTP messages received.",
	})

	SMTPMessagesRejected = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "yunt",
		Name:      "smtp_messages_rejected_total",
		Help:      "Total number of SMTP messages rejected.",
	}, []string{"reason"})

	IMAPActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "yunt",
		Name:      "imap_active_connections",
		Help:      "Number of active IMAP connections.",
	})

	MailboxMessageCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "yunt",
		Name:      "mailbox_message_count",
		Help:      "Number of messages per mailbox.",
	}, []string{"mailbox"})
)
