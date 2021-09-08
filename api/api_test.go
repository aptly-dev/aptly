package api

import (
	"encoding/json"
	ctx "github.com/aptly-dev/aptly/context"
	"github.com/gin-gonic/gin"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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
	file, err := ioutil.TempFile("", "aptly")
	if err != nil {
		return nil
	}
	jsonString, err := json.Marshal(gin.H{
		"architectures": []string{},
	})
	if err != nil {
		return nil
	}
	file.Write(jsonString)
	return file
}

func (s *ApiSuite) SetUpSuite(c *C) {
	file := createTestConfig()
	c.Assert(file, NotNil)
	s.configFile = file

	flags := flag.NewFlagSet("fakeFlags", flag.ContinueOnError)
	flags.Bool("no-lock", false, "dummy")
	flags.Int("db-open-attempts", 3, "dummy")
	flags.String("config", s.configFile.Name(), "dummy")
	flags.String("architectures", "", "dummy")
	s.flags = flags

	context, err := ctx.NewContext(s.flags)
	c.Assert(err, IsNil)

	s.context = context
	s.router = Router(context)
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

func (s *ApiSuite) TestGetVersion(c *C) {
	response, err := s.HTTPRequest("GET", "/api/version", nil)
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 200)
	c.Check(response.Body.String(), Matches, ".*Version.*")
}
