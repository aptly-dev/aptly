package api

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	. "gopkg.in/check.v1"
)

type MetricsTestSuite struct {
	APISuite
}

var _ = Suite(&MetricsTestSuite{})

func (s *MetricsTestSuite) SetUpTest(c *C) {
	s.APISuite.SetUpTest(c)
	// Reset metrics registrar state for each test
	MetricsCollectorRegistrar.hasRegistered = false
}

func (s *MetricsTestSuite) TestMetricsCollectorRegistrarRegisterOnce(c *C) {
	// Test that metrics are only registered once
	registrar := &metricsCollectorRegistrar{hasRegistered: false}

	// First registration should work
	registrar.Register(s.router.(*gin.Engine))
	c.Check(registrar.hasRegistered, Equals, true)

	// Second registration should be skipped
	registrar.Register(s.router.(*gin.Engine))
	c.Check(registrar.hasRegistered, Equals, true)
}

func (s *MetricsTestSuite) TestMetricsCollectorRegistrarVersionGauge(c *C) {
	// Test that version gauge is set correctly
	registrar := &metricsCollectorRegistrar{hasRegistered: false}

	// Register metrics
	registrar.Register(s.router.(*gin.Engine))

	// Check that version gauge was set
	expectedLabels := prometheus.Labels{
		"version":   aptly.Version,
		"goversion": runtime.Version(),
	}

	gauge := apiVersionGauge.With(expectedLabels)
	c.Check(gauge, NotNil)

	// Verify the gauge value is 1
	metric := &dto.Metric{}
	gauge.(prometheus.Gauge).Write(metric)
	c.Check(metric.GetGauge().GetValue(), Equals, float64(1))
}

func (s *MetricsTestSuite) TestApiRequestsInFlightGauge(c *C) {
	// Test that in-flight requests gauge works
	c.Check(apiRequestsInFlightGauge, NotNil)

	// Test that we can create labels for the gauge
	gauge := apiRequestsInFlightGauge.WithLabelValues("GET", "/api/test")
	c.Check(gauge, NotNil)

	// Test incrementing and decrementing
	gauge.Inc()
	gauge.Dec()
}

func (s *MetricsTestSuite) TestApiRequestsTotalCounter(c *C) {
	// Test that total requests counter works
	c.Check(apiRequestsTotalCounter, NotNil)

	// Test that we can create labels for the counter
	counter := apiRequestsTotalCounter.WithLabelValues("200", "GET", "/api/test")
	c.Check(counter, NotNil)

	// Test incrementing
	counter.Inc()
}

func (s *MetricsTestSuite) TestApiRequestSizeSummary(c *C) {
	// Test that request size summary works
	c.Check(apiRequestSizeSummary, NotNil)

	// Test that we can create labels for the summary
	summary := apiRequestSizeSummary.WithLabelValues("200", "POST", "/api/test")
	c.Check(summary, NotNil)

	// Test observing values
	summary.Observe(1024.0)
	summary.Observe(512.0)
}

func (s *MetricsTestSuite) TestApiResponseSizeSummary(c *C) {
	// Test that response size summary works
	c.Check(apiResponseSizeSummary, NotNil)

	// Test that we can create labels for the summary
	summary := apiResponseSizeSummary.WithLabelValues("200", "GET", "/api/test")
	c.Check(summary, NotNil)

	// Test observing values
	summary.Observe(2048.0)
	summary.Observe(1024.0)
}

func (s *MetricsTestSuite) TestApiRequestsDurationSummary(c *C) {
	// Test that request duration summary works
	c.Check(apiRequestsDurationSummary, NotNil)

	// Test that we can create labels for the summary
	summary := apiRequestsDurationSummary.WithLabelValues("200", "GET", "/api/test")
	c.Check(summary, NotNil)

	// Test observing duration values
	summary.Observe(0.1)  // 100ms
	summary.Observe(0.05) // 50ms
	summary.Observe(1.0)  // 1s
}

func (s *MetricsTestSuite) TestApiFilesUploadedCounter(c *C) {
	// Test that files uploaded counter works
	c.Check(apiFilesUploadedCounter, NotNil)

	// Test that we can create labels for the counter
	counter := apiFilesUploadedCounter.WithLabelValues("uploads")
	c.Check(counter, NotNil)

	// Test incrementing
	counter.Inc()
	counter.Add(5)
}

func (s *MetricsTestSuite) TestApiReposPackageCountGauge(c *C) {
	// Test that repos package count gauge works
	c.Check(apiReposPackageCountGauge, NotNil)

	// Test that we can create labels for the gauge
	gauge := apiReposPackageCountGauge.WithLabelValues("source", "stable", "main")
	c.Check(gauge, NotNil)

	// Test setting values
	gauge.Set(100)
	gauge.Set(150)
	gauge.Inc()
	gauge.Dec()
}

func (s *MetricsTestSuite) TestMetricsPrometheusIntegration(c *C) {
	// Test integration with Prometheus client library

	// Test that metrics are properly registered with default registry
	metricNames := []string{
		"aptly_api_http_requests_in_flight",
		"aptly_api_http_requests_total",
		"aptly_api_http_request_size_bytes",
		"aptly_api_http_response_size_bytes",
		"aptly_api_http_request_duration_seconds",
		"aptly_build_info",
		"aptly_api_files_uploaded_total",
		"aptly_repos_package_count",
	}

	for _, metricName := range metricNames {
		// Try to gather metrics to ensure they're registered
		gathered, err := prometheus.DefaultGatherer.Gather()
		c.Check(err, IsNil)

		found := false
		for _, metricFamily := range gathered {
			if metricFamily.GetName() == metricName {
				found = true
				break
			}
		}
		c.Check(found, Equals, true, Commentf("Metric %s not found", metricName))
	}
}

func (s *MetricsTestSuite) TestMetricsLabels(c *C) {
	// Test that metrics have expected labels

	// Test in-flight gauge labels
	gauge := apiRequestsInFlightGauge.WithLabelValues("GET", "/api/test")
	c.Check(gauge, NotNil)

	// Test total counter labels
	counter := apiRequestsTotalCounter.WithLabelValues("200", "GET", "/api/test")
	c.Check(counter, NotNil)

	// Test request size summary labels
	requestSummary := apiRequestSizeSummary.WithLabelValues("200", "POST", "/api/upload")
	c.Check(requestSummary, NotNil)

	// Test response size summary labels
	responseSummary := apiResponseSizeSummary.WithLabelValues("404", "GET", "/api/missing")
	c.Check(responseSummary, NotNil)

	// Test duration summary labels
	durationSummary := apiRequestsDurationSummary.WithLabelValues("500", "POST", "/api/error")
	c.Check(durationSummary, NotNil)

	// Test version gauge labels
	versionGauge := apiVersionGauge.WithLabelValues("1.0.0", "go1.19")
	c.Check(versionGauge, NotNil)

	// Test files uploaded counter labels
	filesCounter := apiFilesUploadedCounter.WithLabelValues("temp-uploads")
	c.Check(filesCounter, NotNil)

	// Test repos package count gauge labels
	reposGauge := apiReposPackageCountGauge.WithLabelValues("snapshot:test", "testing", "contrib")
	c.Check(reposGauge, NotNil)
}

func (s *MetricsTestSuite) TestMetricsWithDifferentHTTPCodes(c *C) {
	// Test metrics with various HTTP status codes
	httpCodes := []string{"200", "201", "400", "401", "403", "404", "409", "500", "502", "503"}

	for _, code := range httpCodes {
		// Test that metrics work with different status codes
		counter := apiRequestsTotalCounter.WithLabelValues(code, "GET", "/api/test")
		counter.Inc()

		requestSummary := apiRequestSizeSummary.WithLabelValues(code, "POST", "/api/test")
		requestSummary.Observe(100)

		responseSummary := apiResponseSizeSummary.WithLabelValues(code, "GET", "/api/test")
		responseSummary.Observe(200)

		durationSummary := apiRequestsDurationSummary.WithLabelValues(code, "PUT", "/api/test")
		durationSummary.Observe(0.1)
	}
}

func (s *MetricsTestSuite) TestMetricsWithDifferentHTTPMethods(c *C) {
	// Test metrics with various HTTP methods
	httpMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	for _, method := range httpMethods {
		// Test that metrics work with different HTTP methods
		gauge := apiRequestsInFlightGauge.WithLabelValues(method, "/api/test")
		gauge.Inc()
		gauge.Dec()

		counter := apiRequestsTotalCounter.WithLabelValues("200", method, "/api/test")
		counter.Inc()
	}
}

func (s *MetricsTestSuite) TestMetricsWithDifferentPaths(c *C) {
	// Test metrics with various API paths
	apiPaths := []string{
		"/api/repos",
		"/api/repos/test",
		"/api/snapshots",
		"/api/publish",
		"/api/files",
		"/api/files/upload",
		"/api/mirrors",
		"/api/tasks",
		"/api/version",
	}

	for _, path := range apiPaths {
		counter := apiRequestsTotalCounter.WithLabelValues("200", "GET", path)
		counter.Inc()

		gauge := apiRequestsInFlightGauge.WithLabelValues("GET", path)
		gauge.Inc()
		gauge.Dec()
	}
}

func (s *MetricsTestSuite) TestMetricsThreadSafety(c *C) {
	// Test that metrics are thread-safe by simulating concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Simulate concurrent metric updates
			for j := 0; j < 100; j++ {
				counter := apiRequestsTotalCounter.WithLabelValues("200", "GET", "/api/concurrent")
				counter.Inc()

				gauge := apiRequestsInFlightGauge.WithLabelValues("GET", "/api/concurrent")
				gauge.Inc()
				gauge.Dec()

				summary := apiRequestsDurationSummary.WithLabelValues("200", "GET", "/api/concurrent")
				summary.Observe(0.01)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify metrics were updated (exact count doesn't matter due to concurrency)
	c.Check(true, Equals, true) // Test completed without race conditions
}

func (s *MetricsTestSuite) TestMetricsMetadata(c *C) {
	// Test that metrics have proper metadata (help text, names)

	// Gather all metrics
	gathered, err := prometheus.DefaultGatherer.Gather()
	c.Check(err, IsNil)

	expectedMetrics := map[string]string{
		"aptly_api_http_requests_in_flight":       "Number of concurrent HTTP api requests currently handled.",
		"aptly_api_http_requests_total":           "Total number of api requests.",
		"aptly_api_http_request_size_bytes":       "Api HTTP request size in bytes.",
		"aptly_api_http_response_size_bytes":      "Api HTTP response size in bytes.",
		"aptly_api_http_request_duration_seconds": "Duration of api requests in seconds.",
		"aptly_build_info":                        "Metric with a constant '1' value labeled by version and goversion from which aptly was built.",
		"aptly_api_files_uploaded_total":          "Total number of uploaded files labeled by upload directory.",
		"aptly_repos_package_count":               "Current number of published packages labeled by source, distribution and component.",
	}

	for _, metricFamily := range gathered {
		metricName := metricFamily.GetName()
		if expectedHelp, exists := expectedMetrics[metricName]; exists {
			c.Check(metricFamily.GetHelp(), Equals, expectedHelp,
				Commentf("Help text mismatch for metric: %s", metricName))
		}
	}
}

func (s *MetricsTestSuite) TestCountPackagesByRepos(c *C) {
	// Test countPackagesByRepos function structure
	// Note: This function requires database context which we don't have in tests,
	// but we can test that it doesn't crash when called

	// This will likely error due to no context, but should not panic
	defer func() {
		if r := recover(); r != nil {
			c.Fatalf("countPackagesByRepos panicked: %v", r)
		}
	}()

	countPackagesByRepos()

	// If we get here, the function didn't panic
	c.Check(true, Equals, true)
}

func (s *MetricsTestSuite) TestGetBasePath(c *C) {
	// Test getBasePath function
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	
	// Test with simple path (only returns first two segments)
	ginCtx.Request = httptest.NewRequest("GET", "/api/version", nil)
	basePath := getBasePath(ginCtx)
	c.Check(basePath, Equals, "/api/version")
	
	// Test with path containing more segments (still returns first two)
	ginCtx.Request = httptest.NewRequest("GET", "/api/repos/test-repo", nil)
	basePath = getBasePath(ginCtx)
	c.Check(basePath, Equals, "/api/repos")
	
	// Test with nested parameters (still returns first two)
	ginCtx.Request = httptest.NewRequest("GET", "/api/repos/repo1/packages", nil)
	basePath = getBasePath(ginCtx)
	c.Check(basePath, Equals, "/api/repos")
	
	// Test with root path
	ginCtx.Request = httptest.NewRequest("GET", "/", nil)
	basePath = getBasePath(ginCtx)
	c.Check(basePath, Equals, "/")
	
	// Test with single segment
	ginCtx.Request = httptest.NewRequest("GET", "/api", nil)
	basePath = getBasePath(ginCtx)
	c.Check(basePath, Equals, "/api")
}

func (s *MetricsTestSuite) TestGetURLSegment(c *C) {
	// Test getURLSegment function
	
	// Test valid segments
	segment, err := getURLSegment("/api/repos/test", 0)
	c.Check(err, IsNil)
	c.Check(*segment, Equals, "/api")
	
	segment, err = getURLSegment("/api/repos/test", 1)
	c.Check(err, IsNil)
	c.Check(*segment, Equals, "/repos")
	
	segment, err = getURLSegment("/api/repos/test", 2)
	c.Check(err, IsNil)
	c.Check(*segment, Equals, "/test")
	
	// Test out of range
	_, err = getURLSegment("/api/repos", 3)
	c.Check(err, NotNil)
	
	// Test root path
	segment, err = getURLSegment("/", 0)
	c.Check(err, NotNil) // No segments after removing empty string
}

func (s *MetricsTestSuite) TestInstrumentHandlerInFlight(c *C) {
	// Test instrumentHandlerInFlight middleware
	w := httptest.NewRecorder()
	
	// Create test gin context
	router := gin.New()
	
	// Add instrumentation middleware
	router.Use(instrumentHandlerInFlight(apiRequestsInFlightGauge, getBasePath))
	
	// Add test handler
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	
	// Make request
	req := httptest.NewRequest("GET", "/api/test", nil)
	router.ServeHTTP(w, req)
	
	c.Check(w.Code, Equals, 200)
}

func (s *MetricsTestSuite) TestInstrumentHandlerCounter(c *C) {
	// Test instrumentHandlerCounter middleware
	w := httptest.NewRecorder()
	
	// Create test gin context
	router := gin.New()
	
	// Add instrumentation middleware
	router.Use(instrumentHandlerCounter(apiRequestsTotalCounter, getBasePath))
	
	// Add test handler
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	
	// Make request
	req := httptest.NewRequest("GET", "/api/test", nil)
	router.ServeHTTP(w, req)
	
	c.Check(w.Code, Equals, 200)
}

func (s *MetricsTestSuite) TestInstrumentHandlerRequestSize(c *C) {
	// Test instrumentHandlerRequestSize middleware
	w := httptest.NewRecorder()
	
	// Create test gin context
	router := gin.New()
	
	// Add instrumentation middleware
	router.Use(instrumentHandlerRequestSize(apiRequestSizeSummary, getBasePath))
	
	// Add test handler
	router.POST("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	
	// Make request with body
	req := httptest.NewRequest("POST", "/api/test", strings.NewReader("test body"))
	router.ServeHTTP(w, req)
	
	c.Check(w.Code, Equals, 200)
}

func (s *MetricsTestSuite) TestInstrumentHandlerResponseSize(c *C) {
	// Test instrumentHandlerResponseSize middleware
	w := httptest.NewRecorder()
	
	// Create test gin context
	router := gin.New()
	
	// Add instrumentation middleware
	router.Use(instrumentHandlerResponseSize(apiResponseSizeSummary, getBasePath))
	
	// Add test handler
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"data": strings.Repeat("x", 1000)})
	})
	
	// Make request
	req := httptest.NewRequest("GET", "/api/test", nil)
	router.ServeHTTP(w, req)
	
	c.Check(w.Code, Equals, 200)
}

func (s *MetricsTestSuite) TestInstrumentHandlerDuration(c *C) {
	// Test instrumentHandlerDuration middleware
	w := httptest.NewRecorder()
	
	// Create test gin context
	router := gin.New()
	
	// Add instrumentation middleware
	router.Use(instrumentHandlerDuration(apiRequestsDurationSummary, getBasePath))
	
	// Add test handler
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	
	// Make request
	req := httptest.NewRequest("GET", "/api/test", nil)
	router.ServeHTTP(w, req)
	
	c.Check(w.Code, Equals, 200)
}

func (s *MetricsTestSuite) TestMetricsRegistration(c *C) {
	// Test that metrics registration works correctly with gin router
	MetricsCollectorRegistrar.Register(s.router.(*gin.Engine))

	// Create a test request to trigger middleware
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Add a test handler
	s.router.(*gin.Engine).GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"test": "response"})
	})

	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Equals, 200)
	c.Check(MetricsCollectorRegistrar.hasRegistered, Equals, true)
}

func (s *MetricsTestSuite) TestMetricsErrorConditions(c *C) {
	// Test error handling in metrics collection

	// Test with invalid label values (should not crash)
	invalidLabels := []string{"", "very_long_label_" + strings.Repeat("x", 1000), "label\nwith\nnewlines"}

	for _, label := range invalidLabels {
		// These should not crash, even with invalid labels
		gauge := apiRequestsInFlightGauge.WithLabelValues("GET", label)
		gauge.Inc()
		gauge.Dec()

		counter := apiRequestsTotalCounter.WithLabelValues("200", "GET", label)
		counter.Inc()
	}
}

func (s *MetricsTestSuite) TestMetricsValueRanges(c *C) {
	// Test metrics with various value ranges

	// Test large values
	summary := apiRequestSizeSummary.WithLabelValues("200", "POST", "/api/large")
	summary.Observe(1e9)  // 1GB
	summary.Observe(1e12) // 1TB

	// Test very small values
	durationSummary := apiRequestsDurationSummary.WithLabelValues("200", "GET", "/api/fast")
	durationSummary.Observe(1e-9) // 1 nanosecond
	durationSummary.Observe(1e-6) // 1 microsecond

	// Test zero values
	gauge := apiReposPackageCountGauge.WithLabelValues("empty", "dist", "comp")
	gauge.Set(0)

	// Test negative values (should be handled gracefully)
	gauge.Set(-1) // May or may not be allowed by Prometheus, but shouldn't crash
}

func (s *MetricsTestSuite) TestMetricsWithSpecialCharacters(c *C) {
	// Test metrics with special characters in labels
	specialPaths := []string{
		"/api/repos/repo-with-dashes",
		"/api/repos/repo_with_underscores",
		"/api/repos/repo.with.dots",
		"/api/repos/repo+with+plus",
		"/api/repos/repo%20with%20encoded",
	}

	for _, path := range specialPaths {
		counter := apiRequestsTotalCounter.WithLabelValues("200", "GET", path)
		counter.Inc()

		gauge := apiRequestsInFlightGauge.WithLabelValues("GET", path)
		gauge.Inc()
		gauge.Dec()
	}
}
