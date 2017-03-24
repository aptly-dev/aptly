package files

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"github.com/smira/aptly/utils"

	. "gopkg.in/check.v1"
)

type PublishedStorageSuite struct {
	root            string
	storage         *PublishedStorage
	storageSymlink  *PublishedStorage
	storageCopy     *PublishedStorage
	storageCopySize *PublishedStorage
}

var _ = Suite(&PublishedStorageSuite{})

func (s *PublishedStorageSuite) SetUpTest(c *C) {
	s.root = c.MkDir()
	s.storage = NewPublishedStorage(filepath.Join(s.root, "public"), "", "")
	s.storageSymlink = NewPublishedStorage(filepath.Join(s.root, "public_symlink"), "symlink", "")
	s.storageCopy = NewPublishedStorage(filepath.Join(s.root, "public_copy"), "copy", "")
	s.storageCopySize = NewPublishedStorage(filepath.Join(s.root, "public_copysize"), "copy", "size")
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

func (s *PublishedStorageSuite) TestRemoveDirs(c *C) {
	err := s.storage.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	err = s.storage.PutFile("ppa/dists/squeeze/Release", "/dev/null")
	c.Assert(err, IsNil)

	err = s.storage.RemoveDirs("ppa/dists/", nil)

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

	_, err = os.Stat(filepath.Join(s.storage.rootPath, "ppa/dists/squeeze/Release"))
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)
}

func (s *PublishedStorageSuite) TestLinkFromPool(c *C) {
	tests := []struct {
		prefix           string
		component        string
		sourcePath       string
		poolDirectory    string
		expectedFilename string
	}{
		{ // package name regular
			prefix:           "",
			component:        "main",
			sourcePath:       "pool/01/ae/mars-invaders_1.03.deb",
			poolDirectory:    "m/mars-invaders",
			expectedFilename: "pool/main/m/mars-invaders/mars-invaders_1.03.deb",
		},
		{ // lib-like filename
			prefix:           "",
			component:        "main",
			sourcePath:       "pool/01/ae/libmars-invaders_1.03.deb",
			poolDirectory:    "libm/libmars-invaders",
			expectedFilename: "pool/main/libm/libmars-invaders/libmars-invaders_1.03.deb",
		},
		{ // duplicate link, shouldn't panic
			prefix:           "",
			component:        "main",
			sourcePath:       "pool/01/ae/mars-invaders_1.03.deb",
			poolDirectory:    "m/mars-invaders",
			expectedFilename: "pool/main/m/mars-invaders/mars-invaders_1.03.deb",
		},
		{ // prefix & component
			prefix:           "ppa",
			component:        "contrib",
			sourcePath:       "pool/01/ae/libmars-invaders_1.04.deb",
			poolDirectory:    "libm/libmars-invaders",
			expectedFilename: "pool/contrib/libm/libmars-invaders/libmars-invaders_1.04.deb",
		},
	}

	pool := NewPackagePool(s.root)

	for _, t := range tests {
		t.sourcePath = filepath.Join(s.root, t.sourcePath)

		err := os.MkdirAll(filepath.Dir(t.sourcePath), 0755)
		c.Assert(err, IsNil)

		err = ioutil.WriteFile(t.sourcePath, []byte("Contents"), 0644)
		c.Assert(err, IsNil)

		sourceChecksum, err := utils.ChecksumsForFile(t.sourcePath)
		c.Assert(err, IsNil)

		err = s.storage.LinkFromPool(filepath.Join(t.prefix, "pool", t.component, t.poolDirectory), pool, t.sourcePath, sourceChecksum, false)
		c.Assert(err, IsNil)

		st, err := os.Stat(filepath.Join(s.storage.rootPath, t.prefix, t.expectedFilename))
		c.Assert(err, IsNil)

		info := st.Sys().(*syscall.Stat_t)
		c.Check(int(info.Nlink), Equals, 2)

		// Test using symlinks
		err = s.storageSymlink.LinkFromPool(filepath.Join(t.prefix, "pool", t.component, t.poolDirectory), pool, t.sourcePath, sourceChecksum, false)
		c.Assert(err, IsNil)

		st, err = os.Stat(filepath.Join(s.storageSymlink.rootPath, t.prefix, t.expectedFilename))
		c.Assert(err, IsNil)

		info = st.Sys().(*syscall.Stat_t)
		c.Check(int(info.Nlink), Equals, 2)

		// Test using copy with checksum verification
		err = s.storageCopy.LinkFromPool(filepath.Join(t.prefix, "pool", t.component, t.poolDirectory), pool, t.sourcePath, sourceChecksum, false)
		c.Assert(err, IsNil)

		st, err = os.Stat(filepath.Join(s.storageCopy.rootPath, t.prefix, t.expectedFilename))
		c.Assert(err, IsNil)

		info = st.Sys().(*syscall.Stat_t)
		c.Check(int(info.Nlink), Equals, 1)

		// Test using copy with size verification
		err = s.storageCopySize.LinkFromPool(filepath.Join(t.prefix, "pool", t.component, t.poolDirectory), pool, t.sourcePath, sourceChecksum, false)
		c.Assert(err, IsNil)

		st, err = os.Stat(filepath.Join(s.storageCopySize.rootPath, t.prefix, t.expectedFilename))
		c.Assert(err, IsNil)

		info = st.Sys().(*syscall.Stat_t)
		c.Check(int(info.Nlink), Equals, 1)
	}

	// test linking files to duplicate final name
	sourcePath := filepath.Join(s.root, "pool/02/bc/mars-invaders_1.03.deb")
	err := os.MkdirAll(filepath.Dir(sourcePath), 0755)
	c.Assert(err, IsNil)

	// use same size to ensure copy with size check will fail on this one
	err = ioutil.WriteFile(sourcePath, []byte("cONTENTS"), 0644)
	c.Assert(err, IsNil)

	sourceChecksum, err := utils.ChecksumsForFile(sourcePath)
	c.Assert(err, IsNil)

	err = s.storage.LinkFromPool(filepath.Join("", "pool", "main", "m/mars-invaders"), pool, sourcePath, sourceChecksum, false)
	c.Check(err, ErrorMatches, ".*file already exists and is different")

	st, err := os.Stat(sourcePath)
	c.Assert(err, IsNil)

	info := st.Sys().(*syscall.Stat_t)
	c.Check(int(info.Nlink), Equals, 1)

	// linking with force
	err = s.storage.LinkFromPool(filepath.Join("", "pool", "main", "m/mars-invaders"), pool, sourcePath, sourceChecksum, true)
	c.Check(err, IsNil)

	st, err = os.Stat(sourcePath)
	c.Assert(err, IsNil)

	info = st.Sys().(*syscall.Stat_t)
	c.Check(int(info.Nlink), Equals, 2)

	// Test using symlinks
	err = s.storageSymlink.LinkFromPool(filepath.Join("", "pool", "main", "m/mars-invaders"), pool, sourcePath, sourceChecksum, false)
	c.Check(err, ErrorMatches, ".*file already exists and is different")

	err = s.storageSymlink.LinkFromPool(filepath.Join("", "pool", "main", "m/mars-invaders"), pool, sourcePath, sourceChecksum, true)
	c.Check(err, IsNil)

	// Test using copy with checksum verification
	err = s.storageCopy.LinkFromPool(filepath.Join("", "pool", "main", "m/mars-invaders"), pool, sourcePath, sourceChecksum, false)
	c.Check(err, ErrorMatches, ".*file already exists and is different")

	err = s.storageCopy.LinkFromPool(filepath.Join("", "pool", "main", "m/mars-invaders"), pool, sourcePath, sourceChecksum, true)
	c.Check(err, IsNil)

	// Test using copy with size verification (this will NOT detect the difference)
	err = s.storageCopySize.LinkFromPool(filepath.Join("", "pool", "main", "m/mars-invaders"), pool, sourcePath, sourceChecksum, false)
	c.Check(err, IsNil)
}
