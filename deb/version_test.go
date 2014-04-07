package deb

import (
	. "launchpad.net/gocheck"
)

type VersionSuite struct {
	stanza Stanza
}

var _ = Suite(&VersionSuite{})

func (s *VersionSuite) TestParseVersion(c *C) {
	e, u, d := parseVersion("1.3.4")
	c.Check([]string{e, u, d}, DeepEquals, []string{"", "1.3.4", ""})

	e, u, d = parseVersion("4:1.3:4")
	c.Check([]string{e, u, d}, DeepEquals, []string{"4", "1.3:4", ""})

	e, u, d = parseVersion("1.3.4-1")
	c.Check([]string{e, u, d}, DeepEquals, []string{"", "1.3.4", "1"})

	e, u, d = parseVersion("1.3-pre4-1")
	c.Check([]string{e, u, d}, DeepEquals, []string{"", "1.3-pre4", "1"})

	e, u, d = parseVersion("4:1.3-pre4-1")
	c.Check([]string{e, u, d}, DeepEquals, []string{"4", "1.3-pre4", "1"})
}

func (s *VersionSuite) TestCompareLexicographic(c *C) {
	c.Check(compareLexicographic("", ""), Equals, 0)
	c.Check(compareLexicographic("pre", "pre"), Equals, 0)

	c.Check(compareLexicographic("pr", "pre"), Equals, -1)
	c.Check(compareLexicographic("pre", "pr"), Equals, 1)

	c.Check(compareLexicographic("pra", "prb"), Equals, -1)
	c.Check(compareLexicographic("prb", "pra"), Equals, 1)

	c.Check(compareLexicographic("prx", "pr+"), Equals, -1)
	c.Check(compareLexicographic("pr+", "prx"), Equals, 1)

	c.Check(compareLexicographic("pr~", "pra"), Equals, -1)
	c.Check(compareLexicographic("pra", "pr~"), Equals, 1)

	c.Check(compareLexicographic("~~", "~~a"), Equals, -1)
	c.Check(compareLexicographic("~~a", "~"), Equals, -1)
	c.Check(compareLexicographic("~", ""), Equals, -1)

	c.Check(compareLexicographic("~~a", "~~"), Equals, 1)
	c.Check(compareLexicographic("~", "~~a"), Equals, 1)
	c.Check(compareLexicographic("", "~"), Equals, 1)
}

func (s *VersionSuite) TestCompareVersionPart(c *C) {
	c.Check(compareVersionPart("", ""), Equals, 0)
	c.Check(compareVersionPart("pre", "pre"), Equals, 0)
	c.Check(compareVersionPart("12", "12"), Equals, 0)
	c.Check(compareVersionPart("1.3.5", "1.3.5"), Equals, 0)
	c.Check(compareVersionPart("1.3.5-pre1", "1.3.5-pre1"), Equals, 0)

	c.Check(compareVersionPart("1.0~beta1~svn1245", "1.0~beta1"), Equals, -1)
	c.Check(compareVersionPart("1.0~beta1", "1.0"), Equals, -1)

	c.Check(compareVersionPart("1.0~beta1", "1.0~beta1~svn1245"), Equals, 1)
	c.Check(compareVersionPart("1.0", "1.0~beta1"), Equals, 1)

	c.Check(compareVersionPart("1.pr", "1.pre"), Equals, -1)
	c.Check(compareVersionPart("1.pre", "1.pr"), Equals, 1)

	c.Check(compareVersionPart("1.pra", "1.prb"), Equals, -1)
	c.Check(compareVersionPart("1.prb", "1.pra"), Equals, 1)

	c.Check(compareVersionPart("3.prx", "3.pr+"), Equals, -1)
	c.Check(compareVersionPart("3.pr+", "3.prx"), Equals, 1)

	c.Check(compareVersionPart("3.pr~", "3.pra"), Equals, -1)
	c.Check(compareVersionPart("3.pra", "3.pr~"), Equals, 1)

	c.Check(compareVersionPart("2~~", "2~~a"), Equals, -1)
	c.Check(compareVersionPart("2~~a", "2~"), Equals, -1)
	c.Check(compareVersionPart("2~", "2"), Equals, -1)

	c.Check(compareVersionPart("2~~a", "2~~"), Equals, 1)
	c.Check(compareVersionPart("2~", "2~~a"), Equals, 1)
	c.Check(compareVersionPart("2", "2~"), Equals, 1)
}

func (s *VersionSuite) TestCompareVersions(c *C) {
	c.Check(CompareVersions("3:1.0~beta1~svn1245-1", "3:1.0~beta1~svn1245-1"), Equals, 0)

	c.Check(CompareVersions("1:1.0~beta1~svn1245-1", "3:1.0~beta1~svn1245-1"), Equals, -1)
	c.Check(CompareVersions("1:1.0~beta1~svn1245-1", "1.0~beta1~svn1245-1"), Equals, 1)
	c.Check(CompareVersions("1.0~beta1~svn1245-1", "1.0~beta1~svn1245-2"), Equals, -1)
	c.Check(CompareVersions("3:1.0~beta1~svn1245-1", "3:1.0~beta1-1"), Equals, -1)

	c.Check(CompareVersions("1.0~beta1~svn1245", "1.0~beta1"), Equals, -1)
	c.Check(CompareVersions("1.0~beta1", "1.0"), Equals, -1)
}

func (s *VersionSuite) TestParseDependency(c *C) {
	d, e := ParseDependency("dpkg (>= 1.6)")
	c.Check(e, IsNil)
	c.Check(d.Pkg, Equals, "dpkg")
	c.Check(d.Relation, Equals, VersionGreaterOrEqual)
	c.Check(d.Version, Equals, "1.6")
	c.Check(d.Architecture, Equals, "")

	d, e = ParseDependency("dpkg(>>1.6)")
	c.Check(e, IsNil)
	c.Check(d.Pkg, Equals, "dpkg")
	c.Check(d.Relation, Equals, VersionGreater)
	c.Check(d.Version, Equals, "1.6")

	d, e = ParseDependency("dpkg(1.6)")
	c.Check(e, IsNil)
	c.Check(d.Pkg, Equals, "dpkg")
	c.Check(d.Relation, Equals, VersionEqual)
	c.Check(d.Version, Equals, "1.6")

	d, e = ParseDependency("dpkg ( 1.6)")
	c.Check(e, IsNil)
	c.Check(d.Pkg, Equals, "dpkg")
	c.Check(d.Relation, Equals, VersionEqual)
	c.Check(d.Version, Equals, "1.6")

	d, e = ParseDependency("dpkg (> 1.6)")
	c.Check(e, IsNil)
	c.Check(d.Pkg, Equals, "dpkg")
	c.Check(d.Relation, Equals, VersionGreaterOrEqual)
	c.Check(d.Version, Equals, "1.6")

	d, e = ParseDependency("dpkg (< 1.6)")
	c.Check(e, IsNil)
	c.Check(d.Pkg, Equals, "dpkg")
	c.Check(d.Relation, Equals, VersionLessOrEqual)
	c.Check(d.Version, Equals, "1.6")

	d, e = ParseDependency("dpkg (= 1.6)")
	c.Check(e, IsNil)
	c.Check(d.Pkg, Equals, "dpkg")
	c.Check(d.Relation, Equals, VersionEqual)
	c.Check(d.Version, Equals, "1.6")

	d, e = ParseDependency("dpkg (<< 1.6)")
	c.Check(e, IsNil)
	c.Check(d.Pkg, Equals, "dpkg")
	c.Check(d.Relation, Equals, VersionLess)
	c.Check(d.Version, Equals, "1.6")

	d, e = ParseDependency("dpkg(>>1.6)")
	c.Check(e, IsNil)
	c.Check(d.Pkg, Equals, "dpkg")
	c.Check(d.Relation, Equals, VersionGreater)
	c.Check(d.Version, Equals, "1.6")

	d, e = ParseDependency("dpkg (>>1.6) {i386}")
	c.Check(e, IsNil)
	c.Check(d.Pkg, Equals, "dpkg")
	c.Check(d.Relation, Equals, VersionGreater)
	c.Check(d.Version, Equals, "1.6")
	c.Check(d.Architecture, Equals, "i386")

	d, e = ParseDependency("dpkg{i386}")
	c.Check(e, IsNil)
	c.Check(d.Pkg, Equals, "dpkg")
	c.Check(d.Relation, Equals, VersionDontCare)
	c.Check(d.Version, Equals, "")
	c.Check(d.Architecture, Equals, "i386")

	d, e = ParseDependency("dpkg ")
	c.Check(e, IsNil)
	c.Check(d.Pkg, Equals, "dpkg")
	c.Check(d.Relation, Equals, VersionDontCare)
	c.Check(d.Version, Equals, "")

	d, e = ParseDependency("dpkg(==1.6)")
	c.Check(e, ErrorMatches, "relation unknown.*")

	d, e = ParseDependency("dpkg==1.6)")
	c.Check(e, ErrorMatches, "unable to parse.*")

	d, e = ParseDependency("dpkg i386}")
	c.Check(e, ErrorMatches, "unable to parse.*")

	d, e = ParseDependency("dpkg ) {i386}")
	c.Check(e, ErrorMatches, "unable to parse.*")
}

func (s *VersionSuite) TestParseDependencyVariants(c *C) {
	l, e := ParseDependencyVariants("dpkg (>= 1.6)")
	c.Check(e, IsNil)
	c.Check(l, HasLen, 1)
	c.Check(l[0].Pkg, Equals, "dpkg")
	c.Check(l[0].Relation, Equals, VersionGreaterOrEqual)
	c.Check(l[0].Version, Equals, "1.6")

	l, e = ParseDependencyVariants("dpkg (>= 1.6) | mailer-agent")
	c.Check(e, IsNil)
	c.Check(l, HasLen, 2)
	c.Check(l[0].Pkg, Equals, "dpkg")
	c.Check(l[0].Relation, Equals, VersionGreaterOrEqual)
	c.Check(l[0].Version, Equals, "1.6")
	c.Check(l[1].Pkg, Equals, "mailer-agent")
	c.Check(l[1].Relation, Equals, VersionDontCare)

	_, e = ParseDependencyVariants("dpkg(==1.6)")
	c.Check(e, ErrorMatches, "relation unknown.*")
}

func (s *VersionSuite) TestDependencyString(c *C) {
	d, _ := ParseDependency("dpkg(>>1.6)")
	d.Architecture = "i386"
	c.Check(d.String(), Equals, "dpkg (>> 1.6) [i386]")

	d, _ = ParseDependency("dpkg")
	d.Architecture = "i386"
	c.Check(d.String(), Equals, "dpkg [i386]")
}
