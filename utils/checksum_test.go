package utils

import (
	"os"
	"strings"

	. "gopkg.in/check.v1"
)

type ChecksumSuite struct {
	tempfile *os.File

	tempfileLarge *os.File
}

var _ = Suite(&ChecksumSuite{})

func (s *ChecksumSuite) SetUpTest(c *C) {
	s.tempfile, _ = os.CreateTemp(c.MkDir(), "aptly-test")
	s.tempfileLarge, _ = os.CreateTemp(c.MkDir(), "aptly-test-large")

	_, _ = s.tempfile.WriteString(testString)
	_, _ = s.tempfileLarge.WriteString(strings.Repeat(testString, 1000))
}

func (s *ChecksumSuite) TearDownTest(c *C) {
	_ = s.tempfile.Close()
	_ = s.tempfileLarge.Close()
}

func (s *ChecksumSuite) TestChecksumsForFile(c *C) {
	info, err := ChecksumsForFile(s.tempfile.Name(), nil)

	c.Assert(err, IsNil)
	c.Check(info.Size, Equals, int64(83))
	c.Check(info.MD5, Equals, "43470766afbfdca292440eecdceb80fb")
	c.Check(info.SHA1, Equals, "1743f8408261b4f1eff88e0fca15a7077223fa79")
	c.Check(info.SHA256, Equals, "f2775692fd3b70bd0faa4054b7afa92d427bf994cd8629741710c4864ee4dc95")

	var buf [512 * 1024]byte
	infoLarge, err := ChecksumsForFile(s.tempfileLarge.Name(), buf[:])
	c.Assert(err, IsNil)
	c.Check(infoLarge.Size, Equals, int64(83000))
	c.Check(infoLarge.MD5, Equals, "6998f2565c7f9b614adbdf25aaa5ff60")
	c.Check(infoLarge.SHA1, Equals, "3905bc72d4df3e9b42a5fbb23d5dde26b3bc4d31")
	c.Check(infoLarge.SHA256, Equals, "b5a7126d6c41a71122728837452f54c0979cd004d5c572f3365ae875085eb50d")
}

func (s *ChecksumSuite) TestMD5ChecksumForFile(c *C) {
	md5sum, err := MD5ChecksumForFile(s.tempfile.Name())

	c.Assert(err, IsNil)
	c.Check(md5sum, Equals, "43470766afbfdca292440eecdceb80fb")
}
