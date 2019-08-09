// Package database provides KV database for meta-information
package database

import "errors"

// Errors for Storage
var (
	ErrNotFound = errors.New("key not found")
)

// StorageProcessor is a function to process one single storage entry
type StorageProcessor func(key []byte, value []byte) error

// Reader provides KV read calls
type Reader interface {
	Get(key []byte) ([]byte, error)
}

// PrefixReader provides prefixed operations
type PrefixReader interface {
	HasPrefix(prefix []byte) bool
	ProcessByPrefix(prefix []byte, proc StorageProcessor) error
	KeysByPrefix(prefix []byte) [][]byte
	FetchByPrefix(prefix []byte) [][]byte
}

// Writer provides KV update/delete calls
type Writer interface {
	Put(key []byte, value []byte) error
	Delete(key []byte) error
}

// ReaderWriter combines Reader and Writer
type ReaderWriter interface {
	Reader
	Writer
}

// Storage is an interface to KV storage
type Storage interface {
	Reader
	Writer

	PrefixReader

	CreateBatch() Batch
	OpenTransaction() (Transaction, error)

	CreateTemporary() (Storage, error)

	Open() error
	Close() error
	CompactDB() error
	Drop() error
}

// Batch provides a way to pack many writes.
type Batch interface {
	Writer

	// Write closes batch and send accumulated writes to the database
	Write() error
}

// Transaction provides isolated atomic way to perform updates.
//
// Transactions might be expensive.
// Transaction should always finish with either Discard() or Commit()
type Transaction interface {
	Reader
	Writer

	Commit() error
	Discard()
}
