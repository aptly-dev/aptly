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

func (s *LevelDBSuite) TestDelete(c *C) {
	var (
		key   = []byte("key")
		value = []byte("value")
	)

	err := s.db.Put(key, value)
	c.Assert(err, IsNil)

	err = s.db.Delete(key)
	c.Assert(err, IsNil)

	_, err = s.db.Get(key)
	c.Assert(err, ErrorMatches, "key not found")

	err = s.db.Delete(key)
	c.Assert(err, IsNil)
}

func (s *LevelDBSuite) TestByPrefix(c *C) {
	c.Check(s.db.FetchByPrefix([]byte{0x80}), DeepEquals, [][]byte{})

	s.db.Put([]byte{0x80, 0x01}, []byte{0x01})
	s.db.Put([]byte{0x80, 0x03}, []byte{0x03})
	s.db.Put([]byte{0x80, 0x02}, []byte{0x02})
	c.Check(s.db.FetchByPrefix([]byte{0x80}), DeepEquals, [][]byte{{0x01}, {0x02}, {0x03}})
	c.Check(s.db.KeysByPrefix([]byte{0x80}), DeepEquals, [][]byte{{0x80, 0x01}, {0x80, 0x02}, {0x80, 0x03}})

	s.db.Put([]byte{0x90, 0x01}, []byte{0x04})
	c.Check(s.db.FetchByPrefix([]byte{0x80}), DeepEquals, [][]byte{{0x01}, {0x02}, {0x03}})
	c.Check(s.db.KeysByPrefix([]byte{0x80}), DeepEquals, [][]byte{{0x80, 0x01}, {0x80, 0x02}, {0x80, 0x03}})

	s.db.Put([]byte{0x00, 0x01}, []byte{0x05})
	c.Check(s.db.FetchByPrefix([]byte{0x80}), DeepEquals, [][]byte{{0x01}, {0x02}, {0x03}})
	c.Check(s.db.KeysByPrefix([]byte{0x80}), DeepEquals, [][]byte{{0x80, 0x01}, {0x80, 0x02}, {0x80, 0x03}})

	c.Check(s.db.FetchByPrefix([]byte{0xa0}), DeepEquals, [][]byte{})
	c.Check(s.db.KeysByPrefix([]byte{0xa0}), DeepEquals, [][]byte{})
}

func (s *LevelDBSuite) TestBatch(c *C) {
	var (
		key    = []byte("key")
		key2   = []byte("key2")
		value  = []byte("value")
		value2 = []byte("value2")
	)

	err := s.db.Put(key, value)
	c.Assert(err, IsNil)

	s.db.StartBatch()
	s.db.Put(key2, value2)
	s.db.Delete(key)

	v, err := s.db.Get(key)
	c.Check(err, IsNil)
	c.Check(v, DeepEquals, value)

	_, err = s.db.Get(key2)
	c.Check(err, ErrorMatches, "key not found")

	err = s.db.FinishBatch()
	c.Check(err, IsNil)

	v2, err := s.db.Get(key2)
	c.Check(err, IsNil)
	c.Check(v2, DeepEquals, value2)

	_, err = s.db.Get(key)
	c.Check(err, ErrorMatches, "key not found")

	c.Check(func() { s.db.FinishBatch() }, Panics, "no batch")

	s.db.StartBatch()
	c.Check(func() { s.db.StartBatch() }, Panics, "batch already started")
}
