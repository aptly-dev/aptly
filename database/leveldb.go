// Package database provides KV database for meta-information
package database

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// Errors for Storage
var (
	ErrNotFound = errors.New("key not found")
)

// StorageProcessor is a function to process one single storage entry
type StorageProcessor func(key []byte, value []byte) error

// Storage is an interface to KV storage
type Storage interface {
	CreateTemporary() (Storage, error)
	Get(key []byte) ([]byte, error)
	Put(key []byte, value []byte) error
	Delete(key []byte) error
	HasPrefix(prefix []byte) bool
	ProcessByPrefix(prefix []byte, proc StorageProcessor) error
	KeysByPrefix(prefix []byte) [][]byte
	FetchByPrefix(prefix []byte) [][]byte
	Open() error
	Close() error
	StartBatch()
	FinishBatch() error
	CompactDB() error
	Drop() error
}

type levelDB struct {
	path  string
	db    *leveldb.DB
	batch *leveldb.Batch
}

// Check interface
var (
	_ Storage = &levelDB{}
)

func internalOpen(path string, throttleCompaction bool) (*leveldb.DB, error) {
	o := &opt.Options{
		Filter:                 filter.NewBloomFilter(10),
		OpenFilesCacheCapacity: 256,
	}

	if throttleCompaction {
		o.CompactionL0Trigger = 32
		o.WriteL0PauseTrigger = 96
		o.WriteL0SlowdownTrigger = 64
	}

	return leveldb.OpenFile(path, o)
}

// NewDB creates new instance of DB, but doesn't open it (yet)
func NewDB(path string) (Storage, error) {
	return &levelDB{path: path}, nil
}

// NewOpenDB creates new instance of DB and opens it
func NewOpenDB(path string) (Storage, error) {
	db, err := NewDB(path)
	if err != nil {
		return nil, err
	}

	return db, db.Open()
}

// RecoverDB recovers LevelDB database from corruption
func RecoverDB(path string) error {
	stor, err := storage.OpenFile(path, false)
	if err != nil {
		return err
	}

	db, err := leveldb.Recover(stor, nil)
	if err != nil {
		return err
	}

	db.Close()
	stor.Close()

	return nil
}

// CreateTemporary creates new DB of the same type in temp dir
func (l *levelDB) CreateTemporary() (Storage, error) {
	tempdir, err := ioutil.TempDir("", "aptly")
	if err != nil {
		return nil, err
	}

	db, err := internalOpen(tempdir, true)
	if err != nil {
		return nil, err
	}
	return &levelDB{db: db, path: tempdir}, nil
}

// Get key value from database
func (l *levelDB) Get(key []byte) ([]byte, error) {
	value, err := l.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return value, nil
}

// Put saves key to database, if key has the same value in DB already, it is not saved
func (l *levelDB) Put(key []byte, value []byte) error {
	if l.batch != nil {
		l.batch.Put(key, value)
		return nil
	}
	old, err := l.db.Get(key, nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return err
		}
	} else {
		if bytes.Equal(old, value) {
			return nil
		}
	}
	return l.db.Put(key, value, nil)
}

// Delete removes key from DB
func (l *levelDB) Delete(key []byte) error {
	if l.batch != nil {
		l.batch.Delete(key)
		return nil
	}
	return l.db.Delete(key, nil)
}

// KeysByPrefix returns all keys that start with prefix
func (l *levelDB) KeysByPrefix(prefix []byte) [][]byte {
	result := make([][]byte, 0, 20)

	iterator := l.db.NewIterator(nil, nil)
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
func (l *levelDB) FetchByPrefix(prefix []byte) [][]byte {
	result := make([][]byte, 0, 20)

	iterator := l.db.NewIterator(nil, nil)
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
func (l *levelDB) HasPrefix(prefix []byte) bool {
	iterator := l.db.NewIterator(nil, nil)
	defer iterator.Release()
	return iterator.Seek(prefix) && bytes.HasPrefix(iterator.Key(), prefix)
}

// ProcessByPrefix iterates through all entries where key starts with prefix and calls
// StorageProcessor on key value pair
func (l *levelDB) ProcessByPrefix(prefix []byte, proc StorageProcessor) error {
	iterator := l.db.NewIterator(nil, nil)
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
func (l *levelDB) Close() error {
	if l.db == nil {
		return nil
	}
	err := l.db.Close()
	l.db = nil
	return err
}

// Reopen tries to open (re-open) the database
func (l *levelDB) Open() error {
	if l.db != nil {
		return nil
	}

	var err error
	l.db, err = internalOpen(l.path, false)
	return err
}

// StartBatch starts batch processing of keys
//
// All subsequent Get, Put and Delete would work on batch
func (l *levelDB) StartBatch() {
	if l.batch != nil {
		panic("batch already started")
	}
	l.batch = new(leveldb.Batch)
}

// FinishBatch finalizes the batch, saving operations
func (l *levelDB) FinishBatch() error {
	if l.batch == nil {
		panic("no batch")
	}
	err := l.db.Write(l.batch, nil)
	l.batch = nil
	return err
}

// CompactDB compacts database by merging layers
func (l *levelDB) CompactDB() error {
	return l.db.CompactRange(util.Range{})
}

// Drop removes all the DB files (DANGEROUS!)
func (l *levelDB) Drop() error {
	if l.db != nil {
		return errors.New("DB is still open")
	}

	return os.RemoveAll(l.path)
}
