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
		MaxCallSendMsgSize:   (2048 * 1024 * 1024) - 1,
		MaxCallRecvMsgSize:   (2048 * 1024 * 1024) - 1,
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
	return &EtcDStorage{url, cli, ""}, nil
}

func NewOpenDB(url string) (database.Storage, error) {
	db, err := NewDB(url)
	if err != nil {
		return nil, err
	}

	return db, nil
}
