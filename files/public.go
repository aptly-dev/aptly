package files

import (
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

// Remove removes single file under public path
func (storage *PublishedStorage) Remove(path string) error {
	filepath := filepath.Join(storage.rootPath, path)
	return os.Remove(filepath)
}

// RemoveDirs removes directory structure under public path
func (storage *PublishedStorage) RemoveDirs(path string, progress aptly.Progress) error {
	filepath := filepath.Join(storage.rootPath, path)
	if progress != nil {
		progress.Printf("Removing %s...\n", filepath)
	}
	return os.RemoveAll(filepath)
}

// LinkFromPool links package file from pool to dist's pool location
//
// publishedDirectory is desired location in pool (like prefix/pool/component/liba/libav/)
// sourcePool is instance of aptly.PackagePool
// sourcePath is filepath to package file in package pool
//
// LinkFromPool returns relative path for the published file to be included in package index
func (storage *PublishedStorage) LinkFromPool(publishedDirectory string, sourcePool aptly.PackagePool, sourcePath string) error {
	// verify that package pool is local pool is filesystem pool
	_ = sourcePool.(*PackagePool)

	baseName := filepath.Base(sourcePath)
	poolPath := filepath.Join(storage.rootPath, publishedDirectory)

	err := os.MkdirAll(poolPath, 0755)
	if err != nil {
		return err
	}

	_, err = os.Stat(filepath.Join(poolPath, baseName))
	if err == nil { // already exists, skip
		return nil
	}

	return os.Link(sourcePath, filepath.Join(poolPath, baseName))
}

// Filelist returns list of files under prefix
func (storage *PublishedStorage) Filelist(prefix string) ([]string, error) {
	root := filepath.Join(storage.rootPath, prefix)
	result := []string{}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			result = append(result, path[len(root)+1:])
		}
		return nil
	})

	return result, err
}

// ChecksumsForFile proxies requests to utils.ChecksumsForFile, joining public path
func (storage *PublishedStorage) ChecksumsForFile(path string) (utils.ChecksumInfo, error) {
	return utils.ChecksumsForFile(filepath.Join(storage.rootPath, path))
}

// RenameFile renames (moves) file
func (storage *PublishedStorage) RenameFile(oldName, newName string) error {
	return os.Rename(filepath.Join(storage.rootPath, oldName), filepath.Join(storage.rootPath, newName))
}
