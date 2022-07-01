package api

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// Only use base path as label value (e.g.: /api/repos) because of time series cardinality
// See https://prometheus.io/docs/practices/naming/#labels
func getBasePath(c *gin.Context) string {
	return fmt.Sprintf("%s%s", getURLSegment(c.Request.URL.Path, 0), getURLSegment(c.Request.URL.Path, 1))
}

func getURLSegment(url string, idx int) string {
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
