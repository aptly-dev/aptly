package ssdb

import (
	"os"

	"github.com/aptly-dev/aptly/database"
	"github.com/aptly-dev/aptly/database/goleveldb"
	"github.com/seefan/gossdb/v2"
	"github.com/seefan/gossdb/v2/conf"
	"github.com/seefan/gossdb/v2/pool"
)

type Storage struct {
	cfg *conf.Config
	db  *pool.Client
}

// CreateTemporary creates new DB of the same type in temp dir
func (s *Storage) CreateTemporary() (database.Storage, error) {
	// use leveldb as temp db
	tmpPath := os.Getenv("SSDB_TMPDB_PATH")
	if tmpPath == "" {
		tmpPath = "/tmp/ssdb_tmpdb_path"
	}
	gdb, err := goleveldb.NewDB(tmpPath)
	if err != nil {
		return nil, err
	}

	return gdb.CreateTemporary()
}

// Get key value from ssdb
func (s *Storage) Get(key []byte) (value []byte, err error) {
	// ssdbLog("ssdb origin db get key:", string(key))
	getResp, err := s.db.Get(string(key))
	if err != nil {
		return
	}

	value = getResp.Bytes()

	if len(value) == 0 {
		err = database.ErrNotFound
		return
	}
	return
}

// Put saves key to ssdb, if key has the same value in DB already, it is not saved
func (s *Storage) Put(key []byte, value []byte) (err error) {
	//ssdbLog("ssdb origin db put key:", string(key), " value: ", string(value))
	err = s.db.Set(string(key), value)
	if err != nil {
		return
	}
	return
}

// Delete removes key from ssdb
func (s *Storage) Delete(key []byte) (err error) {
	//ssdbLog("ssdb origin db del key:", string(key))
	err = s.db.Del(string(key))
	if err != nil {
		return
	}
	return
}

// KeysByPrefix returns all keys that start with prefix
func (s *Storage) KeysByPrefix(prefix []byte) [][]byte {
	result := make([][]byte, 0)
	getResp, err := s.db.Keys(string(prefix), string(prefix)+"}", -1)
	if err != nil {
		return nil
	}
	for _, ev := range getResp {
		key := []byte(ev)
		keyc := make([]byte, len(key))
		copy(keyc, key)
		result = append(result, key)
	}
	return result
}

// FetchByPrefix returns all values with keys that start with prefix
func (s *Storage) FetchByPrefix(prefix []byte) [][]byte {
	result := make([][]byte, 0)
	getResp, err := s.db.Scan(string(prefix), string(prefix)+"}", -1)
	if err != nil {
		return nil
	}
	for _, ev := range getResp {
		value := ev.Bytes()
		valuec := make([]byte, len(value))
		copy(valuec, value)
		result = append(result, valuec)
	}
	return result
}

// HasPrefix checks whether it can find any key with given prefix and returns true if one exists
func (s *Storage) HasPrefix(prefix []byte) bool {
	//ssdbLog("HasPrefix", string(prefix), string(prefix)+"}")
	getResp, err := s.db.Keys(string(prefix), string(prefix)+"}", -1)
	if err != nil {
		return false
	}
	//ssdbLog("HasPrefix", len(getResp))
	if len(getResp) > 0 {
		return true
	}
	return false
}

// ProcessByPrefix iterates through all entries where key starts with prefix and calls
// StorageProcessor on key value pair
func (s *Storage) ProcessByPrefix(prefix []byte, proc database.StorageProcessor) error {
	getResp, err := s.db.Scan(string(prefix), string(prefix)+"}", -1)
	if err != nil {
		return err
	}

	for k, v := range getResp {
		err := proc([]byte(k), v.Bytes())
		if err != nil {
			return err
		}
	}
	return nil
}

// Close finishes ssdb connect
func (s *Storage) Close() error {
	ssdbLog("ssdb close")
	if s.db != nil {
		s.db.Close()
		s.db = nil
	}
	gossdb.Shutdown()
	return nil
}

// Reopen tries to open (re-open) the database
func (s *Storage) Open() error {
	ssdbLog("ssdb open")
	if s.db != nil && s.db.IsOpen() {
		ssdbLog("ssdb opened")
		return nil
	}

	var err error
	s.db, err = internalOpen(s.cfg)
	return err
}

// CreateBatch creates a Batch object
func (s *Storage) CreateBatch() database.Batch {
	Batch := internalOpenBatch(s)
	Batch.cfg = s.cfg
	Batch.db = s.db
	return Batch
}

// OpenTransaction creates new transaction.
func (s *Storage) OpenTransaction() (database.Transaction, error) {
	return internalOpenTransaction(s)
}

// CompactDB compacts database by merging layers
func (s *Storage) CompactDB() error {
	return nil
}

// Drop removes all the ssdb files (DANGEROUS!)
func (s *Storage) Drop() error {
	return nil
}

// Check interface
var (
	_ database.Storage = &Storage{}
)
