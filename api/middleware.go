package api

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

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
)

// Only use base path as label value (e.g.: /api/repos) because of time series cardinality
// See https://prometheus.io/docs/practices/naming/#labels
func getBasePath(c *gin.Context) string {
	return fmt.Sprintf("%s%s", getUrlSegment(c.Request.URL.Path, 0), getUrlSegment(c.Request.URL.Path, 1))
}

func getUrlSegment(url string, idx int) string {
	var urlSegments = strings.Split(url, "/")

	// Remove segment at index 0 because it's an empty string
	var segmentAtIndex = urlSegments[1:cap(urlSegments)][idx]
	return fmt.Sprintf("/%s", segmentAtIndex)
}

func instrumentHandlerInFlight(g *prometheus.GaugeVec, pathFunc func(*gin.Context) string) func(*gin.Context) {
	return func(c *gin.Context) {
		g.WithLabelValues(c.Request.Method, pathFunc(c)).Inc()
		defer g.WithLabelValues(c.Request.Method, pathFunc(c)).Dec()
		c.Next()
	}
}

func instrumentHandlerCounter(counter *prometheus.CounterVec, pathFunc func(*gin.Context) string) func(*gin.Context) {
	return func(c *gin.Context) {
		c.Next()
		counter.WithLabelValues(strconv.Itoa(c.Writer.Status()), c.Request.Method, pathFunc(c)).Inc()
	}
}

func instrumentHandlerRequestSize(obs prometheus.ObserverVec, pathFunc func(*gin.Context) string) func(*gin.Context) {
	return func(c *gin.Context) {
		c.Next()
		obs.WithLabelValues(strconv.Itoa(c.Writer.Status()), c.Request.Method, pathFunc(c)).Observe(float64(c.Request.ContentLength))
	}
}

func instrumentHandlerResponseSize(obs prometheus.ObserverVec, pathFunc func(*gin.Context) string) func(*gin.Context) {
	return func(c *gin.Context) {
		c.Next()
		var responseSize = math.Max(float64(c.Writer.Size()), 0)
		obs.WithLabelValues(strconv.Itoa(c.Writer.Status()), c.Request.Method, pathFunc(c)).Observe(responseSize)
	}
}

func instrumentHandlerDuration(obs prometheus.ObserverVec, pathFunc func(*gin.Context) string) func(*gin.Context) {
	return func(c *gin.Context) {
		now := time.Now()
		c.Next()
		obs.WithLabelValues(strconv.Itoa(c.Writer.Status()), c.Request.Method, pathFunc(c)).Observe(time.Since(now).Seconds())
	}
}
