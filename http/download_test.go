package http

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"

	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/console"
	"github.com/smira/aptly/utils"

	. "gopkg.in/check.v1"
)

type DownloaderSuiteBase struct {
	tempfile *os.File
	l        net.Listener
	url      string
	ch       chan struct{}
	progress aptly.Progress
	d        aptly.Downloader
}

func (s *DownloaderSuiteBase) SetUpTest(c *C) {
	s.tempfile, _ = ioutil.TempFile(os.TempDir(), "aptly-test")
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

	s.progress = console.NewProgress()
	s.progress.Start()

	s.d = NewDownloader(0, s.progress)
}

func (s *DownloaderSuiteBase) TearDownTest(c *C) {
	s.progress.Shutdown()

	s.l.Close()
	<-s.ch

	os.Remove(s.tempfile.Name())
	s.tempfile.Close()
}

type DownloaderSuite struct {
	DownloaderSuiteBase
}

var _ = Suite(&DownloaderSuite{})

func (s *DownloaderSuite) SetUpTest(c *C) {
	s.DownloaderSuiteBase.SetUpTest(c)
}

func (s *DownloaderSuite) TearDownTest(c *C) {
	s.DownloaderSuiteBase.TearDownTest(c)
}

func (s *DownloaderSuite) TestDownloadOK(c *C) {
	c.Assert(s.d.Download(s.url+"/test", s.tempfile.Name()), IsNil)
}

func (s *DownloaderSuite) TestDownloadWithChecksum(c *C) {
	c.Assert(s.d.DownloadWithChecksum(s.url+"/test", s.tempfile.Name(), &utils.ChecksumInfo{}, false, 1),
		ErrorMatches, ".*size check mismatch 12 != 0")

	c.Assert(s.d.DownloadWithChecksum(s.url+"/test", s.tempfile.Name(), &utils.ChecksumInfo{Size: 12, MD5: "abcdef"}, false, 1),
		ErrorMatches, ".*md5 hash mismatch \"a1acb0fe91c7db45ec4d775192ec5738\" != \"abcdef\"")

	c.Assert(s.d.DownloadWithChecksum(s.url+"/test", s.tempfile.Name(), &utils.ChecksumInfo{Size: 12, MD5: "abcdef"}, true, 1),
		IsNil)

	c.Assert(s.d.DownloadWithChecksum(s.url+"/test", s.tempfile.Name(), &utils.ChecksumInfo{Size: 12, MD5: "a1acb0fe91c7db45ec4d775192ec5738"}, false, 1),
		IsNil)

	c.Assert(s.d.DownloadWithChecksum(s.url+"/test", s.tempfile.Name(), &utils.ChecksumInfo{Size: 12, MD5: "a1acb0fe91c7db45ec4d775192ec5738", SHA1: "abcdef"}, false, 1),
		ErrorMatches, ".*sha1 hash mismatch \"921893bae6ad6fd818401875d6779254ef0ff0ec\" != \"abcdef\"")

	c.Assert(s.d.DownloadWithChecksum(s.url+"/test", s.tempfile.Name(), &utils.ChecksumInfo{Size: 12, MD5: "a1acb0fe91c7db45ec4d775192ec5738",
		SHA1: "921893bae6ad6fd818401875d6779254ef0ff0ec"}, false, 1),
		IsNil)

	c.Assert(s.d.DownloadWithChecksum(s.url+"/test", s.tempfile.Name(), &utils.ChecksumInfo{Size: 12, MD5: "a1acb0fe91c7db45ec4d775192ec5738",
		SHA1: "921893bae6ad6fd818401875d6779254ef0ff0ec", SHA256: "abcdef"}, false, 1),
		ErrorMatches, ".*sha256 hash mismatch \"b3c92ee1246176ed35f6e8463cd49074f29442f5bbffc3f8591cde1dcc849dac\" != \"abcdef\"")

	checksums := utils.ChecksumInfo{Size: 12, MD5: "a1acb0fe91c7db45ec4d775192ec5738",
		SHA1: "921893bae6ad6fd818401875d6779254ef0ff0ec", SHA256: "b3c92ee1246176ed35f6e8463cd49074f29442f5bbffc3f8591cde1dcc849dac"}
	c.Assert(s.d.DownloadWithChecksum(s.url+"/test", s.tempfile.Name(), &checksums, false, 1),
		IsNil)
	// download backfills missing checksums
	c.Check(checksums.SHA512, Equals, "bac18bf4e564856369acc2ed57300fecba3a2c1af5ae8304021e4252488678feb18118466382ee4e1210fe1f065080210e453a80cfb37ccb8752af3269df160e")
}

func (s *DownloaderSuite) TestDownload404(c *C) {
	c.Assert(s.d.Download(s.url+"/doesntexist", s.tempfile.Name()),
		ErrorMatches, "HTTP code 404.*")
}

func (s *DownloaderSuite) TestDownloadConnectError(c *C) {
	c.Assert(s.d.Download("http://nosuch.localhost/", s.tempfile.Name()),
		ErrorMatches, ".*no such host")
}

func (s *DownloaderSuite) TestDownloadFileError(c *C) {
	c.Assert(s.d.Download(s.url+"/test", "/"),
		ErrorMatches, ".*permission denied")
}
