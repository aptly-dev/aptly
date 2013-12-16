package utils

import (
	. "launchpad.net/gocheck"
)

type ListSuite struct {
}

var _ = Suite(&ListSuite{})

func (s *ListSuite) TestStringsIsSubset(c *C) {
	err := StringsIsSubset([]string{"a", "b"}, []string{"a", "b", "c"}, "[%s]")
	c.Assert(err, IsNil)

	err = StringsIsSubset([]string{"b", "a"}, []string{"b", "c"}, "[%s]")
	c.Assert(err, ErrorMatches, "\\[a\\]")
}
