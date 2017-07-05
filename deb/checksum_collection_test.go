package deb

import (
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/utils"

	. "gopkg.in/check.v1"
)

type ChecksumCollectionSuite struct {
	collection *ChecksumCollection
	c          utils.ChecksumInfo
	db         database.Storage
}

var _ = Suite(&ChecksumCollectionSuite{})

func (s *ChecksumCollectionSuite) SetUpTest(c *C) {
	s.c = utils.ChecksumInfo{
		Size:   124,
		MD5:    "da39a3ee5e6b4b0d3255bfef95601890afd80709",
		SHA1:   "da39a3ee5e6b4b0d3255bfef95601890afd80709",
		SHA256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}
	s.db, _ = database.NewOpenDB(c.MkDir())
	s.collection = NewChecksumCollection(s.db)
}

func (s *ChecksumCollectionSuite) TearDownTest(c *C) {
	s.db.Close()
}

func (s *ChecksumCollectionSuite) TestFlow(c *C) {
	// checksum not stored
	checksum, err := s.collection.Get("some/path")
	c.Assert(err, IsNil)
	c.Check(checksum, IsNil)

	// store checksum
	err = s.collection.Update("some/path", &s.c)
	c.Assert(err, IsNil)

	// load it back
	checksum, err = s.collection.Get("some/path")
	c.Assert(err, IsNil)
	c.Check(*checksum, DeepEquals, s.c)
}
