package deb

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/utils"
	. "gopkg.in/check.v1"
)

type IndexFilesSuite struct {
	tempDir          string
	publishedStorage *MockPublishedStorage
	indexFiles       *indexFiles
}

var _ = Suite(&IndexFilesSuite{})

func (s *IndexFilesSuite) SetUpTest(c *C) {
	s.tempDir = c.MkDir()
	s.publishedStorage = &MockPublishedStorage{
		files:    make(map[string]string),
		dirs:     make(map[string]bool),
		links:    make(map[string]string),
		symlinks: make(map[string]string),
	}
	s.indexFiles = newIndexFiles(s.publishedStorage, "dists/test", s.tempDir, "", false, false)
}

func (s *IndexFilesSuite) TestNewIndexFiles(c *C) {
	// Test creation of indexFiles struct
	basePath := "dists/testing"
	tempDir := "/tmp/test"
	suffix := ".new"
	acquireByHash := true
	skipBz2 := true

	files := newIndexFiles(s.publishedStorage, basePath, tempDir, suffix, acquireByHash, skipBz2)

	c.Check(files.publishedStorage, Equals, s.publishedStorage)
	c.Check(files.basePath, Equals, basePath)
	c.Check(files.tempDir, Equals, tempDir)
	c.Check(files.suffix, Equals, suffix)
	c.Check(files.acquireByHash, Equals, acquireByHash)
	c.Check(files.skipBz2, Equals, skipBz2)
	c.Check(files.renameMap, NotNil)
	c.Check(files.generatedFiles, NotNil)
	c.Check(files.indexes, NotNil)
	c.Check(len(files.renameMap), Equals, 0)
	c.Check(len(files.generatedFiles), Equals, 0)
	c.Check(len(files.indexes), Equals, 0)
}

func (s *IndexFilesSuite) TestIndexFileBufWriter(c *C) {
	// Test indexFile BufWriter creation
	file := &indexFile{
		parent:       s.indexFiles,
		relativePath: "main/binary-amd64/Packages",
	}

	// First call should create the writer
	writer, err := file.BufWriter()
	c.Check(err, IsNil)
	c.Check(writer, NotNil)
	c.Check(file.w, Equals, writer)
	c.Check(file.tempFile, NotNil)
	c.Check(file.tempFilename, Matches, ".*main_binary-amd64_Packages")

	// Second call should return the same writer
	writer2, err := file.BufWriter()
	c.Check(err, IsNil)
	c.Check(writer2, Equals, writer)

	// Clean up
	file.tempFile.Close()
}

func (s *IndexFilesSuite) TestIndexFileBufWriterError(c *C) {
	// Test BufWriter creation with invalid temp directory
	invalidFiles := newIndexFiles(s.publishedStorage, "dists/test", "/invalid/path", "", false, false)
	file := &indexFile{
		parent:       invalidFiles,
		relativePath: "main/binary-amd64/Packages",
	}

	_, err := file.BufWriter()
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to create temporary index file.*")
}

func (s *IndexFilesSuite) TestIndexFileFinalize(c *C) {
	// Test basic finalization of index file
	file := &indexFile{
		parent:        s.indexFiles,
		relativePath:  "main/binary-amd64/Packages",
		compressable:  false,
		detachedSign:  false,
		clearSign:     false,
		acquireByHash: false,
	}

	// Write some content to the file
	writer, err := file.BufWriter()
	c.Check(err, IsNil)
	writer.WriteString("Package: test-package\nVersion: 1.0\n\n")

	err = file.Finalize(nil)
	c.Check(err, IsNil)

	// Check that file was published
	c.Check(s.publishedStorage.files["dists/test/main/binary-amd64/Packages"], NotNil)
	c.Check(s.publishedStorage.dirs["dists/test/main/binary-amd64"], Equals, true)

	// Check that checksums were generated
	c.Check(s.indexFiles.generatedFiles["main/binary-amd64/Packages"], NotNil)
}

func (s *IndexFilesSuite) TestIndexFileFinalizeCompressable(c *C) {
	// Test finalization with compression
	file := &indexFile{
		parent:        s.indexFiles,
		relativePath:  "main/binary-amd64/Packages",
		compressable:  true,
		detachedSign:  false,
		clearSign:     false,
		acquireByHash: false,
		onlyGzip:      false,
	}

	// Write content and finalize
	writer, err := file.BufWriter()
	c.Check(err, IsNil)
	writer.WriteString("Package: test-package\nVersion: 1.0\n\n")

	err = file.Finalize(nil)
	c.Check(err, IsNil)

	// Check that compressed files were published
	c.Check(s.publishedStorage.files["dists/test/main/binary-amd64/Packages"], NotNil)
	c.Check(s.publishedStorage.files["dists/test/main/binary-amd64/Packages.gz"], NotNil)

	// With skipBz2 = false, should also have .bz2
	if !s.indexFiles.skipBz2 {
		c.Check(s.publishedStorage.files["dists/test/main/binary-amd64/Packages.bz2"], NotNil)
	}

	// Check checksums for all variants
	c.Check(s.indexFiles.generatedFiles["main/binary-amd64/Packages"], NotNil)
	c.Check(s.indexFiles.generatedFiles["main/binary-amd64/Packages.gz"], NotNil)
}

func (s *IndexFilesSuite) TestIndexFileFinalizeOnlyGzip(c *C) {
	// Test finalization with only gzip compression
	file := &indexFile{
		parent:        s.indexFiles,
		relativePath:  "main/Contents-amd64",
		compressable:  true,
		onlyGzip:      true,
		detachedSign:  false,
		clearSign:     false,
		acquireByHash: false,
	}

	writer, err := file.BufWriter()
	c.Check(err, IsNil)
	writer.WriteString("some content data\n")

	err = file.Finalize(nil)
	c.Check(err, IsNil)

	// Should only have .gz file, not .bz2
	c.Check(s.publishedStorage.files["dists/test/main/Contents-amd64.gz"], NotNil)
	_, hasBz2 := s.publishedStorage.files["dists/test/main/Contents-amd64.bz2"]
	c.Check(hasBz2, Equals, false)

	// Checksums should include both uncompressed and compressed
	c.Check(s.indexFiles.generatedFiles["main/Contents-amd64"], NotNil)
	c.Check(s.indexFiles.generatedFiles["main/Contents-amd64.gz"], NotNil)
}

func (s *IndexFilesSuite) TestIndexFileFinalizeDiscardable(c *C) {
	// Test finalization of discardable file (should create empty file)
	file := &indexFile{
		parent:        s.indexFiles,
		relativePath:  "main/debian-installer/binary-amd64/Release",
		discardable:   true,
		compressable:  false,
		detachedSign:  false,
		clearSign:     false,
		acquireByHash: false,
	}

	// Don't write any content, just finalize
	err := file.Finalize(nil)
	c.Check(err, IsNil)

	// Should still create the file
	c.Check(s.publishedStorage.files["dists/test/main/debian-installer/binary-amd64/Release"], NotNil)
}

func (s *IndexFilesSuite) TestIndexFileFinalizeSigning(c *C) {
	// Test finalization with signing
	mockSigner := &MockSigner{}
	file := &indexFile{
		parent:        s.indexFiles,
		relativePath:  "Release",
		compressable:  false,
		detachedSign:  true,
		clearSign:     true,
		acquireByHash: false,
	}

	writer, err := file.BufWriter()
	c.Check(err, IsNil)
	writer.WriteString("Suite: test\nCodename: test\n")

	err = file.Finalize(mockSigner)
	c.Check(err, IsNil)

	// Check that signed files were created
	c.Check(s.publishedStorage.files["dists/test/Release"], NotNil)
	c.Check(s.publishedStorage.files["dists/test/Release.gpg"], NotNil)
	c.Check(s.publishedStorage.files["dists/test/InRelease"], NotNil)

	// Check that signer methods were called
	c.Check(mockSigner.DetachedSignCalled, Equals, true)
	c.Check(mockSigner.ClearSignCalled, Equals, true)
}

func (s *IndexFilesSuite) TestIndexFileFinalizeWithSuffix(c *C) {
	// Test finalization with suffix (for atomic updates)
	s.indexFiles.suffix = ".new"
	file := &indexFile{
		parent:        s.indexFiles,
		relativePath:  "main/binary-amd64/Packages",
		compressable:  false,
		detachedSign:  false,
		clearSign:     false,
		acquireByHash: false,
	}

	writer, err := file.BufWriter()
	c.Check(err, IsNil)
	writer.WriteString("Package: test\n")

	err = file.Finalize(nil)
	c.Check(err, IsNil)

	// Check that file was published with suffix
	c.Check(s.publishedStorage.files["dists/test/main/binary-amd64/Packages.new"], NotNil)

	// Check that rename mapping was created
	expectedTarget := "dists/test/main/binary-amd64/Packages"
	c.Check(s.indexFiles.renameMap["dists/test/main/binary-amd64/Packages.new"], Equals, expectedTarget)
}

func (s *IndexFilesSuite) TestPackageIndex(c *C) {
	// Test PackageIndex creation for binary packages
	file := s.indexFiles.PackageIndex("main", "amd64", false, false, "")
	c.Check(file, NotNil)
	c.Check(file.relativePath, Equals, "main/binary-amd64/Packages")
	c.Check(file.compressable, Equals, true)
	c.Check(file.discardable, Equals, false)
	c.Check(file.detachedSign, Equals, false)

	// Test that same call returns cached instance
	file2 := s.indexFiles.PackageIndex("main", "amd64", false, false, "")
	c.Check(file2, Equals, file)
}

func (s *IndexFilesSuite) TestPackageIndexSource(c *C) {
	// Test PackageIndex creation for source packages
	file := s.indexFiles.PackageIndex("main", ArchitectureSource, false, false, "")
	c.Check(file, NotNil)
	c.Check(file.relativePath, Equals, "main/source/Sources")
	c.Check(file.compressable, Equals, true)
	c.Check(file.discardable, Equals, false)
}

func (s *IndexFilesSuite) TestPackageIndexUdeb(c *C) {
	// Test PackageIndex creation for udeb packages
	file := s.indexFiles.PackageIndex("main", "amd64", true, false, "")
	c.Check(file, NotNil)
	c.Check(file.relativePath, Equals, "main/debian-installer/binary-amd64/Packages")
	c.Check(file.compressable, Equals, true)
}

func (s *IndexFilesSuite) TestPackageIndexInstaller(c *C) {
	// Test PackageIndex creation for installer images
	file := s.indexFiles.PackageIndex("main", "amd64", false, true, "")
	c.Check(file, NotNil)
	c.Check(file.relativePath, Equals, "main/installer-amd64/current/images/SHA256SUMS")
	c.Check(file.compressable, Equals, false)
	c.Check(file.detachedSign, Equals, true)

	// Test focal distribution special case
	fileFocal := s.indexFiles.PackageIndex("main", "amd64", false, true, aptly.DistributionFocal)
	c.Check(fileFocal, NotNil)
	c.Check(fileFocal.relativePath, Equals, "main/installer-amd64/current/legacy-images/SHA256SUMS")
}

func (s *IndexFilesSuite) TestReleaseIndex(c *C) {
	// Test ReleaseIndex creation for binary architecture
	file := s.indexFiles.ReleaseIndex("main", "amd64", false)
	c.Check(file, NotNil)
	c.Check(file.relativePath, Equals, "main/binary-amd64/Release")
	c.Check(file.compressable, Equals, false)
	c.Check(file.discardable, Equals, false)

	// Test that same call returns cached instance
	file2 := s.indexFiles.ReleaseIndex("main", "amd64", false)
	c.Check(file2, Equals, file)
}

func (s *IndexFilesSuite) TestReleaseIndexSource(c *C) {
	// Test ReleaseIndex creation for source architecture
	file := s.indexFiles.ReleaseIndex("main", ArchitectureSource, false)
	c.Check(file, NotNil)
	c.Check(file.relativePath, Equals, "main/source/Release")
	c.Check(file.compressable, Equals, false)
}

func (s *IndexFilesSuite) TestReleaseIndexUdeb(c *C) {
	// Test ReleaseIndex creation for udeb (should be discardable)
	file := s.indexFiles.ReleaseIndex("main", "amd64", true)
	c.Check(file, NotNil)
	c.Check(file.relativePath, Equals, "main/debian-installer/binary-amd64/Release")
	c.Check(file.discardable, Equals, true)
}

func (s *IndexFilesSuite) TestContentsIndex(c *C) {
	// Test ContentsIndex creation for regular packages
	file := s.indexFiles.ContentsIndex("main", "amd64", false)
	c.Check(file, NotNil)
	c.Check(file.relativePath, Equals, "main/Contents-amd64")
	c.Check(file.compressable, Equals, true)
	c.Check(file.onlyGzip, Equals, true)
	c.Check(file.discardable, Equals, true)

	// Test that same call returns cached instance
	file2 := s.indexFiles.ContentsIndex("main", "amd64", false)
	c.Check(file2, Equals, file)
}

func (s *IndexFilesSuite) TestContentsIndexUdeb(c *C) {
	// Test ContentsIndex creation for udeb packages
	file := s.indexFiles.ContentsIndex("main", "amd64", true)
	c.Check(file, NotNil)
	c.Check(file.relativePath, Equals, "main/Contents-udeb-amd64")
	c.Check(file.compressable, Equals, true)
	c.Check(file.onlyGzip, Equals, true)
}

func (s *IndexFilesSuite) TestContentsIndexSource(c *C) {
	// Test ContentsIndex for source architecture (should not have udeb)
	file := s.indexFiles.ContentsIndex("main", ArchitectureSource, true)
	c.Check(file, NotNil)
	c.Check(file.relativePath, Equals, "main/Contents-source")
	// udeb flag should be ignored for source
}

func (s *IndexFilesSuite) TestLegacyContentsIndex(c *C) {
	// Test LegacyContentsIndex creation
	file := s.indexFiles.LegacyContentsIndex("amd64", false)
	c.Check(file, NotNil)
	c.Check(file.relativePath, Equals, "Contents-amd64")
	c.Check(file.compressable, Equals, true)
	c.Check(file.onlyGzip, Equals, true)
	c.Check(file.discardable, Equals, true)

	// Test that same call returns cached instance
	file2 := s.indexFiles.LegacyContentsIndex("amd64", false)
	c.Check(file2, Equals, file)
}

func (s *IndexFilesSuite) TestLegacyContentsIndexUdeb(c *C) {
	// Test LegacyContentsIndex for udeb
	file := s.indexFiles.LegacyContentsIndex("amd64", true)
	c.Check(file, NotNil)
	c.Check(file.relativePath, Equals, "Contents-udeb-amd64")
}

func (s *IndexFilesSuite) TestSkelIndex(c *C) {
	// Test SkelIndex creation
	file := s.indexFiles.SkelIndex("main", "extra/file.txt")
	c.Check(file, NotNil)
	c.Check(file.relativePath, Equals, "main/extra/file.txt")
	c.Check(file.compressable, Equals, false)
	c.Check(file.discardable, Equals, false)

	// Test that same call returns cached instance
	file2 := s.indexFiles.SkelIndex("main", "extra/file.txt")
	c.Check(file2, Equals, file)
}

func (s *IndexFilesSuite) TestReleaseFile(c *C) {
	// Test ReleaseFile creation (should not be cached)
	file := s.indexFiles.ReleaseFile()
	c.Check(file, NotNil)
	c.Check(file.relativePath, Equals, "Release")
	c.Check(file.compressable, Equals, false)
	c.Check(file.detachedSign, Equals, true)
	c.Check(file.clearSign, Equals, true)

	// Test that new call returns different instance (not cached)
	file2 := s.indexFiles.ReleaseFile()
	c.Check(file2, Not(Equals), file)
	c.Check(file2.relativePath, Equals, "Release")
}

func (s *IndexFilesSuite) TestFinalizeAll(c *C) {
	// Test finalizing all index files
	mockSigner := &MockSigner{}
	mockProgress := &MockProgress{}

	// Create some index files
	file1 := s.indexFiles.PackageIndex("main", "amd64", false, false, "")
	file2 := s.indexFiles.ContentsIndex("main", "amd64", false)

	// Write content to files
	writer1, _ := file1.BufWriter()
	writer1.WriteString("Package: test1\n")
	writer2, _ := file2.BufWriter()
	writer2.WriteString("test1 section/file")

	err := s.indexFiles.FinalizeAll(mockProgress, mockSigner)
	c.Check(err, IsNil)

	// Check that files were published
	c.Check(len(s.publishedStorage.files) > 0, Equals, true)

	// Check that progress was tracked
	c.Check(mockProgress.InitBarCalled, Equals, true)
	c.Check(mockProgress.ShutdownBarCalled, Equals, true)
	c.Check(mockProgress.AddBarCount >= 2, Equals, true)

	// Check that indexes map is cleared
	c.Check(len(s.indexFiles.indexes), Equals, 0)
}

func (s *IndexFilesSuite) TestFinalizeAllNoProgress(c *C) {
	// Test finalizing without progress tracking
	file := s.indexFiles.PackageIndex("main", "amd64", false, false, "")
	writer, _ := file.BufWriter()
	writer.WriteString("Package: test\n")

	err := s.indexFiles.FinalizeAll(nil, nil)
	c.Check(err, IsNil)

	c.Check(len(s.publishedStorage.files) > 0, Equals, true)
	c.Check(len(s.indexFiles.indexes), Equals, 0)
}

func (s *IndexFilesSuite) TestRenameFiles(c *C) {
	// Test file renaming functionality
	s.indexFiles.renameMap["old/path"] = "new/path"
	s.indexFiles.renameMap["another/old"] = "another/new"

	err := s.indexFiles.RenameFiles()
	c.Check(err, IsNil)

	// Check that rename operations were performed
	c.Check(s.publishedStorage.RenameOperations["old/path"], Equals, "new/path")
	c.Check(s.publishedStorage.RenameOperations["another/old"], Equals, "another/new")
}

func (s *IndexFilesSuite) TestRenameFilesError(c *C) {
	// Test rename error handling
	s.publishedStorage.SimulateRenameError = true
	s.indexFiles.renameMap["will/fail"] = "target"

	err := s.indexFiles.RenameFiles()
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to rename.*")
}

func (s *IndexFilesSuite) TestAcquireByHashFeature(c *C) {
	// Test acquire-by-hash functionality
	s.indexFiles.acquireByHash = true

	file := &indexFile{
		parent:        s.indexFiles,
		relativePath:  "main/binary-amd64/Packages",
		compressable:  true,
		acquireByHash: true,
	}

	writer, _ := file.BufWriter()
	writer.WriteString("Package: test-hash\nVersion: 1.0\n")

	err := file.Finalize(nil)
	c.Check(err, IsNil)

	// Check that by-hash directories were created
	c.Check(s.publishedStorage.dirs["dists/test/main/binary-amd64/by-hash/MD5Sum"], Equals, true)
	c.Check(s.publishedStorage.dirs["dists/test/main/binary-amd64/by-hash/SHA1"], Equals, true)
	c.Check(s.publishedStorage.dirs["dists/test/main/binary-amd64/by-hash/SHA256"], Equals, true)
	c.Check(s.publishedStorage.dirs["dists/test/main/binary-amd64/by-hash/SHA512"], Equals, true)
}

func (s *IndexFilesSuite) TestPackageIndexByHashFunction(c *C) {
	// Test packageIndexByHash function directly
	s.indexFiles.generatedFiles["main/binary-amd64/Packages"] = utils.ChecksumInfo{
		MD5:    "d41d8cd98f00b204e9800998ecf8427e",
		SHA1:   "da39a3ee5e6b4b0d3255bfef95601890afd80709",
		SHA256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		SHA512: "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e",
	}

	file := &indexFile{
		parent:       s.indexFiles,
		relativePath: "main/binary-amd64/Packages",
	}

	err := packageIndexByHash(file, "", "SHA256", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	c.Check(err, IsNil)

	// Check that hard link was created
	expectedSrc := "dists/test/main/binary-amd64/Packages"
	expectedDst := "dists/test/main/binary-amd64/by-hash/SHA256/e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	c.Check(s.publishedStorage.HardLinks[expectedDst], Equals, expectedSrc)
}

func (s *IndexFilesSuite) TestSkipBz2Feature(c *C) {
	// Test skipBz2 functionality
	s.indexFiles.skipBz2 = true

	file := &indexFile{
		parent:       s.indexFiles,
		relativePath: "main/binary-amd64/Packages",
		compressable: true,
		onlyGzip:     false,
	}

	writer, _ := file.BufWriter()
	writer.WriteString("Package: no-bz2\n")

	err := file.Finalize(nil)
	c.Check(err, IsNil)

	// Should have .gz but not .bz2
	c.Check(s.publishedStorage.files["dists/test/main/binary-amd64/Packages.gz"], NotNil)
	_, hasBz2 := s.publishedStorage.files["dists/test/main/binary-amd64/Packages.bz2"]
	c.Check(hasBz2, Equals, false)
}

// Mock implementations for testing

type MockPublishedStorage struct {
	files                 map[string]string
	dirs                  map[string]bool
	links                 map[string]string
	symlinks              map[string]string
	HardLinks             map[string]string
	RenameOperations      map[string]string
	SimulateRenameError   bool
	SimulateFileError     bool
	SimulateSymlinkExists bool
}

func (m *MockPublishedStorage) MkDir(path string) error {
	if m.dirs == nil {
		m.dirs = make(map[string]bool)
	}
	m.dirs[path] = true
	return nil
}

func (m *MockPublishedStorage) PutFile(path, source string) error {
	if m.SimulateFileError {
		return fmt.Errorf("simulated file error")
	}
	if m.files == nil {
		m.files = make(map[string]string)
	}
	// Read source content (simplified for test)
	content, err := ioutil.ReadFile(source)
	if err != nil {
		// Create dummy content for missing files
		content = []byte("mock content")
	}
	m.files[path] = string(content)
	return nil
}

func (m *MockPublishedStorage) Remove(path string) error {
	delete(m.files, path)
	delete(m.links, path)
	delete(m.symlinks, path)
	return nil
}

func (m *MockPublishedStorage) RenameFile(oldName, newName string) error {
	if m.SimulateRenameError {
		return fmt.Errorf("simulated rename error")
	}
	if m.RenameOperations == nil {
		m.RenameOperations = make(map[string]string)
	}
	m.RenameOperations[oldName] = newName
	if content, exists := m.files[oldName]; exists {
		m.files[newName] = content
		delete(m.files, oldName)
	}
	return nil
}

func (m *MockPublishedStorage) FileExists(path string) (bool, error) {
	_, exists := m.files[path]
	if !exists {
		_, exists = m.symlinks[path]
	}
	if m.SimulateSymlinkExists {
		return true, nil
	}
	return exists, nil
}

func (m *MockPublishedStorage) HardLink(src, dst string) error {
	if m.HardLinks == nil {
		m.HardLinks = make(map[string]string)
	}
	m.HardLinks[dst] = src
	return nil
}

func (m *MockPublishedStorage) SymLink(src, dst string) error {
	if m.symlinks == nil {
		m.symlinks = make(map[string]string)
	}
	m.symlinks[dst] = src
	return nil
}

func (m *MockPublishedStorage) ReadLink(path string) (string, error) {
	if target, exists := m.symlinks[path]; exists {
		return target, nil
	}
	return "", fmt.Errorf("not a symlink")
}

func (m *MockPublishedStorage) Filelist(prefix string) ([]string, error) {
	var files []string
	for path := range m.files {
		if strings.HasPrefix(path, prefix) {
			files = append(files, path)
		}
	}
	return files, nil
}

func (m *MockPublishedStorage) LinkFromPool(publishedPrefix, publishedRelPath, fileName string, sourcePool aptly.PackagePool, sourcePath string, sourceChecksums utils.ChecksumInfo, force bool) error {
	// Mock implementation - just track that it was called
	if m.files == nil {
		m.files = make(map[string]string)
	}
	m.files[publishedRelPath] = "linked from pool"
	return nil
}

func (m *MockPublishedStorage) RemoveDirs(path string, progress aptly.Progress) error {
	// Mock implementation - remove files with path prefix
	for filePath := range m.files {
		if strings.HasPrefix(filePath, path) {
			delete(m.files, filePath)
		}
	}
	for dirPath := range m.dirs {
		if strings.HasPrefix(dirPath, path) {
			delete(m.dirs, dirPath)
		}
	}
	return nil
}

type MockSigner struct {
	DetachedSignCalled bool
	ClearSignCalled    bool
}

func (m *MockSigner) Init() error                         { return nil }
func (m *MockSigner) SetKey(keyRef string)                {}
func (m *MockSigner) SetKeyRing(keyring, secretKeyring string) {}
func (m *MockSigner) SetPassphrase(passphrase, passphraseFile string) {}
func (m *MockSigner) SetBatch(batch bool)                 {}

func (m *MockSigner) DetachedSign(source, signature string) error {
	m.DetachedSignCalled = true
	// Create mock signature file
	return ioutil.WriteFile(signature, []byte("mock signature"), 0644)
}

func (m *MockSigner) ClearSign(source, signature string) error {
	m.ClearSignCalled = true
	// Create mock clear-signed file
	return ioutil.WriteFile(signature, []byte("mock clear signature"), 0644)
}

type MockProgress struct {
	InitBarCalled     bool
	ShutdownBarCalled bool
	AddBarCount       int
}

func (m *MockProgress) InitBar(count int64, isBytes bool, barType aptly.BarType) {
	m.InitBarCalled = true
}

func (m *MockProgress) ShutdownBar() {
	m.ShutdownBarCalled = true
}

func (m *MockProgress) AddBar(count int) {
	m.AddBarCount += count
}

func (m *MockProgress) SetBar(count int) {}

func (m *MockProgress) PrintfBar(msg string, a ...interface{}) {}

func (m *MockProgress) ColoredPrintf(msg string, a ...interface{}) {}

func (m *MockProgress) Printf(msg string, a ...interface{}) {}

func (m *MockProgress) Flush() {}

func (m *MockProgress) PrintfStdErr(msg string, a ...interface{}) {}

func (m *MockProgress) Start() {}

func (m *MockProgress) Shutdown() {}

func (m *MockProgress) Write(p []byte) (n int, err error) {
	return len(p), nil
}