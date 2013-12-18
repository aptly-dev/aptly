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

func (s *ListSuite) TestStrSlicesEqual(c *C) {
	c.Check(StrSlicesEqual(nil, nil), Equals, true)
	c.Check(StrSlicesEqual(nil, []string{}), Equals, true)
	c.Check(StrSlicesEqual([]string{}, nil), Equals, true)
	c.Check(StrSlicesEqual([]string{"a", "b"}, []string{"a", "b"}), Equals, true)

	c.Check(StrSlicesEqual(nil, []string{"a"}), Equals, false)
	c.Check(StrSlicesEqual([]string{"a", "c"}, []string{"a", "b"}), Equals, false)
}

func (s *ListSuite) TestStrMapsEqual(c *C) {
	c.Check(StrMapsEqual(map[string]string{}, nil), Equals, true)
	c.Check(StrMapsEqual(nil, map[string]string{}), Equals, true)
	c.Check(StrMapsEqual(nil, nil), Equals, true)
	c.Check(StrMapsEqual(map[string]string{"a": "1", "b": "2"}, map[string]string{"a": "1", "b": "2"}), Equals, true)

	c.Check(StrMapsEqual(map[string]string{"a": "1", "b": "2"}, map[string]string{"a": "1", "b": "3"}), Equals, false)
	c.Check(StrMapsEqual(map[string]string{"a": "1", "b": "2"}, map[string]string{"a": "1", "c": "2"}), Equals, false)
	c.Check(StrMapsEqual(map[string]string{"a": "1", "b": "2"}, map[string]string{"a": "1"}), Equals, false)
}
