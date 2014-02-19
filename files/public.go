package files

import (
	"fmt"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/utils"
	"os"
	"path/filepath"
)

// PublishedStorage abstract file system with public dirs (published repos)
type PublishedStorage struct {
	rootPath string
}

// Check interface
var (
	_ aptly.PublishedStorage = (*PublishedStorage)(nil)
)

// NewPublishedStorage creates new instance of PublishedStorage which specified root
func NewPublishedStorage(root string) *PublishedStorage {
	return &PublishedStorage{rootPath: filepath.Join(root, "public")}
}

// PublicPath returns root of public part
func (storage *PublishedStorage) PublicPath() string {
	return storage.rootPath
}

// MkDir creates directory recursively under public path
func (storage *PublishedStorage) MkDir(path string) error {
	return os.MkdirAll(filepath.Join(storage.rootPath, path), 0755)
}

// CreateFile creates file for writing under public path
func (storage *PublishedStorage) CreateFile(path string) (*os.File, error) {
	return os.Create(filepath.Join(storage.rootPath, path))
}

// RemoveDirs removes directory structure under public path
func (storage *PublishedStorage) RemoveDirs(path string) error {
	filepath := filepath.Join(storage.rootPath, path)
	fmt.Printf("Removing %s...\n", filepath)
	return os.RemoveAll(filepath)
}

// LinkFromPool links package file from pool to dist's pool location
//
// prefix is publishing prefix for this repo (e.g. empty or "ppa/")
// component is component name when publishing (e.g. main)
// poolDirectory is desired location in pool (like liba/libav/)
// sourcePool is instance of aptly.PackagePool
// sourcePath is filepath to package file in package pool
//
// LinkFromPool returns relative path for the published file to be included in package index
func (storage *PublishedStorage) LinkFromPool(prefix string, component string, poolDirectory string, sourcePool aptly.PackagePool, sourcePath string) (string, error) {
	// verify that package pool is local pool is filesystem pool
	_ = sourcePool.(*PackagePool)

	baseName := filepath.Base(sourcePath)

	relPath := filepath.Join("pool", component, poolDirectory, baseName)
	poolPath := filepath.Join(storage.rootPath, prefix, "pool", component, poolDirectory)

	err := os.MkdirAll(poolPath, 0755)
	if err != nil {
		return "", err
	}

	_, err = os.Stat(filepath.Join(poolPath, baseName))
	if err == nil { // already exists, skip
		return relPath, nil
	}

	err = os.Link(sourcePath, filepath.Join(poolPath, baseName))
	return relPath, err
}

// ChecksumsForFile proxies requests to utils.ChecksumsForFile, joining public path
func (storage *PublishedStorage) ChecksumsForFile(path string) (utils.ChecksumInfo, error) {
	return utils.ChecksumsForFile(filepath.Join(storage.rootPath, path))
}
