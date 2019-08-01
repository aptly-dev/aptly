package goleveldb

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/aptly-dev/aptly/database"
)

type batch struct {
	db *leveldb.DB
	b  *leveldb.Batch
}

func (b *batch) Put(key, value []byte) error {
	b.b.Put(key, value)

	return nil
}

func (b *batch) Delete(key []byte) error {
	b.b.Delete(key)

	return nil
}

func (b *batch) Write() error {
	return b.db.Write(b.b, &opt.WriteOptions{})
}

// batch should implement database.Batch
var (
	_ database.Batch = &batch{}
)
