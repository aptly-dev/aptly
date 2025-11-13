package files

import (
	"bytes"
	"errors"
	"io"
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

type fakeProgress struct{ bytes.Buffer }

func (p *fakeProgress) Start()    {}
func (p *fakeProgress) Shutdown() {}
func (p *fakeProgress) Flush()    {}
func (p *fakeProgress) InitBar(count int64, isBytes bool, barType aptly.BarType) {
}
func (p *fakeProgress) ShutdownBar()     {}
func (p *fakeProgress) AddBar(count int) {}
func (p *fakeProgress) SetBar(count int) {}
func (p *fakeProgress) Printf(msg string, a ...interface{}) {
}
func (p *fakeProgress) ColoredPrintf(msg string, a ...interface{}) {
}
func (p *fakeProgress) PrintfStdErr(msg string, a ...interface{}) {
}

type fakeRSC struct {
	*bytes.Reader
	closeErr error
}

func (r *fakeRSC) Close() error { return r.closeErr }

type fakePool struct {
	sizeErr error
	openFn  func(string) (aptly.ReadSeekerCloser, error)
}

type fakeLocalPool struct {
	fakePool
	statErr error
}

func (p *fakeLocalPool) Stat(path string) (os.FileInfo, error) { return nil, p.statErr }
func (p *fakeLocalPool) GenerateTempPath(filename string) (string, error) {
	return "", nil
}
func (p *fakeLocalPool) Link(path, dstPath string) error    { return nil }
func (p *fakeLocalPool) Symlink(path, dstPath string) error { return nil }
func (p *fakeLocalPool) FullPath(path string) string        { return path }

func (p *fakePool) Verify(poolPath, basename string, checksums *utils.ChecksumInfo, checksumStorage aptly.ChecksumStorage) (string, bool, error) {
	return "", false, nil
}

func (p *fakePool) Import(srcPath, basename string, checksums *utils.ChecksumInfo, move bool, storage aptly.ChecksumStorage) (string, error) {
	return "", nil
}

func (p *fakePool) LegacyPath(filename string, checksums *utils.ChecksumInfo) (string, error) {
	return "", nil
}

func (p *fakePool) Size(path string) (int64, error) {
	if p.sizeErr != nil {
		return 0, p.sizeErr
	}
	return int64(len(path)), nil
}

func (p *fakePool) Open(path string) (aptly.ReadSeekerCloser, error) {
	if p.openFn != nil {
		return p.openFn(path)
	}
	return nil, io.EOF
}

func (p *fakePool) FilepathList(progress aptly.Progress) ([]string, error) { return nil, nil }
func (p *fakePool) Remove(path string) (int64, error)                      { return 0, nil }

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

func (s *PublishedStorageSuite) TestPutFileReturnsErrorIfSourceMissing(c *C) {
	err := s.storage.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	err = s.storage.PutFile("ppa/dists/squeeze/Release", filepath.Join(s.root, "no-such-file"))
	c.Assert(err, NotNil)
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

func (s *PublishedStorageSuite) TestReadLinkReturnsErrorOnMissingPath(c *C) {
	_, err := s.storage.ReadLink("does/not/exist")
	c.Assert(err, NotNil)
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

func (s *PublishedStorageSuite) TestRemoveDirsWithProgress(c *C) {
	err := s.storage.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	err = s.storage.PutFile("ppa/dists/squeeze/Release", "/dev/null")
	c.Assert(err, IsNil)

	p := &fakeProgress{}
	err = s.storage.RemoveDirs("ppa/dists/", p)
	c.Assert(err, IsNil)
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

// DiskFullSuite uses a loopback mount; requires Linux + root.

type DiskFullSuite struct {
	root string
}

var _ = Suite(&DiskFullSuite{})

func (s *DiskFullSuite) SetUpTest(c *C) {
	if runtime.GOOS != "linux" {
		c.Skip("disk full tests only run on Linux")
	}
	if os.Geteuid() != 0 {
		c.Skip("disk full tests require root privileges")
	}

	s.root = c.MkDir()
}

func (s *DiskFullSuite) TestPutFileOutOfSpace(c *C) {
	mountPoint := filepath.Join(s.root, "smallfs")
	err := os.MkdirAll(mountPoint, 0777)
	c.Assert(err, IsNil)
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

	storage := NewPublishedStorage(mountPoint, "", "")
	largeFile := filepath.Join(s.root, "largefile")
	cmd = exec.Command("dd", "if=/dev/zero", "of="+largeFile, "bs=1M", "count=2")
	err = cmd.Run()
	c.Assert(err, IsNil)

	err = storage.PutFile("testfile", largeFile)
	c.Assert(err, NotNil)
	c.Check(strings.Contains(err.Error(), "no space left on device") ||
		strings.Contains(err.Error(), "sync"), Equals, true,
		Commentf("Expected disk full error, got: %v", err))
}

func (s *DiskFullSuite) TestLinkFromPoolCopyOutOfSpace(c *C) {
	mountPoint := filepath.Join(s.root, "smallfs")
	err := os.MkdirAll(mountPoint, 0777)
	c.Assert(err, IsNil)
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

	storage := NewPublishedStorage(mountPoint, "copy", "")

	poolPath := filepath.Join(s.root, "pool")
	pool := NewPackagePool(poolPath, false)
	cs := NewMockChecksumStorage()

	largeFile := filepath.Join(s.root, "package.deb")
	cmd = exec.Command("dd", "if=/dev/zero", "of="+largeFile, "bs=1M", "count=2")
	err = cmd.Run()
	c.Assert(err, IsNil)

	sourceChecksum, err := utils.ChecksumsForFile(largeFile)
	c.Assert(err, IsNil)

	srcPoolPath, err := pool.Import(largeFile, "package.deb",
		&utils.ChecksumInfo{MD5: "d41d8cd98f00b204e9800998ecf8427e"}, false, cs)
	c.Assert(err, IsNil)

	err = storage.LinkFromPool("", "pool/main/p/package", "package.deb",
		pool, srcPoolPath, sourceChecksum, false)
	c.Assert(err, NotNil)
	c.Check(strings.Contains(err.Error(), "no space left on device") ||
		strings.Contains(err.Error(), "sync"), Equals, true,
		Commentf("Expected disk full error, got: %v", err))
}

type DiskFullNoRootSuite struct {
	root string
}

var _ = Suite(&DiskFullNoRootSuite{})

func (s *DiskFullNoRootSuite) SetUpTest(c *C) {
	s.root = c.MkDir()
}

func (s *DiskFullNoRootSuite) TestSyncIsCalled(c *C) {
	storage := NewPublishedStorage(s.root, "", "")
	sourceFile := filepath.Join(s.root, "source.txt")
	err := os.WriteFile(sourceFile, []byte("test content"), 0644)
	c.Assert(err, IsNil)
	err = storage.PutFile("dest.txt", sourceFile)
	c.Assert(err, IsNil)
	content, err := os.ReadFile(filepath.Join(s.root, "dest.txt"))
	c.Assert(err, IsNil)
	c.Check(string(content), Equals, "test content")
}

func (s *DiskFullNoRootSuite) TestLinkFromPoolCopySyncIsCalled(c *C) {
	storage := NewPublishedStorage(s.root, "copy", "")
	poolPath := filepath.Join(s.root, "pool")
	pool := NewPackagePool(poolPath, false)
	cs := NewMockChecksumStorage()

	pkgFile := filepath.Join(s.root, "package.deb")
	err := os.WriteFile(pkgFile, []byte("package content"), 0644)
	c.Assert(err, IsNil)

	sourceChecksum, err := utils.ChecksumsForFile(pkgFile)
	c.Assert(err, IsNil)

	srcPoolPath, err := pool.Import(pkgFile, "package.deb",
		&utils.ChecksumInfo{MD5: "d41d8cd98f00b204e9800998ecf8427e"}, false, cs)
	c.Assert(err, IsNil)

	err = storage.LinkFromPool("", "pool/main/p/package", "package.deb",
		pool, srcPoolPath, sourceChecksum, false)
	c.Assert(err, IsNil)

	destPath := filepath.Join(s.root, "pool/main/p/package/package.deb")
	content, err := os.ReadFile(destPath)
	c.Assert(err, IsNil)
	c.Check(string(content), Equals, "package content")
}

func (s *DiskFullNoRootSuite) TestPutFileSyncErrorIsReturned(c *C) {
	storage := NewPublishedStorage(s.root, "", "")

	sourceFile := filepath.Join(s.root, "source-syncfail.txt")
	err := os.WriteFile(sourceFile, []byte("test content"), 0644)
	c.Assert(err, IsNil)

	oldSyncFile := syncFile
	syncFile = func(_ *os.File) error { return syscall.ENOSPC }
	defer func() { syncFile = oldSyncFile }()

	err = storage.PutFile("dest-syncfail.txt", sourceFile)
	c.Assert(err, NotNil)
	c.Check(strings.Contains(err.Error(), "error syncing file"), Equals, true)
}

func (s *DiskFullNoRootSuite) TestLinkFromPoolCopySyncErrorIsReturned(c *C) {
	storage := NewPublishedStorage(s.root, "copy", "")
	poolPath := filepath.Join(s.root, "pool")
	pool := NewPackagePool(poolPath, false)
	cs := NewMockChecksumStorage()

	pkgFile := filepath.Join(s.root, "package-syncfail.deb")
	err := os.WriteFile(pkgFile, []byte("package content"), 0644)
	c.Assert(err, IsNil)

	sourceChecksum, err := utils.ChecksumsForFile(pkgFile)
	c.Assert(err, IsNil)

	srcPoolPath, err := pool.Import(pkgFile, "package-syncfail.deb",
		&utils.ChecksumInfo{MD5: "d41d8cd98f00b204e9800998ecf8427e"}, false, cs)
	c.Assert(err, IsNil)

	oldSyncFile := syncFile
	syncFile = func(_ *os.File) error { return syscall.ENOSPC }
	defer func() { syncFile = oldSyncFile }()

	err = storage.LinkFromPool("", "pool/main/p/package", "package-syncfail.deb",
		pool, srcPoolPath, sourceChecksum, false)
	c.Assert(err, NotNil)
	c.Check(strings.Contains(err.Error(), "error syncing file"), Equals, true)
}

func (s *DiskFullNoRootSuite) TestGetFileLockReusesMutex(c *C) {
	a := getFileLock(filepath.Join(s.root, "a"))
	b := getFileLock(filepath.Join(s.root, "a"))
	c.Check(a == b, Equals, true)

	c1 := getFileLock(filepath.Join(s.root, "c1"))
	c2 := getFileLock(filepath.Join(s.root, "c2"))
	c.Check(c1 == c2, Equals, false)
}

func (s *DiskFullNoRootSuite) TestPutFileFailsIfDestinationDirMissing(c *C) {
	storage := NewPublishedStorage(s.root, "", "")

	sourceFile := filepath.Join(s.root, "src.txt")
	err := os.WriteFile(sourceFile, []byte("x"), 0644)
	c.Assert(err, IsNil)

	err = storage.PutFile("missingdir/dest.txt", sourceFile)
	c.Assert(err, NotNil)
}

func (s *DiskFullNoRootSuite) TestLinkFromPoolRejectsNonLocalPoolForHardlink(c *C) {
	storage := NewPublishedStorage(s.root, "", "")
	pool := &fakePool{}

	err := storage.LinkFromPool("", "pool/main/p/pkg", "x.deb", pool, "x", utils.ChecksumInfo{MD5: "x"}, false)
	c.Assert(err, NotNil)
	c.Check(strings.Contains(err.Error(), "cannot link"), Equals, true)
}

func (s *DiskFullNoRootSuite) TestLinkFromPoolCopyReturnsErrorIfOpenFails(c *C) {
	storage := NewPublishedStorage(s.root, "copy", "")
	pool := &fakePool{openFn: func(string) (aptly.ReadSeekerCloser, error) { return nil, io.ErrUnexpectedEOF }}

	err := storage.LinkFromPool("", "pool/main/p/pkg", "x.deb", pool, "x", utils.ChecksumInfo{MD5: "x"}, false)
	c.Assert(err, NotNil)
}

func (s *DiskFullNoRootSuite) TestLinkFromPoolCopyReturnsErrorIfReaderCloseFails(c *C) {
	storage := NewPublishedStorage(s.root, "copy", "")

	pool := &fakePool{openFn: func(string) (aptly.ReadSeekerCloser, error) {
		return &fakeRSC{Reader: bytes.NewReader([]byte("data")), closeErr: io.ErrClosedPipe}, nil
	}}

	err := storage.LinkFromPool("", "pool/main/p/pkg", "x.deb", pool, "x", utils.ChecksumInfo{MD5: "x"}, false)
	c.Assert(err, NotNil)
	c.Check(err, Equals, io.ErrClosedPipe)
}

func (s *DiskFullNoRootSuite) TestLinkFromPoolCopyReturnsErrorIfSizeFailsWhenDestExists(c *C) {
	storage := NewPublishedStorage(s.root, "copy", "size")
	pool := &fakePool{sizeErr: io.ErrUnexpectedEOF, openFn: func(string) (aptly.ReadSeekerCloser, error) {
		return &fakeRSC{Reader: bytes.NewReader([]byte("data")), closeErr: nil}, nil
	}}

	destDir := filepath.Join(s.root, "pool/main/p/pkg")
	c.Assert(os.MkdirAll(destDir, 0777), IsNil)
	c.Assert(os.WriteFile(filepath.Join(destDir, "x.deb"), []byte("old"), 0644), IsNil)

	err := storage.LinkFromPool("", "pool/main/p/pkg", "x.deb", pool, "x", utils.ChecksumInfo{MD5: "x"}, false)
	c.Assert(err, NotNil)
	c.Check(err, Equals, io.ErrUnexpectedEOF)
}

func (s *DiskFullNoRootSuite) TestLinkFromPoolCopyChecksumReturnsErrorIfDstMD5Fails(c *C) {
	storage := NewPublishedStorage(s.root, "copy", "")
	pool := &fakePool{openFn: func(string) (aptly.ReadSeekerCloser, error) {
		return &fakeRSC{Reader: bytes.NewReader([]byte("data")), closeErr: nil}, nil
	}}

	// Make destinationPath a directory so MD5ChecksumForFile fails.
	destDir := filepath.Join(s.root, "pool/main/p/pkg")
	c.Assert(os.MkdirAll(destDir, 0777), IsNil)
	c.Assert(os.MkdirAll(filepath.Join(destDir, "x.deb"), 0777), IsNil)

	err := storage.LinkFromPool("", "pool/main/p/pkg", "x.deb", pool, "x", utils.ChecksumInfo{MD5: "x"}, false)
	c.Assert(err, NotNil)
}

func (s *DiskFullNoRootSuite) TestLinkFromPoolHardlinkReturnsErrorIfStatFailsWhenDestExists(c *C) {
	storage := NewPublishedStorage(c.MkDir(), "hardlink", "")
	pool := &fakeLocalPool{statErr: errors.New("stat failed")}

	destDir := filepath.Join(storage.rootPath, "pool", "main", "p", "pkg")
	c.Assert(os.MkdirAll(destDir, 0777), IsNil)
	c.Assert(os.WriteFile(filepath.Join(destDir, "x.deb"), []byte("x"), 0644), IsNil)

	err := storage.LinkFromPool("", "pool/main/p/pkg", "x.deb", pool, "x", utils.ChecksumInfo{MD5: "x"}, false)
	c.Assert(err, NotNil)
}
