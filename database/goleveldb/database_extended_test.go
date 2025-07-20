package goleveldb_test

import (
	"errors"
	"os"
	"path/filepath"

	. "gopkg.in/check.v1"

	"github.com/aptly-dev/aptly/database"
	"github.com/aptly-dev/aptly/database/goleveldb"
)

type ExtendedLevelDBSuite struct {
	tempDir string
}

var _ = Suite(&ExtendedLevelDBSuite{})

func (s *ExtendedLevelDBSuite) SetUpTest(c *C) {
	s.tempDir = c.MkDir()
}

func (s *ExtendedLevelDBSuite) TestNewDB(c *C) {
	// Test NewDB function
	dbPath := filepath.Join(s.tempDir, "test-db")

	db, err := goleveldb.NewDB(dbPath)
	c.Check(err, IsNil)
	c.Check(db, NotNil)

	// DB should not be open yet
	_, err = db.Get([]byte("test"))
	c.Check(err, NotNil) // Should error because DB is not open

	// Open the database
	err = db.Open()
	c.Check(err, IsNil)

	// Now should work
	_, err = db.Get([]byte("test"))
	c.Check(err, Equals, database.ErrNotFound) // Key not found but no open error

	err = db.Close()
	c.Check(err, IsNil)
}

func (s *ExtendedLevelDBSuite) TestNewOpenDB(c *C) {
	// Test NewOpenDB function
	dbPath := filepath.Join(s.tempDir, "test-open-db")

	db, err := goleveldb.NewOpenDB(dbPath)
	c.Check(err, IsNil)
	c.Check(db, NotNil)

	// DB should be open and ready to use
	_, err = db.Get([]byte("test"))
	c.Check(err, Equals, database.ErrNotFound) // Key not found but no open error

	err = db.Close()
	c.Check(err, IsNil)
}

func (s *ExtendedLevelDBSuite) TestRecoverDBError(c *C) {
	// Test RecoverDB with invalid path
	invalidPath := "/invalid/nonexistent/path"

	err := goleveldb.RecoverDB(invalidPath)
	c.Check(err, NotNil) // Should error with invalid path
}

func (s *ExtendedLevelDBSuite) TestRecoverDBValidPath(c *C) {
	// Test RecoverDB with valid database
	dbPath := filepath.Join(s.tempDir, "recover-test")

	// First create a database
	db, err := goleveldb.NewOpenDB(dbPath)
	c.Check(err, IsNil)

	// Add some data
	err = db.Put([]byte("key1"), []byte("value1"))
	c.Check(err, IsNil)

	err = db.Close()
	c.Check(err, IsNil)

	// Now recover it
	err = goleveldb.RecoverDB(dbPath)
	c.Check(err, IsNil)

	// Verify data is still there after recovery
	db2, err := goleveldb.NewOpenDB(dbPath)
	c.Check(err, IsNil)

	value, err := db2.Get([]byte("key1"))
	c.Check(err, IsNil)
	c.Check(value, DeepEquals, []byte("value1"))

	err = db2.Close()
	c.Check(err, IsNil)
}

func (s *ExtendedLevelDBSuite) TestCreateTemporaryError(c *C) {
	// Test CreateTemporary with limited permissions (if possible)
	dbPath := filepath.Join(s.tempDir, "test-temp")

	db, err := goleveldb.NewOpenDB(dbPath)
	c.Check(err, IsNil)

	tempDB, err := db.CreateTemporary()
	c.Check(err, IsNil)
	c.Check(tempDB, NotNil)

	// Temporary DB should be usable
	err = tempDB.Put([]byte("temp-key"), []byte("temp-value"))
	c.Check(err, IsNil)

	value, err := tempDB.Get([]byte("temp-key"))
	c.Check(err, IsNil)
	c.Check(value, DeepEquals, []byte("temp-value"))

	err = tempDB.Close()
	c.Check(err, IsNil)

	err = tempDB.Drop()
	c.Check(err, IsNil)

	err = db.Close()
	c.Check(err, IsNil)
}

func (s *ExtendedLevelDBSuite) TestStoragePutOptimization(c *C) {
	// Test Put optimization (doesn't save if value is same)
	dbPath := filepath.Join(s.tempDir, "put-optimization")

	db, err := goleveldb.NewOpenDB(dbPath)
	c.Check(err, IsNil)

	key := []byte("optimization-key")
	value := []byte("same-value")

	// First put
	err = db.Put(key, value)
	c.Check(err, IsNil)

	// Second put with same value (should be optimized)
	err = db.Put(key, value)
	c.Check(err, IsNil)

	// Third put with different value
	newValue := []byte("different-value")
	err = db.Put(key, newValue)
	c.Check(err, IsNil)

	// Verify final value
	result, err := db.Get(key)
	c.Check(err, IsNil)
	c.Check(result, DeepEquals, newValue)

	err = db.Close()
	c.Check(err, IsNil)
}

func (s *ExtendedLevelDBSuite) TestStorageCloseMultiple(c *C) {
	// Test calling Close multiple times
	dbPath := filepath.Join(s.tempDir, "close-multiple")

	db, err := goleveldb.NewOpenDB(dbPath)
	c.Check(err, IsNil)

	// First close should work
	err = db.Close()
	c.Check(err, IsNil)

	// Second close should not error
	err = db.Close()
	c.Check(err, IsNil)

	// Third close should not error
	err = db.Close()
	c.Check(err, IsNil)
}

func (s *ExtendedLevelDBSuite) TestStorageOpenMultiple(c *C) {
	// Test calling Open multiple times
	dbPath := filepath.Join(s.tempDir, "open-multiple")

	db, err := goleveldb.NewDB(dbPath)
	c.Check(err, IsNil)

	// First open should work
	err = db.Open()
	c.Check(err, IsNil)

	// Second open should not error (already open)
	err = db.Open()
	c.Check(err, IsNil)

	// Should still be functional
	err = db.Put([]byte("test"), []byte("value"))
	c.Check(err, IsNil)

	err = db.Close()
	c.Check(err, IsNil)
}

func (s *ExtendedLevelDBSuite) TestStorageDropError(c *C) {
	// Test Drop when database is still open
	dbPath := filepath.Join(s.tempDir, "drop-error")

	db, err := goleveldb.NewOpenDB(dbPath)
	c.Check(err, IsNil)

	// Try to drop while DB is open (should error)
	err = db.Drop()
	c.Check(err, NotNil)
	c.Check(err.Error(), Equals, "DB is still open")

	// Close and then drop should work
	err = db.Close()
	c.Check(err, IsNil)

	err = db.Drop()
	c.Check(err, IsNil)

	// Verify directory is gone
	_, err = os.Stat(dbPath)
	c.Check(os.IsNotExist(err), Equals, true)
}

func (s *ExtendedLevelDBSuite) TestTransactionInterface(c *C) {
	// Test transaction functionality
	dbPath := filepath.Join(s.tempDir, "transaction-test")

	db, err := goleveldb.NewOpenDB(dbPath)
	c.Check(err, IsNil)

	// Create transaction
	tx, err := db.OpenTransaction()
	c.Check(err, IsNil)
	c.Check(tx, NotNil)

	// Test transaction operations
	key := []byte("tx-key")
	value := []byte("tx-value")

	err = tx.Put(key, value)
	c.Check(err, IsNil)

	// Value should not be visible outside transaction yet
	_, err = db.Get(key)
	c.Check(err, Equals, database.ErrNotFound)

	// But should be visible within transaction
	txValue, err := tx.Get(key)
	c.Check(err, IsNil)
	c.Check(txValue, DeepEquals, value)

	// Commit transaction
	err = tx.Commit()
	c.Check(err, IsNil)

	// Now value should be visible
	finalValue, err := db.Get(key)
	c.Check(err, IsNil)
	c.Check(finalValue, DeepEquals, value)

	err = db.Close()
	c.Check(err, IsNil)
}

func (s *ExtendedLevelDBSuite) TestTransactionDiscard(c *C) {
	// Test transaction discard functionality
	dbPath := filepath.Join(s.tempDir, "transaction-discard")

	db, err := goleveldb.NewOpenDB(dbPath)
	c.Check(err, IsNil)

	// Create transaction
	tx, err := db.OpenTransaction()
	c.Check(err, IsNil)

	key := []byte("discard-key")
	value := []byte("discard-value")

	err = tx.Put(key, value)
	c.Check(err, IsNil)

	// Discard transaction
	tx.Discard()

	// Value should not be visible
	_, err = db.Get(key)
	c.Check(err, Equals, database.ErrNotFound)

	err = db.Close()
	c.Check(err, IsNil)
}

func (s *ExtendedLevelDBSuite) TestProcessByPrefixError(c *C) {
	// Test ProcessByPrefix with processor that returns error
	dbPath := filepath.Join(s.tempDir, "process-error")

	db, err := goleveldb.NewOpenDB(dbPath)
	c.Check(err, IsNil)

	// Add some data
	prefix := []byte("error-")
	err = db.Put(append(prefix, []byte("key1")...), []byte("value1"))
	c.Check(err, IsNil)
	err = db.Put(append(prefix, []byte("key2")...), []byte("value2"))
	c.Check(err, IsNil)

	// Process with error-returning function
	testError := errors.New("processing error")
	processedCount := 0

	err = db.ProcessByPrefix(prefix, func(key, value []byte) error {
		processedCount++
		if processedCount == 1 {
			return testError
		}
		return nil
	})

	c.Check(err, Equals, testError)
	c.Check(processedCount, Equals, 1) // Should stop at first error

	err = db.Close()
	c.Check(err, IsNil)
}

func (s *ExtendedLevelDBSuite) TestPrefixOperationsEmptyDB(c *C) {
	// Test prefix operations on empty database
	dbPath := filepath.Join(s.tempDir, "empty-prefix")

	db, err := goleveldb.NewOpenDB(dbPath)
	c.Check(err, IsNil)

	prefix := []byte("empty")

	// All prefix operations should return empty results
	c.Check(db.HasPrefix(prefix), Equals, false)
	c.Check(db.KeysByPrefix(prefix), DeepEquals, [][]byte{})
	c.Check(db.FetchByPrefix(prefix), DeepEquals, [][]byte{})

	processedCount := 0
	err = db.ProcessByPrefix(prefix, func(key, value []byte) error {
		processedCount++
		return nil
	})
	c.Check(err, IsNil)
	c.Check(processedCount, Equals, 0)

	err = db.Close()
	c.Check(err, IsNil)
}

func (s *ExtendedLevelDBSuite) TestBatchOperations(c *C) {
	// Test batch operations in detail
	dbPath := filepath.Join(s.tempDir, "batch-ops")

	db, err := goleveldb.NewOpenDB(dbPath)
	c.Check(err, IsNil)

	// Create batch
	batch := db.CreateBatch()
	c.Check(batch, NotNil)

	// Add multiple operations to batch
	keys := [][]byte{
		[]byte("batch-key-1"),
		[]byte("batch-key-2"),
		[]byte("batch-key-3"),
	}
	values := [][]byte{
		[]byte("batch-value-1"),
		[]byte("batch-value-2"),
		[]byte("batch-value-3"),
	}

	for i, key := range keys {
		err = batch.Put(key, values[i])
		c.Check(err, IsNil)
	}

	// Values should not be visible before Write
	for _, key := range keys {
		_, err = db.Get(key)
		c.Check(err, Equals, database.ErrNotFound)
	}

	// Write batch
	err = batch.Write()
	c.Check(err, IsNil)

	// Now all values should be visible
	for i, key := range keys {
		value, err := db.Get(key)
		c.Check(err, IsNil)
		c.Check(value, DeepEquals, values[i])
	}

	err = db.Close()
	c.Check(err, IsNil)
}

func (s *ExtendedLevelDBSuite) TestIteratorEdgeCases(c *C) {
	// Test iterator edge cases in prefix operations
	dbPath := filepath.Join(s.tempDir, "iterator-edge")

	db, err := goleveldb.NewOpenDB(dbPath)
	c.Check(err, IsNil)

	// Add data with similar but different prefixes
	prefixes := [][]byte{
		[]byte("test"),
		[]byte("test-"),
		[]byte("test-a"),
		[]byte("test-ab"),
		[]byte("testing"),
		[]byte("totally-different"),
	}

	for i, prefix := range prefixes {
		key := append(prefix, []byte("key")...)
		value := []byte{byte(i)}
		err = db.Put(key, value)
		c.Check(err, IsNil)
	}

	// Test exact prefix matching
	targetPrefix := []byte("test-")
	keys := db.KeysByPrefix(targetPrefix)
	values := db.FetchByPrefix(targetPrefix)

	// Should only match keys that start with "test-"
	expectedCount := 0
	for _, prefix := range prefixes {
		testKey := append(prefix, []byte("key")...)
		if len(testKey) >= len(targetPrefix) {
			if string(testKey[:len(targetPrefix)]) == string(targetPrefix) {
				expectedCount++
			}
		}
	}

	c.Check(len(keys), Equals, expectedCount)
	c.Check(len(values), Equals, expectedCount)

	err = db.Close()
	c.Check(err, IsNil)
}

func (s *ExtendedLevelDBSuite) TestCompactDBError(c *C) {
	// Test CompactDB on closed database
	dbPath := filepath.Join(s.tempDir, "compact-error")

	db, err := goleveldb.NewOpenDB(dbPath)
	c.Check(err, IsNil)

	// Close database
	err = db.Close()
	c.Check(err, IsNil)

	// CompactDB should error on closed database
	err = db.CompactDB()
	c.Check(err, NotNil)
}

func (s *ExtendedLevelDBSuite) TestInterface(c *C) {
	// Test that storage implements database.Storage interface
	dbPath := filepath.Join(s.tempDir, "interface-test")

	var storage database.Storage
	storage, err := goleveldb.NewOpenDB(dbPath)
	c.Check(err, IsNil)
	c.Check(storage, NotNil)

	// Test that all interface methods are available
	_, err = storage.Get([]byte("test"))
	c.Check(err, Equals, database.ErrNotFound)

	err = storage.Put([]byte("test"), []byte("value"))
	c.Check(err, IsNil)

	err = storage.Delete([]byte("test"))
	c.Check(err, IsNil)

	c.Check(storage.HasPrefix([]byte("test")), Equals, false)
	c.Check(storage.KeysByPrefix([]byte("test")), DeepEquals, [][]byte{})
	c.Check(storage.FetchByPrefix([]byte("test")), DeepEquals, [][]byte{})

	err = storage.ProcessByPrefix([]byte("test"), func(k, v []byte) error { return nil })
	c.Check(err, IsNil)

	batch := storage.CreateBatch()
	c.Check(batch, NotNil)

	tx, err := storage.OpenTransaction()
	c.Check(err, IsNil)
	c.Check(tx, NotNil)
	tx.Discard()

	temp, err := storage.CreateTemporary()
	c.Check(err, IsNil)
	c.Check(temp, NotNil)
	temp.Close()
	temp.Drop()

	err = storage.CompactDB()
	c.Check(err, IsNil)

	err = storage.Close()
	c.Check(err, IsNil)

	err = storage.Drop()
	c.Check(err, IsNil)
}
