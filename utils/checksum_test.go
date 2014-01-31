package utils

import (
	"io/ioutil"
	. "launchpad.net/gocheck"
	"os"
)

type ChecksumSuite struct {
	tempfile *os.File
}

var _ = Suite(&ChecksumSuite{})

func (s *ChecksumSuite) SetUpTest(c *C) {
	s.tempfile, _ = ioutil.TempFile(c.MkDir(), "aptly-test")
	s.tempfile.WriteString(testString)
}

func (s *ChecksumSuite) TearDownTest(c *C) {
	s.tempfile.Close()
}

func (s *ChecksumSuite) TestChecksumsForFile(c *C) {
	info, err := ChecksumsForFile(s.tempfile.Name())

	c.Assert(err, IsNil)
	c.Check(info.Size, Equals, int64(83))
	c.Check(info.MD5, Equals, "43470766afbfdca292440eecdceb80fb")
	c.Check(info.SHA1, Equals, "1743f8408261b4f1eff88e0fca15a7077223fa79")
	c.Check(info.SHA256, Equals, "f2775692fd3b70bd0faa4054b7afa92d427bf994cd8629741710c4864ee4dc95")
}
