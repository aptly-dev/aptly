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
	"strings"
	"syscall"

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
	context = aptlyContext
}

func (s *FilesUploadDiskFullSuite) TearDownTest(c *C) {
	if s.configFile != nil {
		_ = os.Remove(s.configFile.Name())
	}
	if s.aptlyContext != nil {
		s.aptlyContext.Shutdown()
	}
}

func (s *FilesUploadDiskFullSuite) TestUploadSuccessWithSync(c *C) {
	testContent := []byte("test file content for upload")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "testfile.txt")
	c.Assert(err, IsNil)

	_, err = part.Write(testContent)
	c.Assert(err, IsNil)

	err = writer.Close()
	c.Assert(err, IsNil)

	req, err := http.NewRequest("POST", "/api/files/testdir", body)
	c.Assert(err, IsNil)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Assert(w.Code, Equals, 200)

	uploadedFile := filepath.Join(s.aptlyContext.Config().GetRootDir(), "upload", "testdir", "testfile.txt")
	content, err := os.ReadFile(uploadedFile)
	c.Assert(err, IsNil)
	c.Check(content, DeepEquals, testContent)
}

func (s *FilesUploadDiskFullSuite) TestUploadVerifiesFileIntegrity(c *C) {
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

	uploadedFile := filepath.Join(s.aptlyContext.Config().GetRootDir(), "upload", "testdir2", "largefile.bin")
	content, err := os.ReadFile(uploadedFile)
	c.Assert(err, IsNil)
	c.Check(len(content), Equals, len(testContent))
	c.Check(content, DeepEquals, testContent)
}

func (s *FilesUploadDiskFullSuite) TestUploadMultipleFilesWithBatchSync(c *C) {
	testFiles := map[string][]byte{
		"file1.txt": []byte("content of file 1"),
		"file2.txt": bytes.Repeat([]byte("B"), 5000),
		"file3.deb": []byte("debian package content"),
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

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

	c.Assert(w.Code, Equals, 200)

	uploadDir := filepath.Join(s.aptlyContext.Config().GetRootDir(), "upload", "multitest")
	for filename, expectedContent := range testFiles {
		uploadedFile := filepath.Join(uploadDir, filename)
		content, err := os.ReadFile(uploadedFile)
		c.Assert(err, IsNil, Commentf("Failed to read %s", filename))
		c.Check(content, DeepEquals, expectedContent, Commentf("Content mismatch for %s", filename))
	}
}

func (s *FilesUploadDiskFullSuite) TestUploadReturnsErrorOnSyncFailure(c *C) {
	oldSyncFile := syncFile
	syncFile = func(f *os.File) error {
		if filepath.Base(f.Name()) == "syncfail.txt" {
			return syscall.ENOSPC
		}
		return nil
	}
	defer func() { syncFile = oldSyncFile }()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part1, err := writer.CreateFormFile("file", "ok.txt")
	c.Assert(err, IsNil)
	_, err = part1.Write([]byte("ok"))
	c.Assert(err, IsNil)

	part2, err := writer.CreateFormFile("file", "syncfail.txt")
	c.Assert(err, IsNil)
	_, err = part2.Write([]byte("will fail on sync"))
	c.Assert(err, IsNil)

	err = writer.Close()
	c.Assert(err, IsNil)

	req, err := http.NewRequest("POST", "/api/files/syncfaildir", body)
	c.Assert(err, IsNil)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Assert(w.Code, Equals, 500)
	c.Check(bytes.Contains(w.Body.Bytes(), []byte("error syncing file")), Equals, true)
}

func (s *FilesUploadDiskFullSuite) TestVerifyPath(c *C) {
	c.Check(verifyPath("a/b/c"), Equals, true)
	c.Check(verifyPath("../x"), Equals, false)
	c.Check(verifyPath("./x"), Equals, true)
	c.Check(verifyPath(".."), Equals, false)
	c.Check(verifyPath("."), Equals, false)
}

func (s *FilesUploadDiskFullSuite) TestListDirsEmptyWhenUploadMissing(c *C) {
	_ = os.RemoveAll(s.aptlyContext.UploadPath())

	req, err := http.NewRequest("GET", "/api/files", nil)
	c.Assert(err, IsNil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Assert(w.Code, Equals, 200)
	c.Check(strings.TrimSpace(w.Body.String()), Equals, "[]")
}

func (s *FilesUploadDiskFullSuite) TestListDirsReturnsDirectories(c *C) {
	uploadRoot := s.aptlyContext.UploadPath()
	c.Assert(os.MkdirAll(filepath.Join(uploadRoot, "d1"), 0777), IsNil)
	c.Assert(os.MkdirAll(filepath.Join(uploadRoot, "d2"), 0777), IsNil)
	c.Assert(os.WriteFile(filepath.Join(uploadRoot, "rootfile"), []byte("x"), 0644), IsNil)

	req, err := http.NewRequest("GET", "/api/files", nil)
	c.Assert(err, IsNil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Assert(w.Code, Equals, 200)
	body := w.Body.String()
	c.Check(strings.Contains(body, "d1"), Equals, true)
	c.Check(strings.Contains(body, "d2"), Equals, true)
}

func (s *FilesUploadDiskFullSuite) TestListFilesNotFound(c *C) {
	req, err := http.NewRequest("GET", "/api/files/does-not-exist", nil)
	c.Assert(err, IsNil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Assert(w.Code, Equals, 404)
}

func (s *FilesUploadDiskFullSuite) TestListFilesReturnsFiles(c *C) {
	base := filepath.Join(s.aptlyContext.UploadPath(), "dir")
	c.Assert(os.MkdirAll(base, 0777), IsNil)
	c.Assert(os.WriteFile(filepath.Join(base, "a.txt"), []byte("a"), 0644), IsNil)
	c.Assert(os.WriteFile(filepath.Join(base, "b.txt"), []byte("b"), 0644), IsNil)

	req, err := http.NewRequest("GET", "/api/files/dir", nil)
	c.Assert(err, IsNil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Assert(w.Code, Equals, 200)
	body := w.Body.String()
	c.Check(strings.Contains(body, "a.txt"), Equals, true)
	c.Check(strings.Contains(body, "b.txt"), Equals, true)
}

func (s *FilesUploadDiskFullSuite) TestDeleteDirRemovesDirectory(c *C) {
	base := filepath.Join(s.aptlyContext.UploadPath(), "todel")
	c.Assert(os.MkdirAll(base, 0777), IsNil)
	c.Assert(os.WriteFile(filepath.Join(base, "a.txt"), []byte("a"), 0644), IsNil)

	req, err := http.NewRequest("DELETE", "/api/files/todel", nil)
	c.Assert(err, IsNil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Assert(w.Code, Equals, 200)

	_, statErr := os.Stat(base)
	c.Check(os.IsNotExist(statErr), Equals, true)
}

func (s *FilesUploadDiskFullSuite) TestDeleteFileRemovesFile(c *C) {
	base := filepath.Join(s.aptlyContext.UploadPath(), "todel2")
	c.Assert(os.MkdirAll(base, 0777), IsNil)
	c.Assert(os.WriteFile(filepath.Join(base, "a.txt"), []byte("a"), 0644), IsNil)

	req, err := http.NewRequest("DELETE", "/api/files/todel2/a.txt", nil)
	c.Assert(err, IsNil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Assert(w.Code, Equals, 200)

	_, statErr := os.Stat(filepath.Join(base, "a.txt"))
	c.Check(os.IsNotExist(statErr), Equals, true)
}

func (s *FilesUploadDiskFullSuite) TestDeleteFileNotFoundStillOk(c *C) {
	base := filepath.Join(s.aptlyContext.UploadPath(), "todel3")
	c.Assert(os.MkdirAll(base, 0777), IsNil)

	req, err := http.NewRequest("DELETE", "/api/files/todel3/nope.txt", nil)
	c.Assert(err, IsNil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Assert(w.Code, Equals, 200)
}

func (s *FilesUploadDiskFullSuite) TestRejectsInvalidDir(c *C) {
	req, err := http.NewRequest("DELETE", "/api/files/..", nil)
	c.Assert(err, IsNil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Assert(w.Code, Equals, 400)
}

func (s *FilesUploadDiskFullSuite) TestRejectsInvalidFileName(c *C) {
	base := filepath.Join(s.aptlyContext.UploadPath(), "dirx")
	c.Assert(os.MkdirAll(base, 0777), IsNil)

	req, err := http.NewRequest("DELETE", "/api/files/dirx/..", nil)
	c.Assert(err, IsNil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Assert(w.Code, Equals, 400)
}

func (s *FilesUploadDiskFullSuite) TestListDirsEmptyIfUploadPathIsNotDir(c *C) {
	_ = os.RemoveAll(s.aptlyContext.UploadPath())
	c.Assert(os.WriteFile(s.aptlyContext.UploadPath(), []byte("not a dir"), 0644), IsNil)

	req, err := http.NewRequest("GET", "/api/files", nil)
	c.Assert(err, IsNil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Assert(w.Code, Equals, 200)
	c.Check(strings.TrimSpace(w.Body.String()), Equals, "[]")
}

func (s *FilesUploadDiskFullSuite) TestListFilesReturns500OnPermissionError(c *C) {
	base := filepath.Join(s.aptlyContext.UploadPath(), "noperms")
	c.Assert(os.MkdirAll(base, 0777), IsNil)
	c.Assert(os.WriteFile(filepath.Join(base, "a.txt"), []byte("a"), 0644), IsNil)
	c.Assert(os.Chmod(base, 0), IsNil)
	defer func() { _ = os.Chmod(base, 0777) }()

	req, err := http.NewRequest("GET", "/api/files/noperms", nil)
	c.Assert(err, IsNil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Assert(w.Code, Equals, 500)
}

func (s *FilesUploadDiskFullSuite) TestDeleteFileReturns500OnNonNotExistError(c *C) {
	base := filepath.Join(s.aptlyContext.UploadPath(), "dirisfile")
	c.Assert(os.MkdirAll(base, 0777), IsNil)
	subdir := filepath.Join(base, "subdir")
	c.Assert(os.MkdirAll(subdir, 0777), IsNil)
	c.Assert(os.WriteFile(filepath.Join(subdir, "x"), []byte("x"), 0644), IsNil)

	req, err := http.NewRequest("DELETE", "/api/files/dirisfile/subdir", nil)
	c.Assert(err, IsNil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Assert(w.Code, Equals, 500)
}

func (s *FilesUploadDiskFullSuite) TestUploadBadMultipartReturns400(c *C) {
	req, err := http.NewRequest("POST", "/api/files/badmultipart", bytes.NewBufferString("not multipart"))
	c.Assert(err, IsNil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=missing")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Assert(w.Code, Equals, 400)
}

func (s *FilesUploadDiskFullSuite) TestUploadRejectsInvalidDir(c *C) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "a.txt")
	c.Assert(err, IsNil)
	_, err = part.Write([]byte("x"))
	c.Assert(err, IsNil)
	c.Assert(writer.Close(), IsNil)

	req, err := http.NewRequest("POST", "/api/files/..", body)
	c.Assert(err, IsNil)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Assert(w.Code, Equals, 400)
}

func (s *FilesUploadDiskFullSuite) TestUploadReturns500IfUploadRootIsNotDir(c *C) {
	_ = os.RemoveAll(s.aptlyContext.UploadPath())
	c.Assert(os.WriteFile(s.aptlyContext.UploadPath(), []byte("not a dir"), 0644), IsNil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "a.txt")
	c.Assert(err, IsNil)
	_, err = part.Write([]byte("x"))
	c.Assert(err, IsNil)
	c.Assert(writer.Close(), IsNil)

	req, err := http.NewRequest("POST", "/api/files/testdir", body)
	c.Assert(err, IsNil)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Assert(w.Code, Equals, 500)
}

func (s *FilesUploadDiskFullSuite) TestUploadReturns500OnFileOpenFailure(c *C) {
	// Pre-populate MultipartForm to inject a FileHeader that fails on Open().
	form := &multipart.Form{
		File: map[string][]*multipart.FileHeader{
			"file": {{Filename: "broken.bin"}},
		},
	}

	req, err := http.NewRequest("POST", "/api/files/openfaildir", nil)
	c.Assert(err, IsNil)
	req.MultipartForm = form

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Assert(w.Code, Equals, 500)
}

func (s *FilesUploadDiskFullSuite) TestUploadReturns500OnCreateFailure(c *C) {
	base := filepath.Join(s.aptlyContext.UploadPath(), "readonly")
	c.Assert(os.MkdirAll(base, 0777), IsNil)
	c.Assert(os.Chmod(base, 0555), IsNil)
	defer func() { _ = os.Chmod(base, 0777) }()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "a.txt")
	c.Assert(err, IsNil)
	_, err = part.Write([]byte("x"))
	c.Assert(err, IsNil)
	c.Assert(writer.Close(), IsNil)

	req, err := http.NewRequest("POST", "/api/files/readonly", body)
	c.Assert(err, IsNil)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Assert(w.Code, Equals, 500)
}

func (s *FilesUploadDiskFullSuite) TestDeleteDirReturns500OnRemoveFailure(c *C) {
	parent := s.aptlyContext.UploadPath()
	base := filepath.Join(parent, "cantremove")
	c.Assert(os.MkdirAll(base, 0777), IsNil)
	c.Assert(os.WriteFile(filepath.Join(base, "a.txt"), []byte("a"), 0644), IsNil)

	c.Assert(os.Chmod(parent, 0555), IsNil)
	defer func() { _ = os.Chmod(parent, 0777) }()

	req, err := http.NewRequest("DELETE", "/api/files/cantremove", nil)
	c.Assert(err, IsNil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	c.Assert(w.Code, Equals, 500)
}
