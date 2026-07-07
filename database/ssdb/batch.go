package ssdb

import (
	"fmt"

	"github.com/aptly-dev/aptly/database"
	"github.com/seefan/gossdb/v2/conf"
	"github.com/seefan/gossdb/v2/pool"
)

const (
	delOpt = "del"
)

type bWriteData struct {
	key   []byte
	value []byte
	opts  string
	err   error
}

type Batch struct {
	cfg *conf.Config
	// key-value chan
	w  chan bWriteData
	p  map[string]interface{}
	d  []string
	db *pool.Client
}

// func internalOpenBatch...
func internalOpenBatch(_ database.Storage) *Batch {
	b := &Batch{
		w: make(chan bWriteData),
		p: make(map[string]interface{}),
	}
	b.run()

	return b
}

func (b *Batch) run() {
	go func() {
		for {
			select {
			case w, ok := <-b.w:
				{
					if !ok {
						ssdbLog("ssdb batch write chan closed")
						return
					}

					if w.opts == "write" {
						ssdbLog("ssdb batch write")
						var err error
						if len(b.p) > 0 && len(b.d) == 0 {
							err = b.db.MultiSet(b.p)
							ssdbLog("ssdb batch set errinfo: ", err)
						} else if len(b.d) > 0 && len(b.p) == 0 {
							err = b.db.MultiDel(b.d...)
							ssdbLog("ssdb batch del errinfo: ", err)
						} else if len(b.p) == 0 && len(b.d) == 0 {
							err = nil
						} else {
							err = fmt.Errorf("ssdb batch does not support both put and delete operations")
						}
						ssdbLog("ssdb batch write errinfo: ", err)
						b.w <- bWriteData{
							err: err,
						}
						ssdbLog("ssdb batch write end")
					} else {
						ssdbLog("ssdb batch", w.opts)
						if w.opts == "put" {
							b.p[string(w.key)] = w.value
						} else if w.opts == delOpt {
							b.d = append(b.d, string(w.key))
						}
					}
				}
			}
		}
	}()
}

func (b *Batch) stop() {
	ssdbLog("ssdb batch stop")
	close(b.w)
}

func (b *Batch) Put(key, value []byte) (err error) {
	// err = b.db.Set(string(key), string(value))
	w := bWriteData{
		key:   key,
		value: value,
		opts:  "put",
	}

	b.w <- w
	return nil
}

func (b *Batch) Delete(key []byte) (err error) {
	/* err = b.db.Del(string(key))
	return */
	w := bWriteData{
		key:  key,
		opts: delOpt,
	}

	b.w <- w
	return nil
}

func (b *Batch) Write() (err error) {
	defer b.stop()
	w := bWriteData{
		opts: "write",
	}

	b.w <- w
	result := <-b.w
	return result.err
}

// batch should implement database.Batch
var (
	_ database.Batch = &Batch{}
)
