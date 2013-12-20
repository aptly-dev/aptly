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

func (s *LevelDBSuite) TestFetchByPrefix(c *C) {
	c.Check(s.db.FetchByPrefix([]byte{0x80}), DeepEquals, [][]byte{})

	s.db.Put([]byte{0x80, 0x01}, []byte{0x01})
	s.db.Put([]byte{0x80, 0x03}, []byte{0x03})
	s.db.Put([]byte{0x80, 0x02}, []byte{0x02})
	c.Check(s.db.FetchByPrefix([]byte{0x80}), DeepEquals, [][]byte{{0x01}, {0x02}, {0x03}})

	s.db.Put([]byte{0x90, 0x01}, []byte{0x04})
	c.Check(s.db.FetchByPrefix([]byte{0x80}), DeepEquals, [][]byte{{0x01}, {0x02}, {0x03}})

	s.db.Put([]byte{0x00, 0x01}, []byte{0x05})
	c.Check(s.db.FetchByPrefix([]byte{0x80}), DeepEquals, [][]byte{{0x01}, {0x02}, {0x03}})

	c.Check(s.db.FetchByPrefix([]byte{0xa0}), DeepEquals, [][]byte{})
}
