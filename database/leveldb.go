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
	FetchByPrefix(prefix []byte) [][]byte
	Close() error
}

type levelDB struct {
	db *leveldb.DB
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
	return l.db.Put(key, value, nil)
}

func (l *levelDB) Delete(key []byte) error {
	return l.db.Delete(key, nil)
}

func (l *levelDB) FetchByPrefix(prefix []byte) [][]byte {
	result := make([][]byte, 0, 20)

	iterator := l.db.NewIterator(nil)
	if iterator.Seek(prefix) {
		for bytes.HasPrefix(iterator.Key(), prefix) {
			val := iterator.Value()
			valc := make([]byte, len(val))
			copy(valc, val)
			result = append(result, valc)
			if !iterator.Next() {
				break
			}
		}
	}

	return result
}

func (l *levelDB) Close() error {
	return l.db.Close()
}
