package main

import (
	"io/ioutil"
	. "launchpad.net/gocheck"
	"os"
	"runtime"
	"testing"
	"time"
)

// Launch gocheck tests
func Test(t *testing.T) {
	TestingT(t)
}

type DownloaderSuite struct {
	tempfile *os.File
}

var _ = Suite(&DownloaderSuite{})

func (s *DownloaderSuite) SetUpTest(c *C) {
	s.tempfile, _ = ioutil.TempFile(os.TempDir(), "aptly-test")
}

func (s *DownloaderSuite) TearDownTest(c *C) {
	os.Remove(s.tempfile.Name())
	s.tempfile.Close()
}

func (s *DownloaderSuite) TestStartupShutdown(c *C) {
	goroutines := runtime.NumGoroutine()

	d := NewDownloader(10)
	d.Shutdown()

	// wait for goroutines to shutdown
	time.Sleep(100 * time.Millisecond)

	if runtime.NumGoroutine()-goroutines > 1 {
		c.Errorf("Number of goroutines %d, expected %d", runtime.NumGoroutine(), goroutines)
	}
}

func (s *DownloaderSuite) TestDownloadOK(c *C) {
	d := NewDownloader(2)
	defer d.Shutdown()

	res := <-d.Download("http://smira.ru/", s.tempfile.Name())
	c.Assert(res, IsNil)
}

func (s *DownloaderSuite) TestDownload404(c *C) {
	d := NewDownloader(2)
	defer d.Shutdown()

	res := <-d.Download("http://smira.ru/doesntexist", s.tempfile.Name())
	c.Assert(res, ErrorMatches, "HTTP code 404.*")
}

func (s *DownloaderSuite) TestDownloadConnectError(c *C) {
	d := NewDownloader(2)
	defer d.Shutdown()

	res := <-d.Download("http://nosuch.smira.ru/", s.tempfile.Name())
	c.Assert(res, ErrorMatches, ".*no such host")
}

func (s *DownloaderSuite) TestDownloadFileError(c *C) {
	d := NewDownloader(2)
	defer d.Shutdown()

	res := <-d.Download("http://smira.ru/", "/no/such/file")
	c.Assert(res, ErrorMatches, ".*no such file or directory")
}
