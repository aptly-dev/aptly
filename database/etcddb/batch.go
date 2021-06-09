package etcddb

import (
	"github.com/aptly-dev/aptly/database"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcDBatch struct {
	db *clientv3.Client
}

type WriteOptions struct {
	NoWriteMerge bool
	Sync         bool
}

func (b *EtcDBatch) Put(key, value []byte) (err error) {
	_, err = b.db.Put(Ctx, string(key), string(value))
	return
}

func (b *EtcDBatch) Delete(key []byte) (err error) {
	_, err = b.db.Delete(Ctx, string(key))
	return
}

func (b *EtcDBatch) Write() (err error) {
	return
}

// batch should implement database.Batch
var (
	_ database.Batch = &EtcDBatch{}
)
