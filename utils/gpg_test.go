package utils

import (
	. "gopkg.in/check.v1"
)

type GpgSuite struct{}

var _ = Suite(&GpgSuite{})

func (s *GpgSuite) TestGpgKeyMatch(c *C) {
	c.Check(GpgKey("EC4B033C70096AD1").Matches(GpgKey("EC4B033C70096AD1")), Equals, true)
	c.Check(GpgKey("37E1C17570096AD1").Matches(GpgKey("EC4B033C70096AD1")), Equals, false)

	c.Check(GpgKey("70096AD1").Matches(GpgKey("70096AD1")), Equals, true)
	c.Check(GpgKey("70096AD1").Matches(GpgKey("EC4B033C")), Equals, false)

	c.Check(GpgKey("37E1C17570096AD1").Matches(GpgKey("70096AD1")), Equals, true)
	c.Check(GpgKey("70096AD1").Matches(GpgKey("EC4B033C70096AD1")), Equals, true)
}
