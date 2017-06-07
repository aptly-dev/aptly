package deb

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"regexp"

	"github.com/smira/aptly/files"

	. "gopkg.in/check.v1"
)

type PackageSuite struct {
	stanza       Stanza
	sourceStanza Stanza
}

var _ = Suite(&PackageSuite{})

func (s *PackageSuite) SetUpTest(c *C) {
	s.stanza = packageStanza.Copy()

	buf := bytes.NewBufferString(sourcePackageMeta)
	s.sourceStanza, _ = NewControlFileReader(buf).ReadStanza(false)
}

func (s *PackageSuite) TestNewFromPara(c *C) {
	p := NewPackageFromControlFile(s.stanza)

	c.Check(p.IsSource, Equals, false)
	c.Check(p.IsUdeb, Equals, false)
	c.Check(p.Name, Equals, "alien-arena-common")
	c.Check(p.Version, Equals, "7.40-2")
	c.Check(p.Architecture, Equals, "i386")
	c.Check(p.Provides, DeepEquals, []string(nil))
	c.Check(p.Files(), HasLen, 1)
	c.Check(p.Files()[0].Filename, Equals, "alien-arena-common_7.40-2_i386.deb")
	c.Check(p.Files()[0].downloadPath, Equals, "pool/contrib/a/alien-arena")
	c.Check(p.Files()[0].Checksums.Size, Equals, int64(187518))
	c.Check(p.Files()[0].Checksums.MD5, Equals, "1e8cba92c41420aa7baa8a5718d67122")
	c.Check(p.deps.Depends, DeepEquals, []string{"libc6 (>= 2.7)", "alien-arena-data (>= 7.40)"})
}

func (s *PackageSuite) TestNewUdebFromPara(c *C) {
	stanza, _ := NewControlFileReader(bytes.NewBufferString(udebPackageMeta)).ReadStanza(false)
	p := NewUdebPackageFromControlFile(stanza)

	c.Check(p.IsSource, Equals, false)
	c.Check(p.IsUdeb, Equals, true)
	c.Check(p.Name, Equals, "dmidecode-udeb")
	c.Check(p.Version, Equals, "2.11-9")
	c.Check(p.Architecture, Equals, "amd64")
	c.Check(p.Provides, DeepEquals, []string(nil))
	c.Check(p.Files(), HasLen, 1)
	c.Check(p.Files()[0].Filename, Equals, "dmidecode-udeb_2.11-9_amd64.udeb")
	c.Check(p.deps.Depends, DeepEquals, []string{"libc6-udeb (>= 2.13)"})
}

func (s *PackageSuite) TestNewSourceFromPara(c *C) {
	p, err := NewSourcePackageFromControlFile(s.sourceStanza)

	c.Check(err, IsNil)
	c.Check(p.IsSource, Equals, true)
	c.Check(p.IsUdeb, Equals, false)
	c.Check(p.Name, Equals, "access-modifier-checker")
	c.Check(p.Version, Equals, "1.0-4")
	c.Check(p.Architecture, Equals, "source")
	c.Check(p.SourceArchitecture, Equals, "all")
	c.Check(p.Provides, IsNil)
	c.Check(p.deps.BuildDepends, DeepEquals, []string{"cdbs", "debhelper (>= 7)", "default-jdk", "maven-debian-helper"})
	c.Check(p.deps.BuildDependsInDep, DeepEquals, []string{"default-jdk-doc", "junit (>= 3.8.1)", "libannotation-indexer-java (>= 1.3)", "libannotation-indexer-java-doc", "libasm3-java", "libmaven-install-plugin-java", "libmaven-javadoc-plugin-java", "libmaven-scm-java", "libmaven2-core-java", "libmaven2-core-java-doc", "libmetainf-services-java", "libmetainf-services-java-doc", "libmaven-plugin-tools-java (>= 2.8)"})
	c.Check(p.Files(), HasLen, 3)

	c.Check(p.Files()[0].Filename, Equals, "access-modifier-checker_1.0-4.debian.tar.gz")
	c.Check(p.Files()[0].downloadPath, Equals, "pool/main/a/access-modifier-checker")

	c.Check(p.Files()[1].Filename, Equals, "access-modifier-checker_1.0-4.dsc")
	c.Check(p.Files()[1].downloadPath, Equals, "pool/main/a/access-modifier-checker")
	c.Check(p.Files()[1].Checksums.Size, Equals, int64(3))
	c.Check(p.Files()[1].Checksums.MD5, Equals, "900150983cd24fb0d6963f7d28e17f72")
	c.Check(p.Files()[1].Checksums.SHA1, Equals, "a9993e364706816aba3e25717850c26c9cd0d89d")
	c.Check(p.Files()[1].Checksums.SHA256, Equals, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad")

	c.Check(p.Files()[2].Filename, Equals, "access-modifier-checker_1.0.orig.tar.gz")
	c.Check(p.Files()[2].downloadPath, Equals, "pool/main/a/access-modifier-checker")
	c.Check(p.Files()[2].Checksums.Size, Equals, int64(4))
	c.Check(p.Files()[2].Checksums.MD5, Equals, "e2fc714c4727ee9395f324cd2e7f331f")
	c.Check(p.Files()[2].Checksums.SHA1, Equals, "81fe8bfe87576c3ecb22426f8e57847382917acf")
	c.Check(p.Files()[2].Checksums.SHA256, Equals, "88d4266fd4e6338d13b845fcf289579d209c897823b9217da3e161936f031589")

	c.Check(p.deps.Depends, IsNil)
}

func (s *PackageSuite) TestWithProvides(c *C) {
	s.stanza["Provides"] = "arena"
	p := NewPackageFromControlFile(s.stanza)

	c.Check(p.Name, Equals, "alien-arena-common")
	c.Check(p.Provides, DeepEquals, []string{"arena"})

	st := p.Stanza()
	c.Check(st["Provides"], Equals, "arena")
}

func (s *PackageSuite) TestKey(c *C) {
	p := NewPackageFromControlFile(s.stanza)

	c.Check(p.Key(""), DeepEquals, []byte("Pi386 alien-arena-common 7.40-2 c8901eedd79ac51b"))
	c.Check(p.Key("xD"), DeepEquals, []byte("xDPi386 alien-arena-common 7.40-2 c8901eedd79ac51b"))

	p.V06Plus = false
	c.Check(p.Key(""), DeepEquals, []byte("Pi386 alien-arena-common 7.40-2"))
	c.Check(p.Key("xD"), DeepEquals, []byte("xDPi386 alien-arena-common 7.40-2"))
}

func (s *PackageSuite) TestShortKey(c *C) {
	p := NewPackageFromControlFile(s.stanza)

	c.Check(p.ShortKey(""), DeepEquals, []byte("Pi386 alien-arena-common 7.40-2"))
	c.Check(p.ShortKey("xD"), DeepEquals, []byte("xDPi386 alien-arena-common 7.40-2"))
}

func (s *PackageSuite) TestStanza(c *C) {
	p := NewPackageFromControlFile(s.stanza.Copy())
	stanza := p.Stanza()

	for k := range s.stanza {
		c.Check(stanza[k], Equals, s.stanza[k])
	}

	c.Assert(stanza, DeepEquals, s.stanza)

	p, _ = NewSourcePackageFromControlFile(s.sourceStanza.Copy())
	stanza = p.Stanza()

	c.Assert(stanza, DeepEquals, s.sourceStanza)
}

func (s *PackageSuite) TestString(c *C) {
	p := NewPackageFromControlFile(s.stanza)
	c.Assert(p.String(), Equals, "alien-arena-common_7.40-2_i386")
}

func (s *PackageSuite) TestGetField(c *C) {
	p := NewPackageFromControlFile(s.stanza.Copy())

	stanza2 := s.stanza.Copy()
	delete(stanza2, "Source")
	stanza2["Provides"] = "app, game"
	p2 := NewPackageFromControlFile(stanza2)

	stanza3 := s.stanza.Copy()
	stanza3["Source"] = "alien-arena (3.5)"
	p3 := NewPackageFromControlFile(stanza3)

	p4, _ := NewSourcePackageFromControlFile(s.sourceStanza.Copy())

	stanza5, _ := NewControlFileReader(bytes.NewBufferString(udebPackageMeta)).ReadStanza(false)
	p5 := NewUdebPackageFromControlFile(stanza5)

	c.Check(p.GetField("$Source"), Equals, "alien-arena")
	c.Check(p2.GetField("$Source"), Equals, "alien-arena-common")
	c.Check(p3.GetField("$Source"), Equals, "alien-arena")
	c.Check(p4.GetField("$Source"), Equals, "")
	c.Check(p5.GetField("$Source"), Equals, "dmidecode")

	c.Check(p.GetField("$SourceVersion"), Equals, "7.40-2")
	c.Check(p2.GetField("$SourceVersion"), Equals, "7.40-2")
	c.Check(p3.GetField("$SourceVersion"), Equals, "3.5")
	c.Check(p4.GetField("$SourceVersion"), Equals, "")
	c.Check(p5.GetField("$SourceVersion"), Equals, "2.11-9")

	c.Check(p.GetField("$Architecture"), Equals, "i386")
	c.Check(p4.GetField("$Architecture"), Equals, "source")
	c.Check(p5.GetField("$Architecture"), Equals, "amd64")

	c.Check(p.GetField("$PackageType"), Equals, "deb")
	c.Check(p4.GetField("$PackageType"), Equals, "source")
	c.Check(p5.GetField("$PackageType"), Equals, "udeb")

	c.Check(p.GetField("Name"), Equals, "alien-arena-common")
	c.Check(p4.GetField("Name"), Equals, "access-modifier-checker")

	c.Check(p.GetField("Architecture"), Equals, "i386")
	c.Check(p4.GetField("Architecture"), Equals, "all")

	c.Check(p.GetField("Version"), Equals, "7.40-2")

	c.Check(p.GetField("Source"), Equals, "alien-arena")
	c.Check(p2.GetField("Source"), Equals, "")
	c.Check(p3.GetField("Source"), Equals, "alien-arena (3.5)")
	c.Check(p4.GetField("Source"), Equals, "")

	c.Check(p.GetField("Depends"), Equals, "libc6 (>= 2.7), alien-arena-data (>= 7.40)")

	c.Check(p.GetField("Provides"), Equals, "")
	c.Check(p2.GetField("Provides"), Equals, "app, game")

	c.Check(p.GetField("Section"), Equals, "contrib/games")
	c.Check(p.GetField("Priority"), Equals, "extra")
}

func (s *PackageSuite) TestEquals(c *C) {
	p := NewPackageFromControlFile(s.stanza)

	p2 := NewPackageFromControlFile(packageStanza.Copy())
	c.Check(p.Equals(p2), Equals, true)

	p2.deps.Depends = []string{"package1"}
	c.Check(p.Equals(p2), Equals, true) // strange, but Equals doesn't check deep

	p2 = NewPackageFromControlFile(packageStanza.Copy())
	files := p2.Files()
	files[0].Checksums.MD5 = "abcdefabcdef"
	p2.UpdateFiles(files)
	c.Check(p.Equals(p2), Equals, false)

	so, _ := NewSourcePackageFromControlFile(s.sourceStanza.Copy())
	so2, _ := NewSourcePackageFromControlFile(s.sourceStanza.Copy())

	c.Check(so.Equals(so2), Equals, true)

	files = so2.Files()
	files[2].Checksums.MD5 = "abcde"
	so2.UpdateFiles(files)
	c.Check(so.Equals(so2), Equals, false)

	so2, _ = NewSourcePackageFromControlFile(s.sourceStanza.Copy())
	files = so2.Files()
	files[1].Filename = "other.deb"
	so2.UpdateFiles(files)
	c.Check(so.Equals(so2), Equals, false)
}

func (s *PackageSuite) TestMatchesArchitecture(c *C) {
	p := NewPackageFromControlFile(s.stanza)
	c.Check(p.MatchesArchitecture("i386"), Equals, true)
	c.Check(p.MatchesArchitecture("amd64"), Equals, false)

	s.stanza = packageStanza.Copy()
	s.stanza["Architecture"] = "all"
	p = NewPackageFromControlFile(s.stanza)
	c.Check(p.MatchesArchitecture("i386"), Equals, true)
	c.Check(p.MatchesArchitecture("amd64"), Equals, true)
	c.Check(p.MatchesArchitecture("source"), Equals, false)

	p, _ = NewSourcePackageFromControlFile(s.sourceStanza)
	c.Check(p.MatchesArchitecture("source"), Equals, true)
	c.Check(p.MatchesArchitecture("amd64"), Equals, false)
}

func (s *PackageSuite) TestMatchesDependency(c *C) {
	p := NewPackageFromControlFile(s.stanza)

	// exact match
	c.Check(p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "i386", Relation: VersionEqual, Version: "7.40-2"}), Equals, true)

	// exact match, same version, no revision specified
	c.Check(p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "i386", Relation: VersionEqual, Version: "7.40"}), Equals, false)

	// different name
	c.Check(p.MatchesDependency(Dependency{Pkg: "alien-arena", Architecture: "i386", Relation: VersionEqual, Version: "7.40-2"}), Equals, false)

	// different version
	c.Check(p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "i386", Relation: VersionEqual, Version: "7.40-3"}), Equals, false)

	// different arch
	c.Check(p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "amd64", Relation: VersionEqual, Version: "7.40-2"}), Equals, false)

	// empty arch
	c.Check(p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "", Relation: VersionEqual, Version: "7.40-2"}), Equals, true)

	// version don't care
	c.Check(p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "i386", Relation: VersionDontCare, Version: ""}), Equals, true)

	// >
	c.Check(p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "i386", Relation: VersionGreater, Version: "7.40-2"}), Equals, false)
	c.Check(p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "i386", Relation: VersionGreater, Version: "7.40-1"}), Equals, true)

	// <
	c.Check(p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "i386", Relation: VersionLess, Version: "7.40-2"}), Equals, false)
	c.Check(p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "i386", Relation: VersionLess, Version: "7.40-3"}), Equals, true)

	// >=
	c.Check(p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "i386", Relation: VersionGreaterOrEqual, Version: "7.40-2"}), Equals, true)
	c.Check(p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "i386", Relation: VersionGreaterOrEqual, Version: "7.40-3"}), Equals, false)

	// <=
	c.Check(p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "i386", Relation: VersionLessOrEqual, Version: "7.40-2"}), Equals, true)
	c.Check(p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "i386", Relation: VersionLessOrEqual, Version: "7.40-1"}), Equals, false)

	// %
	c.Check(p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "i386", Relation: VersionPatternMatch, Version: "7.40-*"}), Equals, true)
	c.Check(p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "i386", Relation: VersionPatternMatch, Version: "7.40-[2]"}), Equals, true)
	c.Check(p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "i386", Relation: VersionPatternMatch, Version: "7.40-[2"}), Equals, false)
	c.Check(p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "i386", Relation: VersionPatternMatch, Version: "7.40-[34]"}), Equals, false)

	// ~
	c.Check(
		p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "i386", Relation: VersionRegexp, Version: "7\\.40-.*",
			Regexp: regexp.MustCompile(`7\.40-.*`)}), Equals, true)
	c.Check(
		p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "i386", Relation: VersionRegexp, Version: "7\\.40-.*",
			Regexp: regexp.MustCompile("40")}), Equals, true)
	c.Check(
		p.MatchesDependency(Dependency{Pkg: "alien-arena-common", Architecture: "i386", Relation: VersionRegexp, Version: "7\\.40-.*",
			Regexp: regexp.MustCompile("39-.*")}), Equals, false)

	// Provides
	c.Check(p.MatchesDependency(Dependency{Pkg: "game", Relation: VersionDontCare}), Equals, false)
	p.Provides = []string{"fun", "game"}
	c.Check(p.MatchesDependency(Dependency{Pkg: "game", Relation: VersionDontCare}), Equals, true)
	c.Check(p.MatchesDependency(Dependency{Pkg: "game", Architecture: "amd64", Relation: VersionDontCare}), Equals, false)
}

func (s *PackageSuite) TestGetDependencies(c *C) {
	p := NewPackageFromControlFile(s.stanza)
	c.Check(p.GetDependencies(0), DeepEquals, []string{"libc6 (>= 2.7)", "alien-arena-data (>= 7.40)", "dpkg (>= 1.6)"})
	c.Check(p.GetDependencies(DepFollowSuggests), DeepEquals, []string{"libc6 (>= 2.7)", "alien-arena-data (>= 7.40)", "dpkg (>= 1.6)", "alien-arena-mars"})
	c.Check(p.GetDependencies(DepFollowSuggests|DepFollowRecommends), DeepEquals, []string{"libc6 (>= 2.7)", "alien-arena-data (>= 7.40)", "dpkg (>= 1.6)", "aliean-arena-luna", "alien-arena-mars"})

	c.Check(p.GetDependencies(DepFollowSource), DeepEquals, []string{"libc6 (>= 2.7)", "alien-arena-data (>= 7.40)", "dpkg (>= 1.6)", "alien-arena (= 7.40-2) {source}"})
	p.Source = "alien-arena (7.40-3)"
	c.Check(p.GetDependencies(DepFollowSource), DeepEquals, []string{"libc6 (>= 2.7)", "alien-arena-data (>= 7.40)", "dpkg (>= 1.6)", "alien-arena (7.40-3) {source}"})
	p.Source = ""
	c.Check(p.GetDependencies(DepFollowSource), DeepEquals, []string{"libc6 (>= 2.7)", "alien-arena-data (>= 7.40)", "dpkg (>= 1.6)", "alien-arena-common (= 7.40-2) {source}"})

	p, _ = NewSourcePackageFromControlFile(s.sourceStanza)
	c.Check(p.GetDependencies(0), DeepEquals, []string{})
	c.Check(p.GetDependencies(DepFollowBuild), DeepEquals, []string{"cdbs", "debhelper (>= 7)", "default-jdk", "maven-debian-helper", "default-jdk-doc", "junit (>= 3.8.1)", "libannotation-indexer-java (>= 1.3)", "libannotation-indexer-java-doc", "libasm3-java", "libmaven-install-plugin-java", "libmaven-javadoc-plugin-java", "libmaven-scm-java", "libmaven2-core-java", "libmaven2-core-java-doc", "libmetainf-services-java", "libmetainf-services-java-doc", "libmaven-plugin-tools-java (>= 2.8)"})
}

func (s *PackageSuite) TestPoolDirectory(c *C) {
	p := NewPackageFromControlFile(s.stanza)
	dir, err := p.PoolDirectory()
	c.Check(err, IsNil)
	c.Check(dir, Equals, "a/alien-arena")

	p = NewPackageFromControlFile(packageStanza.Copy())
	p.Source = ""
	dir, err = p.PoolDirectory()
	c.Check(err, IsNil)
	c.Check(dir, Equals, "a/alien-arena-common")

	p = NewPackageFromControlFile(packageStanza.Copy())
	p.Source = "libarena"
	dir, err = p.PoolDirectory()
	c.Check(err, IsNil)
	c.Check(dir, Equals, "liba/libarena")

	p = NewPackageFromControlFile(packageStanza.Copy())
	p.Source = "gcc-defaults (1.77)"
	dir, err = p.PoolDirectory()
	c.Check(err, IsNil)
	c.Check(dir, Equals, "g/gcc-defaults")

	p = NewPackageFromControlFile(packageStanza.Copy())
	p.Source = "l"
	_, err = p.PoolDirectory()
	c.Check(err, ErrorMatches, ".* too short")
}

func (s *PackageSuite) TestLinkFromPool(c *C) {
	packagePool := files.NewPackagePool(c.MkDir(), false)
	cs := files.NewMockChecksumStorage()
	publishedStorage := files.NewPublishedStorage(c.MkDir(), "", "")
	p := NewPackageFromControlFile(s.stanza)

	tmpFilepath := filepath.Join(c.MkDir(), "file")
	c.Assert(ioutil.WriteFile(tmpFilepath, nil, 0777), IsNil)

	p.Files()[0].PoolPath, _ = packagePool.Import(tmpFilepath, p.Files()[0].Filename, &p.Files()[0].Checksums, false, cs)

	err := p.LinkFromPool(publishedStorage, packagePool, "", "non-free", false)
	c.Check(err, IsNil)
	c.Check(p.Files()[0].Filename, Equals, "alien-arena-common_7.40-2_i386.deb")
	c.Check(p.Files()[0].downloadPath, Equals, "pool/non-free/a/alien-arena")

	p.IsSource = true
	err = p.LinkFromPool(publishedStorage, packagePool, "", "non-free", false)
	c.Check(err, IsNil)
	c.Check(p.Extra()["Directory"], Equals, "pool/non-free/a/alien-arena")
}

func (s *PackageSuite) TestFilepathList(c *C) {
	packagePool := files.NewPackagePool(c.MkDir(), true)
	p := NewPackageFromControlFile(s.stanza)

	list, err := p.FilepathList(packagePool)
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"1e/8c/alien-arena-common_7.40-2_i386.deb"})
}

func (s *PackageSuite) TestDownloadList(c *C) {
	packagePool := files.NewPackagePool(c.MkDir(), false)
	cs := files.NewMockChecksumStorage()
	p := NewPackageFromControlFile(s.stanza)
	p.Files()[0].Checksums.Size = 5

	list, err := p.DownloadList(packagePool, cs)
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []PackageDownloadTask{
		{
			File: &p.Files()[0],
		},
	})

	tmpFilepath := filepath.Join(c.MkDir(), "file")
	c.Assert(ioutil.WriteFile(tmpFilepath, []byte("abcde"), 0777), IsNil)
	p.Files()[0].PoolPath, _ = packagePool.Import(tmpFilepath, p.Files()[0].Filename, &p.Files()[0].Checksums, false, cs)

	list, err = p.DownloadList(packagePool, cs)
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []PackageDownloadTask{})
}

func (s *PackageSuite) TestVerifyFiles(c *C) {
	p := NewPackageFromControlFile(s.stanza)

	packagePool := files.NewPackagePool(c.MkDir(), false)
	cs := files.NewMockChecksumStorage()

	tmpFilepath := filepath.Join(c.MkDir(), "file")
	c.Assert(ioutil.WriteFile(tmpFilepath, []byte("abcde"), 0777), IsNil)

	p.Files()[0].PoolPath, _ = packagePool.Import(tmpFilepath, p.Files()[0].Filename, &p.Files()[0].Checksums, false, cs)

	p.Files()[0].Checksums.Size = 100
	result, err := p.VerifyFiles(packagePool, cs)
	c.Check(err, IsNil)
	c.Check(result, Equals, false)

	p.Files()[0].Checksums.Size = 5

	result, err = p.VerifyFiles(packagePool, cs)
	c.Check(err, IsNil)
	c.Check(result, Equals, true)
}

var packageStanza = Stanza{"Source": "alien-arena", "Pre-Depends": "dpkg (>= 1.6)", "Suggests": "alien-arena-mars", "Recommends": "aliean-arena-luna", "Depends": "libc6 (>= 2.7), alien-arena-data (>= 7.40)", "Filename": "pool/contrib/a/alien-arena/alien-arena-common_7.40-2_i386.deb", "SHA1": "46955e48cad27410a83740a21d766ce362364024", "SHA256": "eb4afb9885cba6dc70cccd05b910b2dbccc02c5900578be5e99f0d3dbf9d76a5", "Priority": "extra", "Maintainer": "Debian Games Team <pkg-games-devel@lists.alioth.debian.org>", "Description": "Common files for Alien Arena client and server ALIEN ARENA is a standalone 3D first person online deathmatch shooter\n crafted from the original source code of Quake II and Quake III, released\n by id Software under the GPL license. With features including 32 bit\n graphics, new particle engine and effects, light blooms, reflective water,\n hi resolution textures and skins, hi poly models, stain maps, ALIEN ARENA\n pushes the envelope of graphical beauty rivaling today's top games.\n .\n This package installs the common files for Alien Arena.\n", "Homepage": "http://red.planetarena.org", "Tag": "role::app-data, role::shared-lib, special::auto-inst-parts", "Installed-Size": "456", "Version": "7.40-2", "Replaces": "alien-arena (<< 7.33-1)", "Size": "187518", "MD5sum": "1e8cba92c41420aa7baa8a5718d67122", "Package": "alien-arena-common", "Section": "contrib/games", "Architecture": "i386"}

const sourcePackageMeta = `Package: access-modifier-checker
Binary: libaccess-modifier-checker-java, libaccess-modifier-checker-java-doc
Version: 1.0-4
Maintainer: Debian Java Maintainers <pkg-java-maintainers@lists.alioth.debian.org>
Uploaders: James Page <james.page@ubuntu.com>
Build-Depends: cdbs, debhelper (>= 7), default-jdk, maven-debian-helper
Build-Depends-Indep: default-jdk-doc, junit (>= 3.8.1), libannotation-indexer-java (>= 1.3), libannotation-indexer-java-doc, libasm3-java, libmaven-install-plugin-java, libmaven-javadoc-plugin-java, libmaven-scm-java, libmaven2-core-java, libmaven2-core-java-doc, libmetainf-services-java, libmetainf-services-java-doc, libmaven-plugin-tools-java (>= 2.8)
Architecture: all
Standards-Version: 3.9.3
Format: 3.0 (quilt)
Files:
 ab56b4d92b40713acc5af89985d4b786 5 access-modifier-checker_1.0-4.debian.tar.gz
 900150983cd24fb0d6963f7d28e17f72 3 access-modifier-checker_1.0-4.dsc
 e2fc714c4727ee9395f324cd2e7f331f 4 access-modifier-checker_1.0.orig.tar.gz
Dm-Upload-Allowed: yes
Vcs-Browser: http://git.debian.org/?p=pkg-java/access-modifier-checker.git
Vcs-Git: git://git.debian.org/git/pkg-java/access-modifier-checker.git
Checksums-Sha1:
 03de6c570bfe24bfc328ccd7ca46b76eadaf4334 5 access-modifier-checker_1.0-4.debian.tar.gz
 a9993e364706816aba3e25717850c26c9cd0d89d 3 access-modifier-checker_1.0-4.dsc
 81fe8bfe87576c3ecb22426f8e57847382917acf 4 access-modifier-checker_1.0.orig.tar.gz
Checksums-Sha256:
 36bbe50ed96841d10443bcb670d6554f0a34b761be67ec9c4a8ad2c0c44ca42c 5 access-modifier-checker_1.0-4.debian.tar.gz
 ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad 3 access-modifier-checker_1.0-4.dsc
 88d4266fd4e6338d13b845fcf289579d209c897823b9217da3e161936f031589 4 access-modifier-checker_1.0.orig.tar.gz
Homepage: https://github.com/kohsuke/access-modifier
Package-List:
 libaccess-modifier-checker-java deb java optional
 libaccess-modifier-checker-java-doc deb doc optional
Directory: pool/main/a/access-modifier-checker
Priority: source
Section: java
`

const udebPackageMeta = `Package: dmidecode-udeb
Source: dmidecode
Version: 2.11-9
Installed-Size: 115
Maintainer: Daniel Baumann <daniel.baumann@progress-technologies.net>
Architecture: amd64
Depends: libc6-udeb (>= 2.13)
Description: SMBIOS/DMI table decoder (udeb)
Description-md5: bdfb786c6a57097be8c8600b800e749f
Section: debian-installer
Priority: optional
Filename: pool/main/d/dmidecode/dmidecode-udeb_2.11-9_amd64.udeb
Size: 29188
MD5sum: ae70341c4d96dcded89fa670bcfea31e
SHA1: 9532ae4226a85805189a671ee0283f719d48a5ba
SHA256: bbb3a2cb07f741c3995b6d4bb08d772d83582b93a0236d4ea7736bc0370fc320`
