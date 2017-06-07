package deb

import (
	"os"
	"path/filepath"

	. "gopkg.in/check.v1"
)

type ChangesSuite struct {
	Dir, Path string
}

var _ = Suite(&ChangesSuite{})

func (s *ChangesSuite) SetUpTest(c *C) {
	s.Dir = c.MkDir()
	s.Path = filepath.Join(s.Dir, "calamares.changes")

	f, err := os.Create(s.Path)
	c.Assert(err, IsNil)

	f.WriteString(changesFile)
	f.Close()
}

func (s *ChangesSuite) TestParseAndVerify(c *C) {
	changes, err := NewChanges(s.Path)
	c.Assert(err, IsNil)

	err = changes.VerifyAndParse(true, true, &NullVerifier{})
	c.Check(err, IsNil)

	c.Check(changes.Distribution, Equals, "sid")
	c.Check(changes.Files, HasLen, 4)
	c.Check(changes.Files[0].Filename, Equals, "calamares_0+git20141127.99.dsc")
	c.Check(changes.Files[0].Checksums.Size, Equals, int64(1106))
	c.Check(changes.Files[0].Checksums.MD5, Equals, "05fd8f3ffe8f362c5ef9bad2f936a56e")
	c.Check(changes.Files[0].Checksums.SHA1, Equals, "79f10e955dab6eb25b7f7bae18213f367a3a0396")
	c.Check(changes.Files[0].Checksums.SHA256, Equals, "35b3280a7b1ffe159a276128cb5c408d687318f60ecbb8ab6dedb2e49c4e82dc")
	c.Check(changes.BasePath, Equals, s.Dir)
	c.Check(changes.Architectures, DeepEquals, []string{"source", "amd64"})
	c.Check(changes.Source, Equals, "calamares")
	c.Check(changes.Binary, DeepEquals, []string{"calamares", "calamares-dbg"})
}

func (s *ChangesSuite) TestPackageQuery(c *C) {
	changes, err := NewChanges(s.Path)
	c.Assert(err, IsNil)

	err = changes.VerifyAndParse(true, true, &NullVerifier{})
	c.Check(err, IsNil)

	q, err := changes.PackageQuery()
	c.Check(err, IsNil)

	c.Check(q.String(), Equals,
		"(($Architecture (= amd64)) | (($Architecture (= source)) | ($Architecture (= )))), ((($PackageType (= source)), (Name (= calamares))) | ((!($PackageType (= source))), (((Name (= calamares-dbg)) | (Name (= calamares))) | ((Source (= calamares)), ((Name (= calamares-dbg-dbgsym)) | (Name (= calamares-dbgsym)))))))")
}

var changesFile = `Format: 1.8
Date: Thu, 27 Nov 2014 13:24:53 +0000
Source: calamares
Binary: calamares calamares-dbg
Architecture: source amd64
Version: 0+git20141127.99
Distribution: sid
Urgency: medium
Maintainer: Rohan Garg <rohan@kde.org>
Changed-By: Rohan <rohan@kde.org>
Description:
 calamares  - distribution-independent installer framework
 calamares-dbg - distribution-independent installer framework -- debug symbols
Changes:
 calamares (0+git20141127.99) sid; urgency=medium
 .
   * Update from git
Checksums-Sha1:
 79f10e955dab6eb25b7f7bae18213f367a3a0396 1106 calamares_0+git20141127.99.dsc
 294c28e2c8e34e72ca9ee0d9da5c14f3bf4188db 2694800 calamares_0+git20141127.99.tar.xz
 d6c26c04b5407c7511f61cb3e3de60c4a1d6c4ff 1698924 calamares_0+git20141127.99_amd64.deb
 a3da632d193007b0d4a1aff73159fde1b532d7a8 12835902 calamares-dbg_0+git20141127.99_amd64.deb
Checksums-Sha256:
 35b3280a7b1ffe159a276128cb5c408d687318f60ecbb8ab6dedb2e49c4e82dc 1106 calamares_0+git20141127.99.dsc
 5576b9caaf814564830f95561227e4f04ee87b31da22c1371aab155cbf7ce395 2694800 calamares_0+git20141127.99.tar.xz
 2e6e2f232ed7ffe52369928ebdf5436d90feb37840286ffba79e87d57a43a2e9 1698924 calamares_0+git20141127.99_amd64.deb
 8dd926080ed7bad2e2439e37e49ce12d5f1357c5041b7da4d860a1041f878a8a 12835902 calamares-dbg_0+git20141127.99_amd64.deb
Files:
 05fd8f3ffe8f362c5ef9bad2f936a56e 1106 devel optional calamares_0+git20141127.99.dsc
 097e55c81abd8e5f30bb2eed90c2c1e9 2694800 devel optional calamares_0+git20141127.99.tar.xz
 827fb3b12534241e119815d331e8197b 1698924 devel optional calamares_0+git20141127.99_amd64.deb
 e6f8ce70f564d1f68cb57758b15b13e3 12835902 debug optional calamares-dbg_0+git20141127.99_amd64.deb`
