package debian

import (
	. "launchpad.net/gocheck"
	"path/filepath"
	"runtime"
)

type DebSuite struct {
	debFile string
}

var _ = Suite(&DebSuite{})

func (s *DebSuite) SetUpSuite(c *C) {
	_, __file__, _, _ := runtime.Caller(0)
	s.debFile = filepath.Join(filepath.Dir(__file__), "../system/files/libboost-program-options-dev_1.49.0.1_i386.deb")
}

func (s *DebSuite) TestGetControlFileFromDeb(c *C) {
	_, err := GetControlFileFromDeb("/no/such/file")
	c.Check(err, ErrorMatches, ".*no such file or directory")

	_, __file__, _, _ := runtime.Caller(0)
	_, err = GetControlFileFromDeb(__file__)
	c.Check(err, ErrorMatches, "unable to read .deb archive: ar: missing global header")

	st, err := GetControlFileFromDeb(s.debFile)
	c.Check(err, IsNil)
	c.Check(st["Version"], Equals, "1.49.0.1")
	c.Check(st["Package"], Equals, "libboost-program-options-dev")
}
