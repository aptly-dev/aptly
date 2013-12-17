package database

import (
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

var (
	ErrNotFound = errors.New("key not found")
)

type Storage interface {
	Get(key []byte) ([]byte, error)
	Put(key []byte, value []byte) error
	Close() error
}

type levelDB struct {
	db *leveldb.DB
}

// Check interface
var (
	_ Storage = &levelDB{}
)

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

func (l *levelDB) Close() error {
	return l.db.Close()
}
