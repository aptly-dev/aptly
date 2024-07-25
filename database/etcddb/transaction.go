package etcddb

import (
	"github.com/aptly-dev/aptly/database"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type transaction struct {
	s     *EtcDStorage
	tmpdb database.Storage
	ops   []clientv3.Op
}

// Get implements database.Reader interface.
func (t *transaction) Get(key []byte) (value []byte, err error) {
	value, err = t.tmpdb.Get(key)
	// if not found, search main db
	if err != nil {
		value, err = t.s.Get(key)
	}
	return
}

// Put implements database.Writer interface.
func (t *transaction) Put(key, value []byte) (err error) {
	err = t.tmpdb.Put(key, value)
	if err != nil {
		return
	}
	t.ops = append(t.ops, clientv3.OpPut(string(key), string(value)))
	return
}

// Delete implements database.Writer interface.
func (t *transaction) Delete(key []byte) (err error) {
	err = t.tmpdb.Delete(key)
	if err != nil {
		return
	}
	t.ops = append(t.ops, clientv3.OpDelete(string(key)))
	return
}

func (t *transaction) Commit() (err error) {
	kv := clientv3.NewKV(t.s.db)

	batchSize := 128
	for i := 0; i < len(t.ops); i += batchSize {
		txn := kv.Txn(Ctx)
		end := i + batchSize
		if end > len(t.ops) {
			end = len(t.ops)
		}

		batch := t.ops[i:end]
		txn.Then(batch...)
		_, err = txn.Commit()
		if err != nil {
			panic(err)
		}
	}
	t.ops = []clientv3.Op{}

	return
}

// Discard is safe to call after Commit(), it would be no-op
func (t *transaction) Discard() {
	t.ops = []clientv3.Op{}
	t.tmpdb.Drop()
	return
}

// transaction should implement database.Transaction
var _ database.Transaction = &transaction{}
