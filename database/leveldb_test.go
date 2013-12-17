package database

import (
	. "launchpad.net/gocheck"
	"testing"
)

// Launch gocheck tests
func Test(t *testing.T) {
	TestingT(t)
}

type LevelDBSuite struct {
	db Storage
}

var _ = Suite(&LevelDBSuite{})

func (s *LevelDBSuite) SetUpTest(c *C) {
	var err error

	s.db, err = OpenDB(c.MkDir())
	c.Assert(err, IsNil)
}

func (s *LevelDBSuite) TearDownTest(c *C) {
	err := s.db.Close()
	c.Assert(err, IsNil)
}

func (s *LevelDBSuite) TestGetPut(c *C) {
	var (
		key   = []byte("key")
		value = []byte("value")
	)

	_, err := s.db.Get(key)
	c.Assert(err, ErrorMatches, "key not found")

	err = s.db.Put(key, value)
	c.Assert(err, IsNil)

	result, err := s.db.Get(key)
	c.Assert(err, IsNil)
	c.Assert(result, DeepEquals, value)
}
