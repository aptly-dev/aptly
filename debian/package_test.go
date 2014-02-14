package debian

import (
	"bytes"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/utils"
	. "launchpad.net/gocheck"
	"os"
	"path/filepath"
)

type PackageSuite struct {
	stanza       Stanza
	sourceStanza Stanza
}

var _ = Suite(&PackageSuite{})

func (s *PackageSuite) SetUpTest(c *C) {
	s.stanza = packageStanza.Copy()

	buf := bytes.NewBufferString(sourcePackageMeta)
	s.sourceStanza, _ = NewControlFileReader(buf).ReadStanza()
}

func (s *PackageSuite) TestPackageFileVerify(c *C) {
	packageRepo := NewRepository(c.MkDir())
	p := NewPackageFromControlFile(s.stanza)
	poolPath, _ := packageRepo.PoolPath(p.Files[0].Filename, p.Files[0].Checksums.MD5)

	result, err := p.Files[0].Verify(packageRepo)
	c.Check(err, IsNil)
	c.Check(result, Equals, false)

	err = os.MkdirAll(filepath.Dir(poolPath), 0755)
	c.Assert(err, IsNil)

	file, err := os.Create(poolPath)
	c.Assert(err, IsNil)
	file.WriteString("abcde")
	file.Close()

	result, err = p.Files[0].Verify(packageRepo)
	c.Check(err, IsNil)
	c.Check(result, Equals, false)

	result, err = p.VerifyFiles(packageRepo)
	c.Check(err, IsNil)
	c.Check(result, Equals, false)

	p.Files[0].Checksums.Size = 5
	result, err = p.Files[0].Verify(packageRepo)
	c.Check(err, IsNil)
	c.Check(result, Equals, true)

	result, err = p.VerifyFiles(packageRepo)
	c.Check(err, IsNil)
	c.Check(result, Equals, true)
}

func (s *PackageSuite) TestNewFromPara(c *C) {
	p := NewPackageFromControlFile(s.stanza)

	c.Check(p.IsSource, Equals, false)
	c.Check(p.Name, Equals, "alien-arena-common")
	c.Check(p.Version, Equals, "7.40-2")
	c.Check(p.Architecture, Equals, "i386")
	c.Check(p.Provides, DeepEquals, []string(nil))
	c.Check(p.Files, HasLen, 1)
	c.Check(p.Files[0].Filename, Equals, "pool/contrib/a/alien-arena/alien-arena-common_7.40-2_i386.deb")
	c.Check(p.Files[0].Checksums.Size, Equals, int64(187518))
	c.Check(p.Files[0].Checksums.MD5, Equals, "1e8cba92c41420aa7baa8a5718d67122")
	c.Check(p.Depends, DeepEquals, []string{"libc6 (>= 2.7)", "alien-arena-data (>= 7.40)"})
}

func (s *PackageSuite) TestNewSourceFromPara(c *C) {
	p, err := NewSourcePackageFromControlFile(s.sourceStanza)

	c.Check(err, IsNil)
	c.Check(p.IsSource, Equals, true)
	c.Check(p.Name, Equals, "access-modifier-checker")
	c.Check(p.Version, Equals, "1.0-4")
	c.Check(p.Architecture, Equals, "source")
	c.Check(p.SourceArchitecture, Equals, "all")
	c.Check(p.Provides, IsNil)
	c.Check(p.BuildDepends, DeepEquals, []string{"cdbs", "debhelper (>= 7)", "default-jdk", "maven-debian-helper"})
	c.Check(p.BuildDependsInDep, DeepEquals, []string{"default-jdk-doc", "junit (>= 3.8.1)", "libannotation-indexer-java (>= 1.3)", "libannotation-indexer-java-doc", "libasm3-java", "libmaven-install-plugin-java", "libmaven-javadoc-plugin-java", "libmaven-scm-java", "libmaven2-core-java", "libmaven2-core-java-doc", "libmetainf-services-java", "libmetainf-services-java-doc", "libmaven-plugin-tools-java (>= 2.8)"})
	c.Check(p.Files, HasLen, 3)

	c.Check(p.Files[0].Filename, Equals, "access-modifier-checker_1.0-4.dsc")
	c.Check(p.Files[0].Checksums.Size, Equals, int64(2583))
	c.Check(p.Files[0].Checksums.MD5, Equals, "eb2a7a57c92ebac9d3d1c2d2d719f290")
	c.Check(p.Files[0].Checksums.SHA1, Equals, "a70b87bba4391c0138cc9cc75efa747c4eeea2d7")
	c.Check(p.Files[0].Checksums.SHA256, Equals, "36ae7f68974bd69450229244510faa8f8056709913770218821069e6cce6b4b8")

	c.Check(p.Files[1].Filename, Equals, "access-modifier-checker_1.0.orig.tar.gz")
	c.Check(p.Files[1].Checksums.Size, Equals, int64(10049))
	c.Check(p.Files[1].Checksums.MD5, Equals, "f280abdc753beea4243d99530f023f12")
	c.Check(p.Files[1].Checksums.SHA1, Equals, "c5bea16095ae1685941193370d20756e036d61a7")
	c.Check(p.Files[1].Checksums.SHA256, Equals, "9d362301892bd647a361755bde28f0a39644a5bbb2ff5300060373ee29d9d87a")

	c.Check(p.Files[2].Filename, Equals, "access-modifier-checker_1.0-4.debian.tar.gz")

	c.Check(p.Depends, IsNil)
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

	c.Check(p.Key(), DeepEquals, []byte("Pi386 alien-arena-common 7.40-2"))
}

func (s *PackageSuite) TestEncodeDecode(c *C) {
	p := NewPackageFromControlFile(s.stanza)
	encoded := p.Encode()
	p2 := &Package{}
	err := p2.Decode(encoded)

	c.Assert(err, IsNil)
	c.Assert(p2, DeepEquals, p)
}

func (s *PackageSuite) TestStanza(c *C) {
	p := NewPackageFromControlFile(s.stanza.Copy())
	stanza := p.Stanza()

	c.Assert(stanza, DeepEquals, s.stanza)

	p, _ = NewSourcePackageFromControlFile(s.sourceStanza.Copy())
	stanza = p.Stanza()

	c.Assert(stanza, DeepEquals, s.sourceStanza)
}

func (s *PackageSuite) TestString(c *C) {
	p := NewPackageFromControlFile(s.stanza)
	c.Assert(p.String(), Equals, "alien-arena-common-7.40-2_i386")
}

func (s *PackageSuite) TestEquals(c *C) {
	p := NewPackageFromControlFile(s.stanza)

	p2 := NewPackageFromControlFile(packageStanza.Copy())
	c.Check(p.Equals(p2), Equals, true)

	p2.Depends = []string{"package1"}
	c.Check(p.Equals(p2), Equals, false)

	p2 = NewPackageFromControlFile(packageStanza.Copy())
	p2.Files[0].Checksums.MD5 = "abcdefabcdef"
	c.Check(p.Equals(p2), Equals, false)

	so, _ := NewSourcePackageFromControlFile(s.sourceStanza.Copy())
	so2, _ := NewSourcePackageFromControlFile(s.sourceStanza.Copy())

	c.Check(so.Equals(so2), Equals, true)
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
}

func (s *PackageSuite) TestGetDependencies(c *C) {
	p := NewPackageFromControlFile(s.stanza)
	c.Check(p.GetDependencies(0), DeepEquals, []string{"libc6 (>= 2.7)", "alien-arena-data (>= 7.40)", "dpkg (>= 1.6)"})
	c.Check(p.GetDependencies(DepFollowSuggests), DeepEquals, []string{"libc6 (>= 2.7)", "alien-arena-data (>= 7.40)", "dpkg (>= 1.6)", "alien-arena-mars"})
	c.Check(p.GetDependencies(DepFollowSuggests|DepFollowRecommends), DeepEquals, []string{"libc6 (>= 2.7)", "alien-arena-data (>= 7.40)", "dpkg (>= 1.6)", "aliean-arena-luna", "alien-arena-mars"})
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
	p.Source = "l"
	_, err = p.PoolDirectory()
	c.Check(err, ErrorMatches, ".* too short")
}

func (s *PackageSuite) TestLinkFromPool(c *C) {
	packageRepo := NewRepository(c.MkDir())
	p := NewPackageFromControlFile(s.stanza)

	poolPath, _ := packageRepo.PoolPath(p.Files[0].Filename, p.Files[0].Checksums.MD5)
	err := os.MkdirAll(filepath.Dir(poolPath), 0755)
	c.Assert(err, IsNil)

	file, err := os.Create(poolPath)
	c.Assert(err, IsNil)
	file.Close()

	err = p.LinkFromPool(packageRepo, "", "non-free")
	c.Check(err, IsNil)
	c.Check(p.Files[0].Filename, Equals, "pool/non-free/a/alien-arena/alien-arena-common_7.40-2_i386.deb")
}

func (s *PackageSuite) TestFilepathList(c *C) {
	packageRepo := NewRepository(c.MkDir())
	p := NewPackageFromControlFile(s.stanza)

	list, err := p.FilepathList(packageRepo)
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"1e/8c/alien-arena-common_7.40-2_i386.deb"})
}

func (s *PackageSuite) TestDownloadList(c *C) {
	packageRepo := NewRepository(c.MkDir())
	p := NewPackageFromControlFile(s.stanza)
	p.Files[0].Checksums.Size = 5
	poolPath, _ := packageRepo.PoolPath(p.Files[0].Filename, p.Files[0].Checksums.MD5)

	list, err := p.DownloadList(packageRepo)
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []PackageDownloadTask{
		PackageDownloadTask{
			RepoURI:         "pool/contrib/a/alien-arena/alien-arena-common_7.40-2_i386.deb",
			DestinationPath: poolPath,
			Checksums: utils.ChecksumInfo{Size: 5,
				MD5:    "1e8cba92c41420aa7baa8a5718d67122",
				SHA1:   "46955e48cad27410a83740a21d766ce362364024",
				SHA256: "eb4afb9885cba6dc70cccd05b910b2dbccc02c5900578be5e99f0d3dbf9d76a5"}}})

	err = os.MkdirAll(filepath.Dir(poolPath), 0755)
	c.Assert(err, IsNil)

	file, err := os.Create(poolPath)
	c.Assert(err, IsNil)
	file.WriteString("abcde")
	file.Close()

	list, err = p.DownloadList(packageRepo)
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []PackageDownloadTask{})
}

type PackageCollectionSuite struct {
	collection *PackageCollection
	p          *Package
	db         database.Storage
}

var _ = Suite(&PackageCollectionSuite{})

func (s *PackageCollectionSuite) SetUpTest(c *C) {
	s.p = NewPackageFromControlFile(packageStanza.Copy())
	s.db, _ = database.OpenDB(c.MkDir())
	s.collection = NewPackageCollection(s.db)
}

func (s *PackageCollectionSuite) TearDownTest(c *C) {
	s.db.Close()
}

func (s *PackageCollectionSuite) TestUpdateByKey(c *C) {
	err := s.collection.Update(s.p)
	c.Assert(err, IsNil)

	p2, err := s.collection.ByKey(s.p.Key())
	c.Assert(err, IsNil)
	c.Assert(p2.Equals(s.p), Equals, true)
}

func (s *PackageCollectionSuite) TestAllPackageRefs(c *C) {
	err := s.collection.Update(s.p)
	c.Assert(err, IsNil)

	refs := s.collection.AllPackageRefs()
	c.Check(refs.Len(), Equals, 1)
	c.Check(refs.Refs[0], DeepEquals, s.p.Key())
}

func (s *PackageCollectionSuite) TestDeleteByKey(c *C) {
	err := s.collection.Update(s.p)
	c.Assert(err, IsNil)

	err = s.collection.DeleteByKey(s.p.Key())
	c.Check(err, IsNil)

	_, err = s.collection.ByKey(s.p.Key())
	c.Check(err, ErrorMatches, "key not found")
}

var packageStanza = Stanza{"Source": "alien-arena", "Pre-Depends": "dpkg (>= 1.6)", "Suggests": "alien-arena-mars", "Recommends": "aliean-arena-luna", "Depends": "libc6 (>= 2.7), alien-arena-data (>= 7.40)", "Filename": "pool/contrib/a/alien-arena/alien-arena-common_7.40-2_i386.deb", "SHA1": " 46955e48cad27410a83740a21d766ce362364024", "SHA256": " eb4afb9885cba6dc70cccd05b910b2dbccc02c5900578be5e99f0d3dbf9d76a5", "Priority": "extra", "Maintainer": "Debian Games Team <pkg-games-devel@lists.alioth.debian.org>", "Description": "Common files for Alien Arena client and server ALIEN ARENA is a standalone 3D first person online deathmatch shooter\n crafted from the original source code of Quake II and Quake III, released\n by id Software under the GPL license. With features including 32 bit\n graphics, new particle engine and effects, light blooms, reflective water,\n hi resolution textures and skins, hi poly models, stain maps, ALIEN ARENA\n pushes the envelope of graphical beauty rivaling today's top games.\n .\n This package installs the common files for Alien Arena.\n", "Homepage": "http://red.planetarena.org", "Tag": "role::app-data, role::shared-lib, special::auto-inst-parts", "Installed-Size": "456", "Version": "7.40-2", "Replaces": "alien-arena (<< 7.33-1)", "Size": "187518", "MD5sum": "1e8cba92c41420aa7baa8a5718d67122", "Package": "alien-arena-common", "Section": "contrib/games", "Architecture": "i386"}

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
 eb2a7a57c92ebac9d3d1c2d2d719f290 2583 access-modifier-checker_1.0-4.dsc
 f280abdc753beea4243d99530f023f12 10049 access-modifier-checker_1.0.orig.tar.gz
 182b18d7a4bfd1a4e6bfda886f986766 4310 access-modifier-checker_1.0-4.debian.tar.gz
Dm-Upload-Allowed: yes
Vcs-Browser: http://git.debian.org/?p=pkg-java/access-modifier-checker.git
Vcs-Git: git://git.debian.org/git/pkg-java/access-modifier-checker.git
Checksums-Sha1:
 a70b87bba4391c0138cc9cc75efa747c4eeea2d7 2583 access-modifier-checker_1.0-4.dsc
 c5bea16095ae1685941193370d20756e036d61a7 10049 access-modifier-checker_1.0.orig.tar.gz
 54033d40ae2f35eb147df0876c649eb9d2eb86bc 4310 access-modifier-checker_1.0-4.debian.tar.gz
Checksums-Sha256:
 36ae7f68974bd69450229244510faa8f8056709913770218821069e6cce6b4b8 2583 access-modifier-checker_1.0-4.dsc
 9d362301892bd647a361755bde28f0a39644a5bbb2ff5300060373ee29d9d87a 10049 access-modifier-checker_1.0.orig.tar.gz
 23852c63f7b5511b4c827a443ff0f92926fc71efc57dc47b49cb2fd9136d8f95 4310 access-modifier-checker_1.0-4.debian.tar.gz
Homepage: https://github.com/kohsuke/access-modifier
Package-List:
 libaccess-modifier-checker-java deb java optional
 libaccess-modifier-checker-java-doc deb doc optional
Directory: pool/main/a/access-modifier-checker
Priority: source
Section: java
`
