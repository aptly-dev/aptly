package utils

import (
	"os"
	"path/filepath"

	. "gopkg.in/check.v1"
)

type CopyfileSuite struct {
	source *os.File
	dest   string
}

var _ = Suite(&CopyfileSuite{})

func (s *CopyfileSuite) SetUpSuite(c *C) {
	s.source, _ = os.CreateTemp(c.MkDir(), "source-file")
	s.dest = filepath.Join(filepath.Dir(s.source.Name()), "destination-file")
}

func (s *CopyfileSuite) TestCopyFile(c *C) {
	err := CopyFile(s.source.Name(), s.dest)
	c.Check(err, Equals, nil)

	_, err = os.Stat(s.dest)
	c.Check(err, Equals, nil)
}
