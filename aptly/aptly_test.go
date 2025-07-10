package aptly

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/aptly-dev/aptly/utils"
	. "gopkg.in/check.v1"
)

// Launch gocheck tests
func Test(t *testing.T) {
	TestingT(t)
}

type AptlySuite struct{}

var _ = Suite(&AptlySuite{})

// Mock implementations for testing interfaces

type MockPackagePool struct {
	verifyFunc       func(string, string, *utils.ChecksumInfo, ChecksumStorage) (string, bool, error)
	importFunc       func(string, string, *utils.ChecksumInfo, bool, ChecksumStorage) (string, error)
	legacyPathFunc   func(string, *utils.ChecksumInfo) (string, error)
	sizeFunc         func(string) (int64, error)
	openFunc         func(string) (ReadSeekerCloser, error)
	filepathListFunc func(Progress) ([]string, error)
	removeFunc       func(string) (int64, error)
}

func (m *MockPackagePool) Verify(poolPath, basename string, checksums *utils.ChecksumInfo, storage ChecksumStorage) (string, bool, error) {
	if m.verifyFunc != nil {
		return m.verifyFunc(poolPath, basename, checksums, storage)
	}
	return poolPath, true, nil
}

func (m *MockPackagePool) Import(srcPath, basename string, checksums *utils.ChecksumInfo, move bool, storage ChecksumStorage) (string, error) {
	if m.importFunc != nil {
		return m.importFunc(srcPath, basename, checksums, move, storage)
	}
	return "imported/path/" + basename, nil
}

func (m *MockPackagePool) LegacyPath(filename string, checksums *utils.ChecksumInfo) (string, error) {
	if m.legacyPathFunc != nil {
		return m.legacyPathFunc(filename, checksums)
	}
	return "legacy/" + filename, nil
}

func (m *MockPackagePool) Size(path string) (int64, error) {
	if m.sizeFunc != nil {
		return m.sizeFunc(path)
	}
	return 1024, nil
}

func (m *MockPackagePool) Open(path string) (ReadSeekerCloser, error) {
	if m.openFunc != nil {
		return m.openFunc(path)
	}
	return &MockReadSeekerCloser{content: []byte("mock file content")}, nil
}

func (m *MockPackagePool) FilepathList(progress Progress) ([]string, error) {
	if m.filepathListFunc != nil {
		return m.filepathListFunc(progress)
	}
	return []string{"file1.deb", "file2.deb"}, nil
}

func (m *MockPackagePool) Remove(path string) (int64, error) {
	if m.removeFunc != nil {
		return m.removeFunc(path)
	}
	return 1024, nil
}

type MockReadSeekerCloser struct {
	content []byte
	pos     int64
	closed  bool
}

func (m *MockReadSeekerCloser) Read(p []byte) (int, error) {
	if m.closed {
		return 0, errors.New("closed")
	}
	if m.pos >= int64(len(m.content)) {
		return 0, io.EOF
	}
	n := copy(p, m.content[m.pos:])
	m.pos += int64(n)
	return n, nil
}

func (m *MockReadSeekerCloser) Seek(offset int64, whence int) (int64, error) {
	if m.closed {
		return 0, errors.New("closed")
	}
	switch whence {
	case io.SeekStart:
		m.pos = offset
	case io.SeekCurrent:
		m.pos += offset
	case io.SeekEnd:
		m.pos = int64(len(m.content)) + offset
	}
	if m.pos < 0 {
		m.pos = 0
	}
	if m.pos > int64(len(m.content)) {
		m.pos = int64(len(m.content))
	}
	return m.pos, nil
}

func (m *MockReadSeekerCloser) Close() error {
	m.closed = true
	return nil
}

type MockPublishedStorage struct {
	mkDirFunc        func(string) error
	putFileFunc      func(string, string) error
	removeDirsFunc   func(string, Progress) error
	removeFunc       func(string) error
	linkFromPoolFunc func(string, string, string, PackagePool, string, utils.ChecksumInfo, bool) error
	filelistFunc     func(string) ([]string, error)
	renameFileFunc   func(string, string) error
	symLinkFunc      func(string, string) error
	hardLinkFunc     func(string, string) error
	fileExistsFunc   func(string) (bool, error)
	readLinkFunc     func(string) (string, error)
}

func (m *MockPublishedStorage) MkDir(path string) error {
	if m.mkDirFunc != nil {
		return m.mkDirFunc(path)
	}
	return nil
}

func (m *MockPublishedStorage) PutFile(path, sourceFilename string) error {
	if m.putFileFunc != nil {
		return m.putFileFunc(path, sourceFilename)
	}
	return nil
}

func (m *MockPublishedStorage) RemoveDirs(path string, progress Progress) error {
	if m.removeDirsFunc != nil {
		return m.removeDirsFunc(path, progress)
	}
	return nil
}

func (m *MockPublishedStorage) Remove(path string) error {
	if m.removeFunc != nil {
		return m.removeFunc(path)
	}
	return nil
}

func (m *MockPublishedStorage) LinkFromPool(publishedPrefix, publishedRelPath, fileName string, sourcePool PackagePool, sourcePath string, sourceChecksums utils.ChecksumInfo, force bool) error {
	if m.linkFromPoolFunc != nil {
		return m.linkFromPoolFunc(publishedPrefix, publishedRelPath, fileName, sourcePool, sourcePath, sourceChecksums, force)
	}
	return nil
}

func (m *MockPublishedStorage) Filelist(prefix string) ([]string, error) {
	if m.filelistFunc != nil {
		return m.filelistFunc(prefix)
	}
	return []string{"file1", "file2"}, nil
}

func (m *MockPublishedStorage) RenameFile(oldName, newName string) error {
	if m.renameFileFunc != nil {
		return m.renameFileFunc(oldName, newName)
	}
	return nil
}

func (m *MockPublishedStorage) SymLink(src, dst string) error {
	if m.symLinkFunc != nil {
		return m.symLinkFunc(src, dst)
	}
	return nil
}

func (m *MockPublishedStorage) HardLink(src, dst string) error {
	if m.hardLinkFunc != nil {
		return m.hardLinkFunc(src, dst)
	}
	return nil
}

func (m *MockPublishedStorage) FileExists(path string) (bool, error) {
	if m.fileExistsFunc != nil {
		return m.fileExistsFunc(path)
	}
	return true, nil
}

func (m *MockPublishedStorage) ReadLink(path string) (string, error) {
	if m.readLinkFunc != nil {
		return m.readLinkFunc(path)
	}
	return "target", nil
}

type MockProgress struct {
	buffer      bytes.Buffer
	started     bool
	barStarted  bool
	barProgress int
}

func (m *MockProgress) Write(p []byte) (n int, err error) {
	return m.buffer.Write(p)
}

func (m *MockProgress) Start() {
	m.started = true
}

func (m *MockProgress) Shutdown() {
	m.started = false
}

func (m *MockProgress) Flush() {
	// Nothing to do in mock
}

func (m *MockProgress) InitBar(count int64, isBytes bool, barType BarType) {
	m.barStarted = true
}

func (m *MockProgress) ShutdownBar() {
	m.barStarted = false
}

func (m *MockProgress) AddBar(count int) {
	m.barProgress += count
}

func (m *MockProgress) SetBar(count int) {
	m.barProgress = count
}

func (m *MockProgress) Printf(msg string, a ...interface{}) {
	fmt.Fprintf(&m.buffer, msg, a...)
}

func (m *MockProgress) ColoredPrintf(msg string, a ...interface{}) {
	// Strip color codes for testing
	cleanMsg := strings.ReplaceAll(msg, "@r", "")
	cleanMsg = strings.ReplaceAll(cleanMsg, "@g", "")
	cleanMsg = strings.ReplaceAll(cleanMsg, "@y", "")
	cleanMsg = strings.ReplaceAll(cleanMsg, "@!", "")
	cleanMsg = strings.ReplaceAll(cleanMsg, "@|", "")
	fmt.Fprintf(&m.buffer, cleanMsg, a...)
}

func (m *MockProgress) PrintfStdErr(msg string, a ...interface{}) {
	fmt.Fprintf(&m.buffer, "[STDERR] "+msg, a...)
}

type MockDownloader struct {
	downloadFunc             func(context.Context, string, string) error
	downloadWithChecksumFunc func(context.Context, string, string, *utils.ChecksumInfo, bool) error
	progress                 Progress
	getLengthFunc            func(context.Context, string) (int64, error)
}

func (m *MockDownloader) Download(ctx context.Context, url, destination string) error {
	if m.downloadFunc != nil {
		return m.downloadFunc(ctx, url, destination)
	}
	return nil
}

func (m *MockDownloader) DownloadWithChecksum(ctx context.Context, url, destination string, expected *utils.ChecksumInfo, ignoreMismatch bool) error {
	if m.downloadWithChecksumFunc != nil {
		return m.downloadWithChecksumFunc(ctx, url, destination, expected, ignoreMismatch)
	}
	return nil
}

func (m *MockDownloader) GetProgress() Progress {
	if m.progress != nil {
		return m.progress
	}
	return &MockProgress{}
}

func (m *MockDownloader) GetLength(ctx context.Context, url string) (int64, error) {
	if m.getLengthFunc != nil {
		return m.getLengthFunc(ctx, url)
	}
	return 1024, nil
}

type MockChecksumStorage struct {
	getFunc    func(string) (*utils.ChecksumInfo, error)
	updateFunc func(string, *utils.ChecksumInfo) error
}

func (m *MockChecksumStorage) Get(path string) (*utils.ChecksumInfo, error) {
	if m.getFunc != nil {
		return m.getFunc(path)
	}
	return &utils.ChecksumInfo{}, nil
}

func (m *MockChecksumStorage) Update(path string, c *utils.ChecksumInfo) error {
	if m.updateFunc != nil {
		return m.updateFunc(path, c)
	}
	return nil
}

// Test interfaces and their basic functionality

func (s *AptlySuite) TestPackagePoolInterface(c *C) {
	// Test PackagePool interface with mock implementation
	var pool PackagePool = &MockPackagePool{}
	
	checksums := &utils.ChecksumInfo{}
	mockStorage := &MockChecksumStorage{}
	
	// Test Verify
	path, exists, err := pool.Verify("test/path", "package.deb", checksums, mockStorage)
	c.Check(err, IsNil)
	c.Check(exists, Equals, true)
	c.Check(path, Equals, "test/path")
	
	// Test Import
	importedPath, err := pool.Import("/src/package.deb", "package.deb", checksums, false, mockStorage)
	c.Check(err, IsNil)
	c.Check(importedPath, Equals, "imported/path/package.deb")
	
	// Test LegacyPath
	legacyPath, err := pool.LegacyPath("package.deb", checksums)
	c.Check(err, IsNil)
	c.Check(legacyPath, Equals, "legacy/package.deb")
	
	// Test Size
	size, err := pool.Size("test/path")
	c.Check(err, IsNil)
	c.Check(size, Equals, int64(1024))
	
	// Test Open
	reader, err := pool.Open("test/path")
	c.Check(err, IsNil)
	c.Check(reader, NotNil)
	reader.Close()
	
	// Test FilepathList
	mockProgress := &MockProgress{}
	files, err := pool.FilepathList(mockProgress)
	c.Check(err, IsNil)
	c.Check(len(files), Equals, 2)
	c.Check(files[0], Equals, "file1.deb")
	
	// Test Remove
	removedSize, err := pool.Remove("test/path")
	c.Check(err, IsNil)
	c.Check(removedSize, Equals, int64(1024))
}

func (s *AptlySuite) TestPublishedStorageInterface(c *C) {
	// Test PublishedStorage interface with mock implementation
	var storage PublishedStorage = &MockPublishedStorage{}
	
	// Test MkDir
	err := storage.MkDir("test/dir")
	c.Check(err, IsNil)
	
	// Test PutFile
	err = storage.PutFile("dest/path", "source/file")
	c.Check(err, IsNil)
	
	// Test RemoveDirs
	mockProgress := &MockProgress{}
	err = storage.RemoveDirs("test/dir", mockProgress)
	c.Check(err, IsNil)
	
	// Test Remove
	err = storage.Remove("test/file")
	c.Check(err, IsNil)
	
	// Test LinkFromPool
	mockPool := &MockPackagePool{}
	checksums := utils.ChecksumInfo{}
	err = storage.LinkFromPool("prefix", "rel/path", "file.deb", mockPool, "pool/path", checksums, false)
	c.Check(err, IsNil)
	
	// Test Filelist
	files, err := storage.Filelist("prefix")
	c.Check(err, IsNil)
	c.Check(len(files), Equals, 2)
	
	// Test RenameFile
	err = storage.RenameFile("old", "new")
	c.Check(err, IsNil)
	
	// Test SymLink
	err = storage.SymLink("src", "dst")
	c.Check(err, IsNil)
	
	// Test HardLink
	err = storage.HardLink("src", "dst")
	c.Check(err, IsNil)
	
	// Test FileExists
	exists, err := storage.FileExists("test/file")
	c.Check(err, IsNil)
	c.Check(exists, Equals, true)
	
	// Test ReadLink
	target, err := storage.ReadLink("link")
	c.Check(err, IsNil)
	c.Check(target, Equals, "target")
}

func (s *AptlySuite) TestProgressInterface(c *C) {
	// Test Progress interface with mock implementation
	var progress Progress = &MockProgress{}
	
	// Test Start/Shutdown
	progress.Start()
	progress.Shutdown()
	
	// Test Write
	n, err := progress.Write([]byte("test"))
	c.Check(err, IsNil)
	c.Check(n, Equals, 4)
	
	// Test progress bar functions
	progress.InitBar(100, false, BarGeneralBuildPackageList)
	progress.AddBar(10)
	progress.SetBar(50)
	progress.ShutdownBar()
	
	// Test Printf functions
	progress.Printf("test %s", "message")
	progress.ColoredPrintf("colored %s", "message")
	progress.PrintfStdErr("error %s", "message")
	
	// Test Flush
	progress.Flush()
}

func (s *AptlySuite) TestDownloaderInterface(c *C) {
	// Test Downloader interface with mock implementation
	var downloader Downloader = &MockDownloader{}
	
	ctx := context.Background()
	
	// Test Download
	err := downloader.Download(ctx, "http://example.com/file", "/tmp/dest")
	c.Check(err, IsNil)
	
	// Test DownloadWithChecksum
	checksums := &utils.ChecksumInfo{}
	err = downloader.DownloadWithChecksum(ctx, "http://example.com/file", "/tmp/dest", checksums, false)
	c.Check(err, IsNil)
	
	// Test GetProgress
	progress := downloader.GetProgress()
	c.Check(progress, NotNil)
	
	// Test GetLength
	length, err := downloader.GetLength(ctx, "http://example.com/file")
	c.Check(err, IsNil)
	c.Check(length, Equals, int64(1024))
}

func (s *AptlySuite) TestChecksumStorageInterface(c *C) {
	// Test ChecksumStorage interface with mock implementation
	var storage ChecksumStorage = &MockChecksumStorage{}
	
	// Test Get
	checksums, err := storage.Get("test/path")
	c.Check(err, IsNil)
	c.Check(checksums, NotNil)
	
	// Test Update
	newChecksums := &utils.ChecksumInfo{}
	err = storage.Update("test/path", newChecksums)
	c.Check(err, IsNil)
}

func (s *AptlySuite) TestConsoleResultReporter(c *C) {
	// Test ConsoleResultReporter implementation
	mockProgress := &MockProgress{}
	reporter := &ConsoleResultReporter{Progress: mockProgress}
	
	// Test interface compliance
	var _ ResultReporter = reporter
	
	// Test Warning
	reporter.Warning("test warning %s", "message")
	output := mockProgress.buffer.String()
	c.Check(strings.Contains(output, "test warning message"), Equals, true)
	c.Check(strings.Contains(output, "[!]"), Equals, true)
	
	// Reset buffer
	mockProgress.buffer.Reset()
	
	// Test Removed
	reporter.Removed("removed %s", "item")
	output = mockProgress.buffer.String()
	c.Check(strings.Contains(output, "removed item"), Equals, true)
	c.Check(strings.Contains(output, "[-]"), Equals, true)
	
	// Reset buffer
	mockProgress.buffer.Reset()
	
	// Test Added
	reporter.Added("added %s", "item")
	output = mockProgress.buffer.String()
	c.Check(strings.Contains(output, "added item"), Equals, true)
	c.Check(strings.Contains(output, "[+]"), Equals, true)
}

func (s *AptlySuite) TestRecordingResultReporter(c *C) {
	// Test RecordingResultReporter implementation
	reporter := &RecordingResultReporter{
		Warnings:     []string{},
		AddedLines:   []string{},
		RemovedLines: []string{},
	}
	
	// Test interface compliance
	var _ ResultReporter = reporter
	
	// Test Warning
	reporter.Warning("test warning %s", "message")
	c.Check(len(reporter.Warnings), Equals, 1)
	c.Check(reporter.Warnings[0], Equals, "test warning message")
	
	// Test Removed
	reporter.Removed("removed %s", "item")
	c.Check(len(reporter.RemovedLines), Equals, 1)
	c.Check(reporter.RemovedLines[0], Equals, "removed item")
	
	// Test Added
	reporter.Added("added %s", "item")
	c.Check(len(reporter.AddedLines), Equals, 1)
	c.Check(reporter.AddedLines[0], Equals, "added item")
	
	// Test multiple entries
	reporter.Warning("second warning")
	reporter.Added("second addition")
	c.Check(len(reporter.Warnings), Equals, 2)
	c.Check(len(reporter.AddedLines), Equals, 2)
	c.Check(reporter.Warnings[1], Equals, "second warning")
	c.Check(reporter.AddedLines[1], Equals, "second addition")
}

func (s *AptlySuite) TestReadSeekerCloserInterface(c *C) {
	// Test ReadSeekerCloser interface with mock implementation
	var rsc ReadSeekerCloser = &MockReadSeekerCloser{
		content: []byte("Hello, World!"),
	}
	
	// Test Read
	buf := make([]byte, 5)
	n, err := rsc.Read(buf)
	c.Check(err, IsNil)
	c.Check(n, Equals, 5)
	c.Check(string(buf), Equals, "Hello")
	
	// Test Seek
	pos, err := rsc.Seek(0, io.SeekStart)
	c.Check(err, IsNil)
	c.Check(pos, Equals, int64(0))
	
	// Test Read again from beginning
	n, err = rsc.Read(buf)
	c.Check(err, IsNil)
	c.Check(string(buf), Equals, "Hello")
	
	// Test Seek to end
	pos, err = rsc.Seek(-6, io.SeekEnd)
	c.Check(err, IsNil)
	c.Check(pos, Equals, int64(7))
	
	// Test Read from near end
	buf = make([]byte, 10)
	n, err = rsc.Read(buf)
	c.Check(err, IsNil)
	c.Check(string(buf[:n]), Equals, "World!")
	
	// Test Close
	err = rsc.Close()
	c.Check(err, IsNil)
	
	// Test Read after close (should error)
	_, err = rsc.Read(buf)
	c.Check(err, NotNil)
}

func (s *AptlySuite) TestBarTypeConstants(c *C) {
	// Test BarType constants are defined and different
	barTypes := []BarType{
		BarGeneralBuildPackageList,
		BarGeneralVerifyDependencies,
		BarGeneralBuildFileList,
		BarCleanupBuildList,
		BarCleanupDeleteUnreferencedFiles,
		BarMirrorUpdateDownloadIndexes,
		BarMirrorUpdateDownloadPackages,
		BarMirrorUpdateBuildPackageList,
		BarMirrorUpdateImportFiles,
		BarMirrorUpdateFinalizeDownload,
		BarPublishGeneratePackageFiles,
		BarPublishFinalizeIndexes,
	}
	
	// Check that all constants are different
	seen := make(map[BarType]bool)
	for _, barType := range barTypes {
		c.Check(seen[barType], Equals, false, Commentf("Duplicate BarType: %v", barType))
		seen[barType] = true
	}
	
	// Check that they are sequential integers starting from 0
	for i, barType := range barTypes {
		c.Check(int(barType), Equals, i, Commentf("BarType not sequential: %v", barType))
	}
}

func (s *AptlySuite) TestErrorHandling(c *C) {
	// Test error handling in mock implementations
	
	// Test PackagePool with errors
	pool := &MockPackagePool{
		verifyFunc: func(string, string, *utils.ChecksumInfo, ChecksumStorage) (string, bool, error) {
			return "", false, errors.New("verify error")
		},
		importFunc: func(string, string, *utils.ChecksumInfo, bool, ChecksumStorage) (string, error) {
			return "", errors.New("import error")
		},
	}
	
	_, _, err := pool.Verify("", "", nil, nil)
	c.Check(err, NotNil)
	c.Check(err.Error(), Equals, "verify error")
	
	_, err = pool.Import("", "", nil, false, nil)
	c.Check(err, NotNil)
	c.Check(err.Error(), Equals, "import error")
	
	// Test PublishedStorage with errors
	storage := &MockPublishedStorage{
		mkDirFunc: func(string) error {
			return errors.New("mkdir error")
		},
		fileExistsFunc: func(string) (bool, error) {
			return false, errors.New("file exists error")
		},
	}
	
	err = storage.MkDir("test")
	c.Check(err, NotNil)
	c.Check(err.Error(), Equals, "mkdir error")
	
	_, err = storage.FileExists("test")
	c.Check(err, NotNil)
	c.Check(err.Error(), Equals, "file exists error")
}

func (s *AptlySuite) TestInterfaceCompatibility(c *C) {
	// Test that our mocks properly implement the interfaces
	
	// PackagePool interface
	var _ PackagePool = &MockPackagePool{}
	
	// PublishedStorage interface 
	var _ PublishedStorage = &MockPublishedStorage{}
	
	// Progress interface
	var _ Progress = &MockProgress{}
	
	// Downloader interface
	var _ Downloader = &MockDownloader{}
	
	// ChecksumStorage interface
	var _ ChecksumStorage = &MockChecksumStorage{}
	
	// ReadSeekerCloser interface
	var _ ReadSeekerCloser = &MockReadSeekerCloser{}
	
	// ResultReporter interface
	var _ ResultReporter = &ConsoleResultReporter{}
	var _ ResultReporter = &RecordingResultReporter{}
	
	// Test that the interface checks pass
	c.Check(true, Equals, true)
}