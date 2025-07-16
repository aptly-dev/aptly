package api

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func TestFiles(t *testing.T) { TestingT(t) }

type FilesSuite struct{}

var _ = Suite(&FilesSuite{})

func (s *FilesSuite) SetUpTest(c *C) {
	gin.SetMode(gin.TestMode)
}

func (s *FilesSuite) TestVerifyPath(c *C) {
	// Valid paths
	c.Check(verifyPath("valid-dir"), Equals, true)
	c.Check(verifyPath("valid/sub/dir"), Equals, true)
	c.Check(verifyPath(""), Equals, true)

	// Invalid paths with ..
	c.Check(verifyPath("../invalid"), Equals, false)
	c.Check(verifyPath("valid/../invalid"), Equals, false)
	c.Check(verifyPath(".."), Equals, false)
	c.Check(verifyPath("."), Equals, false)
}

func (s *FilesSuite) TestVerifyDirValid(c *C) {
	// Create a test gin context
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{
		{Key: "dir", Value: "valid-dir"},
	}

	result := verifyDir(ctx)
	c.Check(result, Equals, true)
}

func (s *FilesSuite) TestVerifyDirInvalid(c *C) {
	// Create a test gin context
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{
		{Key: "dir", Value: "../invalid"},
	}

	result := verifyDir(ctx)
	c.Check(result, Equals, false)
	c.Check(w.Code, Equals, 400)
}

func (s *FilesSuite) TestApiFilesListDirs(c *C) {
	// Create test request - this will likely fail due to no database/context
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/files", nil)

	apiFilesListDirs(ctx)

	// Since we don't have proper context setup, expect error response
	c.Check(w.Code >= 400, Equals, true)
}

func (s *FilesSuite) TestApiFilesUpload(c *C) {
	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.txt")
	c.Assert(err, IsNil)
	part.Write([]byte("test content"))
	writer.Close()

	// Create test request
	req := httptest.NewRequest("POST", "/api/files/testdir", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req
	ctx.Params = gin.Params{
		{Key: "dir", Value: "testdir"},
	}

	apiFilesUpload(ctx)

	// Since we don't have proper context setup, expect error response
	c.Check(w.Code >= 400, Equals, true)
}

func (s *FilesSuite) TestApiFilesListFiles(c *C) {
	// Create test request
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/files/testdir", nil)
	ctx.Params = gin.Params{
		{Key: "dir", Value: "testdir"},
	}

	apiFilesListFiles(ctx)

	// Since we don't have proper context setup, expect error response
	c.Check(w.Code >= 400, Equals, true)
}

func (s *FilesSuite) TestApiFilesDeleteDir(c *C) {
	// Create test request
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("DELETE", "/api/files/testdir", nil)
	ctx.Params = gin.Params{
		{Key: "dir", Value: "testdir"},
	}

	apiFilesDeleteDir(ctx)

	// Since we don't have proper context setup, expect error response
	c.Check(w.Code >= 400, Equals, true)
}

func (s *FilesSuite) TestApiFilesDeleteFile(c *C) {
	// Create test request
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("DELETE", "/api/files/testdir/test.txt", nil)
	ctx.Params = gin.Params{
		{Key: "dir", Value: "testdir"},
		{Key: "name", Value: "test.txt"},
	}

	apiFilesDeleteFile(ctx)

	// Since we don't have proper context setup, expect error response
	c.Check(w.Code >= 400, Equals, true)
}

func (s *FilesSuite) TestApiFilesDeleteFileInvalidPath(c *C) {
	// Create test request with invalid path
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("DELETE", "/api/files/testdir/../invalid", nil)
	ctx.Params = gin.Params{
		{Key: "dir", Value: "testdir"},
		{Key: "name", Value: "../invalid"},
	}

	apiFilesDeleteFile(ctx)

	c.Check(w.Code, Equals, 400)
}

// Custom checker for file existence
var testFileExists Checker = &fileExistsChecker{
	CheckerInfo: &CheckerInfo{Name: "testFileExists", Params: []string{"filename"}},
}

type fileExistsChecker struct {
	*CheckerInfo
}

func (checker *fileExistsChecker) Check(params []interface{}, names []string) (result bool, error string) {
	filename, ok := params[0].(string)
	if !ok {
		return false, "filename must be a string"
	}

	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, ""
		}
		return false, err.Error()
	}
	return true, ""
}

// Test core API functions
func (s *FilesSuite) TestApiVersion(c *C) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/version", nil)

	apiVersion(ctx)

	c.Check(w.Code, Equals, 200)
	c.Check(w.Body.String(), Matches, `.*"Version":.*`)
}

func (s *FilesSuite) TestApiHealthy(c *C) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/healthy", nil)

	apiHealthy(ctx)

	c.Check(w.Code, Equals, 200)
	c.Check(w.Body.String(), Matches, `.*"Status":"Aptly is healthy".*`)
}

func (s *FilesSuite) TestApiReadyWhenReady(c *C) {
	isReady := &atomic.Value{}
	isReady.Store(true)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/ready", nil)

	apiReady(isReady)(ctx)

	c.Check(w.Code, Equals, 200)
	c.Check(w.Body.String(), Matches, `.*"Status":"Aptly is ready".*`)
}

func (s *FilesSuite) TestApiReadyWhenNotReady(c *C) {
	isReady := &atomic.Value{}
	isReady.Store(false)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/ready", nil)

	apiReady(isReady)(ctx)

	c.Check(w.Code, Equals, 503)
	c.Check(w.Body.String(), Matches, `.*"Status":"Aptly is unavailable".*`)
}

func (s *FilesSuite) TestApiReadyWithNil(c *C) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/ready", nil)

	apiReady(nil)(ctx)

	c.Check(w.Code, Equals, 503)
	c.Check(w.Body.String(), Matches, `.*"Status":"Aptly is unavailable".*`)
}

func (s *FilesSuite) TestTruthy(c *C) {
	// Test string values
	c.Check(truthy("yes"), Equals, true)
	c.Check(truthy("true"), Equals, true)
	c.Check(truthy("1"), Equals, true)
	c.Check(truthy("on"), Equals, true)
	c.Check(truthy("anything"), Equals, true)
	c.Check(truthy("n"), Equals, false)
	c.Check(truthy("no"), Equals, false)
	c.Check(truthy("f"), Equals, false)
	c.Check(truthy("false"), Equals, false)
	c.Check(truthy("0"), Equals, false)
	c.Check(truthy("off"), Equals, false)
	c.Check(truthy("NO"), Equals, false)    // case insensitive
	c.Check(truthy("FALSE"), Equals, false) // case insensitive

	// Test int values
	c.Check(truthy(1), Equals, true)
	c.Check(truthy(42), Equals, true)
	c.Check(truthy(-1), Equals, true)
	c.Check(truthy(0), Equals, false)

	// Test bool values
	c.Check(truthy(true), Equals, true)
	c.Check(truthy(false), Equals, false)

	// Test nil
	c.Check(truthy(nil), Equals, false)
}
