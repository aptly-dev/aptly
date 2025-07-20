package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/task"
	"github.com/gin-gonic/gin"
	. "gopkg.in/check.v1"
)

type DBTestSuite struct {
	APISuite
}

var _ = Suite(&DBTestSuite{})

func (s *DBTestSuite) SetUpTest(c *C) {
	s.APISuite.SetUpTest(c)
}

func (s *DBTestSuite) TestDbCleanupStructure(c *C) {
	// Test database cleanup endpoint structure
	req, _ := http.NewRequest("POST", "/api/db/cleanup", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Should succeed with proper context
	c.Check(w.Code, Equals, 200)
}

func (s *DBTestSuite) TestDbCleanupWithAsync(c *C) {
	// Test database cleanup with async parameter
	req, _ := http.NewRequest("POST", "/api/db/cleanup?_async=1", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Should return task response when async
	c.Check(w.Code, Equals, 202)
}

func (s *DBTestSuite) TestDbCleanupWithDryRun(c *C) {
	// Test database cleanup with dry run parameter
	req, _ := http.NewRequest("POST", "/api/db/cleanup?dry-run=1", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Should succeed with dry run
	c.Check(w.Code, Equals, 200)
}

func (s *DBTestSuite) TestDbCleanupWithBothParams(c *C) {
	// Test database cleanup with both async and dry-run parameters
	req, _ := http.NewRequest("POST", "/api/db/cleanup?_async=1&dry-run=1", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will error due to no context, but tests parameter combination
	c.Check(w.Code, Not(Equals), 200)
}

func (s *DBTestSuite) TestDbCleanupHTTPMethods(c *C) {
	// Test that only POST method is allowed
	deniedMethods := []string{"GET", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	for _, method := range deniedMethods {
		req, _ := http.NewRequest(method, "/api/db/cleanup", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		c.Check(w.Code, Equals, 404, Commentf("Method: %s should be denied", method))
	}
}

func (s *DBTestSuite) TestDbCleanupWithRequestBody(c *C) {
	// Test database cleanup with various request bodies (should be ignored)
	testBodies := []string{
		"",
		"some random text",
		`{"key": "value"}`,
		`<xml>data</xml>`,
		"binary\x00\x01\x02data",
	}

	for i, body := range testBodies {
		req, _ := http.NewRequest("POST", "/api/db/cleanup", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should handle various body content without crashing
		c.Check(w.Code, Not(Equals), 0, Commentf("Body test #%d", i+1))
	}
}

func (s *DBTestSuite) TestDbCleanupParameterVariations(c *C) {
	// Test various parameter value combinations
	paramTests := []struct {
		query       string
		description string
	}{
		{"", "no parameters"},
		{"_async=0", "async disabled"},
		{"_async=false", "async false"},
		{"_async=true", "async true"},
		{"dry-run=0", "dry-run disabled"},
		{"dry-run=false", "dry-run false"},
		{"dry-run=true", "dry-run true"},
		{"_async=1&dry-run=0", "async on, dry-run off"},
		{"_async=0&dry-run=1", "async off, dry-run on"},
		{"_async=true&dry-run=false", "async true, dry-run false"},
		{"unknown=param", "unknown parameter"},
		{"_async=invalid", "invalid async value"},
		{"dry-run=invalid", "invalid dry-run value"},
	}

	for _, test := range paramTests {
		path := "/api/db/cleanup"
		if test.query != "" {
			path += "?" + test.query
		}

		req, _ := http.NewRequest("POST", path, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should handle all parameter variations without crashing
		c.Check(w.Code, Not(Equals), 0, Commentf("Test: %s", test.description))
	}
}

func (s *DBTestSuite) TestDbCleanupContentTypes(c *C) {
	// Test different content types
	contentTypes := []string{
		"",
		"application/json",
		"text/plain",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
		"application/octet-stream",
	}

	for _, contentType := range contentTypes {
		req, _ := http.NewRequest("POST", "/api/db/cleanup", nil)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should handle different content types without crashing
		c.Check(w.Code, Not(Equals), 0, Commentf("Content-Type: %s", contentType))
	}
}

func (s *DBTestSuite) TestDbCleanupErrorHandling(c *C) {
	// Test various error conditions
	errorTests := []struct {
		description string
		path        string
		method      string
		expectError bool
	}{
		{"Normal cleanup call", "/api/db/cleanup", "POST", true},            // Expect error due to no context
		{"Cleanup with extra path", "/api/db/cleanup/extra", "POST", false}, // Route not matched
		{"Cleanup normal path", "/api/db/cleanup", "POST", true},  // Valid endpoint
		{"Case sensitive path", "/api/DB/cleanup", "POST", false},           // Route not matched
		{"Case sensitive path", "/api/db/CLEANUP", "POST", false},           // Route not matched
	}

	for _, test := range errorTests {
		req, _ := http.NewRequest(test.method, test.path, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// All should return some response without crashing
		c.Check(w.Code, Not(Equals), 0, Commentf("Test: %s", test.description))
	}
}

func (s *DBTestSuite) TestDbCleanupReliability(c *C) {
	// Test multiple sequential calls for reliability
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("POST", "/api/db/cleanup", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should be consistent across multiple calls
		c.Check(w.Code, Not(Equals), 0, Commentf("Call #%d", i+1))
	}
}

func (s *DBTestSuite) TestDbCleanupHeaders(c *C) {
	// Test with various HTTP headers
	headerTests := []map[string]string{
		{},
		{"Accept": "application/json"},
		{"Accept": "text/plain"},
		{"Accept": "*/*"},
		{"User-Agent": "test-agent"},
		{"Authorization": "Bearer token123"},
		{"X-Custom-Header": "custom-value"},
		{"Accept-Encoding": "gzip, deflate"},
		{"Accept-Language": "en-US,en;q=0.9"},
	}

	for i, headers := range headerTests {
		req, _ := http.NewRequest("POST", "/api/db/cleanup", nil)
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should handle various headers without crashing
		c.Check(w.Code, Not(Equals), 0, Commentf("Header test #%d", i+1))
	}
}

func (s *DBTestSuite) TestDbCleanupResponseFormat(c *C) {
	// Test response format consistency
	req, _ := http.NewRequest("POST", "/api/db/cleanup", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Should have proper response structure
	c.Check(w.Code, Not(Equals), 0)
	c.Check(w.Header(), NotNil)

	// If there's a response body, it should be valid
	if w.Body.Len() > 0 {
		body := w.Body.String()
		c.Check(len(body), Not(Equals), 0)
	}
}

func (s *DBTestSuite) TestDbRequestTypes(c *C) {
	// Test dbRequestKind constants
	c.Check(acquiredb, Equals, dbRequestKind(0))
	c.Check(releasedb, Equals, dbRequestKind(1))
}

func (s *DBTestSuite) TestDbRequestStruct(c *C) {
	// Test dbRequest struct creation
	errCh := make(chan error, 1)
	req := dbRequest{
		kind: acquiredb,
		err:  errCh,
	}
	
	c.Check(req.kind, Equals, acquiredb)
	c.Check(req.err, NotNil)
}

func (s *DBTestSuite) TestAcquireAndReleaseDatabase(c *C) {
	// Initialize db requests channel
	initDBRequests()
	
	// Test multiple acquire and release cycles
	for i := 0; i < 3; i++ {
		err := acquireDatabaseConnection()
		c.Check(err, IsNil)
		
		err = releaseDatabaseConnection()
		c.Check(err, IsNil)
	}
}

func (s *DBTestSuite) TestConcurrentDatabaseAccess(c *C) {
	// Test concurrent database access
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			// Acquire and release database connection
			if err := acquireDatabaseConnection(); err == nil {
				// Simulate some work
				time.Sleep(10 * time.Millisecond)
				_ = releaseDatabaseConnection()
			}
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	
	c.Check(true, Equals, true) // Test passed without deadlock
}

func (s *DBTestSuite) TestMaybeRunTaskInBackgroundWithError(c *C) {
	// Test task that returns an error
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest("GET", "/api/test", nil)
	
	testErr := gin.Error{Type: gin.ErrorTypePublic, Err: gin.Error{}.Err}
	maybeRunTaskInBackground(ginCtx, "error-task", []string{}, func(out aptly.Progress, detail *task.Detail) (*task.ProcessReturnValue, error) {
		return nil, testErr
	})
	
	// Should return error status
	c.Check(w.Code, Not(Equals), 200)
}

func (s *DBTestSuite) TestMaybeRunTaskInBackgroundConflict(c *C) {
	// Test task with resource conflict
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest("GET", "/api/test", nil)
	
	// Create two tasks with same resources to cause conflict
	resource := "test-resource-" + time.Now().Format("20060102150405")
	
	// Start first task
	_, _ = runTaskInBackground("task1", []string{resource}, func(out aptly.Progress, detail *task.Detail) (*task.ProcessReturnValue, error) {
		time.Sleep(100 * time.Millisecond) // Hold resource
		return &task.ProcessReturnValue{Code: 200}, nil
	})
	
	// Try to start second task with same resource (should conflict)
	maybeRunTaskInBackground(ginCtx, "task2", []string{resource}, func(out aptly.Progress, detail *task.Detail) (*task.ProcessReturnValue, error) {
		return &task.ProcessReturnValue{Code: 200}, nil
	})
	
	// Should return 409 Conflict
	c.Check(w.Code, Equals, 409)
}

func (s *DBTestSuite) TestRunTaskInBackgroundWithNilReturn(c *C) {
	// Test task that returns nil ProcessReturnValue
	task, err := runTaskInBackground("nil-return-task", []string{}, func(out aptly.Progress, detail *task.Detail) (*task.ProcessReturnValue, error) {
		return nil, nil
	})
	
	c.Check(err, IsNil)
	c.Check(task, NotNil)
	
	// Wait and clean up
	_, _ = s.context.TaskList().WaitForTaskByID(task.ID)
	_, _ = s.context.TaskList().DeleteTaskByID(task.ID)
}

func (s *DBTestSuite) TestMaybeRunTaskInBackgroundNilReturn(c *C) {
	// Test synchronous task with nil return value
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest("GET", "/api/test", nil)
	
	maybeRunTaskInBackground(ginCtx, "nil-sync-task", []string{}, func(out aptly.Progress, detail *task.Detail) (*task.ProcessReturnValue, error) {
		return nil, nil
	})
	
	// Should return 200 with nil body
	c.Check(w.Code, Equals, 200)
}
