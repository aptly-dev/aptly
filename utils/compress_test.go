package utils

import (
	"compress/bzip2"
	"compress/gzip"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"os"
)

type CompressSuite struct {
	tempfile *os.File
}

var _ = Suite(&CompressSuite{})

const testString = "Quick brown fox jumps over black dog and runs away... Really far away... who knows?"

func (s *CompressSuite) SetUpTest(c *C) {
	s.tempfile, _ = ioutil.TempFile(c.MkDir(), "aptly-test")
	s.tempfile.WriteString(testString)
}

func (s *CompressSuite) TearDownTest(c *C) {
	s.tempfile.Close()
}

func (s *CompressSuite) TestCompress(c *C) {
	err := CompressFile(s.tempfile)
	c.Assert(err, IsNil)

	buf := make([]byte, len(testString))

	file, err := os.Open(s.tempfile.Name() + ".gz")
	c.Assert(err, IsNil)

	gzReader, err := gzip.NewReader(file)
	c.Assert(err, IsNil)

	_, err = gzReader.Read(buf)
	c.Assert(err, IsNil)

	gzReader.Close()
	file.Close()

	c.Check(string(buf), Equals, testString)

	file, err = os.Open(s.tempfile.Name() + ".bz2")
	c.Assert(err, IsNil)

	bzReader := bzip2.NewReader(file)

	_, err = bzReader.Read(buf)
	c.Assert(err, IsNil)

	file.Close()

	c.Check(string(buf), Equals, testString)
}
