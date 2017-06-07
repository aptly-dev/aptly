package files

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/utils"
)

// PublishedStorage abstract file system with public dirs (published repos)
type PublishedStorage struct {
	rootPath     string
	linkMethod   uint
	verifyMethod uint
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
// sourcePath is a relative path to package file in package pool
//
// LinkFromPool returns relative path for the published file to be included in package index
func (storage *PublishedStorage) LinkFromPool(publishedDirectory, baseName string, sourcePool aptly.PackagePool,
	sourcePath string, sourceChecksums utils.ChecksumInfo, force bool) error {

	poolPath := filepath.Join(storage.rootPath, publishedDirectory)

	err := os.MkdirAll(poolPath, 0777)
	if err != nil {
		return err
	}

	var dstStat, srcStat os.FileInfo

	dstStat, err = os.Stat(filepath.Join(poolPath, baseName))
	if err == nil {
		// already exists, check source file
		srcStat, err = sourcePool.Stat(sourcePath)
		if err != nil {
			// source file doesn't exist? problem!
			return err
		}

		if storage.linkMethod == LinkMethodCopy {
			if storage.verifyMethod == VerificationMethodFileSize {
				// if source and destination have the same size, no need to copy
				if srcStat.Size() == dstStat.Size() {
					return nil
				}
			} else {
				// if source and destination have the same checksums, no need to copy
				var dstMD5 string
				dstMD5, err = utils.MD5ChecksumForFile(filepath.Join(poolPath, baseName))

				if err != nil {
					return err
				}

				if dstMD5 == sourceChecksums.MD5 {
					return nil
				}
			}
		} else {
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
			return fmt.Errorf("error linking file to %s: file already exists and is different", filepath.Join(poolPath, baseName))
		}

		// forced, so remove destination
		err = os.Remove(filepath.Join(poolPath, baseName))
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
		dst, err = os.Create(filepath.Join(poolPath, baseName))
		if err != nil {
			r.Close()
			return err
		}

		_, err = io.Copy(dst, r)
		if err != nil {
			r.Close()
			dst.Close()
			return err
		}

		err = r.Close()
		if err != nil {
			dst.Close()
			return err
		}

		err = dst.Close()
	} else if storage.linkMethod == LinkMethodSymLink {
		err = sourcePool.(aptly.LocalPackagePool).Symlink(sourcePath, filepath.Join(poolPath, baseName))
	} else {
		err = sourcePool.(aptly.LocalPackagePool).Link(sourcePath, filepath.Join(poolPath, baseName))
	}

	return err
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
