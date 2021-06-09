/*
 * Copyright (C) 2014 ~ 2021 Deepin Technology Co., Ltd.
 *
 * Author:     hudeng <hudeng@uniontech.com>
 *             zhoufei <zhoufei@uniontech.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package etcddb

import (
	"github.com/aptly-dev/aptly/database"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcDStorage struct {
	url string
	db  *clientv3.Client
}

// CreateTemporary creates new DB of the same type in temp dir
func (s *EtcDStorage) CreateTemporary() (database.Storage, error) {
	return s, nil
}

// Get key value from etcd
func (s *EtcDStorage) Get(key []byte) (value []byte, err error) {
	getResp, err := s.db.Get(Ctx, string(key))
	if err != nil {
		return
	}
	for _, kv := range getResp.Kvs {
		value = kv.Value
	}
	if len(value) == 0 {
		err = database.ErrNotFound
		return
	}
	return
}

// Put saves key to etcd, if key has the same value in DB already, it is not saved
func (s *EtcDStorage) Put(key []byte, value []byte) (err error) {
	_, err = s.db.Put(Ctx, string(key), string(value))
	if err != nil {
		return
	}
	return
}

// Delete removes key from etcd
func (s *EtcDStorage) Delete(key []byte) (err error) {
	_, err = s.db.Delete(Ctx, string(key))
	if err != nil {
		return
	}
	return
}

// KeysByPrefix returns all keys that start with prefix
func (s *EtcDStorage) KeysByPrefix(prefix []byte) [][]byte {
	result := make([][]byte, 0, 20)
	getResp, err := s.db.Get(Ctx, string(prefix), clientv3.WithPrefix())
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
	result := make([][]byte, 0, 20)
	getResp, err := s.db.Get(Ctx, string(prefix), clientv3.WithPrefix())
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
	getResp, err := s.db.Get(Ctx, string(prefix), clientv3.WithPrefix())
	if err != nil {
		return false
	}
	if getResp.Count != 0 {
		return true
	}
	return false
}

// ProcessByPrefix iterates through all entries where key starts with prefix and calls
// StorageProcessor on key value pair
func (s *EtcDStorage) ProcessByPrefix(prefix []byte, proc database.StorageProcessor) error {
	getResp, err := s.db.Get(Ctx, string(prefix), clientv3.WithPrefix())
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
	return &EtcDBatch{
		db: s.db,
	}
}

// OpenTransaction creates new transaction.
func (s *EtcDStorage) OpenTransaction() (database.Transaction, error) {
	cli, err := internalOpen(s.url)
	if err != nil {
		return nil, err
	}
	kvc := clientv3.NewKV(cli)
	return &transaction{t: kvc}, nil
}

// CompactDB compacts database by merging layers
func (s *EtcDStorage) CompactDB() error {
	return nil
}

// Drop removes all the etcd files (DANGEROUS!)
func (s *EtcDStorage) Drop() error {
	return nil
}

// Check interface
var (
	_ database.Storage = &EtcDStorage{}
)
