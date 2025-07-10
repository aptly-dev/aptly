package etcddb

import (
	"os"
	"strconv"
	"time"

	"github.com/aptly-dev/aptly/database"
	"github.com/rs/zerolog/log"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Default timeout for etcd operations
var DefaultTimeout = 60 * time.Second

func init() {
	// Allow timeout configuration via environment variable
	if timeout := os.Getenv("APTLY_ETCD_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			DefaultTimeout = d
			log.Info().Dur("timeout", d).Msg("etcd: using custom timeout")
		} else {
			log.Warn().Str("value", timeout).Err(err).Msg("etcd: invalid timeout value, using default")
		}
	}
}

func internalOpen(url string) (cli *clientv3.Client, err error) {
	// Configure dial timeout
	dialTimeout := 60 * time.Second
	if dt := os.Getenv("APTLY_ETCD_DIAL_TIMEOUT"); dt != "" {
		if d, err := time.ParseDuration(dt); err == nil {
			dialTimeout = d
		}
	}
	
	// Configure keep alive timeout
	keepAliveTimeout := 7200 * time.Second
	if ka := os.Getenv("APTLY_ETCD_KEEPALIVE"); ka != "" {
		if d, err := time.ParseDuration(ka); err == nil {
			keepAliveTimeout = d
		}
	}
	
	// Configure message size
	maxMsgSize := 50 * 1024 * 1024 // 50MiB default
	if size := os.Getenv("APTLY_ETCD_MAX_MSG_SIZE"); size != "" {
		if s, err := strconv.Atoi(size); err == nil && s > 0 {
			maxMsgSize = s
		}
	}
	
	cfg := clientv3.Config{
		Endpoints:            []string{url},
		DialTimeout:          dialTimeout,
		MaxCallSendMsgSize:   maxMsgSize,
		MaxCallRecvMsgSize:   maxMsgSize,
		DialKeepAliveTimeout: keepAliveTimeout,
	}
	
	log.Info().
		Str("endpoint", url).
		Dur("dialTimeout", dialTimeout).
		Dur("keepAlive", keepAliveTimeout).
		Int("maxMsgSize", maxMsgSize).
		Msg("etcd: opening connection")

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
