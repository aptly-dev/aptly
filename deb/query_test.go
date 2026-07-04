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

func (s *QuerySuite) TestVersionCompareGreater(c *C) {
	q := FieldQuery{"Version", VersionGreater, "5.0.0.2", nil}

	p100 := Package{}
	p100.Version = "5.0.0.100"

	p1 := Package{}
	p1.Version = "5.0.0.1"

	c.Check(q.Matches(&p100), Equals, true)
	c.Check(q.Matches(&p1), Equals, false)

	// invalid version on either side of the comparison must not match
	pInvalid := Package{}
	pInvalid.Version = "1.2.3-"
	c.Check(q.Matches(&pInvalid), Equals, false)
}

func (s *QuerySuite) TestVersionCompareGreaterOrEqual(c *C) {
	q := FieldQuery{"Version", VersionGreaterOrEqual, "5.0.0.2", nil}

	p100 := Package{}
	p100.Version = "5.0.0.100"

	pEqual := Package{}
	pEqual.Version = "5.0.0.2"

	p1 := Package{}
	p1.Version = "5.0.0.1"

	c.Check(q.Matches(&p100), Equals, true)
	c.Check(q.Matches(&pEqual), Equals, true)
	c.Check(q.Matches(&p1), Equals, false)

	// invalid version on either side of the comparison must not match
	pInvalid := Package{}
	pInvalid.Version = "1.2.3-"
	c.Check(q.Matches(&pInvalid), Equals, false)
}
