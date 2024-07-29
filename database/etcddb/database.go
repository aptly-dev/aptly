package etcddb

import (
	"context"
	"time"

	"github.com/aptly-dev/aptly/database"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var Ctx = context.TODO()

func internalOpen(url string) (cli *clientv3.Client, err error) {
	cfg := clientv3.Config{
		Endpoints:            []string{url},
		DialTimeout:          30 * time.Second,
		MaxCallSendMsgSize:   2147483647, // (2048 * 1024 * 1024) - 1
		MaxCallRecvMsgSize:   2147483647,
		DialKeepAliveTimeout: 7200 * time.Second,
	}

	cli, err = clientv3.New(cfg)
	return
}

func NewDB(url string) (database.Storage, error) {
	cli, err := internalOpen(url)
	if err != nil {
		return nil, err
	}
	return &EtcDStorage{url, cli, ""}, nil
}
