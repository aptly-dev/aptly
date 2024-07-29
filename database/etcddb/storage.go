package etcddb

import (
	"github.com/aptly-dev/aptly/database"
	"github.com/pborman/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"

	"fmt"
)

type EtcDStorage struct {
	url       string
	db        *clientv3.Client
	tmpPrefix string // prefix for temporary DBs
}

// CreateTemporary creates new DB of the same type in temp dir
func (s *EtcDStorage) CreateTemporary() (database.Storage, error) {
	tmp := uuid.NewRandom().String()
	return &EtcDStorage{
		url:       s.url,
		db:        s.db,
		tmpPrefix: tmp,
	}, nil
}

func (s *EtcDStorage) applyPrefix(key []byte) []byte {
	if len(s.tmpPrefix) != 0 {
		return append([]byte(s.tmpPrefix+"/"), key...)
	}
	return key
}

// Get key value from etcd
func (s *EtcDStorage) Get(key []byte) (value []byte, err error) {
	realKey := s.applyPrefix(key)
	getResp, err := s.db.Get(Ctx, string(realKey))
	if err != nil {
		return
	}
	for _, kv := range getResp.Kvs {
		value = kv.Value
		break
	}
	if len(value) == 0 {
		err = database.ErrNotFound
		return
	}
	return
}

// Put saves key to etcd, if key has the same value in DB already, it is not saved
func (s *EtcDStorage) Put(key []byte, value []byte) (err error) {
	realKey := s.applyPrefix(key)
	_, err = s.db.Put(Ctx, string(realKey), string(value))
	if err != nil {
		return
	}
	return
}

// Delete removes key from etcd
func (s *EtcDStorage) Delete(key []byte) (err error) {
	realKey := s.applyPrefix(key)
	_, err = s.db.Delete(Ctx, string(realKey))
	if err != nil {
		return
	}
	return
}

// KeysByPrefix returns all keys that start with prefix
func (s *EtcDStorage) KeysByPrefix(prefix []byte) [][]byte {
	realPrefix := s.applyPrefix(prefix)
	result := make([][]byte, 0, 20)
	getResp, err := s.db.Get(Ctx, string(realPrefix), clientv3.WithPrefix())
	if err != nil {
		return nil
	}
	for _, ev := range getResp.Kvs {
		key := ev.Key
		keyc := make([]byte, len(key))
		copy(keyc, key)
		result = append(result, key)
	}
	return result
}

// FetchByPrefix returns all values with keys that start with prefix
func (s *EtcDStorage) FetchByPrefix(prefix []byte) [][]byte {
	realPrefix := s.applyPrefix(prefix)
	result := make([][]byte, 0, 20)
	getResp, err := s.db.Get(Ctx, string(realPrefix), clientv3.WithPrefix())
	if err != nil {
		return nil
	}
	for _, kv := range getResp.Kvs {
		valc := make([]byte, len(kv.Value))
		copy(valc, kv.Value)
		result = append(result, kv.Value)
	}

	return result
}

// HasPrefix checks whether it can find any key with given prefix and returns true if one exists
func (s *EtcDStorage) HasPrefix(prefix []byte) bool {
	realPrefix := s.applyPrefix(prefix)
	getResp, err := s.db.Get(Ctx, string(realPrefix), clientv3.WithPrefix())
	if err != nil {
		return false
	}
	return getResp.Count > 0
}

// ProcessByPrefix iterates through all entries where key starts with prefix and calls
// StorageProcessor on key value pair
func (s *EtcDStorage) ProcessByPrefix(prefix []byte, proc database.StorageProcessor) error {
	realPrefix := s.applyPrefix(prefix)
	getResp, err := s.db.Get(Ctx, string(realPrefix), clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for _, kv := range getResp.Kvs {
		err := proc(kv.Key, kv.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

// Close finishes etcd connect
func (s *EtcDStorage) Close() error {
	// do not close temporary db
	if len(s.tmpPrefix) != 0 {
		return nil
	}
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

// Reopen tries to open (re-open) the database
func (s *EtcDStorage) Open() error {
	if s.db != nil {
		return nil
	}
	var err error
	s.db, err = internalOpen(s.url)
	return err
}

// CreateBatch creates a Batch object
func (s *EtcDStorage) CreateBatch() database.Batch {
	if s.db == nil {
		return nil
	}
	return &EtcDBatch{
		s: s,
	}
}

// OpenTransaction creates new transaction.
func (s *EtcDStorage) OpenTransaction() (database.Transaction, error) {
	tmpdb, err := s.CreateTemporary()
	if err != nil {
		return nil, err
	}
	return &transaction{s: s, tmpdb: tmpdb}, nil
}

// CompactDB does nothing for etcd
func (s *EtcDStorage) CompactDB() error {
	return nil
}

// Drop removes only temporary DBs with etcd (i.e. remove all prefixed keys)
func (s *EtcDStorage) Drop() error {
	if len(s.tmpPrefix) != 0 {
		getResp, err := s.db.Get(Ctx, s.tmpPrefix, clientv3.WithPrefix())
		if err != nil {
			return nil
		}
		for _, kv := range getResp.Kvs {
			_, err = s.db.Delete(Ctx, string(kv.Key))
			if err != nil {
				return fmt.Errorf("cannot delete tempdb entry: %s", kv.Key)
			}
		}
	}
	return nil
}

// Check interface
var (
	_ database.Storage = &EtcDStorage{}
)
