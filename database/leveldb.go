// Package database provides KV database for meta-information
package database

import (
	"bytes"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// Errors for Storage
var (
	ErrNotFound = errors.New("key not found")
)

// Storage is an interface to KV storage
type Storage interface {
	Get(key []byte) ([]byte, error)
	Put(key []byte, value []byte) error
	Delete(key []byte) error
	KeysByPrefix(prefix []byte) [][]byte
	FetchByPrefix(prefix []byte) [][]byte
	Close() error
	StartBatch()
	FinishBatch() error
}

type levelDB struct {
	db    *leveldb.DB
	batch *leveldb.Batch
}

// Check interface
var (
	_ Storage = &levelDB{}
)

// OpenDB opens (creates) LevelDB database
func OpenDB(path string) (Storage, error) {
	o := &opt.Options{
		Filter: filter.NewBloomFilter(10),
	}

	db, err := leveldb.OpenFile(path, o)
	if err != nil {
		return nil, err
	}
	return &levelDB{db: db}, nil
}

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

func (l *levelDB) Put(key []byte, value []byte) error {
	if l.batch != nil {
		l.batch.Put(key, value)
		return nil
	}
	return l.db.Put(key, value, nil)
}

func (l *levelDB) Delete(key []byte) error {
	if l.batch != nil {
		l.batch.Delete(key)
		return nil
	}
	return l.db.Delete(key, nil)
}

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

func (l *levelDB) Close() error {
	return l.db.Close()
}

func (l *levelDB) StartBatch() {
	if l.batch != nil {
		panic("batch already started")
	}
	l.batch = new(leveldb.Batch)
}

func (l *levelDB) FinishBatch() error {
	if l.batch == nil {
		panic("no batch")
	}
	err := l.db.Write(l.batch, nil)
	l.batch = nil
	return err
}
