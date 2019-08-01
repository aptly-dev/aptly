package goleveldb

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"

	"github.com/aptly-dev/aptly/database"
)

type storage struct {
	path string
	db   *leveldb.DB
}

// CreateTemporary creates new DB of the same type in temp dir
func (s *storage) CreateTemporary() (database.Storage, error) {
	tempdir, err := ioutil.TempDir("", "aptly")
	if err != nil {
		return nil, err
	}

	db, err := internalOpen(tempdir, true)
	if err != nil {
		return nil, err
	}
	return &storage{db: db, path: tempdir}, nil
}

// Get key value from database
func (s *storage) Get(key []byte) ([]byte, error) {
	value, err := s.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, database.ErrNotFound
		}
		return nil, err
	}

	return value, nil
}

// Put saves key to database, if key has the same value in DB already, it is not saved
func (s *storage) Put(key []byte, value []byte) error {
	old, err := s.db.Get(key, nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return err
		}
	} else {
		if bytes.Equal(old, value) {
			return nil
		}
	}
	return s.db.Put(key, value, nil)
}

// Delete removes key from DB
func (s *storage) Delete(key []byte) error {
	return s.db.Delete(key, nil)
}

// KeysByPrefix returns all keys that start with prefix
func (s *storage) KeysByPrefix(prefix []byte) [][]byte {
	result := make([][]byte, 0, 20)

	iterator := s.db.NewIterator(nil, nil)
	defer iterator.Release()

	for ok := iterator.Seek(prefix); ok && bytes.HasPrefix(iterator.Key(), prefix); ok = iterator.Next() {
		key := iterator.Key()
		keyc := make([]byte, len(key))
		copy(keyc, key)
		result = append(result, keyc)
	}

	return result
}

// FetchByPrefix returns all values with keys that start with prefix
func (s *storage) FetchByPrefix(prefix []byte) [][]byte {
	result := make([][]byte, 0, 20)

	iterator := s.db.NewIterator(nil, nil)
	defer iterator.Release()

	for ok := iterator.Seek(prefix); ok && bytes.HasPrefix(iterator.Key(), prefix); ok = iterator.Next() {
		val := iterator.Value()
		valc := make([]byte, len(val))
		copy(valc, val)
		result = append(result, valc)
	}

	return result
}

// HasPrefix checks whether it can find any key with given prefix and returns true if one exists
func (s *storage) HasPrefix(prefix []byte) bool {
	iterator := s.db.NewIterator(nil, nil)
	defer iterator.Release()
	return iterator.Seek(prefix) && bytes.HasPrefix(iterator.Key(), prefix)
}

// ProcessByPrefix iterates through all entries where key starts with prefix and calls
// StorageProcessor on key value pair
func (s *storage) ProcessByPrefix(prefix []byte, proc database.StorageProcessor) error {
	iterator := s.db.NewIterator(nil, nil)
	defer iterator.Release()

	for ok := iterator.Seek(prefix); ok && bytes.HasPrefix(iterator.Key(), prefix); ok = iterator.Next() {
		err := proc(iterator.Key(), iterator.Value())
		if err != nil {
			return err
		}
	}

	return nil
}

// Close finishes DB work
func (s *storage) Close() error {
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

// Reopen tries to open (re-open) the database
func (s *storage) Open() error {
	if s.db != nil {
		return nil
	}

	var err error
	s.db, err = internalOpen(s.path, false)
	return err
}

// CreateBatch creates a Batch object
func (s *storage) CreateBatch() database.Batch {
	return &batch{
		db: s.db,
		b:  &leveldb.Batch{},
	}
}

// OpenTransaction creates new transaction.
func (s *storage) OpenTransaction() (database.Transaction, error) {
	t, err := s.db.OpenTransaction()
	if err != nil {
		return nil, err
	}

	return &transaction{t: t}, nil
}

// CompactDB compacts database by merging layers
func (s *storage) CompactDB() error {
	return s.db.CompactRange(util.Range{})
}

// Drop removes all the DB files (DANGEROUS!)
func (s *storage) Drop() error {
	if s.db != nil {
		return errors.New("DB is still open")
	}

	return os.RemoveAll(s.path)
}

// Check interface
var (
	_ database.Storage = &storage{}
)
