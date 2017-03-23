// Package aptly provides common infrastructure that doesn't depend directly on
// Debian or CentOS
package aptly

import (
	"io"

	"github.com/smira/aptly/utils"
)

// PackagePool is asbtraction of package pool storage.
//
// PackagePool stores all the package files, deduplicating them.
type PackagePool interface {
	// Path returns full path to package file in pool given any name and hash of file contents
	Path(filename string, hashMD5 string) (string, error)
	// RelativePath returns path relative to pool's root for package files given MD5 and original filename
	RelativePath(filename string, hashMD5 string) (string, error)
	// FilepathList returns file paths of all the files in the pool
	FilepathList(progress Progress) ([]string, error)
	// Remove deletes file in package pool returns its size
	Remove(path string) (size int64, err error)
	// Import copies file into package pool
	Import(path string, hashMD5 string) error
}

// PublishedStorage is abstraction of filesystem storing all published repositories
type PublishedStorage interface {
	// MkDir creates directory recursively under public path
	MkDir(path string) error
	// PutFile puts file into published storage at specified path
	PutFile(path string, sourceFilename string) error
	// RemoveDirs removes directory structure under public path
	RemoveDirs(path string, progress Progress) error
	// Remove removes single file under public path
	Remove(path string) error
	// LinkFromPool links package file from pool to dist's pool location
	LinkFromPool(publishedDirectory string, sourcePool PackagePool, sourcePath, sourceMD5 string, force bool, symlinks bool) error
	// Filelist returns list of files under prefix
	Filelist(prefix string) ([]string, error)
	// RenameFile renames (moves) file
	RenameFile(oldName, newName string) error
}

// LocalPublishedStorage is published storage on local filesystem
type LocalPublishedStorage interface {
	// PublicPath returns root of public part
	PublicPath() string
}

// PublishedStorageProvider is a thing that returns PublishedStorage by name
type PublishedStorageProvider interface {
	// GetPublishedStorage returns PublishedStorage by name
	GetPublishedStorage(name string) PublishedStorage
}

// Progress is a progress displaying entity, it allows progress bars & simple prints
type Progress interface {
	// Writer interface to support progress bar ticking
	io.Writer
	// Start makes progress start its work
	Start()
	// Shutdown shuts down progress display
	Shutdown()
	// Flush returns when all queued messages are sent
	Flush()
	// InitBar starts progressbar for count bytes or count items
	InitBar(count int64, isBytes bool)
	// ShutdownBar stops progress bar and hides it
	ShutdownBar()
	// AddBar increments progress for progress bar
	AddBar(count int)
	// SetBar sets current position for progress bar
	SetBar(count int)
	// Printf does printf but in safe manner: not overwriting progress bar
	Printf(msg string, a ...interface{})
	// ColoredPrintf does printf in colored way + newline
	ColoredPrintf(msg string, a ...interface{})
}

// Downloader is parallel HTTP fetcher
type Downloader interface {
	// Download starts new download task
	Download(url string, destination string, result chan<- error)
	// DownloadWithChecksum starts new download task with checksum verification
	DownloadWithChecksum(url string, destination string, result chan<- error, expected utils.ChecksumInfo, ignoreMismatch bool, maxTries int)
	// Pause pauses task processing
	Pause()
	// Resume resumes task processing
	Resume()
	// Shutdown stops downloader after current tasks are finished,
	// but doesn't process rest of queue
	Shutdown()
	// Abort stops downloader without waiting for shutdown
	Abort()
	// GetProgress returns Progress object
	GetProgress() Progress
}
