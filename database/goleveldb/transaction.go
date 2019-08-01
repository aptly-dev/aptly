package goleveldb

import (
	"bytes"

	"github.com/aptly-dev/aptly/database"
	"github.com/syndtr/goleveldb/leveldb"
)

type transaction struct {
	t *leveldb.Transaction
}

// Get implements database.Reader interface.
func (t *transaction) Get(key []byte) ([]byte, error) {
	value, err := t.t.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, database.ErrNotFound
		}
		return nil, err
	}

	return value, nil
}

// Put implements database.Writer interface.
func (t *transaction) Put(key, value []byte) error {
	old, err := t.t.Get(key, nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return err
		}
	} else {
		if bytes.Equal(old, value) {
			return nil
		}
	}
	return t.t.Put(key, value, nil)
}

// Delete implements database.Writer interface.
func (t *transaction) Delete(key []byte) error {
	return t.t.Delete(key, nil)
}

// Commit finalizes transaction and commits changes to the stable storage.
func (t *transaction) Commit() error {
	return t.t.Commit()
}

// Discard any transaction changes.
//
// Discard is safe to call after Commit(), it would be no-op
func (t *transaction) Discard() {
	t.t.Discard()
}

// transaction should implement database.Transaction
var _ database.Transaction = &transaction{}
