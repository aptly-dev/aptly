// Package aptly provides common infrastructure that doesn't depend directly on
// Debian or CentOS
package aptly

import (
	"context"
	"io"
	"os"

	"github.com/aptly-dev/aptly/database"
	"github.com/aptly-dev/aptly/utils"
)

// ReadSeekerCloser = ReadSeeker + Closer
type ReadSeekerCloser interface {
	io.ReadSeeker
	io.Closer
}

// PackagePool is asbtraction of package pool storage.
//
// PackagePool stores all the package files, deduplicating them.
type PackagePool interface {
	// Verify checks whether file exists in the pool and fills back checksum info
	//
	// if poolPath is empty, poolPath is generated automatically based on checksum info (if available)
	// in any case, if function returns true, it also fills back checksums with complete information about the file in the pool
	Verify(poolPath, basename string, checksums *utils.ChecksumInfo, checksumStorage ChecksumStorage) (string, bool, error)
	// Import copies file into package pool
	//
	// - srcPath is full path to source file as it is now
	// - basename is desired human-readable name (canonical filename)
	// - checksums are used to calculate file placement
	// - move indicates whether srcPath can be removed
	Import(srcPath, basename string, checksums *utils.ChecksumInfo, move bool, storage ChecksumStorage) (path string, err error)
	// LegacyPath returns legacy (pre 1.1) path to package file (relative to root)
	LegacyPath(filename string, checksums *utils.ChecksumInfo) (string, error)
	// Stat returns Unix stat(2) info
	Stat(path string) (os.FileInfo, error)
	// Open returns ReadSeekerCloser to access the file
	Open(path string) (ReadSeekerCloser, error)
	// FilepathList returns file paths of all the files in the pool
	FilepathList(progress Progress) ([]string, error)
	// Remove deletes file in package pool returns its size
	Remove(path string) (size int64, err error)
}

// LocalPackagePool is implemented by PackagePools residing on the same filesystem
type LocalPackagePool interface {
	// GenerateTempPath generates temporary path for download (which is fast to import into package pool later on)
	GenerateTempPath(filename string) (string, error)
	// Link generates hardlink to destination path
	Link(path, dstPath string) error
	// Symlink generates symlink to destination path
	Symlink(path, dstPath string) error
	// FullPath generates full path to the file in pool
	//
	// Please use with care: it's not supposed to be used to access files
	FullPath(path string) string
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
	LinkFromPool(publishedDirectory, fileName string, sourcePool PackagePool, sourcePath string, sourceChecksums utils.ChecksumInfo, force bool) error
	// Filelist returns list of files under prefix
	Filelist(prefix string) ([]string, error)
	// RenameFile renames (moves) file
	RenameFile(oldName, newName string) error
	// SymLink creates a symbolic link, which can be read with ReadLink
	SymLink(src string, dst string) error
	// HardLink creates a hardlink of a file
	HardLink(src string, dst string) error
	// FileExists returns true if path exists
	FileExists(path string) (bool, error)
	// ReadLink returns the symbolic link pointed to by path
	ReadLink(path string) (string, error)
}

// FileSystemPublishedStorage is published storage on filesystem
type FileSystemPublishedStorage interface {
	// PublicPath returns root of public part
	PublicPath() string
}

// PublishedStorageProvider is a thing that returns PublishedStorage by name
type PublishedStorageProvider interface {
	// GetPublishedStorage returns PublishedStorage by name
	GetPublishedStorage(name string) PublishedStorage
}

// BarType used to differentiate between different progress bars
type BarType int

const (
	// BarGeneralBuildPackageList identifies bar for building package list
	BarGeneralBuildPackageList BarType = iota
	// BarGeneralVerifyDependencies identifies bar for verifying dependencies
	BarGeneralVerifyDependencies
	// BarGeneralBuildFileList identifies bar for building file list
	BarGeneralBuildFileList
	// BarCleanupBuildList identifies bar for building list to cleanup
	BarCleanupBuildList
	// BarCleanupDeleteUnreferencedFiles identifies bar for deleting unreferenced files
	BarCleanupDeleteUnreferencedFiles
	// BarMirrorUpdateDownloadIndexes identifies bar for downloading index files
	BarMirrorUpdateDownloadIndexes
	// BarMirrorUpdateDownloadPackages identifies bar for downloading packages
	BarMirrorUpdateDownloadPackages
	// BarMirrorUpdateBuildPackageList identifies bar for building package list of downloaded files
	BarMirrorUpdateBuildPackageList
	// BarMirrorUpdateImportFiles identifies bar for importing package files
	BarMirrorUpdateImportFiles
	// BarMirrorUpdateFinalizeDownload identifies bar for finalizing downloads
	BarMirrorUpdateFinalizeDownload
	// BarPublishGeneratePackageFiles identifies bar for generating package files to publish
	BarPublishGeneratePackageFiles
	// BarPublishFinalizeIndexes identifies bar for finalizing index files
	BarPublishFinalizeIndexes
)

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
	InitBar(count int64, isBytes bool, barType BarType)
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
	// PrintfStdErr does printf but in safe manner to stderr
	PrintfStdErr(msg string, a ...interface{})
}

// Downloader is parallel HTTP fetcher
type Downloader interface {
	// Download starts new download task
	Download(ctx context.Context, url string, destination string) error
	// DownloadWithChecksum starts new download task with checksum verification
	DownloadWithChecksum(ctx context.Context, url string, destination string, expected *utils.ChecksumInfo, ignoreMismatch bool) error
	// GetProgress returns Progress object
	GetProgress() Progress
	// GetLength returns size by heading object with url
	GetLength(ctx context.Context, url string) (int64, error)
}

// ChecksumStorageProvider creates ChecksumStorage based on DB
type ChecksumStorageProvider func(db database.ReaderWriter) ChecksumStorage

// ChecksumStorage is stores checksums in some (persistent) storage
type ChecksumStorage interface {
	// Get finds checksums in DB by path
	Get(path string) (*utils.ChecksumInfo, error)
	// Update adds or updates information about checksum in DB
	Update(path string, c *utils.ChecksumInfo) error
}
