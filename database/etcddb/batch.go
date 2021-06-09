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
