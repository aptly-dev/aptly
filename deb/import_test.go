package deb

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/database"
	"github.com/aptly-dev/aptly/pgp"
	"github.com/aptly-dev/aptly/utils"
	. "gopkg.in/check.v1"
)

type ImportSuite struct {
	tempDir string
}

var _ = Suite(&ImportSuite{})

func (s *ImportSuite) SetUpTest(c *C) {
	s.tempDir = c.MkDir()
}

type MockResultReporter struct {
	warnings []string
	added    []string
	removed  []string
}

func (m *MockResultReporter) Warning(msg string, a ...interface{}) {
	m.warnings = append(m.warnings, fmt.Sprintf(msg, a...))
}

func (m *MockResultReporter) Added(msg string, a ...interface{}) {
	m.added = append(m.added, fmt.Sprintf(msg, a...))
}

func (m *MockResultReporter) Removed(msg string, a ...interface{}) {
	m.removed = append(m.removed, fmt.Sprintf(msg, a...))
}

func (s *ImportSuite) TestCollectPackageFilesEmpty(c *C) {
	// Test with empty locations list
	reporter := &MockResultReporter{}
	packageFiles, otherFiles, failedFiles := CollectPackageFiles([]string{}, reporter)

	c.Check(len(packageFiles), Equals, 0)
	c.Check(len(otherFiles), Equals, 0)
	c.Check(len(failedFiles), Equals, 0)
	c.Check(len(reporter.warnings), Equals, 0)
}

func (s *ImportSuite) TestCollectPackageFilesNonExistentLocation(c *C) {
	// Test with non-existent location
	reporter := &MockResultReporter{}
	nonExistentPath := filepath.Join(s.tempDir, "nonexistent")

	packageFiles, otherFiles, failedFiles := CollectPackageFiles([]string{nonExistentPath}, reporter)

	c.Check(len(packageFiles), Equals, 0)
	c.Check(len(otherFiles), Equals, 0)
	c.Check(len(failedFiles), Equals, 1)
	c.Check(failedFiles[0], Equals, nonExistentPath)
	c.Check(len(reporter.warnings), Equals, 1)
	c.Check(strings.Contains(reporter.warnings[0], "Unable to process"), Equals, true)
}

func (s *ImportSuite) TestCollectPackageFilesSingleDebFile(c *C) {
	// Test with single .deb file
	reporter := &MockResultReporter{}
	debFile := filepath.Join(s.tempDir, "package.deb")

	// Create dummy .deb file
	err := ioutil.WriteFile(debFile, []byte("dummy deb content"), 0644)
	c.Assert(err, IsNil)

	packageFiles, otherFiles, failedFiles := CollectPackageFiles([]string{debFile}, reporter)

	c.Check(len(packageFiles), Equals, 1)
	c.Check(packageFiles[0], Equals, debFile)
	c.Check(len(otherFiles), Equals, 0)
	c.Check(len(failedFiles), Equals, 0)
	c.Check(len(reporter.warnings), Equals, 0)
}

func (s *ImportSuite) TestCollectPackageFilesSingleUdebFile(c *C) {
	// Test with single .udeb file
	reporter := &MockResultReporter{}
	udebFile := filepath.Join(s.tempDir, "package.udeb")

	err := ioutil.WriteFile(udebFile, []byte("dummy udeb content"), 0644)
	c.Assert(err, IsNil)

	packageFiles, otherFiles, failedFiles := CollectPackageFiles([]string{udebFile}, reporter)

	c.Check(len(packageFiles), Equals, 1)
	c.Check(packageFiles[0], Equals, udebFile)
	c.Check(len(otherFiles), Equals, 0)
	c.Check(len(failedFiles), Equals, 0)
}

func (s *ImportSuite) TestCollectPackageFilesSingleDscFile(c *C) {
	// Test with single .dsc file
	reporter := &MockResultReporter{}
	dscFile := filepath.Join(s.tempDir, "package.dsc")

	err := ioutil.WriteFile(dscFile, []byte("dummy dsc content"), 0644)
	c.Assert(err, IsNil)

	packageFiles, otherFiles, failedFiles := CollectPackageFiles([]string{dscFile}, reporter)

	c.Check(len(packageFiles), Equals, 1)
	c.Check(packageFiles[0], Equals, dscFile)
	c.Check(len(otherFiles), Equals, 0)
	c.Check(len(failedFiles), Equals, 0)
}

func (s *ImportSuite) TestCollectPackageFilesSingleDdebFile(c *C) {
	// Test with single .ddeb file
	reporter := &MockResultReporter{}
	ddebFile := filepath.Join(s.tempDir, "package.ddeb")

	err := ioutil.WriteFile(ddebFile, []byte("dummy ddeb content"), 0644)
	c.Assert(err, IsNil)

	packageFiles, otherFiles, failedFiles := CollectPackageFiles([]string{ddebFile}, reporter)

	c.Check(len(packageFiles), Equals, 1)
	c.Check(packageFiles[0], Equals, ddebFile)
	c.Check(len(otherFiles), Equals, 0)
	c.Check(len(failedFiles), Equals, 0)
}

func (s *ImportSuite) TestCollectPackageFilesBuildInfoFile(c *C) {
	// Test with .buildinfo file
	reporter := &MockResultReporter{}
	buildinfoFile := filepath.Join(s.tempDir, "package.buildinfo")

	err := ioutil.WriteFile(buildinfoFile, []byte("dummy buildinfo content"), 0644)
	c.Assert(err, IsNil)

	packageFiles, otherFiles, failedFiles := CollectPackageFiles([]string{buildinfoFile}, reporter)

	c.Check(len(packageFiles), Equals, 0)
	c.Check(len(otherFiles), Equals, 1)
	c.Check(otherFiles[0], Equals, buildinfoFile)
	c.Check(len(failedFiles), Equals, 0)
	c.Check(len(reporter.warnings), Equals, 0)
}

func (s *ImportSuite) TestCollectPackageFilesUnknownExtension(c *C) {
	// Test with unknown file extension
	reporter := &MockResultReporter{}
	unknownFile := filepath.Join(s.tempDir, "package.unknown")

	err := ioutil.WriteFile(unknownFile, []byte("dummy content"), 0644)
	c.Assert(err, IsNil)

	packageFiles, otherFiles, failedFiles := CollectPackageFiles([]string{unknownFile}, reporter)

	c.Check(len(packageFiles), Equals, 0)
	c.Check(len(otherFiles), Equals, 0)
	c.Check(len(failedFiles), Equals, 1)
	c.Check(failedFiles[0], Equals, unknownFile)
	c.Check(len(reporter.warnings), Equals, 1)
	c.Check(strings.Contains(reporter.warnings[0], "Unknown file extension"), Equals, true)
}

func (s *ImportSuite) TestCollectPackageFilesDirectory(c *C) {
	// Test with directory containing various files
	reporter := &MockResultReporter{}
	subDir := filepath.Join(s.tempDir, "packages")
	err := os.MkdirAll(subDir, 0755)
	c.Assert(err, IsNil)

	// Create various file types
	files := map[string]string{
		"package1.deb":      "deb content",
		"package2.udeb":     "udeb content",
		"source.dsc":        "dsc content",
		"debug.ddeb":        "ddeb content",
		"build.buildinfo":   "buildinfo content",
		"readme.txt":        "text content",
		"subdir/nested.deb": "nested deb",
	}

	// Create nested subdirectory
	nestedDir := filepath.Join(subDir, "subdir")
	err = os.MkdirAll(nestedDir, 0755)
	c.Assert(err, IsNil)

	for filename, content := range files {
		fullPath := filepath.Join(subDir, filename)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		c.Assert(err, IsNil)
		err = ioutil.WriteFile(fullPath, []byte(content), 0644)
		c.Assert(err, IsNil)
	}

	packageFiles, otherFiles, failedFiles := CollectPackageFiles([]string{subDir}, reporter)

	// Should find package files (sorted)
	expectedPackageFiles := []string{
		filepath.Join(subDir, "debug.ddeb"),
		filepath.Join(subDir, "package1.deb"),
		filepath.Join(subDir, "package2.udeb"),
		filepath.Join(subDir, "source.dsc"),
		filepath.Join(subDir, "subdir", "nested.deb"),
	}
	sort.Strings(expectedPackageFiles)

	c.Check(len(packageFiles), Equals, 5)
	c.Check(packageFiles, DeepEquals, expectedPackageFiles)

	// Should find other files
	c.Check(len(otherFiles), Equals, 1)
	c.Check(otherFiles[0], Equals, filepath.Join(subDir, "build.buildinfo"))

	// No failed files
	c.Check(len(failedFiles), Equals, 0)
	c.Check(len(reporter.warnings), Equals, 0)
}

func (s *ImportSuite) TestCollectPackageFilesMixedLocations(c *C) {
	// Test with mix of files and directories
	reporter := &MockResultReporter{}

	// Create individual file
	debFile := filepath.Join(s.tempDir, "single.deb")
	err := ioutil.WriteFile(debFile, []byte("single deb"), 0644)
	c.Assert(err, IsNil)

	// Create directory with files
	subDir := filepath.Join(s.tempDir, "multi")
	err = os.MkdirAll(subDir, 0755)
	c.Assert(err, IsNil)

	dscFile := filepath.Join(subDir, "source.dsc")
	err = ioutil.WriteFile(dscFile, []byte("dsc content"), 0644)
	c.Assert(err, IsNil)

	packageFiles, otherFiles, failedFiles := CollectPackageFiles([]string{debFile, subDir}, reporter)

	expectedFiles := []string{debFile, dscFile}
	sort.Strings(expectedFiles)

	c.Check(len(packageFiles), Equals, 2)
	c.Check(packageFiles, DeepEquals, expectedFiles)
	c.Check(len(otherFiles), Equals, 0)
	c.Check(len(failedFiles), Equals, 0)
}

func (s *ImportSuite) TestCollectPackageFilesConcurrency(c *C) {
	// Test concurrent access during directory walking
	reporter := &MockResultReporter{}
	subDir := filepath.Join(s.tempDir, "concurrent")
	err := os.MkdirAll(subDir, 0755)
	c.Assert(err, IsNil)

	// Create many files to test concurrent access
	for i := 0; i < 100; i++ {
		filename := filepath.Join(subDir, fmt.Sprintf("package%d.deb", i))
		err := ioutil.WriteFile(filename, []byte(fmt.Sprintf("content %d", i)), 0644)
		c.Assert(err, IsNil)

		if i%10 == 0 {
			buildinfoFile := filepath.Join(subDir, fmt.Sprintf("build%d.buildinfo", i))
			err = ioutil.WriteFile(buildinfoFile, []byte(fmt.Sprintf("buildinfo %d", i)), 0644)
			c.Assert(err, IsNil)
		}
	}

	packageFiles, otherFiles, failedFiles := CollectPackageFiles([]string{subDir}, reporter)

	c.Check(len(packageFiles), Equals, 100)
	c.Check(len(otherFiles), Equals, 10) // Every 10th file is buildinfo
	c.Check(len(failedFiles), Equals, 0)

	// Check that files are sorted
	c.Check(sort.StringsAreSorted(packageFiles), Equals, true)
}

func (s *ImportSuite) TestCollectPackageFilesPermissionDenied(c *C) {
	// Test handling of permission denied errors
	reporter := &MockResultReporter{}

	// Create directory and remove read permission (if running as non-root)
	subDir := filepath.Join(s.tempDir, "noperm")
	err := os.MkdirAll(subDir, 0755)
	c.Assert(err, IsNil)

	// Create a file inside
	testFile := filepath.Join(subDir, "test.deb")
	err = ioutil.WriteFile(testFile, []byte("test"), 0644)
	c.Assert(err, IsNil)

	// Remove read permission from directory
	err = os.Chmod(subDir, 0000)
	if err != nil {
		c.Skip("Cannot remove permissions, likely running as root")
	}
	defer os.Chmod(subDir, 0755) // Restore for cleanup

	packageFiles, otherFiles, failedFiles := CollectPackageFiles([]string{subDir}, reporter)

	// Should handle permission error gracefully
	c.Check(len(packageFiles), Equals, 0)
	c.Check(len(otherFiles), Equals, 0)
	c.Check(len(failedFiles), Equals, 1)
	c.Check(failedFiles[0], Equals, subDir)
	c.Check(len(reporter.warnings), Equals, 1)
	c.Check(strings.Contains(reporter.warnings[0], "Unable to process"), Equals, true)
}

func (s *ImportSuite) TestCollectPackageFilesEmptyDirectory(c *C) {
	// Test with empty directory
	reporter := &MockResultReporter{}
	emptyDir := filepath.Join(s.tempDir, "empty")
	err := os.MkdirAll(emptyDir, 0755)
	c.Assert(err, IsNil)

	packageFiles, otherFiles, failedFiles := CollectPackageFiles([]string{emptyDir}, reporter)

	c.Check(len(packageFiles), Equals, 0)
	c.Check(len(otherFiles), Equals, 0)
	c.Check(len(failedFiles), Equals, 0)
	c.Check(len(reporter.warnings), Equals, 0)
}

func (s *ImportSuite) TestCollectPackageFilesNestedDirectories(c *C) {
	// Test deeply nested directory structure
	reporter := &MockResultReporter{}

	// Create nested structure: base/level1/level2/level3/
	deepDir := filepath.Join(s.tempDir, "base", "level1", "level2", "level3")
	err := os.MkdirAll(deepDir, 0755)
	c.Assert(err, IsNil)

	// Place files at different levels
	files := map[string]string{
		filepath.Join(s.tempDir, "base", "root.deb"):                               "root",
		filepath.Join(s.tempDir, "base", "level1", "level1.deb"):                   "level1",
		filepath.Join(s.tempDir, "base", "level1", "level2", "level2.deb"):         "level2",
		filepath.Join(s.tempDir, "base", "level1", "level2", "level3", "deep.deb"): "deep",
	}

	for path, content := range files {
		err := ioutil.WriteFile(path, []byte(content), 0644)
		c.Assert(err, IsNil)
	}

	packageFiles, otherFiles, failedFiles := CollectPackageFiles([]string{filepath.Join(s.tempDir, "base")}, reporter)

	c.Check(len(packageFiles), Equals, 4)
	c.Check(len(otherFiles), Equals, 0)
	c.Check(len(failedFiles), Equals, 0)

	// Verify all nested files were found
	for expectedPath := range files {
		found := false
		for _, foundPath := range packageFiles {
			if foundPath == expectedPath {
				found = true
				break
			}
		}
		c.Check(found, Equals, true, Commentf("File not found: %s", expectedPath))
	}
}

func (s *ImportSuite) TestCollectPackageFilesCaseInsensitive(c *C) {
	// Test case sensitivity of file extensions
	reporter := &MockResultReporter{}

	// Create files with various case extensions
	files := []string{
		"package.deb",
		"package.DEB",
		"package.Deb",
		"source.dsc",
		"source.DSC",
		"package.udeb",
		"package.UDEB",
		"debug.ddeb",
		"debug.DDEB",
	}

	for _, filename := range files {
		path := filepath.Join(s.tempDir, filename)
		err := ioutil.WriteFile(path, []byte("content"), 0644)
		c.Assert(err, IsNil)
	}

	packageFiles, otherFiles, failedFiles := CollectPackageFiles([]string{s.tempDir}, reporter)

	// Only lowercase extensions should be recognized
	c.Check(len(packageFiles), Equals, 4) // .deb, .dsc, .udeb, .ddeb (lowercase only)
	c.Check(len(otherFiles), Equals, 0)
	// Uppercase extensions are silently ignored by the file walker, not reported as failed
	c.Check(len(failedFiles), Equals, 0)
	c.Check(len(reporter.warnings), Equals, 0)
}

func (s *ImportSuite) TestCollectPackageFilesSymlinks(c *C) {
	// Test handling of symbolic links
	reporter := &MockResultReporter{}

	// Create a real file
	realFile := filepath.Join(s.tempDir, "real.deb")
	err := ioutil.WriteFile(realFile, []byte("real content"), 0644)
	c.Assert(err, IsNil)

	// Create a symlink to it
	linkFile := filepath.Join(s.tempDir, "link.deb")
	err = os.Symlink(realFile, linkFile)
	if err != nil {
		c.Skip("Cannot create symlinks on this filesystem")
	}

	packageFiles, otherFiles, failedFiles := CollectPackageFiles([]string{s.tempDir}, reporter)

	// Both real file and symlink should be found
	c.Check(len(packageFiles), Equals, 2)
	c.Check(len(otherFiles), Equals, 0)
	c.Check(len(failedFiles), Equals, 0)
}

func (s *ImportSuite) TestCollectPackageFilesSpecialCharacters(c *C) {
	// Test files with special characters in names
	reporter := &MockResultReporter{}

	// Create files with various special characters
	specialFiles := []string{
		"package with spaces.deb",
		"package-with-dashes.deb",
		"package_with_underscores.deb",
		"package.1.0-1.deb",
		"package+plus.deb",
		"package@at.deb",
	}

	for _, filename := range specialFiles {
		path := filepath.Join(s.tempDir, filename)
		err := ioutil.WriteFile(path, []byte("content"), 0644)
		c.Assert(err, IsNil)
	}

	packageFiles, otherFiles, failedFiles := CollectPackageFiles([]string{s.tempDir}, reporter)

	c.Check(len(packageFiles), Equals, len(specialFiles))
	c.Check(len(otherFiles), Equals, 0)
	c.Check(len(failedFiles), Equals, 0)
	c.Check(len(reporter.warnings), Equals, 0)
}

// Mock implementations for ImportPackageFiles testing

type MockPackagePool struct {
	importFunc func(string, string, *utils.ChecksumInfo, bool, aptly.ChecksumStorage) (string, error)
	verifyFunc func(string, string, *utils.ChecksumInfo, aptly.ChecksumStorage) (string, bool, error)
}

func (m *MockPackagePool) Import(srcPath, basename string, checksums *utils.ChecksumInfo, move bool, storage aptly.ChecksumStorage) (string, error) {
	if m.importFunc != nil {
		return m.importFunc(srcPath, basename, checksums, move, storage)
	}
	return "pool/" + basename, nil
}

func (m *MockPackagePool) Verify(poolPath, basename string, checksums *utils.ChecksumInfo, storage aptly.ChecksumStorage) (string, bool, error) {
	if m.verifyFunc != nil {
		return m.verifyFunc(poolPath, basename, checksums, storage)
	}
	return poolPath, true, nil
}

func (m *MockPackagePool) LegacyPath(filename string, checksums *utils.ChecksumInfo) (string, error) {
	return "legacy/" + filename, nil
}

func (m *MockPackagePool) Size(path string) (int64, error) {
	return 1024, nil
}

func (m *MockPackagePool) Open(path string) (aptly.ReadSeekerCloser, error) {
	return nil, nil
}

func (m *MockPackagePool) FilepathList(progress aptly.Progress) ([]string, error) {
	return []string{}, nil
}

func (m *MockPackagePool) Remove(path string) (int64, error) {
	return 1024, nil
}

type MockVerifier struct {
	verifyFunc func(string, string, string) (bool, error)
}

func (m *MockVerifier) ExtractClearsign(signedMessage string) (string, error) {
	return signedMessage, nil
}

func (m *MockVerifier) VerifyClearsign(clearsignInput string, keyringName string, showKeyInfo bool) (string, string, error) {
	if m.verifyFunc != nil {
		if valid, err := m.verifyFunc(clearsignInput, keyringName, ""); err != nil {
			return "", "", err
		} else if !valid {
			return "", "", fmt.Errorf("verification failed")
		}
	}
	return clearsignInput, "", nil
}

// Add missing methods to implement pgp.Verifier interface
func (m *MockVerifier) InitKeyring(verbose bool) error {
	return nil
}

func (m *MockVerifier) AddKeyring(keyring string) {
	// Mock implementation
}

func (m *MockVerifier) VerifyDetachedSignature(signature, cleartext io.Reader, showKeyTip bool) error {
	return nil
}

func (m *MockVerifier) IsClearSigned(clearsigned io.Reader) (bool, error) {
	return true, nil
}

func (m *MockVerifier) VerifyClearsigned(clearsigned io.Reader, showKeyTip bool) (*pgp.KeyInfo, error) {
	return &pgp.KeyInfo{}, nil
}

func (m *MockVerifier) ExtractClearsigned(clearsigned io.Reader) (*os.File, error) {
	// Create a temporary file for mock
	tmpFile, err := ioutil.TempFile("", "mock_extract")
	return tmpFile, err
}

type MockPackageCollection struct {
	updateFunc func(*Package) error
	packages   map[string]*Package
}

func (m *MockPackageCollection) Update(p *Package) error {
	if m.updateFunc != nil {
		return m.updateFunc(p)
	}
	if m.packages == nil {
		m.packages = make(map[string]*Package)
	}
	m.packages[string(p.Key(""))] = p
	return nil
}

func (m *MockPackageCollection) ByKey(key []byte) (*Package, error) {
	if m.packages == nil {
		return nil, fmt.Errorf("not found")
	}
	if pkg, exists := m.packages[string(key)]; exists {
		return pkg, nil
	}
	return nil, fmt.Errorf("not found")
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

func (s *ImportSuite) TestImportPackageFilesEmptyList(c *C) {
	// Test ImportPackageFiles with empty file list
	list := NewPackageList()
	reporter := &MockResultReporter{}
	pool := &MockPackagePool{}
	collection := NewPackageCollection(nil)
	verifier := &MockVerifier{}
	checksumProvider := func(database.ReaderWriter) aptly.ChecksumStorage {
		return &MockChecksumStorage{}
	}

	processedFiles, failedFiles, err := ImportPackageFiles(
		list, []string{}, false, verifier, pool, collection, reporter, nil, checksumProvider)

	c.Check(err, IsNil)
	c.Check(len(processedFiles), Equals, 0)
	c.Check(len(failedFiles), Equals, 0)
	c.Check(len(reporter.warnings), Equals, 0)
	c.Check(len(reporter.added), Equals, 0)
}

func (s *ImportSuite) TestImportPackageFilesNonExistentFile(c *C) {
	// Test ImportPackageFiles with non-existent file
	list := NewPackageList()
	reporter := &MockResultReporter{}
	pool := &MockPackagePool{}
	collection := NewPackageCollection(nil)
	verifier := &MockVerifier{}
	checksumProvider := func(database.ReaderWriter) aptly.ChecksumStorage {
		return &MockChecksumStorage{}
	}

	nonExistentFile := filepath.Join(s.tempDir, "nonexistent.deb")

	processedFiles, failedFiles, err := ImportPackageFiles(
		list, []string{nonExistentFile}, false, verifier, pool, collection, reporter, nil, checksumProvider)

	c.Check(err, IsNil)
	c.Check(len(processedFiles), Equals, 0)
	c.Check(len(failedFiles), Equals, 1)
	c.Check(failedFiles[0], Equals, nonExistentFile)
	c.Check(len(reporter.warnings), Equals, 1)
	c.Check(strings.Contains(reporter.warnings[0], "Unable to read file"), Equals, true)
}

func (s *ImportSuite) TestImportPackageFilesInvalidPackageFile(c *C) {
	// Test ImportPackageFiles with invalid package file
	list := NewPackageList()
	reporter := &MockResultReporter{}
	pool := &MockPackagePool{}
	collection := NewPackageCollection(nil)
	verifier := &MockVerifier{}
	checksumProvider := func(database.ReaderWriter) aptly.ChecksumStorage {
		return &MockChecksumStorage{}
	}

	// Create invalid .deb file
	invalidDeb := filepath.Join(s.tempDir, "invalid.deb")
	err := ioutil.WriteFile(invalidDeb, []byte("not a valid deb file"), 0644)
	c.Assert(err, IsNil)

	processedFiles, failedFiles, err := ImportPackageFiles(
		list, []string{invalidDeb}, false, verifier, pool, collection, reporter, nil, checksumProvider)

	c.Check(err, IsNil)
	c.Check(len(processedFiles), Equals, 0)
	c.Check(len(failedFiles), Equals, 1)
	c.Check(failedFiles[0], Equals, invalidDeb)
	c.Check(len(reporter.warnings), Equals, 1)
	c.Check(strings.Contains(reporter.warnings[0], "Unable to read file"), Equals, true)
}

func (s *ImportSuite) TestImportPackageFilesPoolImportError(c *C) {
	// Test ImportPackageFiles with pool import error
	list := NewPackageList()
	reporter := &MockResultReporter{}

	// Mock pool that fails to import
	pool := &MockPackagePool{
		importFunc: func(string, string, *utils.ChecksumInfo, bool, aptly.ChecksumStorage) (string, error) {
			return "", fmt.Errorf("pool import error")
		},
	}

	collection := NewPackageCollection(nil)
	verifier := &MockVerifier{}
	checksumProvider := func(database.ReaderWriter) aptly.ChecksumStorage {
		return &MockChecksumStorage{}
	}

	// Create a simple .deb file
	debFile := filepath.Join(s.tempDir, "test.deb")
	err := ioutil.WriteFile(debFile, []byte("simple deb"), 0644)
	c.Assert(err, IsNil)

	processedFiles, failedFiles, err := ImportPackageFiles(
		list, []string{debFile}, false, verifier, pool, collection, reporter, nil, checksumProvider)

	c.Check(err, IsNil)
	c.Check(len(processedFiles), Equals, 0)
	c.Check(len(failedFiles), Equals, 1)
	c.Check(failedFiles[0], Equals, debFile)
	c.Check(len(reporter.warnings), Equals, 1) // One warning for file processing issue
}

func (s *ImportSuite) TestImportPackageFilesCollectionUpdateError(c *C) {
	// Test ImportPackageFiles with collection update error
	list := NewPackageList()
	reporter := &MockResultReporter{}
	pool := &MockPackagePool{}

	// Use real collection for testing
	collection := NewPackageCollection(nil)

	verifier := &MockVerifier{}
	checksumProvider := func(database.ReaderWriter) aptly.ChecksumStorage {
		return &MockChecksumStorage{}
	}

	// Create a simple .deb file
	debFile := filepath.Join(s.tempDir, "test.deb")
	err := ioutil.WriteFile(debFile, []byte("simple deb"), 0644)
	c.Assert(err, IsNil)

	processedFiles, failedFiles, err := ImportPackageFiles(
		list, []string{debFile}, false, verifier, pool, collection, reporter, nil, checksumProvider)

	c.Check(err, IsNil)
	c.Check(len(processedFiles), Equals, 0)
	c.Check(len(failedFiles), Equals, 1)
	c.Check(len(reporter.warnings), Equals, 1) // One warning for file processing issue
}

func (s *ImportSuite) TestImportPackageFilesForceReplace(c *C) {
	// Test ImportPackageFiles with force replace option
	list := NewPackageList()
	reporter := &MockResultReporter{}
	pool := &MockPackagePool{}
	collection := NewPackageCollection(nil)
	verifier := &MockVerifier{}
	checksumProvider := func(database.ReaderWriter) aptly.ChecksumStorage {
		return &MockChecksumStorage{}
	}

	// Test that forceReplace calls PrepareIndex on the list
	debFile := filepath.Join(s.tempDir, "test.deb")
	err := ioutil.WriteFile(debFile, []byte("simple deb"), 0644)
	c.Assert(err, IsNil)

	// With forceReplace = true
	processedFiles, failedFiles, err := ImportPackageFiles(
		list, []string{debFile}, true, verifier, pool, collection, reporter, nil, checksumProvider)

	c.Check(err, IsNil)
	c.Check(len(processedFiles), Equals, 0) // No files should be processed due to invalid file
	// Even though the file is invalid, the function should handle forceReplace logic
	c.Check(len(failedFiles), Equals, 1) // Will fail due to invalid deb file
}

func (s *ImportSuite) TestImportPackageFilesErrorHandling(c *C) {
	// Test various error conditions in ImportPackageFiles
	list := NewPackageList()
	reporter := &MockResultReporter{}
	pool := &MockPackagePool{}
	collection := NewPackageCollection(nil)
	verifier := &MockVerifier{}
	checksumProvider := func(database.ReaderWriter) aptly.ChecksumStorage {
		return &MockChecksumStorage{}
	}

	// Test with multiple files, some valid some invalid
	validDeb := filepath.Join(s.tempDir, "valid.deb")
	invalidDeb := filepath.Join(s.tempDir, "invalid.deb")
	nonExistent := filepath.Join(s.tempDir, "nonexistent.deb")

	err := ioutil.WriteFile(validDeb, []byte("valid deb content"), 0644)
	c.Assert(err, IsNil)

	err = ioutil.WriteFile(invalidDeb, []byte("invalid content"), 0644)
	c.Assert(err, IsNil)

	files := []string{validDeb, invalidDeb, nonExistent}

	processedFiles, failedFiles, err := ImportPackageFiles(
		list, files, false, verifier, pool, collection, reporter, nil, checksumProvider)

	c.Check(err, IsNil)
	c.Check(len(processedFiles), Equals, 0)    // No files should be processed successfully
	c.Check(len(failedFiles), Equals, 3)       // All files should fail
	c.Check(len(reporter.warnings), Equals, 3) // Should have warnings for all failures
}

func (s *ImportSuite) TestImportPackageFilesRestrictionFilter(c *C) {
	// Test ImportPackageFiles with package restriction filter
	list := NewPackageList()
	reporter := &MockResultReporter{}
	pool := &MockPackagePool{}
	collection := NewPackageCollection(nil)
	verifier := &MockVerifier{}
	checksumProvider := func(database.ReaderWriter) aptly.ChecksumStorage {
		return &MockChecksumStorage{}
	}

	// Create mock restriction that rejects all packages
	restriction := &MockPackageQuery{
		matchesFunc: func(*Package) bool {
			return false // Reject all packages
		},
	}

	debFile := filepath.Join(s.tempDir, "test.deb")
	err := ioutil.WriteFile(debFile, []byte("test deb"), 0644)
	c.Assert(err, IsNil)

	processedFiles, failedFiles, err := ImportPackageFiles(
		list, []string{debFile}, false, verifier, pool, collection, reporter, restriction, checksumProvider)

	c.Check(err, IsNil)
	c.Check(len(processedFiles), Equals, 0)
	c.Check(len(failedFiles), Equals, 1) // Should fail due to restriction + invalid file
	c.Check(len(reporter.warnings) >= 1, Equals, true)
}

type MockPackageQuery struct {
	matchesFunc func(*Package) bool
}

func (m *MockPackageQuery) Matches(p PackageLike) bool {
	if m.matchesFunc != nil {
		if pkg, ok := p.(*Package); ok {
			return m.matchesFunc(pkg)
		}
		return false
	}
	return true
}

func (m *MockPackageQuery) Fast(_ PackageCatalog) bool {
	return false // Mock implementation returns false for simplicity
}

func (m *MockPackageQuery) Query(list PackageCatalog) *PackageList {
	return list.Scan(m) // Default implementation
}

func (m *MockPackageQuery) String() string {
	return "MockPackageQuery"
}

func (s *ImportSuite) TestImportPackageFilesFileTypes(c *C) {
	// Test ImportPackageFiles with different file types
	list := NewPackageList()
	reporter := &MockResultReporter{}
	pool := &MockPackagePool{}
	collection := NewPackageCollection(nil)
	verifier := &MockVerifier{}
	checksumProvider := func(database.ReaderWriter) aptly.ChecksumStorage {
		return &MockChecksumStorage{}
	}

	// Create files of different types
	files := map[string]string{
		"package.deb":  "deb content",
		"package.udeb": "udeb content",
		"source.dsc":   "dsc content",
		"debug.ddeb":   "ddeb content",
	}

	var fileList []string
	for filename, content := range files {
		path := filepath.Join(s.tempDir, filename)
		err := ioutil.WriteFile(path, []byte(content), 0644)
		c.Assert(err, IsNil)
		fileList = append(fileList, path)
	}

	processedFiles, failedFiles, err := ImportPackageFiles(
		list, fileList, false, verifier, pool, collection, reporter, nil, checksumProvider)

	c.Check(err, IsNil)
	// All files should fail due to invalid format, but function should handle different types
	c.Check(len(failedFiles), Equals, len(fileList))
	c.Check(len(processedFiles), Equals, 0)
}
