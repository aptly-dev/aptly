package etcddb

import (
	"github.com/aptly-dev/aptly/database"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/clientv3util"
)

type transaction struct {
	t clientv3.KV
}

// Get implements database.Reader interface.
func (t *transaction) Get(key []byte) ([]byte, error) {
	getResp, err := t.t.Get(Ctx, string(key))
	if err != nil {
		return nil, err
	}

	var value []byte
	for _, kv := range getResp.Kvs {
		valc := make([]byte, len(kv.Value))
		copy(valc, kv.Value)
		value = valc
	}

	return value, nil
}

// Put implements database.Writer interface.
func (t *transaction) Put(key, value []byte) (err error) {
	_, err = t.t.Txn(Ctx).
		If().Then(clientv3.OpPut(string(key), string(value))).Commit()
	return
}

// Delete implements database.Writer interface.
func (t *transaction) Delete(key []byte) (err error) {
	_, err = t.t.Txn(Ctx).
		If(clientv3util.KeyExists(string(key))).
		Then(clientv3.OpDelete(string(key))).Commit()
	return
}

func (t *transaction) Commit() (err error) {
	return
}

// Discard is safe to call after Commit(), it would be no-op
func (t *transaction) Discard() {
	return
}

// transaction should implement database.Transaction
var _ database.Transaction = &transaction{}
