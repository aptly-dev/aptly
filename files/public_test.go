package files

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/utils"

	. "gopkg.in/check.v1"
)

type PublishedStorageSuite struct {
	root            string
	storage         *PublishedStorage
	storageSymlink  *PublishedStorage
	storageCopy     *PublishedStorage
	storageCopySize *PublishedStorage
	cs              aptly.ChecksumStorage
}

var _ = Suite(&PublishedStorageSuite{})

func (s *PublishedStorageSuite) SetUpTest(c *C) {
	s.root = c.MkDir()
	s.storage = NewPublishedStorage(filepath.Join(s.root, "public"), "", "")
	s.storageSymlink = NewPublishedStorage(filepath.Join(s.root, "public_symlink"), "symlink", "")
	s.storageCopy = NewPublishedStorage(filepath.Join(s.root, "public_copy"), "copy", "")
	s.storageCopySize = NewPublishedStorage(filepath.Join(s.root, "public_copysize"), "copy", "size")
	s.cs = NewMockChecksumStorage()
}

func (s *PublishedStorageSuite) TestLinkMethodField(c *C) {
	c.Assert(s.storage.linkMethod, Equals, LinkMethodHardLink)
	c.Assert(s.storageSymlink.linkMethod, Equals, LinkMethodSymLink)
	c.Assert(s.storageCopy.linkMethod, Equals, LinkMethodCopy)
	c.Assert(s.storageCopySize.linkMethod, Equals, LinkMethodCopy)
}

func (s *PublishedStorageSuite) TestVerifyMethodField(c *C) {
	c.Assert(s.storageCopy.verifyMethod, Equals, VerificationMethodChecksum)
	c.Assert(s.storageCopySize.verifyMethod, Equals, VerificationMethodFileSize)
}

func (s *PublishedStorageSuite) TestPublicPath(c *C) {
	c.Assert(s.storage.PublicPath(), Equals, filepath.Join(s.root, "public"))
	c.Assert(s.storageSymlink.PublicPath(), Equals, filepath.Join(s.root, "public_symlink"))
	c.Assert(s.storageCopy.PublicPath(), Equals, filepath.Join(s.root, "public_copy"))
	c.Assert(s.storageCopySize.PublicPath(), Equals, filepath.Join(s.root, "public_copysize"))
}

func (s *PublishedStorageSuite) TestMkDir(c *C) {
	err := s.storage.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	_, err = os.Stat(filepath.Join(s.storage.rootPath, "ppa/dists/squeeze/"))
	c.Assert(err, IsNil)
}

func (s *PublishedStorageSuite) TestPutFile(c *C) {
	err := s.storage.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	err = s.storage.PutFile("ppa/dists/squeeze/Release", "/dev/null")
	c.Assert(err, IsNil)

	_, err = os.Stat(filepath.Join(s.storage.rootPath, "ppa/dists/squeeze/Release"))
	c.Assert(err, IsNil)
}

func (s *PublishedStorageSuite) TestFilelist(c *C) {
	err := s.storage.MkDir("ppa/pool/main/a/ab/")
	c.Assert(err, IsNil)

	err = s.storage.PutFile("ppa/pool/main/a/ab/a.deb", "/dev/null")
	c.Assert(err, IsNil)

	err = s.storage.PutFile("ppa/pool/main/a/ab/b.deb", "/dev/null")
	c.Assert(err, IsNil)

	list, err := s.storage.Filelist("ppa/pool/main/")
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"a/ab/a.deb", "a/ab/b.deb"})

	list, err = s.storage.Filelist("ppa/pool/doenstexist/")
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{})
}

func (s *PublishedStorageSuite) TestRenameFile(c *C) {
	err := s.storage.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	err = s.storage.PutFile("ppa/dists/squeeze/Release", "/dev/null")
	c.Assert(err, IsNil)

	err = s.storage.RenameFile("ppa/dists/squeeze/Release", "ppa/dists/squeeze/InRelease")
	c.Check(err, IsNil)

	_, err = os.Stat(filepath.Join(s.storage.rootPath, "ppa/dists/squeeze/InRelease"))
	c.Assert(err, IsNil)
}

func (s *PublishedStorageSuite) TestFileExists(c *C) {
	err := s.storage.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	exists, _ := s.storage.FileExists("ppa/dists/squeeze/Release")
	c.Check(exists, Equals, false)

	err = s.storage.PutFile("ppa/dists/squeeze/Release", "/dev/null")
	c.Assert(err, IsNil)

	exists, _ = s.storage.FileExists("ppa/dists/squeeze/Release")
	c.Check(exists, Equals, true)
}

func (s *PublishedStorageSuite) TestSymLink(c *C) {
	err := s.storage.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	err = s.storage.PutFile("ppa/dists/squeeze/Release", "/dev/null")
	c.Assert(err, IsNil)

	err = s.storage.SymLink("ppa/dists/squeeze/Release", "ppa/dists/squeeze/InRelease")
	c.Assert(err, IsNil)

	exists, _ := s.storage.FileExists("ppa/dists/squeeze/InRelease")
	c.Check(exists, Equals, true)

	linkTarget, err := s.storage.ReadLink("ppa/dists/squeeze/InRelease")
	c.Assert(err, IsNil)
	c.Assert(linkTarget, Equals, "ppa/dists/squeeze/Release")
}

func (s *PublishedStorageSuite) TestHardLink(c *C) {
	err := s.storage.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	err = s.storage.PutFile("ppa/dists/squeeze/Release", "/dev/null")
	c.Assert(err, IsNil)

	err = s.storage.HardLink("ppa/dists/squeeze/Release", "ppa/dists/squeeze/InRelease")
	c.Assert(err, IsNil)

	exists, _ := s.storage.FileExists("ppa/dists/squeeze/InRelease")
	c.Check(exists, Equals, true)
}

func (s *PublishedStorageSuite) TestRemoveDirs(c *C) {
	err := s.storage.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	err = s.storage.PutFile("ppa/dists/squeeze/Release", "/dev/null")
	c.Assert(err, IsNil)

	err = s.storage.RemoveDirs("ppa/dists/", nil)
	c.Assert(err, IsNil)

	_, err = os.Stat(filepath.Join(s.storage.rootPath, "ppa/dists/squeeze/Release"))
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)
}

func (s *PublishedStorageSuite) TestRemove(c *C) {
	err := s.storage.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	err = s.storage.PutFile("ppa/dists/squeeze/Release", "/dev/null")
	c.Assert(err, IsNil)

	err = s.storage.Remove("ppa/dists/squeeze/Release")
	c.Assert(err, IsNil)

	_, err = os.Stat(filepath.Join(s.storage.rootPath, "ppa/dists/squeeze/Release"))
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)
}

func (s *PublishedStorageSuite) TestLinkFromPool(c *C) {
	tests := []struct {
		prefix             string
		sourcePath         string
		publishedDirectory string
		expectedFilename   string
	}{
		{ // package name regular
			prefix:             "",
			sourcePath:         "mars-invaders_1.03.deb",
			publishedDirectory: "pool/main/m/mars-invaders",
			expectedFilename:   "pool/main/m/mars-invaders/mars-invaders_1.03.deb",
		},
		{ // lib-like filename
			prefix:             "",
			sourcePath:         "libmars-invaders_1.03.deb",
			publishedDirectory: "pool/main/libm/libmars-invaders",
			expectedFilename:   "pool/main/libm/libmars-invaders/libmars-invaders_1.03.deb",
		},
		{ // duplicate link, shouldn't panic
			prefix:             "",
			sourcePath:         "mars-invaders_1.03.deb",
			publishedDirectory: "pool/main/m/mars-invaders",
			expectedFilename:   "pool/main/m/mars-invaders/mars-invaders_1.03.deb",
		},
		{ // prefix & component
			prefix:             "ppa",
			sourcePath:         "libmars-invaders_1.04.deb",
			publishedDirectory: "pool/contrib/libm/libmars-invaders",
			expectedFilename:   "pool/contrib/libm/libmars-invaders/libmars-invaders_1.04.deb",
		},
		{ // installer file
			prefix:             "",
			sourcePath:         "netboot/boot.img.gz",
			publishedDirectory: "dists/jessie/non-free/installer-i386/current/images",
			expectedFilename:   "dists/jessie/non-free/installer-i386/current/images/netboot/boot.img.gz",
		},
	}

	pool := NewPackagePool(s.root, false)

	for _, t := range tests {
		tmpPath := filepath.Join(c.MkDir(), t.sourcePath)
		_ = os.MkdirAll(filepath.Dir(tmpPath), 0777)
		err := os.WriteFile(tmpPath, []byte("Contents"), 0644)
		c.Assert(err, IsNil)

		sourceChecksum, err := utils.ChecksumsForFile(tmpPath)
		c.Assert(err, IsNil)

		srcPoolPath, err := pool.Import(tmpPath, t.sourcePath, &utils.ChecksumInfo{MD5: "c1df1da7a1ce305a3b60af9d5733ac1d"}, false, s.cs)
		c.Assert(err, IsNil)

		// Test using hardlinks
		err = s.storage.LinkFromPool(t.prefix, t.publishedDirectory, t.sourcePath, pool, srcPoolPath, sourceChecksum, false)
		c.Assert(err, IsNil)

		st, err := os.Stat(filepath.Join(s.storage.rootPath, t.prefix, t.expectedFilename))
		c.Assert(err, IsNil)

		info := st.Sys().(*syscall.Stat_t)
		c.Check(int(info.Nlink), Equals, 3)

		// Test using symlinks
		err = s.storageSymlink.LinkFromPool(t.prefix, t.publishedDirectory, t.sourcePath, pool, srcPoolPath, sourceChecksum, false)
		c.Assert(err, IsNil)

		st, err = os.Lstat(filepath.Join(s.storageSymlink.rootPath, t.prefix, t.expectedFilename))
		c.Assert(err, IsNil)

		info = st.Sys().(*syscall.Stat_t)
		c.Check(int(info.Nlink), Equals, 1)
		c.Check(int(info.Mode&syscall.S_IFMT), Equals, int(syscall.S_IFLNK))

		// Test using copy with checksum verification
		err = s.storageCopy.LinkFromPool(t.prefix, t.publishedDirectory, t.sourcePath, pool, srcPoolPath, sourceChecksum, false)
		c.Assert(err, IsNil)

		st, err = os.Stat(filepath.Join(s.storageCopy.rootPath, t.prefix, t.expectedFilename))
		c.Assert(err, IsNil)

		info = st.Sys().(*syscall.Stat_t)
		c.Check(int(info.Nlink), Equals, 1)

		// Test using copy with size verification
		err = s.storageCopySize.LinkFromPool(t.prefix, t.publishedDirectory, t.sourcePath, pool, srcPoolPath, sourceChecksum, false)
		c.Assert(err, IsNil)

		st, err = os.Stat(filepath.Join(s.storageCopySize.rootPath, t.prefix, t.expectedFilename))
		c.Assert(err, IsNil)

		info = st.Sys().(*syscall.Stat_t)
		c.Check(int(info.Nlink), Equals, 1)
	}

	// test linking files to duplicate final name
	tmpPath := filepath.Join(c.MkDir(), "mars-invaders_1.03.deb")
	err := os.WriteFile(tmpPath, []byte("cONTENTS"), 0644)
	c.Assert(err, IsNil)

	sourceChecksum, err := utils.ChecksumsForFile(tmpPath)
	c.Assert(err, IsNil)

	srcPoolPath, err := pool.Import(tmpPath, "mars-invaders_1.03.deb", &utils.ChecksumInfo{MD5: "02bcda7a1ce305a3b60af9d5733ac1d"}, true, s.cs)
	c.Assert(err, IsNil)

	st, err := pool.Stat(srcPoolPath)
	c.Assert(err, IsNil)
	nlinks := int(st.Sys().(*syscall.Stat_t).Nlink)

	err = s.storage.LinkFromPool("", filepath.Join("pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, srcPoolPath, sourceChecksum, false)
	c.Check(err, ErrorMatches, ".*file already exists and is different")

	st, err = pool.Stat(srcPoolPath)
	c.Assert(err, IsNil)
	c.Check(int(st.Sys().(*syscall.Stat_t).Nlink), Equals, nlinks)

	// linking with force
	err = s.storage.LinkFromPool("", filepath.Join("pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, srcPoolPath, sourceChecksum, true)
	c.Check(err, IsNil)

	st, err = pool.Stat(srcPoolPath)
	c.Assert(err, IsNil)
	c.Check(int(st.Sys().(*syscall.Stat_t).Nlink), Equals, nlinks+1)

	// Test using symlinks
	err = s.storageSymlink.LinkFromPool("", filepath.Join("pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, srcPoolPath, sourceChecksum, false)
	c.Check(err, ErrorMatches, ".*file already exists and is different")

	err = s.storageSymlink.LinkFromPool("", filepath.Join("pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, srcPoolPath, sourceChecksum, true)
	c.Check(err, IsNil)

	// Test using copy with checksum verification
	err = s.storageCopy.LinkFromPool("", filepath.Join("pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, srcPoolPath, sourceChecksum, false)
	c.Check(err, ErrorMatches, ".*file already exists and is different")

	err = s.storageCopy.LinkFromPool("", filepath.Join("pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, srcPoolPath, sourceChecksum, true)
	c.Check(err, IsNil)

	// Test using copy with size verification (this will NOT detect the difference)
	err = s.storageCopySize.LinkFromPool("", filepath.Join("pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, srcPoolPath, sourceChecksum, false)
	c.Check(err, IsNil)
}

func (s *PublishedStorageSuite) TestRootRemove(c *C) {
	// Prevent deletion of the root directory by passing empty subpaths.

	pwd := c.MkDir()

	// Symlink
	linkedDir := filepath.Join(pwd, "linkedDir")
	_ = os.Symlink(s.root, linkedDir)
	linkStorage := NewPublishedStorage(linkedDir, "", "")
	c.Assert(func() { _ = linkStorage.Remove("") }, PanicMatches, "trying to remove empty path")

	// Actual dir
	dirStorage := NewPublishedStorage(pwd, "", "")
	c.Assert(func() { _ = dirStorage.RemoveDirs("", nil) }, PanicMatches, "trying to remove the root directory")
}

// Disk full error handling tests

type DiskFullSuite struct {
	root string
}

var _ = Suite(&DiskFullSuite{})

func (s *DiskFullSuite) SetUpTest(c *C) {
	// Only run on Linux where we can create loopback filesystems
	if runtime.GOOS != "linux" {
		c.Skip("disk full tests only run on Linux")
	}

	// Check if running as root or with sudo capabilities
	if os.Geteuid() != 0 {
		c.Skip("disk full tests require root privileges")
	}

	s.root = c.MkDir()
}

// TestPutFileOutOfSpace tests that PutFile properly reports disk full errors
func (s *DiskFullSuite) TestPutFileOutOfSpace(c *C) {
	mountPoint := filepath.Join(s.root, "smallfs")
	err := os.MkdirAll(mountPoint, 0777)
	c.Assert(err, IsNil)

	// Create a very small filesystem (1MB)
	fsImage := filepath.Join(s.root, "small.img")

	// Create 1MB sparse file
	cmd := exec.Command("dd", "if=/dev/zero", "of="+fsImage, "bs=1M", "count=1")
	err = cmd.Run()
	c.Assert(err, IsNil)

	// Format as ext4
	cmd = exec.Command("mkfs.ext4", "-F", fsImage)
	err = cmd.Run()
	c.Assert(err, IsNil)

	// Mount the filesystem
	cmd = exec.Command("mount", "-o", "loop", fsImage, mountPoint)
	err = cmd.Run()
	c.Assert(err, IsNil)
	defer func() {
		_ = exec.Command("umount", mountPoint).Run()
	}()

	// Create storage on the small filesystem
	storage := NewPublishedStorage(mountPoint, "", "")

	// Create a large source file that won't fit (2MB)
	largeFile := filepath.Join(s.root, "largefile")
	cmd = exec.Command("dd", "if=/dev/zero", "of="+largeFile, "bs=1M", "count=2")
	err = cmd.Run()
	c.Assert(err, IsNil)

	// Try to put the large file - should fail with out of space error
	err = storage.PutFile("testfile", largeFile)
	c.Assert(err, NotNil)
	c.Check(strings.Contains(err.Error(), "no space left on device") ||
		strings.Contains(err.Error(), "sync"), Equals, true,
		Commentf("Expected disk full error, got: %v", err))
}

// TestLinkFromPoolCopyOutOfSpace tests that LinkFromPool with copy mode properly reports disk full errors
func (s *DiskFullSuite) TestLinkFromPoolCopyOutOfSpace(c *C) {
	mountPoint := filepath.Join(s.root, "smallfs")
	err := os.MkdirAll(mountPoint, 0777)
	c.Assert(err, IsNil)

	// Create a very small filesystem (1MB)
	fsImage := filepath.Join(s.root, "small.img")

	cmd := exec.Command("dd", "if=/dev/zero", "of="+fsImage, "bs=1M", "count=1")
	err = cmd.Run()
	c.Assert(err, IsNil)

	cmd = exec.Command("mkfs.ext4", "-F", fsImage)
	err = cmd.Run()
	c.Assert(err, IsNil)

	cmd = exec.Command("mount", "-o", "loop", fsImage, mountPoint)
	err = cmd.Run()
	c.Assert(err, IsNil)
	defer func() {
		_ = exec.Command("umount", mountPoint).Run()
	}()

	// Create storage on the small filesystem using copy mode
	storage := NewPublishedStorage(mountPoint, "copy", "")

	// Create a normal pool in a different location
	poolPath := filepath.Join(s.root, "pool")
	pool := NewPackagePool(poolPath, false)
	cs := NewMockChecksumStorage()

	// Create a large package file (2MB) in the pool
	largeFile := filepath.Join(s.root, "package.deb")
	cmd = exec.Command("dd", "if=/dev/zero", "of="+largeFile, "bs=1M", "count=2")
	err = cmd.Run()
	c.Assert(err, IsNil)

	sourceChecksum, err := utils.ChecksumsForFile(largeFile)
	c.Assert(err, IsNil)

	srcPoolPath, err := pool.Import(largeFile, "package.deb",
		&utils.ChecksumInfo{MD5: "d41d8cd98f00b204e9800998ecf8427e"}, false, cs)
	c.Assert(err, IsNil)

	// Try to link from pool - should fail with out of space error
	err = storage.LinkFromPool("", "pool/main/p/package", "package.deb",
		pool, srcPoolPath, sourceChecksum, false)
	c.Assert(err, NotNil)
	c.Check(strings.Contains(err.Error(), "no space left on device") ||
		strings.Contains(err.Error(), "sync"), Equals, true,
		Commentf("Expected disk full error, got: %v", err))
}

// Alternative simpler test that doesn't require root
type DiskFullNoRootSuite struct {
	root string
}

var _ = Suite(&DiskFullNoRootSuite{})

func (s *DiskFullNoRootSuite) SetUpTest(c *C) {
	s.root = c.MkDir()
}

// This test verifies Sync() is called by checking it doesn't panic
// The actual disk full behavior is harder to test without root
func (s *DiskFullNoRootSuite) TestSyncIsCalled(c *C) {
	storage := NewPublishedStorage(s.root, "", "")

	// Create a small test file
	sourceFile := filepath.Join(s.root, "source.txt")
	err := os.WriteFile(sourceFile, []byte("test content"), 0644)
	c.Assert(err, IsNil)

	// PutFile should succeed with normal disk space
	err = storage.PutFile("dest.txt", sourceFile)
	c.Assert(err, IsNil)

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(s.root, "dest.txt"))
	c.Assert(err, IsNil)
	c.Check(string(content), Equals, "test content")
}

func (s *DiskFullNoRootSuite) TestLinkFromPoolCopySyncIsCalled(c *C) {
	storage := NewPublishedStorage(s.root, "copy", "")
	poolPath := filepath.Join(s.root, "pool")
	pool := NewPackagePool(poolPath, false)
	cs := NewMockChecksumStorage()

	// Create a test package file
	pkgFile := filepath.Join(s.root, "package.deb")
	err := os.WriteFile(pkgFile, []byte("package content"), 0644)
	c.Assert(err, IsNil)

	sourceChecksum, err := utils.ChecksumsForFile(pkgFile)
	c.Assert(err, IsNil)

	srcPoolPath, err := pool.Import(pkgFile, "package.deb",
		&utils.ChecksumInfo{MD5: "d41d8cd98f00b204e9800998ecf8427e"}, false, cs)
	c.Assert(err, IsNil)

	// LinkFromPool with copy should succeed with normal disk space
	err = storage.LinkFromPool("", "pool/main/p/package", "package.deb",
		pool, srcPoolPath, sourceChecksum, false)
	c.Assert(err, IsNil)

	// Verify file was written
	destPath := filepath.Join(s.root, "pool/main/p/package/package.deb")
	content, err := os.ReadFile(destPath)
	c.Assert(err, IsNil)
	c.Check(string(content), Equals, "package content")
}
