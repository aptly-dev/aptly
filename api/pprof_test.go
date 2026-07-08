package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/aptly-dev/aptly/aptly"
	ctx "github.com/aptly-dev/aptly/context"
	"github.com/gin-gonic/gin"
	"github.com/smira/flag"
	. "gopkg.in/check.v1"
)

// newPprofContext builds an aptly context and router from the given config file
// path. It returns an error if context initialisation fails. The caller is
// responsible for calling aptlyCtx.Shutdown() when done.
func newPprofContext(configPath string) (*ctx.AptlyContext, http.Handler, error) {
	flags := flag.NewFlagSet("fakeFlags", flag.ContinueOnError)
	flags.Bool("no-lock", false, "dummy")
	flags.Int("db-open-attempts", 3, "dummy")
	flags.String("config", configPath, "dummy")
	flags.String("architectures", "", "dummy")

	aptlyCtx, err := ctx.NewContext(flags)
	if err != nil {
		return nil, nil, err
	}
	return aptlyCtx, Router(aptlyCtx), nil
}

// PprofEnabledSuite tests pprof endpoints when EnablePprofEndpoint is true.
// It uses a dedicated router built from a config that enables only the pprof
// endpoint, ensuring no interaction with other optional features.
type PprofEnabledSuite struct {
	configFile *os.File
	aptlyCtx   *ctx.AptlyContext
	router     http.Handler
}

var _ = Suite(&PprofEnabledSuite{})

func (s *PprofEnabledSuite) SetUpSuite(c *C) {
	aptly.Version = "testVersion"

	file, err := os.CreateTemp("", "aptly-pprof")
	c.Assert(err, IsNil)
	s.configFile = file

	jsonString, err := json.Marshal(gin.H{
		"architectures":       []string{},
		"enablePprofEndpoint": true,
	})
	c.Assert(err, IsNil)
	_, err = file.Write(jsonString)
	c.Assert(err, IsNil)
	_ = file.Close()

	aptlyCtx, router, err := newPprofContext(file.Name())
	c.Assert(err, IsNil)
	s.aptlyCtx = aptlyCtx
	s.router = router
}

func (s *PprofEnabledSuite) TearDownSuite(c *C) {
	if s.configFile != nil {
		_ = os.Remove(s.configFile.Name())
	}
	if s.aptlyCtx != nil {
		s.aptlyCtx.Shutdown()
	}
}

func (s *PprofEnabledSuite) request(method, url string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	s.router.ServeHTTP(w, req)
	return w
}

// TestPprofIndexReturns200 verifies the pprof index page is served when the
// endpoint is enabled.
func (s *PprofEnabledSuite) TestPprofIndexReturns200(c *C) {
	w := s.request("GET", "/api/debug/pprof/")
	c.Check(w.Code, Equals, 200)
	c.Check(w.Header().Get("Content-Type"), Matches, "text/html.*")
}

// TestPprofGoroutineReturns200 verifies the goroutine sub-profile is served.
func (s *PprofEnabledSuite) TestPprofGoroutineReturns200(c *C) {
	w := s.request("GET", "/api/debug/pprof/goroutine?debug=1")
	c.Check(w.Code, Equals, 200)
}

// TestPprofHeapReturns200 verifies the heap sub-profile is served.
func (s *PprofEnabledSuite) TestPprofHeapReturns200(c *C) {
	w := s.request("GET", "/api/debug/pprof/heap?debug=1")
	c.Check(w.Code, Equals, 200)
}

// TestPprofCmdlineReturns200 verifies the cmdline handler is served.
func (s *PprofEnabledSuite) TestPprofCmdlineReturns200(c *C) {
	w := s.request("GET", "/api/debug/pprof/cmdline")
	c.Check(w.Code, Equals, 200)
}

// TestPprofSymbolReturns200 verifies the symbol lookup handler responds (GET
// with no addresses returns 200 with a count of 0 symbols found).
func (s *PprofEnabledSuite) TestPprofSymbolReturns200(c *C) {
	w := s.request("GET", "/api/debug/pprof/symbol")
	c.Check(w.Code, Equals, 200)
}

// TestPprofTraceReturns200 verifies the execution trace handler is wired
// correctly. The minimum trace duration is 1 second, so this test blocks
// briefly but confirms end-to-end handler registration.
func (s *PprofEnabledSuite) TestPprofTraceReturns200(c *C) {
	w := s.request("GET", "/api/debug/pprof/trace?seconds=1")
	c.Check(w.Code, Equals, 200)
	c.Check(w.Header().Get("Content-Type"), Equals, "application/octet-stream")
}

// TestNonPprofRoutesUnaffectedWhenEnabled verifies that existing API routes
// continue to work correctly when the pprof endpoint is enabled.
func (s *PprofEnabledSuite) TestNonPprofRoutesUnaffectedWhenEnabled(c *C) {
	w := s.request("GET", "/api/version")
	c.Check(w.Code, Equals, 200)
}

// PprofDisabledSuite tests that pprof endpoints are NOT exposed when the config
// option is off. It embeds APISuite, whose config does not set
// enablePprofEndpoint, confirming the default state is safe. The inherited
// APISuite tests also run here, verifying that normal API operation is
// unaffected when pprof is disabled.
type PprofDisabledSuite struct {
	APISuite
}

var _ = Suite(&PprofDisabledSuite{})

// TestPprofIndexNotExposed verifies the pprof index is not reachable when
// enablePprofEndpoint is false (the default).
func (s *PprofDisabledSuite) TestPprofIndexNotExposed(c *C) {
	response, err := s.HTTPRequest("GET", "/api/debug/pprof/", nil)
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 404)
}

// TestPprofGoroutineNotExposed verifies the goroutine profile is not reachable
// when disabled.
func (s *PprofDisabledSuite) TestPprofGoroutineNotExposed(c *C) {
	response, err := s.HTTPRequest("GET", "/api/debug/pprof/goroutine", nil)
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 404)
}

// TestPprofHeapNotExposed verifies the heap profile is not reachable when
// disabled.
func (s *PprofDisabledSuite) TestPprofHeapNotExposed(c *C) {
	response, err := s.HTTPRequest("GET", "/api/debug/pprof/heap", nil)
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 404)
}

// TestPprofCmdlineNotExposed verifies the cmdline endpoint is not reachable
// when disabled.
func (s *PprofDisabledSuite) TestPprofCmdlineNotExposed(c *C) {
	response, err := s.HTTPRequest("GET", "/api/debug/pprof/cmdline", nil)
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 404)
}

// TestPprofTraceNotExposed verifies the trace endpoint is not reachable when
// disabled.
func (s *PprofDisabledSuite) TestPprofTraceNotExposed(c *C) {
	response, err := s.HTTPRequest("GET", "/api/debug/pprof/trace", nil)
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 404)
}
