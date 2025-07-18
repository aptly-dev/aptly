package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func TestFiles(t *testing.T) { TestingT(t) }

type FilesSuite struct {
	APISuite
}

var _ = Suite(&FilesSuite{})

func (s *FilesSuite) SetUpTest(c *C) {
	s.APISuite.SetUpTest(c)
}

func (s *FilesSuite) TearDownTest(c *C) {
	// Clean up any test files
	if s.context != nil {
		uploadPath := s.context.UploadPath()
		if uploadPath != "" {
			os.RemoveAll(uploadPath)
		}
	}
	s.APISuite.TearDownTest(c)
}

func (s *FilesSuite) TestVerifyPath(c *C) {
	// Valid paths
	c.Check(verifyPath("valid-dir"), Equals, true)
	c.Check(verifyPath("valid/sub/dir"), Equals, true)
	c.Check(verifyPath("valid/../other"), Equals, true)  // filepath.Clean normalizes to "other"

	// Invalid paths
	c.Check(verifyPath(""), Equals, false)  // Empty path becomes "."
	c.Check(verifyPath("../invalid"), Equals, false)  // Contains ".."
	c.Check(verifyPath(".."), Equals, false)  // Is ".."
	c.Check(verifyPath("."), Equals, false)  // Is "."
	c.Check(verifyPath("./"), Equals, false)  // Contains "."
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
	// Create upload directory for testing
	uploadPath := s.context.UploadPath()
	err := os.MkdirAll(filepath.Join(uploadPath, "test-dir"), 0755)
	c.Assert(err, IsNil)
	defer os.RemoveAll(uploadPath)
	
	// Create test file
	f, err := os.Create(filepath.Join(uploadPath, "test-file.txt"))
	c.Assert(err, IsNil)
	f.Close()

	// Create test request
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/files", nil)
	s.router.ServeHTTP(w, req)

	// Check response
	c.Check(w.Code, Equals, 200)
	var result []string
	err = json.Unmarshal(w.Body.Bytes(), &result)
	c.Assert(err, IsNil)
	c.Check(len(result), Equals, 1)
	c.Check(result[0], Equals, "test-dir")
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
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/files/testdir", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	s.router.ServeHTTP(w, req)

	// Check response
	c.Check(w.Code, Equals, 200)
	
	// Verify file was uploaded
	uploadPath := filepath.Join(s.context.UploadPath(), "testdir", "test.txt")
	_, err = os.Stat(uploadPath)
	c.Assert(err, IsNil)
	
	// Clean up
	os.RemoveAll(filepath.Join(s.context.UploadPath(), "testdir"))
}

func (s *FilesSuite) TestApiFilesListFiles(c *C) {
	// Create test directory and files
	testDir := filepath.Join(s.context.UploadPath(), "testdir")
	err := os.MkdirAll(testDir, 0755)
	c.Assert(err, IsNil)
	
	// Create test files
	for i := 0; i < 3; i++ {
		f, err := os.Create(filepath.Join(testDir, fmt.Sprintf("test%d.txt", i)))
		c.Assert(err, IsNil)
		f.Close()
	}
	
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/files/testdir", nil)
	s.router.ServeHTTP(w, req)

	// Check response
	c.Check(w.Code, Equals, 200)
	
	var result []string
	err = json.Unmarshal(w.Body.Bytes(), &result)
	c.Assert(err, IsNil)
	c.Check(len(result), Equals, 3)
	
	// Clean up
	os.RemoveAll(testDir)
}

func (s *FilesSuite) TestApiFilesDeleteDir(c *C) {
	// Create test directory
	testDir := filepath.Join(s.context.UploadPath(), "testdir")
	err := os.MkdirAll(testDir, 0755)
	c.Assert(err, IsNil)
	
	// Create test file in directory
	f, err := os.Create(filepath.Join(testDir, "test.txt"))
	c.Assert(err, IsNil)
	f.Close()
	
	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/files/testdir", nil)
	s.router.ServeHTTP(w, req)

	// Check response
	c.Check(w.Code, Equals, 200)
	
	// Verify directory was deleted
	_, err = os.Stat(testDir)
	c.Assert(os.IsNotExist(err), Equals, true)
}

func (s *FilesSuite) TestApiFilesDeleteFile(c *C) {
	// Create test directory and file
	testDir := filepath.Join(s.context.UploadPath(), "testdir")
	err := os.MkdirAll(testDir, 0755)
	c.Assert(err, IsNil)
	
	testFile := filepath.Join(testDir, "test.txt")
	f, err := os.Create(testFile)
	c.Assert(err, IsNil)
	f.Write([]byte("test content"))
	f.Close()
	
	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/files/testdir/test.txt", nil)
	s.router.ServeHTTP(w, req)

	// Check response
	c.Check(w.Code, Equals, 200)
	
	// Verify file was deleted
	_, err = os.Stat(testFile)
	c.Assert(os.IsNotExist(err), Equals, true)
	
	// Clean up
	os.RemoveAll(testDir)
}

func (s *FilesSuite) TestApiFilesDeleteFileInvalidPath(c *C) {
	// Create test request with invalid path
	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/files/testdir/../invalid", nil)
	s.router.ServeHTTP(w, req)

	// Should reject with 404 (not found) or 400 (bad request)
	c.Check(w.Code == 400 || w.Code == 404, Equals, true)
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
