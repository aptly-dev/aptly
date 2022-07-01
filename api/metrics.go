package api

import (
	"runtime"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	apiRequestsInFlightGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aptly_api_http_requests_in_flight",
			Help: "Number of concurrent HTTP api requests currently handled.",
		},
		[]string{"method", "path"},
	)
	apiRequestsTotalCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aptly_api_http_requests_total",
			Help: "Total number of api requests.",
		},
		[]string{"code", "method", "path"},
	)
	apiRequestSizeSummary = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "aptly_api_http_request_size_bytes",
			Help: "Api HTTP request size in bytes.",
		},
		[]string{"code", "method", "path"},
	)
	apiResponseSizeSummary = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "aptly_api_http_response_size_bytes",
			Help: "Api HTTP response size in bytes.",
		},
		[]string{"code", "method", "path"},
	)
	apiRequestsDurationSummary = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "aptly_api_http_request_duration_seconds",
			Help: "Duration of api requests in seconds.",
		},
		[]string{"code", "method", "path"},
	)
	apiVersionGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aptly_build_info",
			Help: "Metric with a constant '1' value labeled by version and goversion from which aptly was built.",
		},
		[]string{"version", "goversion"},
	)
)

type metricsCollectorRegistrar struct {
	hasRegistered bool
}

func (r *metricsCollectorRegistrar) Register(router *gin.Engine) {
	if !r.hasRegistered {
		apiVersionGauge.WithLabelValues(aptly.Version, runtime.Version()).Set(1)
		router.Use(instrumentHandlerInFlight(apiRequestsInFlightGauge, getBasePath))
		router.Use(instrumentHandlerCounter(apiRequestsTotalCounter, getBasePath))
		router.Use(instrumentHandlerRequestSize(apiRequestSizeSummary, getBasePath))
		router.Use(instrumentHandlerResponseSize(apiResponseSizeSummary, getBasePath))
		router.Use(instrumentHandlerDuration(apiRequestsDurationSummary, getBasePath))
		r.hasRegistered = true
	}
}

var MetricsCollectorRegistrar = metricsCollectorRegistrar{hasRegistered: false}
