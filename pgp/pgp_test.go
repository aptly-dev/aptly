package pgp

import (
	"testing"

	. "gopkg.in/check.v1"
)

// Launch gocheck tests
func Test(t *testing.T) {
	TestingT(t)
}

type PGPSuite struct{}

var _ = Suite(&PGPSuite{})

func (s *PGPSuite) TestKeyMatch(c *C) {
	c.Check(Key("EC4B033C70096AD1").Matches(Key("EC4B033C70096AD1")), Equals, true)
	c.Check(Key("37E1C17570096AD1").Matches(Key("EC4B033C70096AD1")), Equals, false)

	c.Check(Key("70096AD1").Matches(Key("70096AD1")), Equals, true)
	c.Check(Key("70096AD1").Matches(Key("EC4B033C")), Equals, false)

	c.Check(Key("37E1C17570096AD1").Matches(Key("70096AD1")), Equals, true)
	c.Check(Key("70096AD1").Matches(Key("EC4B033C70096AD1")), Equals, true)
}
