package goleveldb

import (
	"github.com/aptly-dev/aptly/database"
	. "gopkg.in/check.v1"
)

type TransactionSuite struct {
	storage *storage
	tempDir string
}

var _ = Suite(&TransactionSuite{})

func (s *TransactionSuite) SetUpTest(c *C) {
	s.tempDir = c.MkDir()
	s.storage = &storage{
		path: s.tempDir,
		db:   nil, // Not opened for unit tests
	}
}

func (s *TransactionSuite) TestOpenTransactionNilDB(c *C) {
	// Test opening transaction with nil DB - should fail
	transaction, err := s.storage.OpenTransaction()
	c.Check(err, NotNil) // Expected to fail with nil DB
	c.Check(transaction, IsNil)
}

func (s *TransactionSuite) TestInterfaceCompliance(c *C) {
	// Test that storage implements the transaction interface
	var storageInterface database.Storage = &storage{}
	c.Check(storageInterface, NotNil)

	// Test that we can call OpenTransaction method
	_, err := storageInterface.OpenTransaction()
	c.Check(err, NotNil) // Expected to fail without proper setup
}
