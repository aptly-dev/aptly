package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"runtime"
	"sync/atomic"

	"github.com/aptly-dev/aptly/utils"
	"github.com/gin-gonic/gin"
	. "gopkg.in/check.v1"
)

type MiddlewareSuite struct {
	router    http.Handler
	context   *gin.Context
	logReader *os.File
	logWriter *os.File
}

var _ = Suite(&MiddlewareSuite{})

func (s *MiddlewareSuite) SetUpTest(c *C) {
	r, w, err := os.Pipe()
	c.Assert(err, IsNil)

	l := utils.NewZeroJSONLoggerWithWriter("debug", w)
	mw := JSONLogger(l)

	router := gin.New()
	router.UseRawPath = true
	router.Use(gin.Recovery(), gin.ErrorLogger())
	router.Use(mw)

	root := router.Group("/api")
	isReady := &atomic.Value{}
	isReady.Store(false)
	root.GET("/ready", apiReady(isReady))
	root.GET("/healthy", apiHealthy)

	s.router = router
	s.logReader = r
	s.logWriter = w

}

func (s *MiddlewareSuite) TearDownTest(c *C) {
	s.router = nil
	s.context = nil
	s.logReader = nil
	s.logWriter = nil
}

func (s *MiddlewareSuite) HTTPRequest(method string, url string, body io.Reader) {
	recorder := httptest.NewRecorder()
	s.context, _ = gin.CreateTestContext(recorder)
	req, _ := http.NewRequestWithContext(s.context, method, url, body)
	s.context.Request = req
	req.Header.Add("Content-Type", "application/json")
	s.router.ServeHTTP(httptest.NewRecorder(), req)
}

// Source: https://stackoverflow.com/questions/7052693
func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func (s *MiddlewareSuite) TestLoggerMiddlewareFactory(c *C) {
	l := utils.LoggerFactory("default", "debug")
	mw := LoggerMiddlewareFactory(l)
	c.Check(fmt.Sprintf("%T", gin.DefaultWriter), Equals, "*os.File")
	c.Check(GetFunctionName(mw), Matches, ".*gin.LoggerWithConfig.*")

	l = utils.LoggerFactory("json", "debug")
	mw = LoggerMiddlewareFactory(l)
	c.Check(fmt.Sprintf("%T", gin.DefaultWriter), Equals, fmt.Sprintf("%T", utils.LogWriter{}))
	c.Check(GetFunctionName(mw), Matches, ".*api.JSONLogger.*")
}

func (s *MiddlewareSuite) TestJSONMiddleware4xx(c *C) {
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, s.logReader)
		fmt.Println(buf.String())
		outC <- buf.String()
	}()

	s.HTTPRequest(http.MethodGet, "/", nil)
	s.logWriter.Close()
	capturedOutput := <-outC

	var jsonMap map[string]interface{}
	json.Unmarshal([]byte(capturedOutput), &jsonMap)

	if val, ok := jsonMap["level"]; ok {
		c.Check(val, Equals, "warn")
	} else {
		c.Errorf("Log message didn't have a 'level' key, obtained %s", capturedOutput)
	}

	if val, ok := jsonMap["method"]; ok {
		c.Check(val, Equals, "GET")
	} else {
		c.Errorf("Log message didn't have a 'method' key, obtained %s", capturedOutput)
	}

	if val, ok := jsonMap["path"]; ok {
		c.Check(val, Equals, "/")
	} else {
		c.Errorf("Log message didn't have a 'path' key, obtained %s", capturedOutput)
	}

	if val, ok := jsonMap["protocol"]; ok {
		c.Check(val, Equals, "HTTP/1.1")
	} else {
		c.Errorf("Log message didn't have a 'protocol' key, obtained %s", capturedOutput)
	}

	if val, ok := jsonMap["code"]; ok {
		c.Check(val, Equals, "404")
	} else {
		c.Errorf("Log message didn't have a 'code' key, obtained %s", capturedOutput)
	}

	if _, ok := jsonMap["remote"]; !ok {
		c.Errorf("Log message didn't have a 'remote' key, obtained %s", capturedOutput)
	}

	if _, ok := jsonMap["latency"]; !ok {
		c.Errorf("Log message didn't have a 'latency' key, obtained %s", capturedOutput)
	}

	if _, ok := jsonMap["agent"]; !ok {
		c.Errorf("Log message didn't have a 'agent' key, obtained %s", capturedOutput)
	}

	if _, ok := jsonMap["ts"]; !ok {
		c.Errorf("Log message didn't have a 'ts' key, obtained %s", capturedOutput)
	}
}

func (s *MiddlewareSuite) TestJSONMiddleware2xx(c *C) {
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, s.logReader)
		fmt.Println(buf.String())
		outC <- buf.String()
	}()

	s.HTTPRequest(http.MethodGet, "/api/healthy", nil)
	s.logWriter.Close()
	capturedOutput := <-outC

	var jsonMap map[string]interface{}
	json.Unmarshal([]byte(capturedOutput), &jsonMap)

	if val, ok := jsonMap["level"]; ok {
		c.Check(val, Equals, "info")
	} else {
		c.Errorf("Log message didn't have a 'level' key, obtained %s", capturedOutput)
	}
}

func (s *MiddlewareSuite) TestJSONMiddleware5xx(c *C) {
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, s.logReader)
		fmt.Println(buf.String())
		outC <- buf.String()
	}()

	s.HTTPRequest(http.MethodGet, "/api/ready", nil)
	s.logWriter.Close()
	capturedOutput := <-outC

	var jsonMap map[string]interface{}
	json.Unmarshal([]byte(capturedOutput), &jsonMap)

	if val, ok := jsonMap["level"]; ok {
		c.Check(val, Equals, "error")
	} else {
		c.Errorf("Log message didn't have a 'level' key, obtained %s", capturedOutput)
	}
}

func (s *MiddlewareSuite) TestJSONMiddlewareRaw(c *C) {
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, s.logReader)
		fmt.Println(buf.String())
		outC <- buf.String()
	}()

	s.HTTPRequest(http.MethodGet, "/api/healthy?test=raw", nil)
	s.logWriter.Close()
	capturedOutput := <-outC

	var jsonMap map[string]interface{}
	json.Unmarshal([]byte(capturedOutput), &jsonMap)

	fmt.Println(capturedOutput)

	if val, ok := jsonMap["level"]; ok {
		c.Check(val, Equals, "info")
	} else {
		c.Errorf("Log message didn't have a 'level' key, obtained %s", capturedOutput)
	}
}

func (s *MiddlewareSuite) TestGetBasePath(c *C) {
	s.HTTPRequest(http.MethodGet, "", nil)
	path := getBasePath(s.context)
	c.Check(path, Equals, "/")

	s.HTTPRequest(http.MethodGet, "/", nil)
	path = getBasePath(s.context)
	c.Check(path, Equals, "/")

	s.HTTPRequest(http.MethodGet, "/api", nil)
	path = getBasePath(s.context)
	c.Check(path, Equals, "/api")

	s.HTTPRequest(http.MethodGet, "/api/repos/testRepo", nil)
	path = getBasePath(s.context)
	c.Check(path, Equals, "/api/repos")
}

func (s *MiddlewareSuite) TestGetURLSegment(c *C) {
	url := "/"
	segment, err := getURLSegment(url, 0)
	if err != nil {
		c.Error(err)
	}
	c.Check(*segment, Equals, "/")

	_, err = getURLSegment(url, 1)
	if err == nil {
		c.Error("Invalid return value")
	}

	url = "/api"
	segment, err = getURLSegment(url, 0)
	if err != nil {
		c.Error(err)
	}
	c.Check(*segment, Equals, "/api")

	_, err = getURLSegment(url, 1)
	if err == nil {
		c.Error("Invalid return value")
	}

	url = "/api/repos/testRepo"
	segment, err = getURLSegment(url, 0)
	if err != nil {
		c.Error(err)
	}
	c.Check(*segment, Equals, "/api")

	segment, err = getURLSegment(url, 1)
	if err != nil {
		c.Error(err)
	}
	c.Check(*segment, Equals, "/repos")
}
