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

// Default write retry count
var DefaultWriteRetries = 3

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

	// Allow write retry configuration via environment variable
	if retries := os.Getenv("APTLY_ETCD_WRITE_RETRIES"); retries != "" {
		if r, err := strconv.Atoi(retries); err == nil && r >= 0 {
			DefaultWriteRetries = r
			log.Info().Int("retries", r).Msg("etcd: using custom write retry count")
		} else {
			log.Warn().Str("value", retries).Err(err).Msg("etcd: invalid write retry value, using default")
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
	return &EtcDStorage{
		url:          url,
		db:           cli,
		queuedClient: nil,
		queuedKV:     nil,
		tmpPrefix:    "",
	}, nil
}

// NewDBWithQueue creates a new DB with optional write queue
func NewDBWithQueue(url string, queueConfig *QueueConfig) (database.Storage, error) {
	cli, err := internalOpen(url)
	if err != nil {
		return nil, err
	}
	
	storage := &EtcDStorage{
		url:       url,
		db:        cli,
		tmpPrefix: "",
	}
	
	if queueConfig != nil && queueConfig.Enabled {
		storage.queuedClient = NewQueuedEtcdClient(cli, queueConfig)
		storage.queuedKV = NewQueuedKV(cli.KV, storage.queuedClient.writeQueue, queueConfig)
		log.Info().
			Bool("enabled", queueConfig.Enabled).
			Int("queueSize", queueConfig.WriteQueueSize).
			Int("maxWritesPerSec", queueConfig.MaxWritesPerSec).
			Msg("etcd: write queue enabled")
	}
	
	return storage, nil
}

// ConfigureFromDBConfig applies configuration from DBConfig
func ConfigureFromDBConfig(timeout string, writeRetries int) {
	// Configure timeout if provided
	if timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			DefaultTimeout = d
			log.Info().Dur("timeout", d).Msg("etcd: configured timeout from config")
		} else {
			log.Warn().Str("value", timeout).Err(err).Msg("etcd: invalid timeout in config, keeping current value")
		}
	}

	// Configure write retries if provided
	if writeRetries > 0 {
		DefaultWriteRetries = writeRetries
		log.Info().Int("retries", writeRetries).Msg("etcd: configured write retries from config")
	}
}
