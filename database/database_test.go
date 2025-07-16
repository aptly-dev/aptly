package database

import (
	"errors"
	"testing"

	check "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { check.TestingT(t) }

type DatabaseSuite struct{}

var _ = check.Suite(&DatabaseSuite{})

func (s *DatabaseSuite) TestErrNotFound(c *check.C) {
	// Test that ErrNotFound is properly defined
	c.Check(ErrNotFound, check.NotNil)
	c.Check(ErrNotFound.Error(), check.Equals, "key not found")

	// Test that it's an actual error
	var err error = ErrNotFound
	c.Check(err, check.NotNil)

	// Test comparison with errors.New
	newErr := errors.New("key not found")
	c.Check(ErrNotFound.Error(), check.Equals, newErr.Error())

	// Test that it's not equal to other errors
	otherErr := errors.New("other error")
	c.Check(ErrNotFound.Error(), check.Not(check.Equals), otherErr.Error())
}

func (s *DatabaseSuite) TestStorageProcessor(c *check.C) {
	// Test StorageProcessor function type
	called := false
	var processor StorageProcessor = func(key []byte, value []byte) error {
		called = true
		c.Check(key, check.DeepEquals, []byte("test-key"))
		c.Check(value, check.DeepEquals, []byte("test-value"))
		return nil
	}

	err := processor([]byte("test-key"), []byte("test-value"))
	c.Check(err, check.IsNil)
	c.Check(called, check.Equals, true)
}

func (s *DatabaseSuite) TestStorageProcessorWithError(c *check.C) {
	// Test StorageProcessor that returns an error
	testError := errors.New("processing error")
	var processor StorageProcessor = func(key []byte, value []byte) error {
		return testError
	}

	err := processor([]byte("key"), []byte("value"))
	c.Check(err, check.Equals, testError)
}

func (s *DatabaseSuite) TestStorageProcessorNilInputs(c *check.C) {
	// Test StorageProcessor with nil inputs
	var processor StorageProcessor = func(key []byte, value []byte) error {
		c.Check(key, check.IsNil)
		c.Check(value, check.DeepEquals, []byte("value"))
		return nil
	}

	err := processor(nil, []byte("value"))
	c.Check(err, check.IsNil)
}

func (s *DatabaseSuite) TestStorageProcessorEmptyInputs(c *check.C) {
	// Test StorageProcessor with empty inputs
	var processor StorageProcessor = func(key []byte, value []byte) error {
		c.Check(len(key), check.Equals, 0)
		c.Check(len(value), check.Equals, 0)
		return nil
	}

	err := processor([]byte{}, []byte{})
	c.Check(err, check.IsNil)
}

// Mock implementations to test interface compliance
type mockReader struct {
	data map[string][]byte
}

func (m *mockReader) Get(key []byte) ([]byte, error) {
	if value, exists := m.data[string(key)]; exists {
		return value, nil
	}
	return nil, ErrNotFound
}

type mockWriter struct {
	data map[string][]byte
}

func (m *mockWriter) Put(key []byte, value []byte) error {
	m.data[string(key)] = value
	return nil
}

func (m *mockWriter) Delete(key []byte) error {
	delete(m.data, string(key))
	return nil
}

type mockReaderWriter struct {
	*mockReader
	*mockWriter
}

func (s *DatabaseSuite) TestReaderInterface(c *check.C) {
	// Test Reader interface implementation
	data := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
	}

	var reader Reader = &mockReader{data: data}

	// Test existing key
	value, err := reader.Get([]byte("key1"))
	c.Check(err, check.IsNil)
	c.Check(value, check.DeepEquals, []byte("value1"))

	// Test non-existing key
	value, err = reader.Get([]byte("nonexistent"))
	c.Check(err, check.Equals, ErrNotFound)
	c.Check(value, check.IsNil)
}

func (s *DatabaseSuite) TestWriterInterface(c *check.C) {
	// Test Writer interface implementation
	data := make(map[string][]byte)
	var writer Writer = &mockWriter{data: data}

	// Test Put
	err := writer.Put([]byte("key1"), []byte("value1"))
	c.Check(err, check.IsNil)
	c.Check(data["key1"], check.DeepEquals, []byte("value1"))

	// Test Delete
	err = writer.Delete([]byte("key1"))
	c.Check(err, check.IsNil)
	_, exists := data["key1"]
	c.Check(exists, check.Equals, false)
}

func (s *DatabaseSuite) TestReaderWriterInterface(c *check.C) {
	// Test ReaderWriter interface implementation
	data := make(map[string][]byte)

	var rw ReaderWriter = &mockReaderWriter{
		mockReader: &mockReader{data: data},
		mockWriter: &mockWriter{data: data},
	}

	// Test write then read
	err := rw.Put([]byte("test"), []byte("value"))
	c.Check(err, check.IsNil)

	value, err := rw.Get([]byte("test"))
	c.Check(err, check.IsNil)
	c.Check(value, check.DeepEquals, []byte("value"))

	// Test delete
	err = rw.Delete([]byte("test"))
	c.Check(err, check.IsNil)

	value, err = rw.Get([]byte("test"))
	c.Check(err, check.Equals, ErrNotFound)
	c.Check(value, check.IsNil)
}

// Test that all interfaces are properly defined
func (s *DatabaseSuite) TestInterfaceDefinitions(c *check.C) {
	// This test ensures that all interfaces are properly defined
	// and can be used as interface types

	var reader Reader
	var prefixReader PrefixReader
	var writer Writer
	var readerWriter ReaderWriter
	var storage Storage
	var batch Batch
	var transaction Transaction

	// Test that they are nil by default
	c.Check(reader, check.IsNil)
	c.Check(prefixReader, check.IsNil)
	c.Check(writer, check.IsNil)
	c.Check(readerWriter, check.IsNil)
	c.Check(storage, check.IsNil)
	c.Check(batch, check.IsNil)
	c.Check(transaction, check.IsNil)
}

func (s *DatabaseSuite) TestErrorConstants(c *check.C) {
	// Test that error constants are immutable and consistently defined
	original := ErrNotFound
	c.Check(original, check.NotNil)

	// Verify it maintains its identity
	c.Check(ErrNotFound, check.Equals, original)
	c.Check(ErrNotFound.Error(), check.Equals, original.Error())
}
