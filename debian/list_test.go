package debian

import (
	"errors"
	. "launchpad.net/gocheck"
	"sort"
	"strings"
)

type PackageListSuite struct {
	// Simple list with "real" packages from stanzas
	list                   *PackageList
	p1, p2, p3, p4, p5, p6 *Package

	// Mocked packages in list
	packages       []*Package
	sourcePackages []*Package
	il             *PackageList
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
		&Package{Name: "lib", Version: "1.0", Architecture: "i386", Source: "lib (0.9)", deps: &PackageDependencies{PreDepends: []string{"dpkg (>= 1.6)"}, Depends: []string{"mail-agent"}}},
		&Package{Name: "dpkg", Version: "1.7", Architecture: "i386", Provides: []string{"package-installer"}, deps: &PackageDependencies{}},
		&Package{Name: "data", Version: "1.1~bp1", Architecture: "all", Source: "app", deps: &PackageDependencies{PreDepends: []string{"dpkg (>= 1.6)"}}},
		&Package{Name: "app", Version: "1.1~bp1", Architecture: "i386", deps: &PackageDependencies{PreDepends: []string{"dpkg (>= 1.6)"}, Depends: []string{"lib (>> 0.9)", "data (>= 1.0)"}}},
		&Package{Name: "mailer", Version: "3.5.8", Architecture: "i386", Source: "postfix (1.3)", Provides: []string{"mail-agent"}, deps: &PackageDependencies{}},
		&Package{Name: "app", Version: "1.1~bp1", Architecture: "amd64", deps: &PackageDependencies{PreDepends: []string{"dpkg (>= 1.6)"}, Depends: []string{"lib (>> 0.9)", "data (>= 1.0)"}}},
		&Package{Name: "app", Version: "1.1~bp1", Architecture: "arm", deps: &PackageDependencies{PreDepends: []string{"dpkg (>= 1.6)"}, Depends: []string{"lib (>> 0.9) | libx (>= 1.5)", "data (>= 1.0) | mail-agent"}}},
		&Package{Name: "app", Version: "1.0", Architecture: "s390", deps: &PackageDependencies{PreDepends: []string{"dpkg >= 1.6)"}, Depends: []string{"lib (>> 0.9)", "data (>= 1.0)"}}},
		&Package{Name: "aa", Version: "2.0-1", Architecture: "i386", deps: &PackageDependencies{PreDepends: []string{"dpkg (>= 1.6)"}}},
		&Package{Name: "dpkg", Version: "1.6.1-3", Architecture: "amd64", Provides: []string{"package-installer"}, deps: &PackageDependencies{}},
		&Package{Name: "libx", Version: "1.5", Architecture: "arm", deps: &PackageDependencies{PreDepends: []string{"dpkg (>= 1.6)"}}},
		&Package{Name: "dpkg", Version: "1.6.1-3", Architecture: "arm", Provides: []string{"package-installer"}, deps: &PackageDependencies{}},
		&Package{Name: "dpkg", Version: "1.6.1-3", Architecture: "source", SourceArchitecture: "any", IsSource: true, deps: &PackageDependencies{}},
		&Package{Name: "dpkg", Version: "1.7", Architecture: "source", SourceArchitecture: "any", IsSource: true, deps: &PackageDependencies{}},
	}
	for _, p := range s.packages {
		s.il.Add(p)
	}
	s.il.PrepareIndex()

	s.sourcePackages = []*Package{
		&Package{Name: "postfix", Version: "1.3", Architecture: "source", SourceArchitecture: "any", IsSource: true, deps: &PackageDependencies{}},
		&Package{Name: "app", Version: "1.1~bp1", Architecture: "source", SourceArchitecture: "any", IsSource: true, deps: &PackageDependencies{}},
		&Package{Name: "aa", Version: "2.0-1", Architecture: "source", SourceArchitecture: "any", IsSource: true, deps: &PackageDependencies{}},
		&Package{Name: "lib", Version: "0.9", Architecture: "source", SourceArchitecture: "any", IsSource: true, deps: &PackageDependencies{}},
	}

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

func (s *PackageListSuite) TestFilter(c *C) {
	c.Check(func() { s.list.Filter([]string{"abcd_0.3_i386"}, false, nil, 0, nil) }, Panics, "list not indexed, can't filter")

	_, err := s.il.Filter([]string{"app >3)"}, false, nil, 0, nil)
	c.Check(err, ErrorMatches, "unable to parse dependency.*")

	plString := func(l *PackageList) string {
		list := make([]string, 0, l.Len())
		for _, p := range l.packages {
			list = append(list, p.String())
		}

		sort.Strings(list)

		return strings.Join(list, " ")
	}

	result, err := s.il.Filter([]string{"app_1.1~bp1_i386"}, false, nil, 0, nil)
	c.Check(err, IsNil)
	c.Check(plString(result), Equals, "app_1.1~bp1_i386")

	result, err = s.il.Filter([]string{"app_1.1~bp1_i386", "dpkg_1.7_source", "dpkg_1.8_amd64"}, false, nil, 0, nil)
	c.Check(err, IsNil)
	c.Check(plString(result), Equals, "app_1.1~bp1_i386 dpkg_1.7_source")

	result, err = s.il.Filter([]string{"app", "dpkg (>>1.6.1-3)", "app (>=1.0)", "xyz", "aa (>>3.0)"}, false, nil, 0, nil)
	c.Check(err, IsNil)
	c.Check(plString(result), Equals, "app_1.0_s390 app_1.1~bp1_amd64 app_1.1~bp1_arm app_1.1~bp1_i386 dpkg_1.7_i386 dpkg_1.7_source")

	result, err = s.il.Filter([]string{"app {i386}"}, true, NewPackageList(), 0, []string{"i386"})
	c.Check(err, IsNil)
	c.Check(plString(result), Equals, "app_1.1~bp1_i386 data_1.1~bp1_all dpkg_1.7_i386 lib_1.0_i386 mailer_3.5.8_i386")

	result, err = s.il.Filter([]string{"app (>=0.9)", "lib", "data"}, true, NewPackageList(), 0, []string{"i386", "amd64"})
	c.Check(err, IsNil)
	c.Check(plString(result), Equals, "app_1.0_s390 app_1.1~bp1_amd64 app_1.1~bp1_arm app_1.1~bp1_i386 data_1.1~bp1_all dpkg_1.6.1-3_amd64 dpkg_1.7_i386 lib_1.0_i386 mailer_3.5.8_i386")
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

	for _, p := range s.sourcePackages {
		s.il.Add(p)
	}

	missing, err = s.il.VerifyDependencies(DepFollowSource, []string{"i386", "amd64"}, s.il)
	c.Check(err, IsNil)
	c.Check(missing, DeepEquals, []Dependency{Dependency{Pkg: "lib", Relation: VersionGreater, Version: "0.9", Architecture: "amd64"}})

	missing, err = s.il.VerifyDependencies(DepFollowSource, []string{"arm"}, s.il)
	c.Check(err, IsNil)
	c.Check(missing, DeepEquals, []Dependency{Dependency{Pkg: "libx", Relation: VersionEqual, Version: "1.5", Architecture: "source"}})

	_, err = s.il.VerifyDependencies(0, []string{"i386", "amd64", "s390"}, s.il)
	c.Check(err, ErrorMatches, "unable to process package app_1.0_s390:.*")
}

func (s *PackageListSuite) TestArchitectures(c *C) {
	archs := s.il.Architectures(true)
	sort.Strings(archs)
	c.Check(archs, DeepEquals, []string{"amd64", "arm", "i386", "s390", "source"})

	archs = s.il.Architectures(false)
	sort.Strings(archs)
	c.Check(archs, DeepEquals, []string{"amd64", "arm", "i386", "s390"})
}
