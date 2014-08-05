package files

import (
	"fmt"
	"github.com/smira/aptly/aptly"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// PublishedStorage abstract file system with public dirs (published repos)
type PublishedStorage struct {
	rootPath string
}

// Check interfaces
var (
	_ aptly.PublishedStorage      = (*PublishedStorage)(nil)
	_ aptly.LocalPublishedStorage = (*PublishedStorage)(nil)
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

// PutFile puts file into published storage at specified path
func (storage *PublishedStorage) PutFile(path string, sourceFilename string) error {
	var (
		source, f *os.File
		err       error
	)
	source, err = os.Open(sourceFilename)
	if err != nil {
		return err
	}
	defer source.Close()

	f, err = os.Create(filepath.Join(storage.rootPath, path))
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, source)
	return err
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
func (storage *PublishedStorage) LinkFromPool(publishedDirectory string, sourcePool aptly.PackagePool,
	sourcePath, sourceMD5 string, force bool) error {
	// verify that package pool is local pool is filesystem pool
	_ = sourcePool.(*PackagePool)

	baseName := filepath.Base(sourcePath)
	poolPath := filepath.Join(storage.rootPath, publishedDirectory)

	err := os.MkdirAll(poolPath, 0755)
	if err != nil {
		return err
	}

	var dstStat, srcStat os.FileInfo

	dstStat, err = os.Stat(filepath.Join(poolPath, baseName))
	if err == nil {
		// already exists, check source file
		srcStat, err = os.Stat(sourcePath)
		if err != nil {
			// source file doesn't exist? problem!
			return err
		}

		srcSys := srcStat.Sys().(*syscall.Stat_t)
		dstSys := dstStat.Sys().(*syscall.Stat_t)

		// source and destination inodes match, no need to link
		if srcSys.Ino == dstSys.Ino {
			return nil
		}

		// source and destination have different inodes, if !forced, this is fatal error
		if !force {
			return fmt.Errorf("error linking file to %s: file already exists and is different", filepath.Join(poolPath, baseName))
		}

		// forced, so remove destination
		err = os.Remove(filepath.Join(poolPath, baseName))
		if err != nil {
			return err
		}
	}

	// destination doesn't exist (or forced), create link
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

	if err != nil && os.IsNotExist(err) {
		// file path doesn't exist, consider it empty
		return []string{}, nil
	}

	return result, err
}

// RenameFile renames (moves) file
func (storage *PublishedStorage) RenameFile(oldName, newName string) error {
	return os.Rename(filepath.Join(storage.rootPath, oldName), filepath.Join(storage.rootPath, newName))
}
