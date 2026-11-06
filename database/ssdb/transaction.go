package ssdb

import (
	"fmt"

	"github.com/aptly-dev/aptly/database"
)

type trWriteData struct {
	key   []byte
	value []byte
	opts  string
	err   error
}

type trReadData struct {
	kv  []byte
	err error
}

type transaction struct {
	// for key-value-operation chan
	w chan trWriteData
	// key read chan
	r chan trReadData
	q map[string]trWriteData
	t database.Storage
}

// func internalOpenTransaction...
func internalOpenTransaction(t database.Storage) (*transaction, error) {
	tr := &transaction{
		w: make(chan trWriteData),
		r: make(chan trReadData),
		q: make(map[string]trWriteData),
		t: t,
	}

	return tr, tr.run()
}

// func run...
func (t *transaction) run() error {
	go func() {
		for {
			select {
			case w, ok := <-t.w:
				{
					if !ok {
						ssdbLog("ssdb transaction write chan closed")
						return
					}

					if w.opts == "commit" {
						ssdbLog("ssdb transaction commit")
						var errs []error
						for _, vo := range t.q {
							if vo.opts == "put" {
								err := t.t.Put(vo.key, vo.value)
								if err != nil {
									//ssdbLog(err)
									errs = append(errs, err)
								}
							}

							if vo.opts == delOpt {
								err := t.t.Delete(vo.key)
								if err != nil {
									errs = append(errs, err)
								}
							}
						}
						if len(errs) == 0 {
							t.w <- trWriteData{
								err: nil,
							}
						} else {
							t.w <- trWriteData{
								err: fmt.Errorf("ssdb transaction write errs: %v", errs),
							}
						}
						ssdbLog("ssdb transaction commit end")
					} else {
						ssdbLog("ssdb transaction", w.opts)
						//ssdbLog("ssdb r transaction", w.opts, "key: ", string(w.key), "value: ", string(w.value))
						t.q[string(w.key)] = w
					}
				}
			case r, ok := <-t.r:
				{
					if !ok {
						ssdbLog("ssdb transaction read chan closed")
						return
					}

					if rData, ok := t.q[string(r.kv)]; ok {
						if rData.opts == delOpt {
							// del return not found error
							t.r <- trReadData{
								kv:  nil,
								err: database.ErrNotFound,
							}
						} else {
							t.r <- trReadData{
								kv:  rData.value,
								err: nil,
							}
						}
					} else {
						v, err := t.t.Get(r.kv)
						t.r <- trReadData{
							kv:  v,
							err: err,
						}
					}
				}
			}
		}
	}()

	return nil
}

// Get implements database.Reader interface.
func (t *transaction) Get(key []byte) ([]byte, error) {
	keyc := make([]byte, len(key))
	copy(keyc, key)
	r := trReadData{
		kv:  keyc,
		err: nil,
	}
	t.r <- r
	result := <-t.r
	return result.kv, result.err
}

// Put implements database.Writer interface.
func (t *transaction) Put(key, value []byte) error {
	//ssdbLog("golf*********************ssdb put")
	//ssdbLog("ssdb transaction db put key:", string(key), " value: ", string(value))
	keyc := make([]byte, len(key))
	copy(keyc, key)
	valuec := make([]byte, len(value))
	copy(valuec, value)
	w := trWriteData{
		key:   keyc,
		value: valuec,
		opts:  "put",
	}

	t.w <- w
	return nil
}

// Delete implements database.Writer interface.
func (t *transaction) Delete(key []byte) error {
	//return t.t.Delete(key)
	//ssdbLog("golf*********************ssdb del")
	keyc := make([]byte, len(key))
	copy(keyc, key)
	w := trWriteData{
		key:  keyc,
		opts: delOpt,
	}

	t.w <- w
	return nil
}

func (t *transaction) Commit() error {
	w := trWriteData{
		opts: "commit",
	}

	t.w <- w
	result := <-t.w
	return result.err
}

// Discard is safe to call after Commit(), it would be no-op
func (t *transaction) Discard() {
	ssdbLog("ssdb transaction stop")
	close(t.r)
	close(t.w)
}

// transaction should implement database.Transaction
var _ database.Transaction = &transaction{}
