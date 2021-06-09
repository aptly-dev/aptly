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
