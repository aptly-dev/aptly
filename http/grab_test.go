package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/console"
	"github.com/aptly-dev/aptly/utils"

	. "gopkg.in/check.v1"
)

type GrabDownloaderSuiteBase struct {
	tempfile *os.File
	l        net.Listener
	url      string
	ch       chan struct{}
	progress aptly.Progress
	d        aptly.Downloader
	ctx      context.Context
}

func (s *GrabDownloaderSuiteBase) SetUpTest(c *C) {
	s.tempfile, _ = os.CreateTemp(os.TempDir(), "aptly-test")
	s.l, _ = net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	s.url = fmt.Sprintf("http://localhost:%d", s.l.Addr().(*net.TCPAddr).Port)

	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %s", r.URL.Path)
	})

	s.ch = make(chan struct{})

	go func() {
		http.Serve(s.l, mux)
		close(s.ch)
	}()

	s.progress = console.NewProgress(false)
	s.progress.Start()

	s.d = NewGrabDownloader(0, 1, s.progress)
	s.ctx = context.Background()
}

func (s *GrabDownloaderSuiteBase) TearDownTest(c *C) {
	s.progress.Shutdown()

	s.l.Close()
	<-s.ch

	os.Remove(s.tempfile.Name())
	s.tempfile.Close()
}

type GrabDownloaderSuite struct {
	GrabDownloaderSuiteBase
}

var _ = Suite(&GrabDownloaderSuite{})

func (s *GrabDownloaderSuite) SetUpTest(c *C) {
	s.GrabDownloaderSuiteBase.SetUpTest(c)
}

func (s *GrabDownloaderSuite) TearDownTest(c *C) {
	s.GrabDownloaderSuiteBase.TearDownTest(c)
}

func (s *GrabDownloaderSuite) TestDownloadOK(c *C) {
	c.Assert(s.d.Download(s.ctx, s.url+"/test", s.tempfile.Name()), IsNil)
}

func (s *GrabDownloaderSuite) TestDownloadWithChecksum(c *C) {
	c.Assert(s.d.DownloadWithChecksum(s.ctx, s.url+"/test", s.tempfile.Name(), &utils.ChecksumInfo{Size: 1}, false),
		// ErrorMatches, ".*size check mismatch 12 != 1")
		ErrorMatches, "bad content length")

	c.Assert(s.d.DownloadWithChecksum(s.ctx, s.url+"/test", s.tempfile.Name(), &utils.ChecksumInfo{Size: 12, MD5: "abcdef"}, false),
		ErrorMatches, "checksum mismatch")

	c.Assert(s.d.DownloadWithChecksum(s.ctx, s.url+"/test", s.tempfile.Name(), &utils.ChecksumInfo{Size: 12, MD5: "abcdef"}, true),
		IsNil)

	c.Assert(s.d.DownloadWithChecksum(s.ctx, s.url+"/test", s.tempfile.Name(), &utils.ChecksumInfo{Size: 12, MD5: "a1acb0fe91c7db45ec4d775192ec5738"}, false),
		IsNil)

	c.Assert(s.d.DownloadWithChecksum(s.ctx, s.url+"/test", s.tempfile.Name(), &utils.ChecksumInfo{Size: 12, MD5: "a1acb0fe91c7db45ec4d775192ec5738",
		SHA1: "921893bae6ad6fd818401875d6779254ef0ff0ec"}, false),
		IsNil)

	checksums := utils.ChecksumInfo{Size: 12, MD5: "a1acb0fe91c7db45ec4d775192ec5738",
		SHA1: "921893bae6ad6fd818401875d6779254ef0ff0ec", SHA256: "b3c92ee1246176ed35f6e8463cd49074f29442f5bbffc3f8591cde1dcc849dac"}
	c.Assert(s.d.DownloadWithChecksum(s.ctx, s.url+"/test", s.tempfile.Name(), &checksums, false),
		IsNil)
}

func (s *GrabDownloaderSuite) TestDownload404(c *C) {
	c.Assert(s.d.Download(s.ctx, s.url+"/doesntexist", s.tempfile.Name()),
		ErrorMatches, ".* 404 .*")
}

func (s *GrabDownloaderSuite) TestDownloadConnectError(c *C) {
	c.Assert(s.d.Download(s.ctx, "http://nosuch.host.invalid./", s.tempfile.Name()),
		ErrorMatches, ".*no such host")
}

func (s *GrabDownloaderSuite) TestDownloadFileError(c *C) {
	skipIfRoot(c)
	c.Assert(s.d.Download(s.ctx, s.url+"/test", "/"),
		ErrorMatches, ".*(permission denied|read-only file system)")
}

func (s *GrabDownloaderSuite) TestGetLength(c *C) {
	size, err := s.d.GetLength(s.ctx, s.url+"/test")

	c.Assert(err, IsNil)
	c.Assert(size, Equals, int64(12))
}

func (s *GrabDownloaderSuite) TestGetLength404(c *C) {
	_, err := s.d.GetLength(s.ctx, s.url+"/doesntexist")

	c.Assert(err, ErrorMatches, "HTTP code 404.*")
}

func (s *GrabDownloaderSuite) TestGetLengthConnectError(c *C) {
	_, err := s.d.GetLength(s.ctx, "http://nosuch.host.invalid./")

	c.Assert(err, ErrorMatches, ".*no such host")
}
