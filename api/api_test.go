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
	"github.com/gin-gonic/gin"

	"github.com/smira/flag"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

type ApiSuite struct {
	context    *ctx.AptlyContext
	flags      *flag.FlagSet
	configFile *os.File
	router     http.Handler
}

var _ = Suite(&ApiSuite{})

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
	file.Write(jsonString)
	return file
}

func (s *ApiSuite) setupContext() error {
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

func (s *ApiSuite) SetUpSuite(c *C) {
	err := s.setupContext()
	c.Assert(err, IsNil)
}

func (s *ApiSuite) TearDownSuite(c *C) {
	os.Remove(s.configFile.Name())
	s.context.Shutdown()
}

func (s *ApiSuite) SetUpTest(c *C) {
}

func (s *ApiSuite) TearDownTest(c *C) {
}

func (s *ApiSuite) HTTPRequest(method string, url string, body io.Reader) (*httptest.ResponseRecorder, error) {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	s.router.ServeHTTP(w, req)
	return w, nil
}

func (s *ApiSuite) TestGinRunsInReleaseMode(c *C) {
	c.Check(gin.Mode(), Equals, gin.ReleaseMode)
}

func (s *ApiSuite) TestGetVersion(c *C) {
	response, err := s.HTTPRequest("GET", "/api/version", nil)
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 200)
	c.Check(response.Body.String(), Matches, "{\"Version\":\""+aptly.Version+"\"}")
}

func (s *ApiSuite) TestGetReadiness(c *C) {
	response, err := s.HTTPRequest("GET", "/api/ready", nil)
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 200)
	c.Check(response.Body.String(), Matches, "{\"Status\":\"Aptly is ready\"}")
}

func (s *ApiSuite) TestGetHealthiness(c *C) {
	response, err := s.HTTPRequest("GET", "/api/healthy", nil)
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 200)
	c.Check(response.Body.String(), Matches, "{\"Status\":\"Aptly is healthy\"}")
}

func (s *ApiSuite) TestGetMetrics(c *C) {
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

func (s *ApiSuite) TestRepoCreate(c *C) {
	body, err := json.Marshal(gin.H{
		"Name": "dummy",
	})
	c.Assert(err, IsNil)
	_, err = s.HTTPRequest("POST", "/api/repos", bytes.NewReader(body))
	c.Assert(err, IsNil)
}

func (s *ApiSuite) TestTruthy(c *C) {
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
