package files

import (
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/utils"
)

type mockChecksumStorage struct {
	store map[string]utils.ChecksumInfo
}

// NewMockChecksumtorage creates aptly.ChecksumStorage for tests
func NewMockChecksumtorage() aptly.ChecksumStorage {
	return &mockChecksumStorage{
		store: make(map[string]utils.ChecksumInfo),
	}
}

func (st *mockChecksumStorage) Get(path string) (*utils.ChecksumInfo, error) {
	c, ok := st.store[path]
	if !ok {
		return nil, nil
	}

	return &c, nil
}

func (st *mockChecksumStorage) Update(path string, c *utils.ChecksumInfo) error {
	st.store[path] = *c
	return nil
}

// Check interface
var (
	_ aptly.ChecksumStorage = &mockChecksumStorage{}
)
