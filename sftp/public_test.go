package sftp

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/kr/fs"
	"github.com/smira/aptly/files"

	. "gopkg.in/check.v1"
)

type PublishedStorageSuite struct {
	tmpdir                   string
	root                     string
	storage, prefixedStorage *PublishedStorage
}

var _ = Suite(&PublishedStorageSuite{})

type MockSftp struct {
	root  string
	paths []string
	// NB: this thing needs to expect that the paths are prefixed correctly
	//     as such all functions need to check this!
}

func (s *MockSftp) CreatePath(path string) (sftpFileInterface, error) {
	if !strings.HasPrefix(path, s.root) {
		return nil, errors.New("invalid prefix " + path)
	}
	return os.Create(path)
}

func (s *MockSftp) Mkdir(path string) error {
	if !strings.HasPrefix(path, s.root) {
		return errors.New("invalid prefix " + path)
	}
	return os.Mkdir(path, 0700)
}

func (s *MockSftp) Remove(path string) error {
	if !strings.HasPrefix(path, s.root) {
		return errors.New("invalid prefix " + path)
	}
	return os.Remove(path)
}

func (s *MockSftp) Rename(oldname, newname string) error {
	if !strings.HasPrefix(oldname, s.root) {
		return errors.New("invalid prefix " + oldname)
	}
	if !strings.HasPrefix(newname, s.root) {
		return errors.New("invalid prefix " + newname)
	}
	return os.Rename(oldname, newname)
}

func (s *MockSftp) Stat(p string) (os.FileInfo, error) {
	if !strings.HasPrefix(p, s.root) {
		return nil, errors.New("invalid prefix " + p)
	}
	return os.Stat(p)
}

func (s *MockSftp) Walk(root string) *fs.Walker {
	if !strings.HasPrefix(root, s.root) {
		return nil
	}
	return fs.Walk(root)
}

type MockSSH struct {
}

func (s *PublishedStorageSuite) SetUpTest(c *C) {
	dir, err := ioutil.TempDir("", "example")
	c.Assert(err, IsNil)
	// Change dir to make sure we don't write out of order
	err = os.Chdir(dir)
	c.Assert(err, IsNil)
	s.tmpdir = dir
	s.root = dir
	fmt.Fprintf(os.Stderr, "created tmpdir %v\n", dir)

	url, err := url.Parse("sftp://localhost" + dir)
	c.Assert(err, IsNil)

	ssh := MockSSH{}
	sftp := MockSftp{root: dir}

	s.storage, err = NewPublishedStorageInternal(url, &ssh, &sftp)
	c.Assert(err, IsNil)
}

func (s *PublishedStorageSuite) TearDownTest(c *C) {
	fmt.Fprintf(os.Stderr, "deleting tmpdir %v\n", s.tmpdir)
	os.RemoveAll(s.tmpdir)
}

func (s *PublishedStorageSuite) TestFilelist(c *C) {
	os.Mkdir(s.tmpdir+"/TestFilelist", 0700)
	os.Mkdir(s.tmpdir+"/TestFilelist/empty", 0700)
	os.Mkdir(s.tmpdir+"/TestFilelist/not-empty", 0700)
	os.Mkdir(s.tmpdir+"/TestFilelist/not-empty/dir", 0700)
	ioutil.WriteFile(s.tmpdir+"/TestFilelist/not-empty/dir/file", []byte(""), 0600)
	ioutil.WriteFile(s.tmpdir+"/TestFilelist/not-empty/file", []byte(""), 0600)

	list, err := s.storage.Filelist("TestFilelist")
	c.Assert(err, IsNil)
	c.Assert(list, DeepEquals, []string{"not-empty/dir/file", "not-empty/file"})
	fmt.Fprintf(os.Stderr, "axios %v\n", list)

	list, err = s.storage.Filelist("TestFilelist/not-empty")
	c.Assert(err, IsNil)
	c.Assert(list, DeepEquals, []string{"dir/file", "file"})

	list, err = s.storage.Filelist("TestFilelist/empty")
	c.Assert(err, IsNil)
	c.Assert(list, DeepEquals, []string{})
}

func (s *PublishedStorageSuite) TestRenameFile(c *C) {
	err := s.storage.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	err = s.storage.PutFile("ppa/dists/squeeze/Release", "/dev/null")
	c.Assert(err, IsNil)

	err = s.storage.RenameFile("ppa/dists/squeeze/Release", "ppa/dists/squeeze/InRelease")
	c.Check(err, IsNil)

	_, err = os.Stat(filepath.Join(s.root, "ppa/dists/squeeze/InRelease"))
	c.Assert(err, IsNil)

	// Create a new Release and move it to InRelease, this tests
	// what happens when renaming a file that already exists.
	// Which should work just fine.
	err = s.storage.PutFile("ppa/dists/squeeze/Release", "/dev/null")
	c.Assert(err, IsNil)

	err = s.storage.RenameFile("ppa/dists/squeeze/Release", "ppa/dists/squeeze/InRelease")
	c.Check(err, IsNil)
}

func (s *PublishedStorageSuite) TestRemove(c *C) {
	file := filepath.Join(s.tmpdir, "TestRemove")
	fmt.Fprintf(os.Stderr, "atros %v\n", file)
	err := ioutil.WriteFile(file, []byte("test"), 0600)
	c.Assert(err, IsNil)
	_, err = os.Stat(file)
	c.Assert(err, IsNil)

	err = s.storage.Remove("TestRemove")
	c.Check(err, IsNil)

	_, err = os.Stat(file)
	c.Assert(os.IsNotExist(err), Equals, true)
}

func (s *PublishedStorageSuite) TestRemoveDirs(c *C) {
	os.Mkdir(filepath.Join(s.tmpdir, "TestRemoveDirs"), 0700)
	os.Mkdir(filepath.Join(s.tmpdir, "TestRemoveDirs/a"), 0700)
	os.Mkdir(filepath.Join(s.tmpdir, "TestRemoveDirs/a/a"), 0700)
	os.Mkdir(filepath.Join(s.tmpdir, "TestRemoveDirs/a/b"), 0700)
	os.Mkdir(filepath.Join(s.tmpdir, "TestRemoveDirs/a/b/a"), 0700)
	ioutil.WriteFile(filepath.Join(s.tmpdir, "TestRemoveDirs/a/b/a/file"), []byte(""), 0600)
	os.Mkdir(filepath.Join(s.tmpdir, "TestRemoveDirs/b"), 0700)

	err := s.storage.RemoveDirs("TestRemoveDirs/a", nil)
	c.Assert(err, IsNil)

	list, err := s.storage.Filelist("")
	c.Assert(err, IsNil)
	c.Assert(list, DeepEquals, []string{}) // We only had one file.

	// Make sure b is still there.
	_, err = os.Stat(filepath.Join(s.tmpdir, "TestRemoveDirs/b"))
	c.Assert(err, IsNil)

	// Make sure all of a isn't.
	_, err = os.Stat(filepath.Join(s.tmpdir, "TestRemoveDirs/a"))
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

	pool := files.NewPackagePool(s.root)

	for _, t := range tests {
		t.sourcePath = filepath.Join(s.root, t.sourcePath)

		err := os.MkdirAll(filepath.Dir(t.sourcePath), 0755)
		c.Assert(err, IsNil)

		err = ioutil.WriteFile(t.sourcePath, []byte("Contents"), 0644)
		c.Assert(err, IsNil)

		err = s.storage.LinkFromPool(filepath.Join(t.prefix, "pool", t.component, t.poolDirectory), pool, t.sourcePath, "", false)
		c.Assert(err, IsNil)

		_, err = os.Stat(filepath.Join(s.root, t.prefix, t.expectedFilename))
		c.Assert(err, IsNil)

		// info := st.Sys().(*syscall.Stat_t)
		// c.Check(int(info.Nlink), Equals, 2)
	}

	// test linking files to duplicate final name
	sourcePath := filepath.Join(s.root, "pool/02/bc/mars-invaders_1.03.deb")
	err := os.MkdirAll(filepath.Dir(sourcePath), 0755)
	c.Assert(err, IsNil)

	err = ioutil.WriteFile(sourcePath, []byte("Contents"), 0644)
	c.Assert(err, IsNil)

	// FIXME: not implemented
	// err = s.storage.LinkFromPool(filepath.Join("", "pool", "main", "m/mars-invaders"), pool, sourcePath, "", false)
	// c.Check(err, ErrorMatches, ".*file already exists and is different")

	st, err := os.Stat(sourcePath)
	c.Assert(err, IsNil)

	info := st.Sys().(*syscall.Stat_t)
	c.Check(int(info.Nlink), Equals, 1)

	// linking with force
	err = s.storage.LinkFromPool(filepath.Join("", "pool", "main", "m/mars-invaders"), pool, sourcePath, "", true)
	c.Check(err, IsNil)

	st, err = os.Stat(sourcePath)
	c.Assert(err, IsNil)

	// info = st.Sys().(*syscall.Stat_t)
	// c.Check(int(info.Nlink), Equals, 2)
}

func (s *PublishedStorageSuite) TestString(c *C) {
	exp := fmt.Sprintf("SFTP: %s", s.storage.url.String())
	c.Assert(s.storage.String(), Equals, exp)
}
