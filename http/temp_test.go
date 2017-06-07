package http

import (
	"os"

	"github.com/smira/aptly/utils"

	. "gopkg.in/check.v1"
)

type TempSuite struct {
	DownloaderSuiteBase
}

var _ = Suite(&TempSuite{})

func (s *TempSuite) SetUpTest(c *C) {
	s.DownloaderSuiteBase.SetUpTest(c)
}

func (s *TempSuite) TearDownTest(c *C) {
	s.DownloaderSuiteBase.TearDownTest(c)
}

func (s *TempSuite) TestDownloadTemp(c *C) {
	f, err := DownloadTemp(s.d, s.url+"/test")
	c.Assert(err, IsNil)
	defer f.Close()

	buf := make([]byte, 1)

	f.Read(buf)
	c.Assert(buf, DeepEquals, []byte("H"))

	_, err = os.Stat(f.Name())
	c.Assert(os.IsNotExist(err), Equals, true)
}

func (s *TempSuite) TestDownloadTempWithChecksum(c *C) {
	f, err := DownloadTempWithChecksum(s.d, s.url+"/test", &utils.ChecksumInfo{Size: 12, MD5: "a1acb0fe91c7db45ec4d775192ec5738",
		SHA1: "921893bae6ad6fd818401875d6779254ef0ff0ec", SHA256: "b3c92ee1246176ed35f6e8463cd49074f29442f5bbffc3f8591cde1dcc849dac"}, false, 1)
	c.Assert(err, IsNil)

	c.Assert(f.Close(), IsNil)

	_, err = DownloadTempWithChecksum(s.d, s.url+"/test", &utils.ChecksumInfo{Size: 13}, false, 1)
	c.Assert(err, ErrorMatches, ".*size check mismatch 12 != 13")
}

func (s *TempSuite) TestDownloadTempError(c *C) {
	f, err := DownloadTemp(s.d, s.url+"/doesntexist")
	c.Assert(err, NotNil)
	c.Assert(f, IsNil)
	c.Assert(err, ErrorMatches, "HTTP code 404.*")
}
