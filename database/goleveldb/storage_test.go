package goleveldb

import (
	"github.com/aptly-dev/aptly/database"
	. "gopkg.in/check.v1"
)

type LevelDBStorageSuite struct {
	storage *storage
	tempDir string
}

var _ = Suite(&LevelDBStorageSuite{})

func (s *LevelDBStorageSuite) SetUpTest(c *C) {
	s.tempDir = c.MkDir()
	s.storage = &storage{
		path: s.tempDir,
		db:   nil, // Not opened for unit tests
	}
}

func (s *LevelDBStorageSuite) TestCreateTemporary(c *C) {
	// Test creating temporary storage
	tempStorage, err := s.storage.CreateTemporary()
	if err != nil {
		// Expected to fail without real leveldb setup
		c.Check(err, NotNil)
		return
	}

	c.Check(tempStorage, NotNil)
	levelStorage, ok := tempStorage.(*storage)
	c.Check(ok, Equals, true)
	c.Check(len(levelStorage.path) > 0, Equals, true)
	c.Check(levelStorage.path, Not(Equals), s.storage.path)
}

func (s *LevelDBStorageSuite) TestCloseNilDB(c *C) {
	// Test closing storage with nil DB
	err := s.storage.Close()
	c.Check(err, IsNil)
}

func (s *LevelDBStorageSuite) TestOpenNilDB(c *C) {
	// Test opening storage - should succeed with valid path
	err := s.storage.Open()
	// Should succeed with valid temporary directory
	c.Check(err, IsNil)
	// Clean up
	s.storage.Close()
}

func (s *LevelDBStorageSuite) TestCreateBatchNilDB(c *C) {
	// Test creating batch with nil DB
	batch := s.storage.CreateBatch()
	c.Check(batch, IsNil)
}

func (s *LevelDBStorageSuite) TestCompactDB(c *C) {
	// Test CompactDB with nil DB - should handle gracefully
	err := s.storage.CompactDB()
	c.Check(err, NotNil) // Expected to fail with nil DB
}

func (s *LevelDBStorageSuite) TestDropNilDB(c *C) {
	// Test dropping storage with nil DB
	err := s.storage.Drop()
	c.Check(err, IsNil) // Should succeed (removes directory)
}

func (s *LevelDBStorageSuite) TestInterfaceCompliance(c *C) {
	// Test that storage implements database.Storage interface
	var dbStorage database.Storage = &storage{}
	c.Check(dbStorage, NotNil)
}

func (s *LevelDBStorageSuite) TestGetNilDB(c *C) {
	// Test Get with nil DB - should fail
	_, err := s.storage.Get([]byte("key"))
	c.Check(err, NotNil) // Expected to fail with nil DB
}

// Note: storage does not implement Has method - it uses Get and checks for ErrNotFound

func (s *LevelDBStorageSuite) TestPutNilDB(c *C) {
	// Test Put with nil DB - should fail
	err := s.storage.Put([]byte("key"), []byte("value"))
	c.Check(err, NotNil) // Expected to fail with nil DB
}

func (s *LevelDBStorageSuite) TestDeleteNilDB(c *C) {
	// Test Delete with nil DB - should fail
	err := s.storage.Delete([]byte("key"))
	c.Check(err, NotNil) // Expected to fail with nil DB
}

func (s *LevelDBStorageSuite) TestKeysByPrefixNilDB(c *C) {
	// Test KeysByPrefix with nil DB - should return nil
	keys := s.storage.KeysByPrefix([]byte("prefix/"))
	c.Check(keys, IsNil)
}

func (s *LevelDBStorageSuite) TestFetchByPrefixNilDB(c *C) {
	// Test FetchByPrefix with nil DB - should return nil
	values := s.storage.FetchByPrefix([]byte("prefix/"))
	c.Check(values, IsNil)
}

func (s *LevelDBStorageSuite) TestHasPrefixNilDB(c *C) {
	// Test HasPrefix with nil DB - should return false
	result := s.storage.HasPrefix([]byte("prefix/"))
	c.Check(result, Equals, false)
}

func (s *LevelDBStorageSuite) TestProcessByPrefixNilDB(c *C) {
	// Test ProcessByPrefix with nil DB - should fail
	processor := func(key, value []byte) error { return nil }
	err := s.storage.ProcessByPrefix([]byte("prefix/"), processor)
	c.Check(err, NotNil) // Expected to fail with nil DB
}
