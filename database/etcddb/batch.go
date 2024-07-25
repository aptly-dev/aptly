package etcddb

import (
	"github.com/aptly-dev/aptly/database"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcDBatch struct {
	s   *EtcDStorage
	ops []clientv3.Op
}

type WriteOptions struct {
	NoWriteMerge bool
	Sync         bool
}

func (b *EtcDBatch) Put(key []byte, value []byte) (err error) {
	b.ops = append(b.ops, clientv3.OpPut(string(key), string(value)))
	return
}

func (b *EtcDBatch) Delete(key []byte) (err error) {
	b.ops = append(b.ops, clientv3.OpDelete(string(key)))
	return
}

func (b *EtcDBatch) Write() (err error) {
	kv := clientv3.NewKV(b.s.db)

	batchSize := 128
	for i := 0; i < len(b.ops); i += batchSize {
		txn := kv.Txn(Ctx)
		end := i + batchSize
		if end > len(b.ops) {
			end = len(b.ops)
		}

		batch := b.ops[i:end]
		txn.Then(batch...)
		_, err = txn.Commit()
		if err != nil {
			panic(err)
		}
	}

	return
}

// batch should implement database.Batch
var (
	_ database.Batch = &EtcDBatch{}
)
