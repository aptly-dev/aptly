package deb

import (
	"bytes"
	"strings"

	"github.com/aptly-dev/aptly/database"
	. "gopkg.in/check.v1"
)

type ContentsIndexSuite struct {
	mockDB *MockStorage
}

var _ = Suite(&ContentsIndexSuite{})

func (s *ContentsIndexSuite) SetUpTest(c *C) {
	s.mockDB = &MockStorage{
		data:     make(map[string][]byte),
		prefixes: make(map[string]bool),
	}
}

func (s *ContentsIndexSuite) TestNewContentsIndex(c *C) {
	// Test ContentsIndex creation
	index := NewContentsIndex(s.mockDB)
	
	c.Check(index, NotNil)
	c.Check(index.db, Equals, s.mockDB)
	c.Check(len(index.prefix), Equals, 36) // UUID length
}

func (s *ContentsIndexSuite) TestContentsIndexEmpty(c *C) {
	// Test Empty method
	index := NewContentsIndex(s.mockDB)
	
	// Should be empty initially
	c.Check(index.Empty(), Equals, true)
	
	// Add some data
	s.mockDB.prefixes[string(index.prefix)] = true
	
	// Should not be empty now
	c.Check(index.Empty(), Equals, false)
}

func (s *ContentsIndexSuite) TestContentsIndexPush(c *C) {
	// Test Push method
	index := NewContentsIndex(s.mockDB)
	writer := &MockWriter{storage: s.mockDB}
	
	qualifiedName := []byte("package_1.0_amd64")
	contents := []string{
		"/usr/bin/program",
		"/usr/share/doc/package/README",
		"/etc/package.conf",
	}
	
	err := index.Push(qualifiedName, contents, writer)
	c.Check(err, IsNil)
	
	// Verify data was written
	c.Check(len(s.mockDB.data), Equals, 3)
	
	// Check that keys contain the expected format
	for path := range contents {
		expectedKey := string(index.prefix) + contents[path] + "\x00" + string(qualifiedName)
		_, exists := s.mockDB.data[expectedKey]
		c.Check(exists, Equals, true, Commentf("Missing key for path: %s", contents[path]))
	}
}

func (s *ContentsIndexSuite) TestContentsIndexPushError(c *C) {
	// Test Push method with writer error
	index := NewContentsIndex(s.mockDB)
	writer := &MockWriter{storage: s.mockDB, shouldError: true}
	
	qualifiedName := []byte("package_1.0_amd64")
	contents := []string{"/usr/bin/program"}
	
	err := index.Push(qualifiedName, contents, writer)
	c.Check(err, NotNil)
	c.Check(err.Error(), Equals, "mock writer error")
}

func (s *ContentsIndexSuite) TestContentsIndexWriteTo(c *C) {
	// Test WriteTo method
	index := NewContentsIndex(s.mockDB)
	writer := &MockWriter{storage: s.mockDB}
	
	// Add some packages
	err := index.Push([]byte("package1_1.0_amd64"), []string{"/usr/bin/prog1", "/usr/share/file1"}, writer)
	c.Check(err, IsNil)
	
	err = index.Push([]byte("package2_2.0_amd64"), []string{"/usr/bin/prog2", "/usr/share/file1"}, writer)
	c.Check(err, IsNil)
	
	// Set up processor to simulate database iteration
	s.mockDB.processor = func(prefix []byte, fn database.StorageProcessor) error {
		// Simulate database keys in sorted order
		keys := []string{
			string(prefix) + "/usr/bin/prog1\x00package1_1.0_amd64",
			string(prefix) + "/usr/bin/prog2\x00package2_2.0_amd64", 
			string(prefix) + "/usr/share/file1\x00package1_1.0_amd64",
			string(prefix) + "/usr/share/file1\x00package2_2.0_amd64",
		}
		
		for _, key := range keys {
			err := fn([]byte(key), nil)
			if err != nil {
				return err
			}
		}
		return nil
	}
	
	var buf bytes.Buffer
	n, err := index.WriteTo(&buf)
	c.Check(err, IsNil)
	c.Check(n, Equals, int64(buf.Len()))
	
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	
	// Should have header plus content lines
	c.Check(len(lines), Equals, 4)
	c.Check(lines[0], Equals, "FILE LOCATION")
	c.Check(lines[1], Equals, "/usr/bin/prog1 package1_1.0_amd64")
	c.Check(lines[2], Equals, "/usr/bin/prog2 package2_2.0_amd64")
	c.Check(lines[3], Equals, "/usr/share/file1 package1_1.0_amd64,package2_2.0_amd64")
}

func (s *ContentsIndexSuite) TestContentsIndexWriteToEmpty(c *C) {
	// Test WriteTo with empty index
	index := NewContentsIndex(s.mockDB)
	
	s.mockDB.processor = func(prefix []byte, fn database.StorageProcessor) error {
		// No entries
		return nil
	}
	
	var buf bytes.Buffer
	n, err := index.WriteTo(&buf)
	c.Check(err, IsNil)
	c.Check(n, Equals, int64(buf.Len()))
	
	output := buf.String()
	c.Check(output, Equals, "FILE LOCATION\n")
}

func (s *ContentsIndexSuite) TestContentsIndexWriteToCorruptedEntry(c *C) {
	// Test WriteTo with corrupted database entry
	index := NewContentsIndex(s.mockDB)
	
	s.mockDB.processor = func(prefix []byte, fn database.StorageProcessor) error {
		// Corrupted key without null byte separator
		corruptedKey := string(prefix) + "/usr/bin/prog1package_name"
		return fn([]byte(corruptedKey), nil)
	}
	
	var buf bytes.Buffer
	_, err := index.WriteTo(&buf)
	c.Check(err, NotNil)
	c.Check(err.Error(), Equals, "corrupted index entry")
}

func (s *ContentsIndexSuite) TestContentsIndexPushMultiplePackages(c *C) {
	// Test pushing multiple packages
	index := NewContentsIndex(s.mockDB)
	writer := &MockWriter{storage: s.mockDB}
	
	packages := []struct {
		name     string
		contents []string
	}{
		{"package1_1.0_amd64", []string{"/usr/bin/prog1", "/usr/share/doc1"}},
		{"package2_2.0_amd64", []string{"/usr/bin/prog2", "/usr/share/doc2"}},
		{"package3_3.0_amd64", []string{"/usr/bin/prog3"}},
	}
	
	for _, pkg := range packages {
		err := index.Push([]byte(pkg.name), pkg.contents, writer)
		c.Check(err, IsNil, Commentf("Failed to push package: %s", pkg.name))
	}
	
	// Verify all entries were written
	expectedEntries := 2 + 2 + 1 // Total files across all packages
	c.Check(len(s.mockDB.data), Equals, expectedEntries)
}

func (s *ContentsIndexSuite) TestContentsIndexPushEmptyContents(c *C) {
	// Test pushing package with no contents
	index := NewContentsIndex(s.mockDB)
	writer := &MockWriter{storage: s.mockDB}
	
	err := index.Push([]byte("empty_package"), []string{}, writer)
	c.Check(err, IsNil)
	
	// Should not add any entries
	c.Check(len(s.mockDB.data), Equals, 0)
}

func (s *ContentsIndexSuite) TestContentsIndexSpecialCharacters(c *C) {
	// Test with special characters in paths and package names
	index := NewContentsIndex(s.mockDB)
	writer := &MockWriter{storage: s.mockDB}
	
	qualifiedName := []byte("special-package_1.0+build1_amd64")
	contents := []string{
		"/usr/bin/prog-with-dashes",
		"/usr/share/file with spaces",
		"/etc/config.d/file.conf",
	}
	
	err := index.Push(qualifiedName, contents, writer)
	c.Check(err, IsNil)
	
	c.Check(len(s.mockDB.data), Equals, 3)
}

func (s *ContentsIndexSuite) TestContentsIndexBinaryData(c *C) {
	// Test with binary data in paths (edge case)
	index := NewContentsIndex(s.mockDB)
	writer := &MockWriter{storage: s.mockDB}
	
	// Path with binary data
	binaryPath := "/usr/bin/prog\x00\xFF\xFE"
	qualifiedName := []byte("binary_package_1.0_amd64")
	
	err := index.Push(qualifiedName, []string{binaryPath}, writer)
	c.Check(err, IsNil)
	
	c.Check(len(s.mockDB.data), Equals, 1)
}

// Mock implementations for testing
type MockStorage struct {
	data      map[string][]byte
	prefixes  map[string]bool
	processor func([]byte, database.StorageProcessor) error
}

func (m *MockStorage) Get(key []byte) ([]byte, error) {
	if value, exists := m.data[string(key)]; exists {
		return value, nil
	}
	return nil, database.ErrNotFound
}

func (m *MockStorage) Put(key, value []byte) error {
	m.data[string(key)] = value
	return nil
}

func (m *MockStorage) Delete(key []byte) error {
	delete(m.data, string(key))
	return nil
}

func (m *MockStorage) HasPrefix(prefix []byte) bool {
	if exists, ok := m.prefixes[string(prefix)]; ok {
		return exists
	}
	
	// Check if any key has this prefix
	for key := range m.data {
		if strings.HasPrefix(key, string(prefix)) {
			return true
		}
	}
	return false
}

func (m *MockStorage) ProcessByPrefix(prefix []byte, fn database.StorageProcessor) error {
	if m.processor != nil {
		return m.processor(prefix, fn)
	}
	
	// Default implementation - process matching keys
	for key, value := range m.data {
		if strings.HasPrefix(key, string(prefix)) {
			err := fn([]byte(key), value)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *MockStorage) KeysByPrefix(prefix []byte) [][]byte {
	var keys [][]byte
	for key := range m.data {
		if strings.HasPrefix(key, string(prefix)) {
			keys = append(keys, []byte(key))
		}
	}
	return keys
}

func (m *MockStorage) FetchByPrefix(prefix []byte) [][]byte {
	var values [][]byte
	for key, value := range m.data {
		if strings.HasPrefix(key, string(prefix)) {
			values = append(values, value)
		}
	}
	return values
}

func (m *MockStorage) Close() error {
	return nil
}

func (m *MockStorage) CompactDB() error {
	return nil
}

func (m *MockStorage) Drop() error {
	return nil
}

func (m *MockStorage) Open() error {
	return nil
}

func (m *MockStorage) CreateBatch() database.Batch {
	return &MockBatch{storage: m}
}

func (m *MockStorage) OpenTransaction() (database.Transaction, error) {
	return &MockTransaction{storage: m}, nil
}

func (m *MockStorage) CreateTemporary() (database.Storage, error) {
	return &MockStorage{
		data:     make(map[string][]byte),
		prefixes: make(map[string]bool),
	}, nil
}

type MockBatch struct {
	storage *MockStorage
}

func (m *MockBatch) Put(key, value []byte) error {
	return m.storage.Put(key, value)
}

func (m *MockBatch) Delete(key []byte) error {
	return m.storage.Delete(key)
}

func (m *MockBatch) Write() error {
	return nil
}

type MockTransaction struct {
	storage *MockStorage
}

func (m *MockTransaction) Get(key []byte) ([]byte, error) {
	return m.storage.Get(key)
}

func (m *MockTransaction) Put(key, value []byte) error {
	return m.storage.Put(key, value)
}

func (m *MockTransaction) Delete(key []byte) error {
	return m.storage.Delete(key)
}

func (m *MockTransaction) Commit() error {
	return nil
}

func (m *MockTransaction) Discard() {
}

type MockWriter struct {
	storage     *MockStorage
	shouldError bool
}

func (m *MockWriter) Put(key, value []byte) error {
	if m.shouldError {
		return &MockError{message: "mock writer error"}
	}
	return m.storage.Put(key, value)
}

func (m *MockWriter) Delete(key []byte) error {
	if m.shouldError {
		return &MockError{message: "mock writer error"}
	}
	return m.storage.Delete(key)
}

type MockError struct {
	message string
}

func (e *MockError) Error() string {
	return e.message
}