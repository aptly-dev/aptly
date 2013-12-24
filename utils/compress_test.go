package utils

import (
	"io/ioutil"
	. "launchpad.net/gocheck"
	"os"
)

type CompressSuite struct {
	tempfile *os.File
}

var _ = Suite(&CompressSuite{})

func (s *CompressSuite) SetUpTest(c *C) {
	s.tempfile, _ = ioutil.TempFile(c.MkDir(), "aptly-test")
	s.tempfile.WriteString("Quick brown fox jumps over black dog and runs away... Really far away... who knows?")
}

func (s *CompressSuite) TearDownTest(c *C) {
	s.tempfile.Close()
}

func (s *CompressSuite) TestCompress(c *C) {
	err := CompressFile(s.tempfile)
	c.Assert(err, IsNil)

	st, err := os.Stat(s.tempfile.Name() + ".gz")
	c.Assert(err, IsNil)
	c.Assert(st.Size() < 100, Equals, true)

	st, err = os.Stat(s.tempfile.Name() + ".bz2")
	c.Assert(err, IsNil)
	c.Assert(st.Size() < 120, Equals, true)
}
