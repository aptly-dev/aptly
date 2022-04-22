package deb

import (
	. "gopkg.in/check.v1"
)

type QuerySuite struct {
}

var _ = Suite(&QuerySuite{})

func (s *QuerySuite) TestVersionCompare(c *C) {
	q := FieldQuery{"Version", VersionLess, "5.0.0.2", nil}

	p100 := Package{}
	p100.Version = "5.0.0.100"

	p1 := Package{}
	p1.Version = "5.0.0.1"

	c.Check(q.Matches(&p100), Equals, false)
	c.Check(q.Matches(&p1), Equals, true)
}
