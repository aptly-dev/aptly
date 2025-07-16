package etcddb

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/aptly-dev/aptly/database"
	"github.com/rs/zerolog/log"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type EtcDBatch struct {
	s   *EtcDStorage
	ops []clientv3.Op
}

type WriteOptions struct {
	NoWriteMerge bool
	Sync         bool
}

func (b *EtcDBatch) Put(key []byte, value []byte) (err error) {
	b.ops = append(b.ops, clientv3.OpPut(string(key), string(value)))
	return
}

func (b *EtcDBatch) Delete(key []byte) (err error) {
	b.ops = append(b.ops, clientv3.OpDelete(string(key)))
	return
}

func (b *EtcDBatch) Write() (err error) {
	kv := clientv3.NewKV(b.s.db)

	batchSize := 128
	for i := 0; i < len(b.ops); i += batchSize {
		end := i + batchSize
		if end > len(b.ops) {
			end = len(b.ops)
		}

		batch := b.ops[i:end]

		// Retry logic with exponential backoff
		var lastErr error
		for retry := 0; retry <= DefaultWriteRetries; retry++ {
			ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
			txn := kv.Txn(ctx)
			txn.Then(batch...)
			_, err = txn.Commit()
			cancel()

			if err == nil {
				// Success, move to next batch
				break
			}

			lastErr = err

			// Check if error is retryable
			if !isRetryableError(err) {
				log.Error().Err(err).Int("batch_start", i).Int("batch_end", end).Msg("etcd: non-retryable error during batch write")
				return fmt.Errorf("etcd batch write failed: %w", err)
			}

			if retry < DefaultWriteRetries {
				// Calculate exponential backoff
				backoff := time.Duration(math.Pow(2, float64(retry))) * 100 * time.Millisecond
				if backoff > 5*time.Second {
					backoff = 5 * time.Second
				}

				log.Warn().Err(err).
					Int("retry", retry+1).
					Int("max_retries", DefaultWriteRetries).
					Dur("backoff", backoff).
					Int("batch_start", i).
					Int("batch_end", end).
					Msg("etcd: batch write failed, retrying")

				time.Sleep(backoff)
			}
		}

		// All retries exhausted
		if lastErr != nil {
			log.Error().Err(lastErr).
				Int("batch_start", i).
				Int("batch_end", end).
				Int("retries", DefaultWriteRetries).
				Msg("etcd: batch write failed after all retries")
			return fmt.Errorf("etcd batch write failed after %d retries: %w", DefaultWriteRetries, lastErr)
		}
	}

	return nil
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for gRPC status errors
	if s, ok := status.FromError(err); ok {
		switch s.Code() {
		case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Aborted:
			return true
		}
	}

	// Check for context errors
	if err == context.DeadlineExceeded || err == context.Canceled {
		return true
	}

	// Check for timeout errors in error message
	if errStr := err.Error(); errStr != "" {
		if contains(errStr, "timeout") || contains(errStr, "timed out") ||
			contains(errStr, "unavailable") || contains(errStr, "connection refused") {
			return true
		}
	}

	return false
}

// contains is a simple string contains helper
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// batch should implement database.Batch
var (
	_ database.Batch = &EtcDBatch{}
)
