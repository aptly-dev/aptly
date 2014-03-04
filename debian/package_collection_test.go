package debian

import (
	"github.com/smira/aptly/database"
	. "launchpad.net/gocheck"
)

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

func (s *PackageCollectionSuite) TestUpdate(c *C) {
	// package doesn't exist, update ok
	err := s.collection.Update(s.p)
	c.Assert(err, IsNil)
	res, err := s.collection.ByKey(s.p.Key(""))
	c.Assert(err, IsNil)
	c.Assert(res.Equals(s.p), Equals, true)

	// same package, ok
	p2 := NewPackageFromControlFile(packageStanza.Copy())
	err = s.collection.Update(p2)
	c.Assert(err, IsNil)
	res, err = s.collection.ByKey(p2.Key(""))
	c.Assert(err, IsNil)
	c.Assert(res.Equals(s.p), Equals, true)

	// change some metadata
	p2.Source = "lala"
	err = s.collection.Update(p2)
	c.Assert(err, IsNil)
	res, err = s.collection.ByKey(p2.Key(""))
	c.Assert(err, IsNil)
	c.Assert(res.Equals(s.p), Equals, false)
	c.Assert(res.Equals(p2), Equals, true)

	// change file info
	p2 = NewPackageFromControlFile(packageStanza.Copy())
	p2.UpdateFiles(nil)
	res, err = s.collection.ByKey(p2.Key(""))
	err = s.collection.Update(p2)
	c.Assert(err, ErrorMatches, ".*conflict with existing packge")
	p2 = NewPackageFromControlFile(packageStanza.Copy())
	files := p2.Files()
	files[0].Checksums.MD5 = "abcdef"
	p2.UpdateFiles(files)
	res, err = s.collection.ByKey(p2.Key(""))
	err = s.collection.Update(p2)
	c.Assert(err, ErrorMatches, ".*conflict with existing packge")
}

func (s *PackageCollectionSuite) TestByKey(c *C) {
	err := s.collection.Update(s.p)
	c.Assert(err, IsNil)

	p2, err := s.collection.ByKey(s.p.Key(""))
	c.Assert(err, IsNil)
	c.Assert(p2.Equals(s.p), Equals, true)
}

func (s *PackageCollectionSuite) TestAllPackageRefs(c *C) {
	err := s.collection.Update(s.p)
	c.Assert(err, IsNil)

	refs := s.collection.AllPackageRefs()
	c.Check(refs.Len(), Equals, 1)
	c.Check(refs.Refs[0], DeepEquals, s.p.Key(""))
}

func (s *PackageCollectionSuite) TestDeleteByKey(c *C) {
	err := s.collection.Update(s.p)
	c.Assert(err, IsNil)

	err = s.collection.DeleteByKey(s.p.Key(""))
	c.Check(err, IsNil)

	_, err = s.collection.ByKey(s.p.Key(""))
	c.Check(err, ErrorMatches, "key not found")
}
