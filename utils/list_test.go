package utils

import (
	. "gopkg.in/check.v1"
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

func (s *ListSuite) TestStrSlicesSubstract(c *C) {
	empty := []string(nil)
	l1 := []string{"r1", "r2", "r3", "r4"}
	l2 := []string{"r1", "r3"}
	l3 := []string{"r2", "r4"}
	l4 := []string{"r4", "r5"}
	l5 := []string{"r1", "r2", "r3"}

	c.Check(StrSlicesSubstract(l1, empty), DeepEquals, l1)
	c.Check(StrSlicesSubstract(l1, l2), DeepEquals, l3)
	c.Check(StrSlicesSubstract(l1, l3), DeepEquals, l2)
	c.Check(StrSlicesSubstract(l1, l4), DeepEquals, l5)
	c.Check(StrSlicesSubstract(empty, l1), DeepEquals, empty)
	c.Check(StrSlicesSubstract(l2, l3), DeepEquals, l2)
}
