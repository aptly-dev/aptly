package api

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/aptly-dev/aptly/utils"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// Only use base path as label value (e.g.: /api/repos) because of time series cardinality
// See https://prometheus.io/docs/practices/naming/#labels
func getBasePath(c *gin.Context) string {
	segment0, err := getURLSegment(c.Request.URL.Path, 0)
	if err != nil {
		return "/"
	}
	segment1, err := getURLSegment(c.Request.URL.Path, 1)
	if err != nil {
		return *segment0
	}

	return *segment0 + *segment1
}

func getURLSegment(url string, idx int) (*string, error) {
	urlSegments := strings.Split(url, "/")
	// Remove segment at index 0 because it's an empty string
	urlSegments = urlSegments[1:cap(urlSegments)]

	if len(urlSegments) <= idx {
		return nil, fmt.Errorf("index %d out of range, only has %d url segments", idx, len(urlSegments))
	}

	segmentAtIndex := urlSegments[idx]
	s := fmt.Sprintf("/%s", segmentAtIndex)
	return &s, nil
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

// LoggerMiddlewareFactory returns a logging middleware
func LoggerMiddlewareFactory(logger utils.Logger) gin.HandlerFunc {
	switch logger.(type) {
	case *utils.PlainLogger:
		return gin.Logger()
	case *utils.ZeroJSONLogger:
		gin.DefaultWriter = logger.Writer("debug")
		return JSONLogger(logger)
	}

	return gin.Logger()
}

// JSONLogger is a gin middleware that takes an instance of Logger and uses it for writing access
// logs that include error messages if there are any.
func JSONLogger(logger utils.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		ts := time.Now()
		if raw != "" {
			path = path + "?" + raw
		}

		errorMessage := strings.TrimSuffix(c.Errors.ByType(gin.ErrorTypePrivate).String(), "\n")
		l := logger.WithField("remote", c.ClientIP()).
			WithField("method", c.Request.Method).
			WithField("path", path).
			WithField("protocol", c.Request.Proto).
			WithField("code", fmt.Sprint(c.Writer.Status())).
			WithField("latency", ts.Sub(start).String()).
			WithField("agent", c.Request.UserAgent())

		if c.Writer.Status() >= 400 && c.Writer.Status() < 500 {
			l.Warn(errorMessage)
		} else if c.Writer.Status() >= 500 {
			l.Error(errorMessage)
		} else {
			l.Info(errorMessage)
		}
	}
}
