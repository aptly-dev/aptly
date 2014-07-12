package http

import (
	"errors"
	"fmt"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/console"
	"github.com/smira/aptly/utils"
	"io"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"
)

type DownloaderSuite struct {
	tempfile *os.File
	l        net.Listener
	url      string
	ch       chan bool
	progress aptly.Progress
}

var _ = Suite(&DownloaderSuite{})

func (s *DownloaderSuite) SetUpTest(c *C) {
	s.tempfile, _ = ioutil.TempFile(os.TempDir(), "aptly-test")
	s.l, _ = net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	s.url = fmt.Sprintf("http://localhost:%d", s.l.Addr().(*net.TCPAddr).Port)

	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %s", r.URL.Path)
	})

	s.ch = make(chan bool)

	go func() {
		http.Serve(s.l, mux)
		s.ch <- true
	}()

	s.progress = console.NewProgress()
	s.progress.Start()
}

func (s *DownloaderSuite) TearDownTest(c *C) {
	s.progress.Shutdown()

	s.l.Close()
	<-s.ch

	os.Remove(s.tempfile.Name())
	s.tempfile.Close()
}

func (s *DownloaderSuite) TestStartupShutdown(c *C) {
	goroutines := runtime.NumGoroutine()

	d := NewDownloader(10, 100, s.progress)
	d.Shutdown()

	// wait for goroutines to shutdown
	time.Sleep(100 * time.Millisecond)

	if runtime.NumGoroutine()-goroutines > 1 {
		c.Errorf("Number of goroutines %d, expected %d", runtime.NumGoroutine(), goroutines)
	}
}

func (s *DownloaderSuite) TestPauseResume(c *C) {
	d := NewDownloader(2, 0, s.progress)
	defer d.Shutdown()

	d.Pause()
	d.Resume()
}

func (s *DownloaderSuite) TestDownloadOK(c *C) {
	d := NewDownloader(2, 0, s.progress)
	defer d.Shutdown()
	ch := make(chan error)

	d.Download(s.url+"/test", s.tempfile.Name(), ch)
	res := <-ch
	c.Assert(res, IsNil)
}

func (s *DownloaderSuite) TestDownloadWithChecksum(c *C) {
	d := NewDownloader(2, 0, s.progress)
	defer d.Shutdown()
	ch := make(chan error)

	d.DownloadWithChecksum(s.url+"/test", s.tempfile.Name(), ch, utils.ChecksumInfo{}, false)
	res := <-ch
	c.Assert(res, ErrorMatches, ".*size check mismatch 12 != 0")

	d.DownloadWithChecksum(s.url+"/test", s.tempfile.Name(), ch, utils.ChecksumInfo{Size: 12, MD5: "abcdef"}, false)
	res = <-ch
	c.Assert(res, ErrorMatches, ".*md5 hash mismatch \"a1acb0fe91c7db45ec4d775192ec5738\" != \"abcdef\"")

	d.DownloadWithChecksum(s.url+"/test", s.tempfile.Name(), ch, utils.ChecksumInfo{Size: 12, MD5: "abcdef"}, true)
	res = <-ch
	c.Assert(res, IsNil)

	d.DownloadWithChecksum(s.url+"/test", s.tempfile.Name(), ch, utils.ChecksumInfo{Size: 12, MD5: "a1acb0fe91c7db45ec4d775192ec5738"}, false)
	res = <-ch
	c.Assert(res, IsNil)

	d.DownloadWithChecksum(s.url+"/test", s.tempfile.Name(), ch, utils.ChecksumInfo{Size: 12, MD5: "a1acb0fe91c7db45ec4d775192ec5738", SHA1: "abcdef"}, false)
	res = <-ch
	c.Assert(res, ErrorMatches, ".*sha1 hash mismatch \"921893bae6ad6fd818401875d6779254ef0ff0ec\" != \"abcdef\"")

	d.DownloadWithChecksum(s.url+"/test", s.tempfile.Name(), ch, utils.ChecksumInfo{Size: 12, MD5: "a1acb0fe91c7db45ec4d775192ec5738",
		SHA1: "921893bae6ad6fd818401875d6779254ef0ff0ec"}, false)
	res = <-ch
	c.Assert(res, IsNil)

	d.DownloadWithChecksum(s.url+"/test", s.tempfile.Name(), ch, utils.ChecksumInfo{Size: 12, MD5: "a1acb0fe91c7db45ec4d775192ec5738",
		SHA1: "921893bae6ad6fd818401875d6779254ef0ff0ec", SHA256: "abcdef"}, false)
	res = <-ch
	c.Assert(res, ErrorMatches, ".*sha256 hash mismatch \"b3c92ee1246176ed35f6e8463cd49074f29442f5bbffc3f8591cde1dcc849dac\" != \"abcdef\"")

	d.DownloadWithChecksum(s.url+"/test", s.tempfile.Name(), ch, utils.ChecksumInfo{Size: 12, MD5: "a1acb0fe91c7db45ec4d775192ec5738",
		SHA1: "921893bae6ad6fd818401875d6779254ef0ff0ec", SHA256: "b3c92ee1246176ed35f6e8463cd49074f29442f5bbffc3f8591cde1dcc849dac"}, false)
	res = <-ch
	c.Assert(res, IsNil)
}

func (s *DownloaderSuite) TestDownload404(c *C) {
	d := NewDownloader(2, 0, s.progress)
	defer d.Shutdown()
	ch := make(chan error)

	d.Download(s.url+"/doesntexist", s.tempfile.Name(), ch)
	res := <-ch
	c.Assert(res, ErrorMatches, "HTTP code 404.*")
}

func (s *DownloaderSuite) TestDownloadConnectError(c *C) {
	d := NewDownloader(2, 0, s.progress)
	defer d.Shutdown()
	ch := make(chan error)

	d.Download("http://nosuch.localhost/", s.tempfile.Name(), ch)
	res := <-ch
	c.Assert(res, ErrorMatches, ".*no such host")
}

func (s *DownloaderSuite) TestDownloadFileError(c *C) {
	d := NewDownloader(2, 0, s.progress)
	defer d.Shutdown()
	ch := make(chan error)

	d.Download(s.url+"/test", "/", ch)
	res := <-ch
	c.Assert(res, ErrorMatches, ".*permission denied")
}

func (s *DownloaderSuite) TestDownloadTemp(c *C) {
	d := NewDownloader(2, 0, s.progress)
	defer d.Shutdown()

	f, err := DownloadTemp(d, s.url+"/test")
	c.Assert(err, IsNil)
	defer f.Close()

	buf := make([]byte, 1)

	f.Read(buf)
	c.Assert(buf, DeepEquals, []byte("H"))

	_, err = os.Stat(f.Name())
	c.Assert(os.IsNotExist(err), Equals, true)
}

func (s *DownloaderSuite) TestDownloadTempWithChecksum(c *C) {
	d := NewDownloader(2, 0, s.progress)
	defer d.Shutdown()

	f, err := DownloadTempWithChecksum(d, s.url+"/test", utils.ChecksumInfo{Size: 12, MD5: "a1acb0fe91c7db45ec4d775192ec5738",
		SHA1: "921893bae6ad6fd818401875d6779254ef0ff0ec", SHA256: "b3c92ee1246176ed35f6e8463cd49074f29442f5bbffc3f8591cde1dcc849dac"}, false)
	defer f.Close()
	c.Assert(err, IsNil)

	_, err = DownloadTempWithChecksum(d, s.url+"/test", utils.ChecksumInfo{Size: 13}, false)
	c.Assert(err, ErrorMatches, ".*size check mismatch 12 != 13")
}

func (s *DownloaderSuite) TestDownloadTempError(c *C) {
	d := NewDownloader(2, 0, s.progress)
	defer d.Shutdown()

	f, err := DownloadTemp(d, s.url+"/doesntexist")
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

	expectedChecksums := map[string]utils.ChecksumInfo{
		"file.bz2": utils.ChecksumInfo{Size: int64(len(bzipData))},
		"file.gz":  utils.ChecksumInfo{Size: int64(len(gzipData))},
		"file":     utils.ChecksumInfo{Size: int64(len(rawData))},
	}

	// bzip2 only available
	buf = make([]byte, 4)
	d := NewFakeDownloader()
	d.ExpectResponse("http://example.com/file.bz2", bzipData)
	r, file, err := DownloadTryCompression(d, "http://example.com/file", expectedChecksums, false)
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
	r, file, err = DownloadTryCompression(d, "http://example.com/file", expectedChecksums, false)
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
	r, file, err = DownloadTryCompression(d, "http://example.com/file", expectedChecksums, false)
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
	r, file, err = DownloadTryCompression(d, "http://example.com/file", nil, false)
	c.Assert(err, IsNil)
	defer file.Close()
	io.ReadFull(r, buf)
	c.Assert(string(buf), Equals, "reco")
	c.Assert(d.Empty(), Equals, true)
}

func (s *DownloaderSuite) TestDownloadTryCompressionErrors(c *C) {
	d := NewFakeDownloader()
	_, _, err := DownloadTryCompression(d, "http://example.com/file", nil, false)
	c.Assert(err, ErrorMatches, "unexpected request.*")

	d = NewFakeDownloader()
	d.ExpectError("http://example.com/file.bz2", errors.New("404"))
	d.ExpectError("http://example.com/file.gz", errors.New("404"))
	d.ExpectError("http://example.com/file", errors.New("403"))
	_, _, err = DownloadTryCompression(d, "http://example.com/file", nil, false)
	c.Assert(err, ErrorMatches, "403")

	d = NewFakeDownloader()
	d.ExpectError("http://example.com/file.bz2", errors.New("404"))
	d.ExpectError("http://example.com/file.gz", errors.New("404"))
	d.ExpectResponse("http://example.com/file", rawData)
	_, _, err = DownloadTryCompression(d, "http://example.com/file", map[string]utils.ChecksumInfo{"file": utils.ChecksumInfo{Size: 7}}, false)
	c.Assert(err, ErrorMatches, "checksums don't match.*")
}
