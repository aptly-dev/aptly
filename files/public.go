package files

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/utils"
	"github.com/saracen/walker"
)

// PublishedStorage abstract file system with public dirs (published repos)
type PublishedStorage struct {
	rootPath     string
	linkMethod   uint
	verifyMethod uint
}

// Global mutex map to prevent concurrent access to the same destinationPath in LinkFromPool
var (
	fileLockMutex sync.Mutex
	fileLocks = make(map[string]*sync.Mutex)
)

// getFileLock returns a mutex for a specific file path to prevent concurrent modifications
func getFileLock(filePath string) *sync.Mutex {
	fileLockMutex.Lock()
	defer fileLockMutex.Unlock()

	if mutex, exists := fileLocks[filePath]; exists {
		return mutex
	}

	mutex := &sync.Mutex{}
	fileLocks[filePath] = mutex
	return mutex
}

// Check interfaces
var (
	_ aptly.PublishedStorage           = (*PublishedStorage)(nil)
	_ aptly.FileSystemPublishedStorage = (*PublishedStorage)(nil)
)

// Constants defining the type of creating links
const (
	LinkMethodHardLink uint = iota
	LinkMethodSymLink
	LinkMethodCopy
)

// Constants defining the type of file verification for LinkMethodCopy
const (
	VerificationMethodChecksum uint = iota
	VerificationMethodFileSize
)

// NewPublishedStorage creates new instance of PublishedStorage which specified root
func NewPublishedStorage(root string, linkMethod string, verifyMethod string) *PublishedStorage {
	// Ensure linkMethod is one of 'hardlink', 'symlink', 'copy'
	var verifiedLinkMethod uint

	if strings.EqualFold(linkMethod, "copy") {
		verifiedLinkMethod = LinkMethodCopy
	} else if strings.EqualFold(linkMethod, "symlink") {
		verifiedLinkMethod = LinkMethodSymLink
	} else {
		verifiedLinkMethod = LinkMethodHardLink
	}

	var verifiedVerifyMethod uint

	if strings.EqualFold(verifyMethod, "size") {
		verifiedVerifyMethod = VerificationMethodFileSize
	} else {
		verifiedVerifyMethod = VerificationMethodChecksum
	}

	return &PublishedStorage{rootPath: root, linkMethod: verifiedLinkMethod,
		verifyMethod: verifiedVerifyMethod}
}

// PublicPath returns root of public part
func (storage *PublishedStorage) PublicPath() string {
	return storage.rootPath
}

// MkDir creates directory recursively under public path
func (storage *PublishedStorage) MkDir(path string) error {
	return os.MkdirAll(filepath.Join(storage.rootPath, path), 0777)
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
	defer func() {
		_ = source.Close()
	}()

	f, err = os.Create(filepath.Join(storage.rootPath, path))
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	_, err = io.Copy(f, source)
	return err
}

// Remove removes single file under public path
func (storage *PublishedStorage) Remove(path string) error {
	if len(path) <= 0 {
		panic("trying to remove empty path")
	}
	filepath := filepath.Join(storage.rootPath, path)
	return os.Remove(filepath)
}

// RemoveDirs removes directory structure under public path
func (storage *PublishedStorage) RemoveDirs(path string, progress aptly.Progress) error {
	if len(path) <= 0 {
		panic("trying to remove the root directory")
	}
	filepath := filepath.Join(storage.rootPath, path)
	if progress != nil {
		progress.Printf("Removing %s...\n", filepath)
	}
	return os.RemoveAll(filepath)
}

// LinkFromPool links package file from pool to dist's pool location
//
// publishedPrefix is desired prefix for the location in the pool.
// publishedRelPath is desired location in pool (like pool/component/liba/libav/)
// sourcePool is instance of aptly.PackagePool
// sourcePath is a relative path to package file in package pool
//
// LinkFromPool returns relative path for the published file to be included in package index
func (storage *PublishedStorage) LinkFromPool(publishedPrefix, publishedRelPath, fileName string, sourcePool aptly.PackagePool,
	sourcePath string, sourceChecksums utils.ChecksumInfo, force bool) error {

	baseName := filepath.Base(fileName)
	poolPath := filepath.Join(storage.rootPath, publishedPrefix, publishedRelPath, filepath.Dir(fileName))
	destinationPath := filepath.Join(poolPath, baseName)

	// Acquire file-specific lock to prevent concurrent access to the same file
	fileLock := getFileLock(destinationPath)
	fileLock.Lock()
	defer fileLock.Unlock()

	var localSourcePool aptly.LocalPackagePool
	if storage.linkMethod != LinkMethodCopy {
		pp, ok := sourcePool.(aptly.LocalPackagePool)
		if !ok {
			return fmt.Errorf("cannot link %s from non-local pool %s", baseName, sourcePool)
		}

		localSourcePool = pp
	}

	err := os.MkdirAll(poolPath, 0777)
	if err != nil {
		return err
	}

	var dstStat os.FileInfo

	dstStat, err = os.Stat(destinationPath)
	if err == nil {
		// already exists, check source file

		if storage.linkMethod == LinkMethodCopy {
			srcSize, err := sourcePool.Size(sourcePath)
			if err != nil {
				// source file doesn't exist? problem!
				return err
			}

			if storage.verifyMethod == VerificationMethodFileSize {
				// if source and destination have the same size, no need to copy
				if srcSize == dstStat.Size() {
					return nil
				}
			} else {
				// if source and destination have the same checksums, no need to copy
				var dstMD5 string
				dstMD5, err = utils.MD5ChecksumForFile(destinationPath)

				if err != nil {
					return err
				}

				if dstMD5 == sourceChecksums.MD5 {
					return nil
				}
			}
		} else {
			srcStat, err := localSourcePool.Stat(sourcePath)
			if err != nil {
				// source file doesn't exist? problem!
				return err
			}

			srcSys := srcStat.Sys().(*syscall.Stat_t)
			dstSys := dstStat.Sys().(*syscall.Stat_t)

			// if source and destination inodes match, no need to link

			// Symlink can point to different filesystem with identical inodes
			// so we have to check the device as well.
			if srcSys.Ino == dstSys.Ino && srcSys.Dev == dstSys.Dev {
				return nil
			}
		}

		// source and destination have different inodes, if !forced, this is fatal error
		if !force {
			return fmt.Errorf("error linking file to %s: file already exists and is different", destinationPath)
		}

		// forced, so remove destination
		err = os.Remove(destinationPath)
		if err != nil {
			return err
		}
	}

	// destination doesn't exist (or forced), create link or copy
	if storage.linkMethod == LinkMethodCopy {
		var r aptly.ReadSeekerCloser
		r, err = sourcePool.Open(sourcePath)
		if err != nil {
			return err
		}

		var dst *os.File
		dst, err = os.Create(destinationPath)
		if err != nil {
			_ = r.Close()
			return err
		}

		_, err = io.Copy(dst, r)
		if err != nil {
			_ = r.Close()
			_ = dst.Close()
			return err
		}

		err = r.Close()
		if err != nil {
			_ = dst.Close()
			return err
		}

		err = dst.Close()
	} else if storage.linkMethod == LinkMethodSymLink {
		err = localSourcePool.Symlink(sourcePath, destinationPath)
	} else {
		err = localSourcePool.Link(sourcePath, destinationPath)
	}

	return err
}

// Filelist returns list of files under prefix
func (storage *PublishedStorage) Filelist(prefix string) ([]string, error) {
	root := filepath.Join(storage.rootPath, prefix)
	result := []string{}
	resultLock := &sync.Mutex{}

	err := walker.Walk(root, func(path string, info os.FileInfo) error {
		if !info.IsDir() {
			resultLock.Lock()
			defer resultLock.Unlock()
			result = append(result, path[len(root)+1:])
		}
		return nil
	})

	if err != nil && os.IsNotExist(err) {
		// file path doesn't exist, consider it empty
		return []string{}, nil
	}

	sort.Strings(result)
	return result, err
}

// RenameFile renames (moves) file
func (storage *PublishedStorage) RenameFile(oldName, newName string) error {
	return os.Rename(filepath.Join(storage.rootPath, oldName), filepath.Join(storage.rootPath, newName))
}

// SymLink creates a symbolic link, which can be read with ReadLink
func (storage *PublishedStorage) SymLink(src string, dst string) error {
	return os.Symlink(filepath.Join(storage.rootPath, src), filepath.Join(storage.rootPath, dst))
}

// HardLink creates a hardlink of a file
func (storage *PublishedStorage) HardLink(src string, dst string) error {
	return os.Link(filepath.Join(storage.rootPath, src), filepath.Join(storage.rootPath, dst))
}

// FileExists returns true if path exists
func (storage *PublishedStorage) FileExists(path string) (bool, error) {
	if _, err := os.Lstat(filepath.Join(storage.rootPath, path)); os.IsNotExist(err) {
		return false, nil
	}

	return true, nil
}

// ReadLink returns the symbolic link pointed to by path (relative to storage
// root)
func (storage *PublishedStorage) ReadLink(path string) (string, error) {
	absPath, err := os.Readlink(filepath.Join(storage.rootPath, path))
	if err != nil {
		return absPath, err
	}
	return filepath.Rel(storage.rootPath, absPath)
}
