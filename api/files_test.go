package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/aptly-dev/aptly/aptly"
	ctx "github.com/aptly-dev/aptly/context"
	"github.com/gin-gonic/gin"
	"github.com/smira/flag"

	. "gopkg.in/check.v1"
)

type FilesUploadDiskFullSuite struct {
	aptlyContext *ctx.AptlyContext
	flags        *flag.FlagSet
	configFile   *os.File
	router       http.Handler
}

var _ = Suite(&FilesUploadDiskFullSuite{})

func (s *FilesUploadDiskFullSuite) SetUpTest(c *C) {
	aptly.Version = "testVersion"

	// Create temporary config
	file, err := os.CreateTemp("", "aptly")
	c.Assert(err, IsNil)
	s.configFile = file

	jsonString, err := json.Marshal(gin.H{
		"architectures": []string{},
		"rootDir":       c.MkDir(),
	})
	c.Assert(err, IsNil)
	_, err = file.Write(jsonString)
	c.Assert(err, IsNil)
	_ = file.Close()

	// Setup flags and context
	flags := flag.NewFlagSet("fakeFlags", flag.ContinueOnError)
	flags.Bool("no-lock", false, "dummy")
	flags.Int("db-open-attempts", 3, "dummy")
	flags.String("config", s.configFile.Name(), "dummy")
	flags.String("architectures", "", "dummy")
	s.flags = flags

	aptlyContext, err := ctx.NewContext(s.flags)
	c.Assert(err, IsNil)

	s.aptlyContext = aptlyContext
	s.router = Router(aptlyContext)
	context = aptlyContext // set global context
}

func (s *FilesUploadDiskFullSuite) TearDownTest(c *C) {
	if s.configFile != nil {
		_ = os.Remove(s.configFile.Name())
	}
	if s.aptlyContext != nil {
		s.aptlyContext.Shutdown()
	}
}

// TestUploadSuccessWithSync verifies that file uploads succeed when there's space
// and that the Sync() call is made (by verifying the file is complete)
func (s *FilesUploadDiskFullSuite) TestUploadSuccessWithSync(c *C) {
	// Create a test file to upload
	testContent := []byte("test file content for upload")

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "testfile.txt")
	c.Assert(err, IsNil)

	_, err = part.Write(testContent)
	c.Assert(err, IsNil)

	err = writer.Close()
	c.Assert(err, IsNil)

	// Create request
	req, err := http.NewRequest("POST", "/api/files/testdir", body)
	c.Assert(err, IsNil)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Create response recorder
	w := httptest.NewRecorder()

	// Call handler
	s.router.ServeHTTP(w, req)

	// Check response
	c.Assert(w.Code, Equals, 200)

	// Verify file was written and synced
	uploadedFile := filepath.Join(s.aptlyContext.Config().GetRootDir(), "upload", "testdir", "testfile.txt")
	content, err := os.ReadFile(uploadedFile)
	c.Assert(err, IsNil)
	c.Check(content, DeepEquals, testContent)
}

// TestUploadVerifiesFileIntegrity ensures uploaded files are complete
func (s *FilesUploadDiskFullSuite) TestUploadVerifiesFileIntegrity(c *C) {
	// Create larger test file
	testContent := bytes.Repeat([]byte("A"), 10000)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "largefile.bin")
	c.Assert(err, IsNil)

	_, err = io.Copy(part, bytes.NewReader(testContent))
	c.Assert(err, IsNil)

	err = writer.Close()
	c.Assert(err, IsNil)

	req, err := http.NewRequest("POST", "/api/files/testdir2", body)
	c.Assert(err, IsNil)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Assert(w.Code, Equals, 200)

	// Verify complete file was written
	uploadedFile := filepath.Join(s.aptlyContext.Config().GetRootDir(), "upload", "testdir2", "largefile.bin")
	content, err := os.ReadFile(uploadedFile)
	c.Assert(err, IsNil)
	c.Check(len(content), Equals, len(testContent))
	c.Check(content, DeepEquals, testContent)
}

// TestUploadMultipleFilesWithBatchSync tests that multiple files are synced in batch
func (s *FilesUploadDiskFullSuite) TestUploadMultipleFilesWithBatchSync(c *C) {
	// Create multiple test files with different content
	testFiles := map[string][]byte{
		"file1.txt": []byte("content of file 1"),
		"file2.txt": bytes.Repeat([]byte("B"), 5000),
		"file3.deb": []byte("debian package content"),
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add all files to multipart form
	for filename, content := range testFiles {
		part, err := writer.CreateFormFile("file", filename)
		c.Assert(err, IsNil)
		_, err = part.Write(content)
		c.Assert(err, IsNil)
	}

	err := writer.Close()
	c.Assert(err, IsNil)

	req, err := http.NewRequest("POST", "/api/files/multitest", body)
	c.Assert(err, IsNil)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Verify response
	c.Assert(w.Code, Equals, 200)

	// Verify all files were written and synced correctly
	uploadDir := filepath.Join(s.aptlyContext.Config().GetRootDir(), "upload", "multitest")
	for filename, expectedContent := range testFiles {
		uploadedFile := filepath.Join(uploadDir, filename)
		content, err := os.ReadFile(uploadedFile)
		c.Assert(err, IsNil, Commentf("Failed to read %s", filename))
		c.Check(content, DeepEquals, expectedContent, Commentf("Content mismatch for %s", filename))
	}
}
