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
	"context"
	"time"

	"github.com/aptly-dev/aptly/database"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var Ctx = context.TODO()

func internalOpen(url string) (*clientv3.Client, error) {
	cfg := clientv3.Config{
		Endpoints:            []string{url},
		DialTimeout:          30 * time.Second,
		MaxCallSendMsgSize:   2048 * 1024 * 1024,
		MaxCallRecvMsgSize:   2048 * 1024 * 1024,
		DialKeepAliveTimeout: 7200 * time.Second,
	}

	cli, err := clientv3.New(cfg)
	if err != nil {
		return nil, err
	}

	return cli, nil
}

func NewDB(url string) (database.Storage, error) {
	cli, err := internalOpen(url)
	if err != nil {
		return nil, err
	}
	return &EtcDStorage{url, cli}, nil
}

func NewOpenDB(url string) (database.Storage, error) {
	db, err := NewDB(url)
	if err != nil {
		return nil, err
	}

	return db, nil
}
