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

func (s *ListSuite) TestStrSliceHasIteml(c *C) {
	c.Check(StrSliceHasItem([]string{"a", "b"}, "b"), Equals, true)
	c.Check(StrSliceHasItem([]string{"a", "b"}, "c"), Equals, false)
}

func (s *ListSuite) TestStrMapSortedKeys(c *C) {
	c.Check(StrMapSortedKeys(map[string]string{}), DeepEquals, []string{})
	c.Check(StrMapSortedKeys(map[string]string{"x": "1", "a": "3", "y": "4"}), DeepEquals, []string{"a", "x", "y"})
}

func (s *ListSuite) TestStrSliceDeduplicate(c *C) {
	c.Check(StrSliceDeduplicate([]string{}), DeepEquals, []string{})
	c.Check(StrSliceDeduplicate([]string{"a"}), DeepEquals, []string{"a"})
	c.Check(StrSliceDeduplicate([]string{"a", "b"}), DeepEquals, []string{"a", "b"})
	c.Check(StrSliceDeduplicate([]string{"a", "a"}), DeepEquals, []string{"a"})
	c.Check(StrSliceDeduplicate([]string{"a", "b", "c", "a", "a", "b"}), DeepEquals, []string{"a", "b", "c"})
	c.Check(StrSliceDeduplicate([]string{"a", "b", "c", "d", "e", "f"}), DeepEquals, []string{"a", "b", "c", "d", "e", "f"})
}
