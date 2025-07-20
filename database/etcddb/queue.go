package etcddb

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"github.com/rs/zerolog/log"
)

// QueueConfig contains configuration for the write queue
type QueueConfig struct {
	Enabled         bool
	WriteQueueSize  int
	MaxWritesPerSec int
	BatchMaxSize    int
	BatchMaxWait    time.Duration
}

// DefaultQueueConfig returns default queue configuration
func DefaultQueueConfig() *QueueConfig {
	return &QueueConfig{
		Enabled:         false,
		WriteQueueSize:  1000,
		MaxWritesPerSec: 100,
		BatchMaxSize:    50,
		BatchMaxWait:    10 * time.Millisecond,
	}
}

// writeOp represents a queued write operation
type writeOp struct {
	fn     func() error
	result chan error
}

// QueuedEtcdClient wraps an etcd client with write queueing
type QueuedEtcdClient struct {
	client     *clientv3.Client
	kv         clientv3.KV
	writeQueue chan writeOp
	config     *QueueConfig
	wg         sync.WaitGroup
	done       chan struct{}
	closed     atomic.Bool
}

// NewQueuedEtcdClient creates a new queued etcd client
func NewQueuedEtcdClient(client *clientv3.Client, config *QueueConfig) *QueuedEtcdClient {
	if config == nil {
		config = DefaultQueueConfig()
	}

	qc := &QueuedEtcdClient{
		client:     client,
		kv:         client.KV,
		writeQueue: make(chan writeOp, config.WriteQueueSize),
		config:     config,
		done:       make(chan struct{}),
	}

	if config.Enabled {
		qc.wg.Add(1)
		go qc.processQueue()
	}

	return qc
}

// processQueue processes write operations sequentially
func (qc *QueuedEtcdClient) processQueue() {
	defer qc.wg.Done()

	ticker := time.NewTicker(time.Second / time.Duration(qc.config.MaxWritesPerSec))
	defer ticker.Stop()

	for {
		select {
		case <-qc.done:
			// Cancel remaining operations
			for len(qc.writeQueue) > 0 {
				select {
				case op := <-qc.writeQueue:
					op.result <- context.Canceled
				default:
					return
				}
			}
			return

		case op := <-qc.writeQueue:
			if qc.closed.Load() {
				op.result <- context.Canceled
				continue
			}
			qc.executeOp(op)
			<-ticker.C // Rate limiting after operation
		}
	}
}

// executeOp executes a single write operation
func (qc *QueuedEtcdClient) executeOp(op writeOp) {
	start := time.Now()
	err := op.fn()
	duration := time.Since(start)

	if err != nil {
		log.Warn().Err(err).Dur("duration", duration).Msg("etcd write operation failed")
	} else {
		log.Debug().Dur("duration", duration).Msg("etcd write operation completed")
	}

	op.result <- err
}

// Close closes the queued client
func (qc *QueuedEtcdClient) Close() error {
	if qc.config.Enabled {
		qc.closed.Store(true)
		close(qc.done)
		
		// Wait for queue to drain with timeout
		done := make(chan struct{})
		go func() {
			qc.wg.Wait()
			close(done)
		}()
		
		select {
		case <-done:
			// Queue drained successfully
		case <-time.After(5 * time.Second):
			// Timeout - log warning but continue
			log.Warn().Msg("etcd: queue close timeout, some operations may be lost")
		}
	}
	return qc.client.Close()
}

// QueuedKV implements clientv3.KV with write queueing
type QueuedKV struct {
	kv         clientv3.KV
	writeQueue chan writeOp
	config     *QueueConfig
}

// NewQueuedKV creates a new queued KV interface
func NewQueuedKV(kv clientv3.KV, writeQueue chan writeOp, config *QueueConfig) *QueuedKV {
	return &QueuedKV{
		kv:         kv,
		writeQueue: writeQueue,
		config:     config,
	}
}

// Put queues a put operation
func (qkv *QueuedKV) Put(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	if !qkv.config.Enabled {
		return qkv.kv.Put(ctx, key, val, opts...)
	}

	resultChan := make(chan error, 1)
	respChan := make(chan *clientv3.PutResponse, 1)

	select {
	case qkv.writeQueue <- writeOp{
		fn: func() error {
			resp, err := qkv.kv.Put(ctx, key, val, opts...)
			if err == nil {
				respChan <- resp
			}
			return err
		},
		result: resultChan,
	}:
		// Successfully queued
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	select {
	case err := <-resultChan:
		if err != nil {
			return nil, err
		}
		return <-respChan, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Get performs a get operation (not queued)
func (qkv *QueuedKV) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	return qkv.kv.Get(ctx, key, opts...)
}

// Delete queues a delete operation
func (qkv *QueuedKV) Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
	if !qkv.config.Enabled {
		return qkv.kv.Delete(ctx, key, opts...)
	}

	resultChan := make(chan error, 1)
	respChan := make(chan *clientv3.DeleteResponse, 1)

	select {
	case qkv.writeQueue <- writeOp{
		fn: func() error {
			resp, err := qkv.kv.Delete(ctx, key, opts...)
			if err == nil {
				respChan <- resp
			}
			return err
		},
		result: resultChan,
	}:
		// Successfully queued
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	select {
	case err := <-resultChan:
		if err != nil {
			return nil, err
		}
		return <-respChan, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Txn creates a transaction (will be queued)
func (qkv *QueuedKV) Txn(ctx context.Context) clientv3.Txn {
	return &QueuedTxn{
		txn:        qkv.kv.Txn(ctx),
		writeQueue: qkv.writeQueue,
		config:     qkv.config,
		ctx:        ctx,
	}
}

// Do performs a generic operation
func (qkv *QueuedKV) Do(ctx context.Context, op clientv3.Op) (clientv3.OpResponse, error) {
	// Determine if this is a write operation
	if op.IsGet() {
		// Read operations are not queued
		return qkv.kv.Do(ctx, op)
	}

	if !qkv.config.Enabled {
		return qkv.kv.Do(ctx, op)
	}

	// Queue write operations
	resultChan := make(chan error, 1)
	respChan := make(chan clientv3.OpResponse, 1)

	select {
	case qkv.writeQueue <- writeOp{
		fn: func() error {
			resp, err := qkv.kv.Do(ctx, op)
			if err == nil {
				respChan <- resp
			}
			return err
		},
		result: resultChan,
	}:
		// Successfully queued
	case <-ctx.Done():
		return clientv3.OpResponse{}, ctx.Err()
	}

	err := <-resultChan
	if err != nil {
		return clientv3.OpResponse{}, err
	}
	return <-respChan, nil
}

// Compact queues a compact operation
func (qkv *QueuedKV) Compact(ctx context.Context, rev int64, opts ...clientv3.CompactOption) (*clientv3.CompactResponse, error) {
	if !qkv.config.Enabled {
		return qkv.kv.Compact(ctx, rev, opts...)
	}

	resultChan := make(chan error, 1)
	respChan := make(chan *clientv3.CompactResponse, 1)

	select {
	case qkv.writeQueue <- writeOp{
		fn: func() error {
			resp, err := qkv.kv.Compact(ctx, rev, opts...)
			if err == nil {
				respChan <- resp
			}
			return err
		},
		result: resultChan,
	}:
		// Successfully queued
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	select {
	case err := <-resultChan:
		if err != nil {
			return nil, err
		}
		return <-respChan, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// QueuedTxn wraps a transaction with queueing
type QueuedTxn struct {
	txn        clientv3.Txn
	writeQueue chan writeOp
	config     *QueueConfig
	ctx        context.Context
}

// If sets the comparison target
func (qtxn *QueuedTxn) If(cs ...clientv3.Cmp) clientv3.Txn {
	qtxn.txn = qtxn.txn.If(cs...)
	return qtxn
}

// Then sets the success operations
func (qtxn *QueuedTxn) Then(ops ...clientv3.Op) clientv3.Txn {
	qtxn.txn = qtxn.txn.Then(ops...)
	return qtxn
}

// Else sets the failure operations
func (qtxn *QueuedTxn) Else(ops ...clientv3.Op) clientv3.Txn {
	qtxn.txn = qtxn.txn.Else(ops...)
	return qtxn
}

// Commit queues the transaction commit
func (qtxn *QueuedTxn) Commit() (*clientv3.TxnResponse, error) {
	if !qtxn.config.Enabled {
		return qtxn.txn.Commit()
	}

	resultChan := make(chan error, 1)
	respChan := make(chan *clientv3.TxnResponse, 1)

	select {
	case qtxn.writeQueue <- writeOp{
		fn: func() error {
			resp, err := qtxn.txn.Commit()
			if err == nil {
				respChan <- resp
			}
			return err
		},
		result: resultChan,
	}:
		// Successfully queued
	case <-qtxn.ctx.Done():
		return nil, qtxn.ctx.Err()
	}

	select {
	case err := <-resultChan:
		if err != nil {
			return nil, err
		}
		return <-respChan, nil
	case <-qtxn.ctx.Done():
		return nil, qtxn.ctx.Err()
	}
}