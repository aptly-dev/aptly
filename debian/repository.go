package debian

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Repository abstract file system with package pool and published package repos
type Repository struct {
	RootPath string
}

// NewRepository creates new instance of repository which specified root
func NewRepository(root string) *Repository {
	return &Repository{RootPath: root}
}

// PoolPath returns full path to package file in pool
//
// PoolPath checks that final path doesn't go out of repository root path
func (r *Repository) PoolPath(filename string) (string, error) {
	filename = filepath.Clean(filename)
	if strings.HasPrefix(filename, ".") {
		return "", fmt.Errorf("filename %s starts with dot", filename)
	}

	if filepath.IsAbs(filename) {
		return "", fmt.Errorf("absolute filename %s not supported", filename)
	}

	if strings.HasPrefix(filename, "pool/") {
		filename = filename[5:]
	}

	return filepath.Join(r.RootPath, "pool", filename), nil
}
