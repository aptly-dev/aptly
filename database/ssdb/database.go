package ssdb

import (
	"os"
	"strconv"

	"github.com/aptly-dev/aptly/database"
	"github.com/seefan/gossdb/v2"
	"github.com/seefan/gossdb/v2/conf"
	"github.com/seefan/gossdb/v2/pool"
)

var defaultBufSize = 102400
var defaultPoolSize = 1

func internalOpen(cfg *conf.Config) (*pool.Client, error) {
	ssdbLog("internalOpen")

	cfg.ReadBufferSize = defaultBufSize
	cfg.WriteBufferSize = defaultBufSize
	cfg.MaxPoolSize = defaultPoolSize
	cfg.PoolSize = defaultPoolSize
	cfg.MinPoolSize = defaultPoolSize
	cfg.MaxWaitSize = 100 * defaultPoolSize
	cfg.RetryEnabled = true

	//override by env
	if os.Getenv("SSDB_READBUFFERSIZE") != "" {
		readBufSize, err := strconv.Atoi(os.Getenv("SSDB_READBUFFERSIZE"))
		if err != nil {
			cfg.ReadBufferSize = readBufSize
		}
	}

	if os.Getenv("SSDB_WRITEBUFFERSIZE") != "" {
		writeBufSize, err := strconv.Atoi(os.Getenv("SSDB_WRITEBUFFERSIZE"))
		if err != nil {
			cfg.WriteBufferSize = writeBufSize
		}
	}

	var cfgs = []*conf.Config{cfg}
	err := gossdb.Start(cfgs...)
	if err != nil {
		return nil, err
	}

	return gossdb.NewClient()
}

func NewDB(cfg *conf.Config) (database.Storage, error) {
	return &Storage{cfg: cfg}, nil
}

func NewOpenDB(cfg *conf.Config) (database.Storage, error) {
	db, err := NewDB(cfg)
	if err != nil {
		return nil, err
	}

	return db, db.Open()
}
