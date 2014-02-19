// Package aptly provides common infrastructure that doesn't depend directly on
// Debian or CentOS
package aptly

import (
	"github.com/smira/aptly/utils"
	"os"
)

// PackagePool is asbtraction of package pool storage.
//
// PackagePool stores all the package files, deduplicating them.
type PackagePool interface {
	Path(filename string, hashMD5 string) (string, error)
	RelativePath(filename string, hashMD5 string) (string, error)
	FilepathList(progress *utils.Progress) ([]string, error)
	Remove(path string) (size int64, err error)
}

// PublishedStorage is abstraction of filesystem storing all published repositories
type PublishedStorage interface {
	PublicPath() string
	MkDir(path string) error
	CreateFile(path string) (*os.File, error)
	RemoveDirs(path string) error
	LinkFromPool(prefix string, component string, poolDirectory string, sourcePool PackagePool, sourcePath string) (string, error)
	ChecksumsForFile(path string) (utils.ChecksumInfo, error)
}
