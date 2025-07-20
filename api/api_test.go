package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/aptly-dev/aptly/aptly"
	ctx "github.com/aptly-dev/aptly/context"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/task"
	"github.com/gin-gonic/gin"

	"github.com/smira/flag"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

type APISuite struct {
	context    *ctx.AptlyContext
	flags      *flag.FlagSet
	configFile *os.File
	router     http.Handler
}

var _ = Suite(&APISuite{})

func createTestConfig() *os.File {
	file, err := os.CreateTemp("", "aptly")
	if err != nil {
		return nil
	}
	jsonString, err := json.Marshal(gin.H{
		"architectures":         []string{},
		"enableMetricsEndpoint": true,
	})
	if err != nil {
		return nil
	}
	_, _ = file.Write(jsonString)
	return file
}

func (s *APISuite) setupContext() error {
	aptly.Version = "testVersion"
	file := createTestConfig()
	if nil == file {
		return fmt.Errorf("unable to create the test configuration file")
	}
	s.configFile = file

	flags := flag.NewFlagSet("fakeFlags", flag.ContinueOnError)
	flags.Bool("no-lock", false, "dummy")
	flags.Int("db-open-attempts", 3, "dummy")
	flags.String("config", s.configFile.Name(), "dummy")
	flags.String("architectures", "", "dummy")
	s.flags = flags

	context, err := ctx.NewContext(s.flags)
	if nil != err {
		return err
	}

	s.context = context
	s.router = Router(context)

	return nil
}

func (s *APISuite) SetUpSuite(c *C) {
	err := s.setupContext()
	c.Assert(err, IsNil)
}

func (s *APISuite) TearDownSuite(c *C) {
	_ = os.Remove(s.configFile.Name())
	s.context.Shutdown()
}

func (s *APISuite) SetUpTest(c *C) {
}

func (s *APISuite) TearDownTest(c *C) {
}

func (s *APISuite) HTTPRequest(method string, url string, body io.Reader) (*httptest.ResponseRecorder, error) {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	s.router.ServeHTTP(w, req)
	return w, nil
}

func (s *APISuite) TestGinRunsInReleaseMode(c *C) {
	c.Check(gin.Mode(), Equals, gin.ReleaseMode)
}

func (s *APISuite) TestGetVersion(c *C) {
	response, err := s.HTTPRequest("GET", "/api/version", nil)
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 200)
	c.Check(response.Body.String(), Matches, "{\"Version\":\""+aptly.Version+"\"}")
}

func (s *APISuite) TestGetReadiness(c *C) {
	response, err := s.HTTPRequest("GET", "/api/ready", nil)
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 200)
	c.Check(response.Body.String(), Matches, "{\"Status\":\"Aptly is ready\"}")
}

func (s *APISuite) TestGetHealthiness(c *C) {
	response, err := s.HTTPRequest("GET", "/api/healthy", nil)
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 200)
	c.Check(response.Body.String(), Matches, "{\"Status\":\"Aptly is healthy\"}")
}

func (s *APISuite) TestGetMetrics(c *C) {
	response, err := s.HTTPRequest("GET", "/api/metrics", nil)
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 200)
	b := strings.Replace(response.Body.String(), "\n", "", -1)
	c.Check(b, Matches, ".*# TYPE aptly_api_http_requests_in_flight gauge.*")
	c.Check(b, Matches, ".*# TYPE aptly_api_http_requests_total counter.*")
	c.Check(b, Matches, ".*# TYPE aptly_api_http_request_size_bytes summary.*")
	c.Check(b, Matches, ".*# TYPE aptly_api_http_response_size_bytes summary.*")
	c.Check(b, Matches, ".*# TYPE aptly_api_http_request_duration_seconds summary.*")
	c.Check(b, Matches, ".*# TYPE aptly_build_info gauge.*")
	c.Check(b, Matches, ".*aptly_build_info.*version=\"testVersion\".*")
}

func (s *APISuite) TestRepoCreate(c *C) {
	body, err := json.Marshal(gin.H{
		"Name": "dummy",
	})
	c.Assert(err, IsNil)
	resp, err := s.HTTPRequest("POST", "/api/repos", bytes.NewReader(body))
	c.Assert(err, IsNil)
	c.Check(resp.Code, Equals, 201)
	
	// Clean up: delete the created repo
	resp, err = s.HTTPRequest("DELETE", "/api/repos/dummy?force=1", nil)
	c.Assert(err, IsNil)
	c.Check(resp.Code, Equals, 200)
}

func (s *APISuite) TestTruthy(c *C) {
	c.Check(truthy("no"), Equals, false)
	c.Check(truthy("n"), Equals, false)
	c.Check(truthy("off"), Equals, false)
	c.Check(truthy("false"), Equals, false)
	c.Check(truthy("0"), Equals, false)
	c.Check(truthy(false), Equals, false)
	c.Check(truthy(0), Equals, false)

	c.Check(truthy("y"), Equals, true)
	c.Check(truthy("yes"), Equals, true)
	c.Check(truthy("t"), Equals, true)
	c.Check(truthy("true"), Equals, true)
	c.Check(truthy("1"), Equals, true)
	c.Check(truthy(true), Equals, true)
	c.Check(truthy(1), Equals, true)

	c.Check(truthy(nil), Equals, false)

	c.Check(truthy("foobar"), Equals, true)
	c.Check(truthy(-1), Equals, true)
	c.Check(truthy(gin.H{}), Equals, true)
}

func (s *APISuite) TestDatabaseConnectionFunctions(c *C) {
	// Test acquire and release database connection
	err := acquireDatabaseConnection()
	c.Check(err, IsNil)
	
	err = releaseDatabaseConnection()
	c.Check(err, IsNil)
}

func (s *APISuite) TestConcurrentDatabaseRequests(c *C) {
	// Test concurrent database acquisition
	done := make(chan bool, 5)
	
	for i := 0; i < 5; i++ {
		go func() {
			defer func() { done <- true }()
			
			err := acquireDatabaseConnection()
			if err == nil {
				_ = releaseDatabaseConnection()
			}
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
	
	c.Check(true, Equals, true) // If we get here, no deadlock occurred
}

func (s *APISuite) TestMaybeRunTaskInBackground(c *C) {
	// Test synchronous task execution
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest("GET", "/api/test", nil)
	
	called := false
	maybeRunTaskInBackground(ginCtx, "test-task", []string{}, func(out aptly.Progress, detail *task.Detail) (*task.ProcessReturnValue, error) {
		called = true
		return &task.ProcessReturnValue{Code: 200, Value: gin.H{"status": "ok"}}, nil
	})
	
	c.Check(called, Equals, true)
	c.Check(w.Code, Equals, 200)
}

func (s *APISuite) TestMaybeRunTaskInBackgroundAsync(c *C) {
	// Test asynchronous task execution
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest("GET", "/api/test?_async=true", nil)
	
	maybeRunTaskInBackground(ginCtx, "test-async-task", []string{}, func(out aptly.Progress, detail *task.Detail) (*task.ProcessReturnValue, error) {
		return &task.ProcessReturnValue{Code: 200, Value: gin.H{"status": "ok"}}, nil
	})
	
	// For async, should return 202 Accepted
	c.Check(w.Code, Equals, 202)
}

func (s *APISuite) TestAbortWithJSONError(c *C) {
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	
	testErr := fmt.Errorf("test error message")
	AbortWithJSONError(ginCtx, 400, testErr)
	
	c.Check(w.Code, Equals, 400)
	c.Check(w.Header().Get("Content-Type"), Equals, "application/json; charset=utf-8")
}

func (s *APISuite) TestShowPackagesWithNilList(c *C) {
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest("GET", "/api/test", nil)
	
	showPackages(ginCtx, nil, s.context.NewCollectionFactory())
	
	// Should return error when reflist is nil
	c.Check(w.Code, Equals, 404)
}

func (s *APISuite) TestAPIVersionConstant(c *C) {
	// Test that apiVersion struct is properly defined
	version := aptlyVersion{Version: "test-version"}
	c.Check(version.Version, Equals, "test-version")
}

func (s *APISuite) TestAPIStatusConstant(c *C) {
	// Test that aptlyStatus struct is properly defined
	status := aptlyStatus{Status: "test-status"}
	c.Check(status.Status, Equals, "test-status")
}

func (s *APISuite) TestRunTaskInBackground(c *C) {
	// Test running task in background
	task, err := runTaskInBackground("background-test", []string{}, func(out aptly.Progress, detail *task.Detail) (*task.ProcessReturnValue, error) {
		return &task.ProcessReturnValue{Code: 200, Value: gin.H{"done": true}}, nil
	})
	
	c.Check(err, IsNil)
	c.Check(task, NotNil)
	c.Check(task.Name, Equals, "background-test")
	
	// Wait for task to complete
	_, _ = s.context.TaskList().WaitForTaskByID(task.ID)
	
	// Clean up
	_, _ = s.context.TaskList().DeleteTaskByID(task.ID)
}

func (s *APISuite) TestInitDBRequests(c *C) {
	// Test that initDBRequests can be called multiple times safely
	initDBRequests()
	initDBRequests() // Should not panic
	
	c.Check(dbRequests, NotNil)
}

func (s *APISuite) TestShowPackagesWithQuery(c *C) {
	// Create a test gin context
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest("GET", "/api/test?q=Name&format=details", nil)
	
	// Create empty reflist
	reflist := deb.NewPackageRefList()
	
	showPackages(ginCtx, reflist, s.context.NewCollectionFactory())
	
	// Should succeed with empty list
	c.Check(w.Code, Equals, 200)
	
	var result []*deb.Package
	err := json.Unmarshal(w.Body.Bytes(), &result)
	c.Check(err, IsNil)
	c.Check(len(result), Equals, 0)
}

func (s *APISuite) TestShowPackagesCompactFormat(c *C) {
	// Test compact format (default)
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest("GET", "/api/test", nil)
	
	reflist := deb.NewPackageRefList()
	showPackages(ginCtx, reflist, s.context.NewCollectionFactory())
	
	c.Check(w.Code, Equals, 200)
	
	var result []string
	err := json.Unmarshal(w.Body.Bytes(), &result)
	c.Check(err, IsNil)
	c.Check(len(result), Equals, 0)
}

func (s *APISuite) TestTruthyEdgeCases(c *C) {
	// Test edge cases for truthy function
	c.Check(truthy("F"), Equals, false) // capital F
	c.Check(truthy("FALSE"), Equals, false) // all caps
	c.Check(truthy("False"), Equals, false) // mixed case
	c.Check(truthy("NO"), Equals, false) // capital NO
	c.Check(truthy("Off"), Equals, false) // mixed case off
	
	// Test empty string
	c.Check(truthy(""), Equals, true) // empty string is truthy
	
	// Test other types
	c.Check(truthy(struct{}{}), Equals, true) // empty struct
	c.Check(truthy([]int{}), Equals, true) // empty slice
	c.Check(truthy(map[string]int{}), Equals, true) // empty map
}
