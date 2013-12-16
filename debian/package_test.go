package debian

import (
	debc "github.com/smira/godebiancontrol"
	. "launchpad.net/gocheck"
)

var packagePara = debc.Paragraph{"Source": "alien-arena", "Depends": "libc6 (>= 2.7), alien-arena-data (>= 7.40)", "Filename": "pool/contrib/a/alien-arena/alien-arena-common_7.40-2_i386.deb", "SHA1": "46955e48cad27410a83740a21d766ce362364024", "SHA256": "eb4afb9885cba6dc70cccd05b910b2dbccc02c5900578be5e99f0d3dbf9d76a5", "Priority": "extra", "Maintainer": "Debian Games Team <pkg-games-devel@lists.alioth.debian.org>", "Description": "Common files for Alien Arena client and server ALIEN ARENA is a standalone 3D first person online deathmatch shooter\n crafted from the original source code of Quake II and Quake III, released\n by id Software under the GPL license. With features including 32 bit\n graphics, new particle engine and effects, light blooms, reflective water,\n hi resolution textures and skins, hi poly models, stain maps, ALIEN ARENA\n pushes the envelope of graphical beauty rivaling today's top games.\n .\n This package installs the common files for Alien Arena.\n", "Homepage": "http://red.planetarena.org", "Tag": "role::app-data, role::shared-lib, special::auto-inst-parts", "Installed-Size": "456", "Version": "7.40-2", "Replaces": "alien-arena (<< 7.33-1)", "Size": "187518", "MD5sum": "1e8cba92c41420aa7baa8a5718d67122", "Package": "alien-arena-common", "Section": "contrib/games", "Architecture": "i386"}

type PackageSuite struct {
	para debc.Paragraph
}

var _ = Suite(&PackageSuite{})

func (s *PackageSuite) SetUpTest(c *C) {
	s.para = make(debc.Paragraph)
	for k, v := range packagePara {
		s.para[k] = v
	}
}

func (s *PackageSuite) TestNewFromPara(c *C) {
	p := NewPackageFromControlFile(s.para)

	c.Check(p.Name, Equals, "alien-arena-common")
	c.Check(p.Version, Equals, "7.40-2")
	c.Check(p.Architecture, Equals, "i386")
	c.Check(p.Filename, Equals, "pool/contrib/a/alien-arena/alien-arena-common_7.40-2_i386.deb")
	c.Check(p.Depends, DeepEquals, []string{"libc6 (>= 2.7)", "alien-arena-data (>= 7.40)"})
	c.Check(p.Suggests, IsNil)
}

func (s *PackageSuite) TestKey(c *C) {
	p := NewPackageFromControlFile(s.para)

	c.Check(p.Key(), DeepEquals, []byte("alien-arena-common 7.40-2"))
}

func (s *PackageSuite) TestEncodeDecode(c *C) {
	p := NewPackageFromControlFile(s.para)
	encoded := p.Encode()
	p2 := &Package{}
	err := p2.Decode(encoded)

	c.Assert(err, IsNil)
	c.Assert(p2, DeepEquals, p)
}
