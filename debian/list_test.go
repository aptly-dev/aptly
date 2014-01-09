package debian

import (
	"errors"
	"github.com/smira/aptly/database"
	. "launchpad.net/gocheck"
	"sort"
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

func (s *PackageListSuite) TestNewPackageListFromRefList(c *C) {
	db, _ := database.OpenDB(c.MkDir())
	coll := NewPackageCollection(db)
	coll.Update(s.p1)
	coll.Update(s.p3)

	s.list.Add(s.p1)
	s.list.Add(s.p3)
	s.list.Add(s.p5)
	s.list.Add(s.p6)

	reflist := NewPackageRefListFromPackageList(s.list)

	_, err := NewPackageListFromRefList(reflist, coll)
	c.Assert(err, ErrorMatches, "unable to load package with key.*")

	coll.Update(s.p5)
	coll.Update(s.p6)

	list, err := NewPackageListFromRefList(reflist, coll)
	c.Assert(err, IsNil)
	c.Check(list.Len(), Equals, 4)
	c.Check(list.Add(s.p4), ErrorMatches, "conflict in package.*")
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

type PackageIndexedListSuite struct {
	packages []*Package
	pl       *PackageList
	list     *PackageIndexedList
}

var _ = Suite(&PackageIndexedListSuite{})

func (s *PackageIndexedListSuite) SetUpTest(c *C) {
	s.pl = NewPackageList()
	s.packages = []*Package{
		&Package{Name: "lib", Version: "1.0", Architecture: "i386", PreDepends: []string{"dpkg (>= 1.6)"}, Depends: []string{"mail-agent"}},
		&Package{Name: "dpkg", Version: "1.7", Architecture: "i386"},
		&Package{Name: "data", Version: "1.1~bp1", Architecture: "all", PreDepends: []string{"dpkg (>= 1.6)"}},
		&Package{Name: "app", Version: "1.1~bp1", Architecture: "i386", PreDepends: []string{"dpkg (>= 1.6)"}, Depends: []string{"lib (>> 0.9)", "data (>= 1.0)"}},
		&Package{Name: "mailer", Version: "3.5.8", Architecture: "i386", Provides: "mail-agent"},
		&Package{Name: "app", Version: "1.1~bp1", Architecture: "amd64", PreDepends: []string{"dpkg (>= 1.6)"}, Depends: []string{"lib (>> 0.9)", "data (>= 1.0)"}},
		&Package{Name: "app", Version: "1.1~bp1", Architecture: "arm", PreDepends: []string{"dpkg (>= 1.6)"}, Depends: []string{"lib (>> 0.9) | libx (>= 1.5)", "data (>= 1.0) | mail-agent"}},
		&Package{Name: "app", Version: "1.0", Architecture: "s390", PreDepends: []string{"dpkg >= 1.6)"}, Depends: []string{"lib (>> 0.9)", "data (>= 1.0)"}},
		&Package{Name: "aa", Version: "2.0-1", Architecture: "i386", PreDepends: []string{"dpkg (>= 1.6)"}},
		&Package{Name: "dpkg", Version: "1.6.1-3", Architecture: "amd64"},
		&Package{Name: "libx", Version: "1.5", Architecture: "arm", PreDepends: []string{"dpkg (>= 1.6)"}},
		&Package{Name: "dpkg", Version: "1.6.1-3", Architecture: "arm"},
	}
	for _, p := range s.packages {
		s.pl.Add(p)
	}

	s.list = NewPackageIndexedList()
	s.list.Append(s.pl)
	s.list.PrepareIndex()
}

func (s *PackageIndexedListSuite) TestIndex(c *C) {
	c.Check(len(s.list.providesList), Equals, 1)
	c.Check(len(s.list.providesList["mail-agent"]), Equals, 1)
	c.Check(s.list.packages[0], Equals, s.packages[8])
}

func (s *PackageIndexedListSuite) TestSearch(c *C) {
	c.Check(s.list.Search(Dependency{Architecture: "i386", Pkg: "app"}), Equals, s.packages[3])
	c.Check(s.list.Search(Dependency{Architecture: "i386", Pkg: "mail-agent"}), Equals, s.packages[4])
	c.Check(s.list.Search(Dependency{Architecture: "i386", Pkg: "puppy"}), IsNil)

	c.Check(s.list.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionEqual, Version: "1.1~bp1"}), Equals, s.packages[3])
	c.Check(s.list.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionEqual, Version: "1.1~bp2"}), IsNil)

	c.Check(s.list.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionLess, Version: "1.1"}), Equals, s.packages[3])
	c.Check(s.list.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionLess, Version: "1.1~~"}), IsNil)

	c.Check(s.list.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionLessOrEqual, Version: "1.1"}), Equals, s.packages[3])
	c.Check(s.list.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionLessOrEqual, Version: "1.1~bp1"}), Equals, s.packages[3])
	c.Check(s.list.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionLessOrEqual, Version: "1.1~~"}), IsNil)

	c.Check(s.list.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionGreater, Version: "1.0"}), Equals, s.packages[3])
	c.Check(s.list.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionGreater, Version: "1.2"}), IsNil)

	c.Check(s.list.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionGreaterOrEqual, Version: "1.0"}), Equals, s.packages[3])
	c.Check(s.list.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionGreaterOrEqual, Version: "1.1~bp1"}), Equals, s.packages[3])
	c.Check(s.list.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionGreaterOrEqual, Version: "1.2"}), IsNil)
}

func (s *PackageIndexedListSuite) TestVerifyDependencies(c *C) {
	missing, err := s.pl.VerifyDependencies(0, []string{"i386"}, s.list)

	c.Check(err, IsNil)
	c.Check(missing, DeepEquals, []Dependency{})

	missing, err = s.pl.VerifyDependencies(0, []string{"i386", "amd64"}, s.list)

	c.Check(err, IsNil)
	c.Check(missing, DeepEquals, []Dependency{Dependency{Pkg: "lib", Relation: VersionGreater, Version: "0.9", Architecture: "amd64"}})

	missing, err = s.pl.VerifyDependencies(0, []string{"arm"}, s.list)

	c.Check(err, IsNil)
	c.Check(missing, DeepEquals, []Dependency{})

	missing, err = s.pl.VerifyDependencies(DepFollowAllVariants, []string{"arm"}, s.list)

	c.Check(err, IsNil)
	c.Check(missing, DeepEquals, []Dependency{Dependency{Pkg: "lib", Relation: VersionGreater, Version: "0.9", Architecture: "arm"},
		Dependency{Pkg: "mail-agent", Relation: VersionDontCare, Version: "", Architecture: "arm"}})

	_, err = s.pl.VerifyDependencies(0, []string{"i386", "amd64", "s390"}, s.list)

	c.Check(err, ErrorMatches, "unable to process package app-1.0_s390:.*")
}

func (s *PackageIndexedListSuite) TestArchitectures(c *C) {
	archs := s.pl.Architectures()
	sort.Strings(archs)
	c.Check(archs, DeepEquals, []string{"amd64", "arm", "i386", "s390"})
}
