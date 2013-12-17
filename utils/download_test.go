package utils

import (
	"errors"
	"io"
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

	res := <-d.Download("http://smira.ru/", "/")
	c.Assert(res, ErrorMatches, ".*permission denied")
}

func (s *DownloaderSuite) TestDownloadTemp(c *C) {
	d := NewDownloader(2)
	defer d.Shutdown()

	f, err := DownloadTemp(d, "http://smira.ru/")
	c.Assert(err, IsNil)
	defer f.Close()

	buf := make([]byte, 1)

	f.Read(buf)
	c.Assert(buf, DeepEquals, []byte("<"))

	_, err = os.Stat(f.Name())
	c.Assert(os.IsNotExist(err), Equals, true)
}

func (s *DownloaderSuite) TestDownloadTempError(c *C) {
	d := NewDownloader(2)
	defer d.Shutdown()

	f, err := DownloadTemp(d, "http://smira.ru/doesntexist")
	c.Assert(err, NotNil)
	c.Assert(f, IsNil)
	c.Assert(err, ErrorMatches, "HTTP code 404.*")
}

const (
	bzipData = "BZh91AY&SY\xcc\xc3q\xd4\x00\x00\x02A\x80\x00\x10\x02\x00\x0c\x00 \x00!\x9ah3M\x19\x97\x8b\xb9\"\x9c(Hfa\xb8\xea\x00"
	gzipData = "\x1f\x8b\x08\x00\xc8j\xb0R\x00\x03+I-.\xe1\x02\x00\xc65\xb9;\x05\x00\x00\x00"
	rawData  = "test"
)

func (s *DownloaderSuite) TestDownloadTryCompression(c *C) {
	var buf []byte

	// bzip2 only available
	buf = make([]byte, 4)
	d := NewFakeDownloader()
	d.ExpectResponse("http://example.com/file.bz2", bzipData)
	r, file, err := DownloadTryCompression(d, "http://example.com/file")
	c.Assert(err, IsNil)
	defer file.Close()
	io.ReadFull(r, buf)
	c.Assert(string(buf), Equals, rawData)
	c.Assert(d.Empty(), Equals, true)

	// bzip2 not available, but gz is
	buf = make([]byte, 4)
	d = NewFakeDownloader()
	d.ExpectError("http://example.com/file.bz2", errors.New("404"))
	d.ExpectResponse("http://example.com/file.gz", gzipData)
	r, file, err = DownloadTryCompression(d, "http://example.com/file")
	c.Assert(err, IsNil)
	defer file.Close()
	io.ReadFull(r, buf)
	c.Assert(string(buf), Equals, rawData)
	c.Assert(d.Empty(), Equals, true)

	// bzip2 & gzip not available, but raw is
	buf = make([]byte, 4)
	d = NewFakeDownloader()
	d.ExpectError("http://example.com/file.bz2", errors.New("404"))
	d.ExpectError("http://example.com/file.gz", errors.New("404"))
	d.ExpectResponse("http://example.com/file", rawData)
	r, file, err = DownloadTryCompression(d, "http://example.com/file")
	c.Assert(err, IsNil)
	defer file.Close()
	io.ReadFull(r, buf)
	c.Assert(string(buf), Equals, rawData)
	c.Assert(d.Empty(), Equals, true)

	// gzip available, but broken
	buf = make([]byte, 4)
	d = NewFakeDownloader()
	d.ExpectError("http://example.com/file.bz2", errors.New("404"))
	d.ExpectResponse("http://example.com/file.gz", "x")
	d.ExpectResponse("http://example.com/file", "recovered")
	r, file, err = DownloadTryCompression(d, "http://example.com/file")
	c.Assert(err, IsNil)
	defer file.Close()
	io.ReadFull(r, buf)
	c.Assert(string(buf), Equals, "reco")
	c.Assert(d.Empty(), Equals, true)
}

func (s *DownloaderSuite) TestDownloadTryCompressionErrors(c *C) {
	d := NewFakeDownloader()
	_, _, err := DownloadTryCompression(d, "http://example.com/file")
	c.Assert(err, ErrorMatches, "unexpected request.*")

	d = NewFakeDownloader()
	d.ExpectError("http://example.com/file.bz2", errors.New("404"))
	d.ExpectError("http://example.com/file.gz", errors.New("404"))
	d.ExpectError("http://example.com/file", errors.New("403"))
	_, _, err = DownloadTryCompression(d, "http://example.com/file")
	c.Assert(err, ErrorMatches, "403")
}
