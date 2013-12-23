package debian

import (
	"github.com/smira/aptly/database"
	. "launchpad.net/gocheck"
)

var packageStanza = Stanza{"Source": "alien-arena", "Depends": "libc6 (>= 2.7), alien-arena-data (>= 7.40)", "Filename": "pool/contrib/a/alien-arena/alien-arena-common_7.40-2_i386.deb", "SHA1": "46955e48cad27410a83740a21d766ce362364024", "SHA256": "eb4afb9885cba6dc70cccd05b910b2dbccc02c5900578be5e99f0d3dbf9d76a5", "Priority": "extra", "Maintainer": "Debian Games Team <pkg-games-devel@lists.alioth.debian.org>", "Description": "Common files for Alien Arena client and server ALIEN ARENA is a standalone 3D first person online deathmatch shooter\n crafted from the original source code of Quake II and Quake III, released\n by id Software under the GPL license. With features including 32 bit\n graphics, new particle engine and effects, light blooms, reflective water,\n hi resolution textures and skins, hi poly models, stain maps, ALIEN ARENA\n pushes the envelope of graphical beauty rivaling today's top games.\n .\n This package installs the common files for Alien Arena.\n", "Homepage": "http://red.planetarena.org", "Tag": "role::app-data, role::shared-lib, special::auto-inst-parts", "Installed-Size": "456", "Version": "7.40-2", "Replaces": "alien-arena (<< 7.33-1)", "Size": "187518", "MD5sum": "1e8cba92c41420aa7baa8a5718d67122", "Package": "alien-arena-common", "Section": "contrib/games", "Architecture": "i386"}

type PackageSuite struct {
	para Stanza
}

var _ = Suite(&PackageSuite{})

func (s *PackageSuite) SetUpTest(c *C) {
	s.para = packageStanza.Copy()
}

func (s *PackageSuite) TestNewFromPara(c *C) {
	p := NewPackageFromControlFile(s.para)

	c.Check(p.Name, Equals, "alien-arena-common")
	c.Check(p.Version, Equals, "7.40-2")
	c.Check(p.Architecture, Equals, "i386")
	c.Check(p.Filename, Equals, "pool/contrib/a/alien-arena/alien-arena-common_7.40-2_i386.deb")
	c.Check(p.Depends, DeepEquals, []string{"libc6 (>= 2.7)", "alien-arena-data (>= 7.40)"})
	c.Check(p.Suggests, IsNil)
	c.Check(p.Filesize, Equals, int64(187518))
}

func (s *PackageSuite) TestKey(c *C) {
	p := NewPackageFromControlFile(s.para)

	c.Check(p.Key(), DeepEquals, []byte("Palien-arena-common 7.40-2 i386"))
}

func (s *PackageSuite) TestEncodeDecode(c *C) {
	p := NewPackageFromControlFile(s.para)
	encoded := p.Encode()
	p2 := &Package{}
	err := p2.Decode(encoded)

	c.Assert(err, IsNil)
	c.Assert(p2, DeepEquals, p)
}

func (s *PackageSuite) TestString(c *C) {
	p := NewPackageFromControlFile(s.para)
	c.Assert(p.String(), Equals, "alien-arena-common-7.40-2_i386")
}

func (s *PackageSuite) TestEquals(c *C) {
	p := NewPackageFromControlFile(s.para)

	stanza2 := packageStanza.Copy()
	p2 := NewPackageFromControlFile(stanza2)
	c.Check(p.Equals(p2), Equals, true)

	p2.Depends = []string{"package1"}
	c.Check(p.Equals(p2), Equals, false)
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
