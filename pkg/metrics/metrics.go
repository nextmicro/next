package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	DefaultNamespace   = "next"
	ComponentNamespace = "component"

	BuildInfoGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: DefaultNamespace,
		Subsystem: "app",
		Name:      "build_info",
	}, []string{"app_id", "app_name", "app_version", "deploy_env", "go_version", "next_version", "start_time", "build_time"})

	// ClientMetricMillisecond is a prometheus histogram for measuring the duration of a request.
	ClientMetricMillisecond = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: DefaultNamespace,
		Subsystem: "client_requests",
		Name:      "duration_ms",
		Help:      "requests duration(ms).",
		Buckets:   []float64{0.1, 0.5, 1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
	}, []string{"kind", "callee", "method"})

	// ClientMetricRequests  is a counter vector of requests.
	ClientMetricRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: DefaultNamespace,
		Subsystem: "client_requests",
		Name:      "total",
		Help:      "The total number of processed requests",
	}, []string{"kind", "callee", "method", "status"})

	// ServerMetricMillisecond is a prometheus histogram for measuring the duration of a request.
	ServerMetricMillisecond = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: DefaultNamespace,
		Subsystem: "server_requests",
		Name:      "duration_ms",
		Help:      "requests duration(ms).",
		Buckets:   []float64{0.1, 0.5, 1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
	}, []string{"kind", "caller", "method"})

	// ServerMetricRequests  is a counter vector of requests.
	ServerMetricRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: DefaultNamespace,
		Subsystem: "server_requests",
		Name:      "total",
		Help:      "The total number of processed requests",
	}, []string{"kind", "caller", "method", "status"})

	// MetricRateLimitTotal is a counter vector of rate limit.
	MetricRateLimitTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: DefaultNamespace,
		Subsystem: "requests_ratelimit",
		Name:      "total",
		Help:      "The total number of ratelimit denied requests",
	}, []string{"kind", "caller", "method"})

	// DBSystemMetricMillisecond is a prometheus histogram for measuring the duration of a request.
	DBSystemMetricMillisecond = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: ComponentNamespace,
		Subsystem: "db_system_requests",
		Name:      "duration_ms",
		Help:      "requests duration(ms).",
		Buckets:   []float64{0.1, 0.5, 1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
	}, []string{"kind", "name", "addr", "command"})

	// DBSystemMetricRequests  is a counter vector of requests.
	DBSystemMetricRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: ComponentNamespace,
		Subsystem: "db_system_requests",
		Name:      "total",
		Help:      "The total number of processed requests",
	}, []string{"kind", "name", "addr", "command", "status"})

	// DBSystemStatsGauge is a db stats
	DBSystemStatsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: ComponentNamespace,
		Name:      "db_system_stats",
		Help:      "The contains database statistics.",
	}, []string{"kind", "name", "addr", "index"})

	// MessagingProducerMetricMillisecond is a prometheus histogram for measuring the duration of a request.
	MessagingProducerMetricMillisecond = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: ComponentNamespace,
		Subsystem: "messaging_producer_requests",
		Name:      "duration_ms",
		Help:      "requests duration(ms).",
		Buckets:   []float64{0.1, 0.5, 1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
	}, []string{"kind", "addr", "destination"})

	// MessagingProducerMetricRequests  is a counter vector of requests.
	MessagingProducerMetricRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: ComponentNamespace,
		Subsystem: "messaging_producer_requests",
		Name:      "total",
		Help:      "The total number of processed requests",
	}, []string{"kind", "addr", "destination", "status"})

	// MessagingConsumerMetricMillisecond is a prometheus histogram for measuring the duration of a request.
	MessagingConsumerMetricMillisecond = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: ComponentNamespace,
		Subsystem: "messaging_consumer_requests",
		Name:      "duration_ms",
		Help:      "requests duration(ms).",
		Buckets:   []float64{0.1, 0.5, 1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
	}, []string{"kind", "addr", "destination", "queue"})

	// MessagingConsumerMetricRequests  is a counter vector of requests.
	MessagingConsumerMetricRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: ComponentNamespace,
		Subsystem: "messaging_consumer_requests",
		Name:      "total",
		Help:      "The total number of processed requests",
	}, []string{"kind", "addr", "destination", "queue", "status"})
)

func init() {
	prometheus.MustRegister(
		MetricRateLimitTotal,
		ClientMetricMillisecond, ClientMetricRequests, // client metrics
		ServerMetricMillisecond, ServerMetricRequests, // server metrics
		DBSystemMetricMillisecond, DBSystemMetricRequests, // db client metrics
		MessagingProducerMetricMillisecond, MessagingProducerMetricRequests, // messaging producer
		MessagingConsumerMetricMillisecond, MessagingConsumerMetricRequests, // messaging consumer
		BuildInfoGauge, DBSystemStatsGauge,
	)
}
