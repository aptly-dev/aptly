package files

import (
	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/utils"
)

type MockChecksumStorage struct {
	Store map[string]utils.ChecksumInfo
}

// NewMockChecksumStorage creates aptly.ChecksumStorage for tests
func NewMockChecksumStorage() aptly.ChecksumStorage {
	return &MockChecksumStorage{
		Store: make(map[string]utils.ChecksumInfo),
	}
}

func (st *MockChecksumStorage) Get(path string) (*utils.ChecksumInfo, error) {
	c, ok := st.Store[path]
	if !ok {
		return nil, nil
	}

	return &c, nil
}

func (st *MockChecksumStorage) Update(path string, c *utils.ChecksumInfo) error {
	st.Store[path] = *c
	return nil
}

// Check interface
var (
	_ aptly.ChecksumStorage = &MockChecksumStorage{}
)
