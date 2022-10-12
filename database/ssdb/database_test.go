package ssdb_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/aptly-dev/aptly/database"
	"github.com/aptly-dev/aptly/database/ssdb"
	"github.com/seefan/gossdb/v2/conf"
	. "gopkg.in/check.v1"
)

// Launch gocheck tests
func Test(t *testing.T) {
	TestingT(t)
}

func setUpSsdb() error {
	setUpStr := `
	#!/bin/bash
	if [ ! -e /tmp/ssdb-master/ssdb-master ]; then
		mkdir -p /tmp/ssdb-master
		wget --no-check-certificate https://github.com/ideawu/ssdb/archive/master.zip -O /tmp/ssdb-master/master.zip
		cd /tmp/ssdb-master && unzip master && cd ssdb-master && make all
	fi
	cd /tmp/ssdb-master/ssdb-master && ./ssdb-server -d ssdb.conf -s restart
	sleep 2`

	tmpShell, err := ioutil.TempFile("/tmp", "ssdbSetup")
	if err != nil {
		return err
	}
	defer os.Remove(tmpShell.Name())

	_, err = tmpShell.WriteString(setUpStr)
	if err != nil {
		return err
	}

	cmd := exec.Command("/bin/bash", tmpShell.Name())
	fmt.Println(cmd.String())
	output, err := cmd.Output()
	fmt.Println(string(output))
	if err != nil {
		return err
	}

	return nil
}

func TestMain(m *testing.M) {
	setUpSsdb()
	m.Run()
}

type SSDBSuite struct {
	cfg *conf.Config
	db  database.Storage
}

var _ = Suite(&SSDBSuite{cfg: &conf.Config{
	Host: "127.0.0.1",
	Port: 8888,
}})

func (s *SSDBSuite) SetUpTest(c *C) {
	var err error
	s.db, err = ssdb.NewOpenDB(s.cfg)
	c.Assert(err, IsNil)
}

func (s *SSDBSuite) TestSetUpTest(c *C) {
	var err error
	s.db, err = ssdb.NewOpenDB(s.cfg)
	c.Assert(err, IsNil)
}

func (s *SSDBSuite) TestGetPut(c *C) {
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

func (s *SSDBSuite) TestTemporaryDelete(c *C) {
	fmt.Println("TestTemporaryDelete")
	var (
		key   = []byte("key")
		value = []byte("value")
	)

	temp, err := s.db.CreateTemporary()
	c.Assert(err, IsNil)

	c.Check(temp.HasPrefix([]byte(nil)), Equals, false)

	err = temp.Put(key, value)
	c.Assert(err, IsNil)
	c.Check(temp.HasPrefix([]byte(nil)), Equals, true)

	c.Assert(temp.Close(), IsNil)
	c.Assert(temp.Drop(), IsNil)
}

func (s *SSDBSuite) TestDelete(c *C) {
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

func (s *SSDBSuite) TestByPrefix(c *C) {
	//c.Check(s.db.FetchByPrefix([]byte{0x80}), DeepEquals, [][]byte{})

	s.db.Put([]byte{0x80, 0x01}, []byte{0x01})
	s.db.Put([]byte{0x80, 0x03}, []byte{0x03})
	s.db.Put([]byte{0x80, 0x02}, []byte{0x02})
	c.Check(len(s.db.FetchByPrefix([]byte{0x80})), DeepEquals, len([][]byte{{0x01}, {0x02}, {0x03}}))
	c.Check(len(s.db.KeysByPrefix([]byte{0x80})), DeepEquals, len([][]byte{{0x80, 0x01}, {0x80, 0x02}, {0x80, 0x03}}))

	s.db.Put([]byte{0x90, 0x01}, []byte{0x04})
	c.Check(len(s.db.FetchByPrefix([]byte{0x80})), DeepEquals, len([][]byte{{0x01}, {0x02}, {0x03}}))
	c.Check(len(s.db.KeysByPrefix([]byte{0x80})), DeepEquals, len([][]byte{{0x80, 0x01}, {0x80, 0x02}, {0x80, 0x03}}))

	s.db.Put([]byte{0x00, 0x01}, []byte{0x05})
	c.Check(len(s.db.FetchByPrefix([]byte{0x80})), DeepEquals, len([][]byte{{0x01}, {0x02}, {0x03}}))
	c.Check(len(s.db.KeysByPrefix([]byte{0x80})), DeepEquals, len([][]byte{{0x80, 0x01}, {0x80, 0x02}, {0x80, 0x03}}))

	keys := [][]byte{}
	values := [][]byte{}

	c.Check(s.db.ProcessByPrefix([]byte{0x80}, func(k, v []byte) error {
		keys = append(keys, append([]byte(nil), k...))
		values = append(values, append([]byte(nil), v...))
		return nil
	}), IsNil)

	c.Check(len(values), DeepEquals, len([][]byte{{0x01}, {0x02}, {0x03}}))
	c.Check(len(keys), DeepEquals, len([][]byte{{0x80, 0x01}, {0x80, 0x02}, {0x80, 0x03}}))

	c.Check(s.db.ProcessByPrefix([]byte{0x80}, func(k, v []byte) error {
		return database.ErrNotFound
	}), Equals, database.ErrNotFound)

	c.Check(s.db.ProcessByPrefix([]byte{0xa0}, func(k, v []byte) error {
		return database.ErrNotFound
	}), IsNil)

	c.Check(s.db.FetchByPrefix([]byte{0xa0}), DeepEquals, [][]byte{})
	c.Check(s.db.KeysByPrefix([]byte{0xa0}), DeepEquals, [][]byte{})
}

func (s *SSDBSuite) TestHasPrefix(c *C) {
	s.db.Put([]byte{0x80, 0x01}, []byte{0x01})

	//c.Check(s.db.HasPrefix([]byte("")), Equals, true)
	c.Check(s.db.HasPrefix([]byte{0x80}), Equals, true)
	c.Check(s.db.HasPrefix([]byte{0x79}), Equals, false)
}

func (s *SSDBSuite) TestTransactionCommit(c *C) {
	var (
		key    = []byte("key")
		key2   = []byte("key2")
		value  = []byte("value")
		value2 = []byte("value2")
	)
	s.db.Delete(key)
	s.db.Delete(key2)
	transaction, err := s.db.OpenTransaction()
	c.Assert(err, IsNil)
	defer transaction.Discard()

	err = s.db.Put(key, value)
	c.Assert(err, IsNil)

	v, err := s.db.Get(key)
	c.Assert(err, IsNil)
	c.Check(v, DeepEquals, value)

	err = transaction.Put(key2, value2)
	c.Assert(err, IsNil)
	v, err = transaction.Get(key2)
	c.Check(err, IsNil)
	c.Check(v, DeepEquals, value2)
	_, err = s.db.Get(key2)
	c.Assert(err, ErrorMatches, "key not found")

	err = transaction.Delete(key)
	c.Assert(err, IsNil)
	_, err = transaction.Get(key)
	c.Assert(err, ErrorMatches, "key not found")
	v, err = s.db.Get(key)
	c.Assert(err, IsNil)
	c.Check(v, DeepEquals, value)

	err = transaction.Commit()
	c.Check(err, IsNil)

	v, err = s.db.Get(key2)
	c.Check(err, IsNil)
	c.Check(v, DeepEquals, value2)

	_, err = s.db.Get(key)
	c.Assert(err, ErrorMatches, "key not found")
}

func (s *SSDBSuite) TestBatch(c *C) {
	var (
		key    = []byte("bkey")
		key2   = []byte("bkey2")
		value  = []byte("bvalue")
		value2 = []byte("bvalue2")
	)

	err := s.db.Put(key, value)
	c.Check(err, IsNil)

	batch := s.db.CreateBatch()
	batch.Put(key2, value2)
	v, err := s.db.Get(key)
	c.Check(err, IsNil)
	c.Check(v, DeepEquals, value)
	_, err = s.db.Get(key2)
	c.Check(err, ErrorMatches, "key not found")

	err = batch.Write()
	c.Check(err, IsNil)

	v, err = s.db.Get(key2)
	c.Check(err, IsNil)
	c.Check(v, DeepEquals, value2)

	batch = s.db.CreateBatch()
	batch.Delete(key)
	batch.Delete(key2)
	c.Check(err, IsNil)
	v, err = s.db.Get(key)
	c.Check(err, IsNil)
	c.Check(v, DeepEquals, value)
	c.Check(err, IsNil)
	v, err = s.db.Get(key2)
	c.Check(err, IsNil)
	c.Check(v, DeepEquals, value2)

	err = batch.Write()
	c.Check(err, IsNil)

	_, err = s.db.Get(key2)
	c.Check(err, ErrorMatches, "key not found")
	_, err = s.db.Get(key)
	c.Check(err, ErrorMatches, "key not found")
}
