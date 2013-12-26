package debian

import (
	"errors"
	. "launchpad.net/gocheck"
)

type PackageListSuite struct {
	list                   *PackageList
	p1, p2, p3, p4, p5, p6 *Package
}

var _ = Suite(&PackageListSuite{})

func (s *PackageListSuite) SetUpTest(c *C) {
	s.list = NewPackageList()

	s.p1 = NewPackageFromControlFile(packageStanza.Copy())
	s.p2 = NewPackageFromControlFile(packageStanza.Copy())
	stanza := packageStanza.Copy()
	stanza["Package"] = "mars-invaders"
	s.p3 = NewPackageFromControlFile(stanza)
	stanza = packageStanza.Copy()
	stanza["Size"] = "42"
	s.p4 = NewPackageFromControlFile(stanza)
	stanza = packageStanza.Copy()
	stanza["Package"] = "lonely-strangers"
	s.p5 = NewPackageFromControlFile(stanza)
	stanza = packageStanza.Copy()
	stanza["Version"] = "99.1"
	s.p6 = NewPackageFromControlFile(stanza)
}

func (s *PackageListSuite) TestAddLen(c *C) {
	c.Check(s.list.Len(), Equals, 0)
	c.Check(s.list.Add(s.p1), IsNil)
	c.Check(s.list.Len(), Equals, 1)
	c.Check(s.list.Add(s.p2), IsNil)
	c.Check(s.list.Len(), Equals, 1)
	c.Check(s.list.Add(s.p3), IsNil)
	c.Check(s.list.Len(), Equals, 2)
	c.Check(s.list.Add(s.p4), ErrorMatches, "conflict in package.*")
}

func (s *PackageListSuite) TestForeach(c *C) {
	s.list.Add(s.p1)
	s.list.Add(s.p3)

	Len := 0
	err := s.list.ForEach(func(*Package) error {
		Len++
		return nil
	})

	c.Check(Len, Equals, 2)
	c.Check(err, IsNil)

	e := errors.New("a")

	err = s.list.ForEach(func(*Package) error {
		return e
	})

	c.Check(err, Equals, e)

}

func (s *PackageListSuite) TestNewPackageRefList(c *C) {
	s.list.Add(s.p1)
	s.list.Add(s.p3)
	s.list.Add(s.p5)
	s.list.Add(s.p6)

	reflist := NewPackageRefListFromPackageList(s.list)
	c.Assert(reflist.Len(), Equals, 4)
	c.Check(reflist.Refs[0], DeepEquals, []byte(s.p1.Key()))
	c.Check(reflist.Refs[1], DeepEquals, []byte(s.p6.Key()))
	c.Check(reflist.Refs[2], DeepEquals, []byte(s.p5.Key()))
	c.Check(reflist.Refs[3], DeepEquals, []byte(s.p3.Key()))
}

func (s *PackageListSuite) TestPackageRefListEncodeDecode(c *C) {
	s.list.Add(s.p1)
	s.list.Add(s.p3)
	s.list.Add(s.p5)
	s.list.Add(s.p6)

	reflist := NewPackageRefListFromPackageList(s.list)

	reflist2 := &PackageRefList{}
	err := reflist2.Decode(reflist.Encode())
	c.Assert(err, IsNil)
	c.Check(reflist2.Len(), Equals, reflist.Len())
	c.Check(reflist2.Refs, DeepEquals, reflist.Refs)
}

func (s *PackageListSuite) TestPackageRefListForeach(c *C) {
	s.list.Add(s.p1)
	s.list.Add(s.p3)
	s.list.Add(s.p5)
	s.list.Add(s.p6)

	reflist := NewPackageRefListFromPackageList(s.list)

	Len := 0
	err := reflist.ForEach(func([]byte) error {
		Len++
		return nil
	})

	c.Check(Len, Equals, 4)
	c.Check(err, IsNil)

	e := errors.New("b")

	err = reflist.ForEach(func([]byte) error {
		return e
	})

	c.Check(err, Equals, e)
}
