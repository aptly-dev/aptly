// Package aptly provides common infrastructure that doesn't depend directly on
// Debian or CentOS
package aptly

import (
	"github.com/smira/aptly/utils"
	"io"
	"os"
)

// PackagePool is asbtraction of package pool storage.
//
// PackagePool stores all the package files, deduplicating them.
type PackagePool interface {
	Path(filename string, hashMD5 string) (string, error)
	RelativePath(filename string, hashMD5 string) (string, error)
	FilepathList(progress Progress) ([]string, error)
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

// Progress is a progress displaying entity, it allows progress bars & simple prints
type Progress interface {
	// Writer interface to support progress bar ticking
	io.Writer
	// Start makes progress start its work
	Start()
	// Shutdown shuts down progress display
	Shutdown()
	// InitBar starts progressbar for count bytes or count items
	InitBar(count int64, isBytes bool)
	// ShutdownBar stops progress bar and hides it
	ShutdownBar()
	// AddBar increments progress for progress bar
	AddBar(count int)
	// Printf does printf but in safe manner: not overwriting progress bar
	Printf(msg string, a ...interface{})
}

// Downloader is parallel HTTP fetcher
type Downloader interface {
	Download(url string, destination string, result chan<- error)
	DownloadWithChecksum(url string, destination string, result chan<- error, expected utils.ChecksumInfo, ignoreMismatch bool)
	Pause()
	Resume()
	Shutdown()
}
