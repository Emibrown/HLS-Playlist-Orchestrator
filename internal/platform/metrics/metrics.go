package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds Prometheus counters and gauges for the HLS orchestrator.
type Metrics struct {
	registry                *prometheus.Registry
	requestsTotal           prometheus.Counter
	segmentsRegisteredTotal prometheus.Counter
	streamsEndedTotal       prometheus.Counter
	activeStreams           prometheus.Gauge
	errorsTotal             prometheus.Counter
}

// New creates and registers Prometheus metrics for the orchestrator.
func New() *Metrics {
	registry := prometheus.NewRegistry()

	requestsTotal := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "hls_requests_total",
		Help: "Total number of HTTP requests received",
	})
	segmentsRegisteredTotal := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "hls_segments_registered_total",
		Help: "Total number of segments successfully registered",
	})
	streamsEndedTotal := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "hls_streams_ended_total",
		Help: "Total number of streams ended",
	})
	activeStreams := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "hls_active_streams",
		Help: "Number of streams that are not ended",
	})
	errorsTotal := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "hls_errors_total",
		Help: "Total number of HTTP responses with error status (4xx or 5xx)",
	})

	registry.MustRegister(
		requestsTotal,
		segmentsRegisteredTotal,
		streamsEndedTotal,
		activeStreams,
		errorsTotal,
	)

	return &Metrics{
		registry:                registry,
		requestsTotal:           requestsTotal,
		segmentsRegisteredTotal: segmentsRegisteredTotal,
		streamsEndedTotal:       streamsEndedTotal,
		activeStreams:           activeStreams,
		errorsTotal:             errorsTotal,
	}
}

// IncRequests increments the total request counter.
func (m *Metrics) IncRequests() {
	m.requestsTotal.Inc()
}

// IncSegmentsRegistered increments the segments registered counter.
func (m *Metrics) IncSegmentsRegistered() {
	m.segmentsRegisteredTotal.Inc()
}

// IncStreamsEnded increments the streams ended counter.
func (m *Metrics) IncStreamsEnded() {
	m.streamsEndedTotal.Inc()
}

// SetActiveStreams sets the active streams gauge.
func (m *Metrics) SetActiveStreams(n int) {
	m.activeStreams.Set(float64(n))
}

// IncErrors increments the errors counter.
func (m *Metrics) IncErrors() {
	m.errorsTotal.Inc()
}

// Handler returns an http.Handler that serves Prometheus metrics.
// updateGauges is called before each scrape to refresh gauge values (e.g. active streams).
func (m *Metrics) Handler(updateGauges func()) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if updateGauges != nil {
			updateGauges()
		}
		promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	})
}
