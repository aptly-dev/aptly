package goleveldb

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	leveldbstorage "github.com/syndtr/goleveldb/leveldb/storage"

	"github.com/aptly-dev/aptly/database"
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
func NewDB(path string) (database.Storage, error) {
	return &storage{path: path}, nil
}

// NewOpenDB creates new instance of DB and opens it
func NewOpenDB(path string) (database.Storage, error) {
	db, err := NewDB(path)
	if err != nil {
		return nil, err
	}

	return db, db.Open()
}

// RecoverDB recovers LevelDB database from corruption
func RecoverDB(path string) error {
	stor, err := leveldbstorage.OpenFile(path, false)
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
