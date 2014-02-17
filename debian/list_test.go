package debian

import (
	"errors"
	"github.com/smira/aptly/database"
	. "launchpad.net/gocheck"
	"sort"
)

type PackageListSuite struct {
	// Simple list with "real" packages from stanzas
	list                   *PackageList
	p1, p2, p3, p4, p5, p6 *Package

	// Mocked packages in list
	packages []*Package
	il       *PackageList
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

	s.il = NewPackageList()
	s.packages = []*Package{
		&Package{Name: "lib", Version: "1.0", Architecture: "i386", PreDepends: []string{"dpkg (>= 1.6)"}, Depends: []string{"mail-agent"}},
		&Package{Name: "dpkg", Version: "1.7", Architecture: "i386", Source: "dpkg", Provides: []string{"package-installer"}},
		&Package{Name: "data", Version: "1.1~bp1", Architecture: "all", PreDepends: []string{"dpkg (>= 1.6)"}},
		&Package{Name: "app", Version: "1.1~bp1", Architecture: "i386", PreDepends: []string{"dpkg (>= 1.6)"}, Depends: []string{"lib (>> 0.9)", "data (>= 1.0)"}},
		&Package{Name: "mailer", Version: "3.5.8", Architecture: "i386", Provides: []string{"mail-agent"}},
		&Package{Name: "app", Version: "1.1~bp1", Architecture: "amd64", PreDepends: []string{"dpkg (>= 1.6)"}, Depends: []string{"lib (>> 0.9)", "data (>= 1.0)"}},
		&Package{Name: "app", Version: "1.1~bp1", Architecture: "arm", PreDepends: []string{"dpkg (>= 1.6)"}, Depends: []string{"lib (>> 0.9) | libx (>= 1.5)", "data (>= 1.0) | mail-agent"}},
		&Package{Name: "app", Version: "1.0", Architecture: "s390", PreDepends: []string{"dpkg >= 1.6)"}, Depends: []string{"lib (>> 0.9)", "data (>= 1.0)"}},
		&Package{Name: "aa", Version: "2.0-1", Architecture: "i386", PreDepends: []string{"dpkg (>= 1.6)"}},
		&Package{Name: "dpkg", Version: "1.6.1-3", Architecture: "amd64", Source: "dpkg", Provides: []string{"package-installer"}},
		&Package{Name: "libx", Version: "1.5", Architecture: "arm", Source: "libx", PreDepends: []string{"dpkg (>= 1.6)"}},
		&Package{Name: "dpkg", Version: "1.6.1-3", Architecture: "arm", Source: "dpkg", Provides: []string{"package-installer"}},
		&Package{Name: "dpkg", Version: "1.6.1-3", Architecture: "source", SourceArchitecture: "any"},
		&Package{Name: "dpkg", Version: "1.7", Architecture: "source", SourceArchitecture: "any"},
	}
	for _, p := range s.packages {
		s.il.Add(p)
	}
	s.il.PrepareIndex()

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

func (s *PackageListSuite) TestRemove(c *C) {
	c.Check(s.list.Add(s.p1), IsNil)
	c.Check(s.list.Add(s.p3), IsNil)
	c.Check(s.list.Len(), Equals, 2)

	s.list.Remove(s.p1)
	c.Check(s.list.Len(), Equals, 1)
}

func (s *PackageListSuite) TestAddWhenIndexed(c *C) {
	c.Check(s.list.Len(), Equals, 0)
	s.list.PrepareIndex()

	c.Check(s.list.Add(&Package{Name: "a1st", Version: "1.0", Architecture: "i386", Provides: []string{"fa", "fb"}}), IsNil)
	c.Check(s.list.packagesIndex[0].Name, Equals, "a1st")
	c.Check(s.list.providesIndex["fa"][0].Name, Equals, "a1st")
	c.Check(s.list.providesIndex["fb"][0].Name, Equals, "a1st")

	c.Check(s.list.Add(&Package{Name: "c3rd", Version: "1.0", Architecture: "i386", Provides: []string{"fa"}}), IsNil)
	c.Check(s.list.packagesIndex[0].Name, Equals, "a1st")
	c.Check(s.list.packagesIndex[1].Name, Equals, "c3rd")
	c.Check(s.list.providesIndex["fa"][0].Name, Equals, "a1st")
	c.Check(s.list.providesIndex["fa"][1].Name, Equals, "c3rd")
	c.Check(s.list.providesIndex["fb"][0].Name, Equals, "a1st")

	c.Check(s.list.Add(&Package{Name: "b2nd", Version: "1.0", Architecture: "i386"}), IsNil)
	c.Check(s.list.packagesIndex[0].Name, Equals, "a1st")
	c.Check(s.list.packagesIndex[1].Name, Equals, "b2nd")
	c.Check(s.list.packagesIndex[2].Name, Equals, "c3rd")
	c.Check(s.list.providesIndex["fa"][0].Name, Equals, "a1st")
	c.Check(s.list.providesIndex["fa"][1].Name, Equals, "c3rd")
	c.Check(s.list.providesIndex["fb"][0].Name, Equals, "a1st")
}

func (s *PackageListSuite) TestRemoveWhenIndexed(c *C) {
	s.il.Remove(s.packages[0])
	names := make([]string, s.il.Len())
	for i, p := range s.il.packagesIndex {
		names[i] = p.Name
	}
	c.Check(names, DeepEquals, []string{"aa", "app", "app", "app", "app", "data", "dpkg", "dpkg", "dpkg", "dpkg", "dpkg", "libx", "mailer"})

	s.il.Remove(s.packages[4])
	names = make([]string, s.il.Len())
	for i, p := range s.il.packagesIndex {
		names[i] = p.Name
	}
	c.Check(names, DeepEquals, []string{"aa", "app", "app", "app", "app", "data", "dpkg", "dpkg", "dpkg", "dpkg", "dpkg", "libx"})
	c.Check(s.il.providesIndex["mail-agent"], DeepEquals, []*Package{})

	s.il.Remove(s.packages[9])
	names = make([]string, s.il.Len())
	for i, p := range s.il.packagesIndex {
		names[i] = p.Name
	}
	c.Check(names, DeepEquals, []string{"aa", "app", "app", "app", "app", "data", "dpkg", "dpkg", "dpkg", "dpkg", "libx"})
	c.Check(s.il.providesIndex["package-installer"], HasLen, 2)

	s.il.Remove(s.packages[1])
	names = make([]string, s.il.Len())
	for i, p := range s.il.packagesIndex {
		names[i] = p.Name
	}
	c.Check(names, DeepEquals, []string{"aa", "app", "app", "app", "app", "data", "dpkg", "dpkg", "dpkg", "libx"})
	c.Check(s.il.providesIndex["package-installer"], DeepEquals, []*Package{s.packages[11]})
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

func (s *PackageListSuite) TestIndex(c *C) {
	c.Check(len(s.il.providesIndex), Equals, 2)
	c.Check(len(s.il.providesIndex["mail-agent"]), Equals, 1)
	c.Check(len(s.il.providesIndex["package-installer"]), Equals, 3)
	c.Check(s.il.packagesIndex[0], Equals, s.packages[8])
}

func (s *PackageListSuite) TestAppend(c *C) {
	s.list.Add(s.p1)
	s.list.Add(s.p3)

	err := s.list.Append(s.il)
	c.Check(err, IsNil)
	c.Check(s.list.Len(), Equals, 16)

	list := NewPackageList()
	list.Add(s.p4)

	err = s.list.Append(list)
	c.Check(err, ErrorMatches, "conflict.*")

	s.list.PrepareIndex()
	c.Check(func() { s.list.Append(s.il) }, Panics, "Append not supported when indexed")
}

func (s *PackageListSuite) TestSearch(c *C) {
	c.Check(func() { s.list.Search(Dependency{Architecture: "i386", Pkg: "app"}) }, Panics, "list not indexed, can't search")

	c.Check(s.il.Search(Dependency{Architecture: "i386", Pkg: "app"}), Equals, s.packages[3])
	c.Check(s.il.Search(Dependency{Architecture: "i386", Pkg: "mail-agent"}), Equals, s.packages[4])
	c.Check(s.il.Search(Dependency{Architecture: "i386", Pkg: "puppy"}), IsNil)

	c.Check(s.il.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionEqual, Version: "1.1~bp1"}), Equals, s.packages[3])
	c.Check(s.il.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionEqual, Version: "1.1~bp2"}), IsNil)

	c.Check(s.il.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionLess, Version: "1.1"}), Equals, s.packages[3])
	c.Check(s.il.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionLess, Version: "1.1~~"}), IsNil)

	c.Check(s.il.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionLessOrEqual, Version: "1.1"}), Equals, s.packages[3])
	c.Check(s.il.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionLessOrEqual, Version: "1.1~bp1"}), Equals, s.packages[3])
	c.Check(s.il.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionLessOrEqual, Version: "1.1~~"}), IsNil)

	c.Check(s.il.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionGreater, Version: "1.0"}), Equals, s.packages[3])
	c.Check(s.il.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionGreater, Version: "1.2"}), IsNil)

	c.Check(s.il.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionGreaterOrEqual, Version: "1.0"}), Equals, s.packages[3])
	c.Check(s.il.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionGreaterOrEqual, Version: "1.1~bp1"}), Equals, s.packages[3])
	c.Check(s.il.Search(Dependency{Architecture: "i386", Pkg: "app", Relation: VersionGreaterOrEqual, Version: "1.2"}), IsNil)
}

func (s *PackageListSuite) TestVerifyDependencies(c *C) {
	missing, err := s.il.VerifyDependencies(0, []string{"i386"}, s.il)

	c.Check(err, IsNil)
	c.Check(missing, DeepEquals, []Dependency{})

	missing, err = s.il.VerifyDependencies(0, []string{"i386", "amd64"}, s.il)

	c.Check(err, IsNil)
	c.Check(missing, DeepEquals, []Dependency{Dependency{Pkg: "lib", Relation: VersionGreater, Version: "0.9", Architecture: "amd64"}})

	missing, err = s.il.VerifyDependencies(0, []string{"arm"}, s.il)

	c.Check(err, IsNil)
	c.Check(missing, DeepEquals, []Dependency{})

	missing, err = s.il.VerifyDependencies(DepFollowAllVariants, []string{"arm"}, s.il)

	c.Check(err, IsNil)
	c.Check(missing, DeepEquals, []Dependency{Dependency{Pkg: "lib", Relation: VersionGreater, Version: "0.9", Architecture: "arm"},
		Dependency{Pkg: "mail-agent", Relation: VersionDontCare, Version: "", Architecture: "arm"}})

	missing, err = s.il.VerifyDependencies(DepFollowSource, []string{"i386", "amd64"}, s.il)

	c.Check(err, IsNil)
	c.Check(missing, DeepEquals, []Dependency{Dependency{Pkg: "lib", Relation: VersionGreater, Version: "0.9", Architecture: "amd64"}})

	missing, err = s.il.VerifyDependencies(DepFollowSource, []string{"arm"}, s.il)

	c.Check(err, IsNil)
	c.Check(missing, DeepEquals, []Dependency{Dependency{Pkg: "libx", Relation: VersionEqual, Version: "1.5", Architecture: "source"}})

	_, err = s.il.VerifyDependencies(0, []string{"i386", "amd64", "s390"}, s.il)

	c.Check(err, ErrorMatches, "unable to process package app-1.0_s390:.*")
}

func (s *PackageListSuite) TestArchitectures(c *C) {
	archs := s.il.Architectures(true)
	sort.Strings(archs)
	c.Check(archs, DeepEquals, []string{"amd64", "arm", "i386", "s390", "source"})

	archs = s.il.Architectures(false)
	sort.Strings(archs)
	c.Check(archs, DeepEquals, []string{"amd64", "arm", "i386", "s390"})
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

	reflist = NewPackageRefList()
	c.Check(reflist.Len(), Equals, 0)
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

func (s *PackageListSuite) TestSubstract(c *C) {
	r1 := []byte("r1")
	r2 := []byte("r2")
	r3 := []byte("r3")
	r4 := []byte("r4")
	r5 := []byte("r5")

	empty := &PackageRefList{Refs: [][]byte{}}
	l1 := &PackageRefList{Refs: [][]byte{r1, r2, r3, r4}}
	l2 := &PackageRefList{Refs: [][]byte{r1, r3}}
	l3 := &PackageRefList{Refs: [][]byte{r2, r4}}
	l4 := &PackageRefList{Refs: [][]byte{r4, r5}}
	l5 := &PackageRefList{Refs: [][]byte{r1, r2, r3}}

	c.Check(l1.Substract(empty), DeepEquals, l1)
	c.Check(l1.Substract(l2), DeepEquals, l3)
	c.Check(l1.Substract(l3), DeepEquals, l2)
	c.Check(l1.Substract(l4), DeepEquals, l5)
	c.Check(empty.Substract(l1), DeepEquals, empty)
	c.Check(l2.Substract(l3), DeepEquals, l2)
}

func (s *PackageListSuite) TestDiff(c *C) {
	db, _ := database.OpenDB(c.MkDir())
	coll := NewPackageCollection(db)

	packages := []*Package{
		&Package{Name: "lib", Version: "1.0", Architecture: "i386"},      //0
		&Package{Name: "dpkg", Version: "1.7", Architecture: "i386"},     //1
		&Package{Name: "data", Version: "1.1~bp1", Architecture: "all"},  //2
		&Package{Name: "app", Version: "1.1~bp1", Architecture: "i386"},  //3
		&Package{Name: "app", Version: "1.1~bp2", Architecture: "i386"},  //4
		&Package{Name: "app", Version: "1.1~bp2", Architecture: "amd64"}, //5
		&Package{Name: "xyz", Version: "3.0", Architecture: "sparc"},     //6
	}

	for _, p := range packages {
		coll.Update(p)
	}

	listA := NewPackageList()
	listA.Add(packages[0])
	listA.Add(packages[1])
	listA.Add(packages[2])
	listA.Add(packages[3])
	listA.Add(packages[6])

	listB := NewPackageList()
	listB.Add(packages[0])
	listB.Add(packages[2])
	listB.Add(packages[4])
	listB.Add(packages[5])

	reflistA := NewPackageRefListFromPackageList(listA)
	reflistB := NewPackageRefListFromPackageList(listB)

	diffAA, err := reflistA.Diff(reflistA, coll)
	c.Check(err, IsNil)
	c.Check(diffAA, HasLen, 0)

	diffAB, err := reflistA.Diff(reflistB, coll)
	c.Check(err, IsNil)
	c.Check(diffAB, HasLen, 4)

	c.Check(diffAB[0].Left, IsNil)
	c.Check(diffAB[0].Right.String(), Equals, "app-1.1~bp2_amd64")

	c.Check(diffAB[1].Left.String(), Equals, "app-1.1~bp1_i386")
	c.Check(diffAB[1].Right.String(), Equals, "app-1.1~bp2_i386")

	c.Check(diffAB[2].Left.String(), Equals, "dpkg-1.7_i386")
	c.Check(diffAB[2].Right, IsNil)

	c.Check(diffAB[3].Left.String(), Equals, "xyz-3.0_sparc")
	c.Check(diffAB[3].Right, IsNil)

	diffBA, err := reflistB.Diff(reflistA, coll)
	c.Check(err, IsNil)
	c.Check(diffBA, HasLen, 4)

	c.Check(diffBA[0].Right, IsNil)
	c.Check(diffBA[0].Left.String(), Equals, "app-1.1~bp2_amd64")

	c.Check(diffBA[1].Right.String(), Equals, "app-1.1~bp1_i386")
	c.Check(diffBA[1].Left.String(), Equals, "app-1.1~bp2_i386")

	c.Check(diffBA[2].Right.String(), Equals, "dpkg-1.7_i386")
	c.Check(diffBA[2].Left, IsNil)

	c.Check(diffBA[3].Right.String(), Equals, "xyz-3.0_sparc")
	c.Check(diffBA[3].Left, IsNil)

}

func (s *PackageListSuite) TestMerge(c *C) {
	db, _ := database.OpenDB(c.MkDir())
	coll := NewPackageCollection(db)

	packages := []*Package{
		&Package{Name: "lib", Version: "1.0", Architecture: "i386"},      //0
		&Package{Name: "dpkg", Version: "1.7", Architecture: "i386"},     //1
		&Package{Name: "data", Version: "1.1~bp1", Architecture: "all"},  //2
		&Package{Name: "app", Version: "1.1~bp1", Architecture: "i386"},  //3
		&Package{Name: "app", Version: "1.1~bp2", Architecture: "i386"},  //4
		&Package{Name: "app", Version: "1.1~bp2", Architecture: "amd64"}, //5
		&Package{Name: "dpkg", Version: "1.0", Architecture: "i386"},     //6
		&Package{Name: "xyz", Version: "1.0", Architecture: "sparc"},     //7
	}

	for _, p := range packages {
		coll.Update(p)
	}

	listA := NewPackageList()
	listA.Add(packages[0])
	listA.Add(packages[1])
	listA.Add(packages[2])
	listA.Add(packages[3])
	listA.Add(packages[7])

	listB := NewPackageList()
	listB.Add(packages[0])
	listB.Add(packages[2])
	listB.Add(packages[4])
	listB.Add(packages[5])
	listB.Add(packages[6])

	reflistA := NewPackageRefListFromPackageList(listA)
	reflistB := NewPackageRefListFromPackageList(listB)

	toStrSlice := func(reflist *PackageRefList) (result []string) {
		result = make([]string, reflist.Len())
		for i, r := range reflist.Refs {
			result[i] = string(r)
		}
		return
	}

	mergeAB := reflistA.Merge(reflistB, true)
	mergeBA := reflistB.Merge(reflistA, true)

	c.Check(toStrSlice(mergeAB), DeepEquals,
		[]string{"Pall data 1.1~bp1", "Pamd64 app 1.1~bp2", "Pi386 app 1.1~bp2", "Pi386 dpkg 1.0", "Pi386 lib 1.0", "Psparc xyz 1.0"})
	c.Check(toStrSlice(mergeBA), DeepEquals,
		[]string{"Pall data 1.1~bp1", "Pamd64 app 1.1~bp2", "Pi386 app 1.1~bp1", "Pi386 dpkg 1.7", "Pi386 lib 1.0", "Psparc xyz 1.0"})

	mergeABall := reflistA.Merge(reflistB, false)
	mergeBAall := reflistB.Merge(reflistA, false)

	c.Check(mergeABall, DeepEquals, mergeBAall)
	c.Check(toStrSlice(mergeBAall), DeepEquals,
		[]string{"Pall data 1.1~bp1", "Pamd64 app 1.1~bp2", "Pi386 app 1.1~bp1", "Pi386 app 1.1~bp2", "Pi386 dpkg 1.0", "Pi386 dpkg 1.7", "Pi386 lib 1.0", "Psparc xyz 1.0"})
}
