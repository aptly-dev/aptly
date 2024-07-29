package etcddb_test

import (
	"testing"

	"github.com/aptly-dev/aptly/database"
	"github.com/aptly-dev/aptly/database/etcddb"
	. "gopkg.in/check.v1"
)

// Launch gocheck tests
func Test(t *testing.T) {
	TestingT(t)
}

type EtcDDBSuite struct {
	url string
	db  database.Storage
}

var _ = Suite(&EtcDDBSuite{})

func (s *EtcDDBSuite) SetUpTest(c *C) {
	var err error
	s.db, err = etcddb.NewDB("127.0.0.1:2379")
	c.Assert(err, IsNil)
}

func (s *EtcDDBSuite) TestSetUpTest(c *C) {
	var err error
	s.db, err = etcddb.NewDB("127.0.0.1:2379")
	c.Assert(err, IsNil)
}

func (s *EtcDDBSuite) TestGetPut(c *C) {
	var (
		key   = []byte("key")
		value = []byte("value")
	)
	var err error

	err = s.db.Put(key, value)
	c.Assert(err, IsNil)

	result, err := s.db.Get(key)
	c.Assert(err, IsNil)
	c.Assert(result, DeepEquals, value)
}

func (s *EtcDDBSuite) TestDelete(c *C) {
	var (
		key   = []byte("key")
		value = []byte("value")
	)

	err := s.db.Put(key, value)
	c.Assert(err, IsNil)

	_, err = s.db.Get(key)
	c.Assert(err, IsNil)

	err = s.db.Delete(key)
	c.Assert(err, IsNil)

}

func (s *EtcDDBSuite) TestByPrefix(c *C) {
	//c.Check(s.db.FetchByPrefix([]byte{0x80}), DeepEquals, [][]byte{})

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

	keys := [][]byte{}
	values := [][]byte{}

	c.Check(s.db.ProcessByPrefix([]byte{0x80}, func(k, v []byte) error {
		keys = append(keys, append([]byte(nil), k...))
		values = append(values, append([]byte(nil), v...))
		return nil
	}), IsNil)

	c.Check(values, DeepEquals, [][]byte{{0x01}, {0x02}, {0x03}})
	c.Check(keys, DeepEquals, [][]byte{{0x80, 0x01}, {0x80, 0x02}, {0x80, 0x03}})

	c.Check(s.db.ProcessByPrefix([]byte{0x80}, func(k, v []byte) error {
		return database.ErrNotFound
	}), Equals, database.ErrNotFound)

	c.Check(s.db.ProcessByPrefix([]byte{0xa0}, func(k, v []byte) error {
		return database.ErrNotFound
	}), IsNil)

	c.Check(s.db.FetchByPrefix([]byte{0xa0}), DeepEquals, [][]byte{})
	c.Check(s.db.KeysByPrefix([]byte{0xa0}), DeepEquals, [][]byte{})
}

func (s *EtcDDBSuite) TestHasPrefix(c *C) {
	//c.Check(s.db.HasPrefix([]byte(nil)), Equals, false)
	//c.Check(s.db.HasPrefix([]byte{0x80}), Equals, false)

	s.db.Put([]byte{0x80, 0x01}, []byte{0x01})

	c.Check(s.db.HasPrefix([]byte(nil)), Equals, true)
	c.Check(s.db.HasPrefix([]byte{0x80}), Equals, true)
	c.Check(s.db.HasPrefix([]byte{0x79}), Equals, false)
}

func (s *EtcDDBSuite) TestTransactionCommit(c *C) {
	var (
		key    = []byte("key")
		key2   = []byte("key2")
		value  = []byte("value")
		value2 = []byte("value2")
	)
	transaction, err := s.db.OpenTransaction()

	err = s.db.Put(key, value)
	c.Assert(err, IsNil)

	c.Assert(err, IsNil)
	transaction.Put(key2, value2)
	v, err := s.db.Get(key)
	c.Check(v, DeepEquals, value)
        err = transaction.Delete(key)
	c.Assert(err, IsNil)

	_, err = transaction.Get(key2)
	c.Assert(err, IsNil)

	v2, err := transaction.Get(key2)
	c.Check(err, IsNil)
	c.Check(v2, DeepEquals, value2)

	_, err = transaction.Get(key)
	c.Assert(err, IsNil)

	err = transaction.Commit()
	c.Check(err, IsNil)

	v2, err = transaction.Get(key2)
	c.Check(err, IsNil)
	c.Check(v2, DeepEquals, value2)

	_, err = transaction.Get(key)
	c.Assert(err, NotNil)
}

