package s3

import (
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3/s3test"
	"github.com/smira/aptly/files"
	"io/ioutil"
	"os"
	"path/filepath"

	. "gopkg.in/check.v1"
)

type PublishedStorageSuite struct {
	srv                      *s3test.Server
	storage, prefixedStorage *PublishedStorage
}

var _ = Suite(&PublishedStorageSuite{})

func (s *PublishedStorageSuite) SetUpTest(c *C) {
	var err error
	s.srv, err = s3test.NewServer(&s3test.Config{})
	c.Assert(err, IsNil)
	c.Assert(s.srv, NotNil)

	auth, _ := aws.GetAuth("aa", "bb")
	s.storage, err = NewPublishedStorageRaw(auth, aws.Region{Name: "test-1", S3Endpoint: s.srv.URL(), S3LocationConstraint: true}, "test", "", "", "", "", false, true)
	c.Assert(err, IsNil)

	s.prefixedStorage, err = NewPublishedStorageRaw(auth, aws.Region{Name: "test-1", S3Endpoint: s.srv.URL(), S3LocationConstraint: true}, "test", "", "lala", "", "", false, true)
	c.Assert(err, IsNil)

	err = s.storage.s3.Bucket("test").PutBucket("private")
	c.Assert(err, IsNil)
}

func (s *PublishedStorageSuite) TearDownTest(c *C) {
	s.srv.Quit()
}

func (s *PublishedStorageSuite) TestNewPublishedStorage(c *C) {
	stor, err := NewPublishedStorage("aa", "bbb", "", "", "", "", "", "", "", false, false)
	c.Check(stor, IsNil)
	c.Check(err, ErrorMatches, "unknown region: .*")
}

func (s *PublishedStorageSuite) TestPutFile(c *C) {
	dir := c.MkDir()
	err := ioutil.WriteFile(filepath.Join(dir, "a"), []byte("welcome to s3!"), 0644)
	c.Assert(err, IsNil)

	err = s.storage.PutFile("a/b.txt", filepath.Join(dir, "a"))
	c.Check(err, IsNil)

	data, err := s.storage.bucket.Get("a/b.txt")
	c.Check(err, IsNil)
	c.Check(data, DeepEquals, []byte("welcome to s3!"))

	err = s.prefixedStorage.PutFile("a/b.txt", filepath.Join(dir, "a"))
	c.Check(err, IsNil)

	data, err = s.storage.bucket.Get("lala/a/b.txt")
	c.Check(err, IsNil)
	c.Check(data, DeepEquals, []byte("welcome to s3!"))
}

func (s *PublishedStorageSuite) TestPutFilePlusWorkaround(c *C) {
	s.storage.plusWorkaround = true

	dir := c.MkDir()
	err := ioutil.WriteFile(filepath.Join(dir, "a"), []byte("welcome to s3!"), 0644)
	c.Assert(err, IsNil)

	err = s.storage.PutFile("a/b+c.txt", filepath.Join(dir, "a"))
	c.Check(err, IsNil)

	data, err := s.storage.bucket.Get("a/b+c.txt")
	c.Check(err, IsNil)
	c.Check(data, DeepEquals, []byte("welcome to s3!"))

	data, err = s.storage.bucket.Get("a/b c.txt")
	c.Check(err, IsNil)
	c.Check(data, DeepEquals, []byte("welcome to s3!"))
}

func (s *PublishedStorageSuite) TestFilelist(c *C) {
	paths := []string{"a", "b", "c", "testa", "test/a", "test/b", "lala/a", "lala/b", "lala/c"}
	for _, path := range paths {
		err := s.storage.bucket.Put(path, []byte("test"), "binary/octet-stream", "private")
		c.Check(err, IsNil)
	}

	list, err := s.storage.Filelist("")
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"a", "b", "c", "lala/a", "lala/b", "lala/c", "test/a", "test/b", "testa"})

	list, err = s.storage.Filelist("test")
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"a", "b"})

	list, err = s.storage.Filelist("test2")
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{})

	list, err = s.prefixedStorage.Filelist("")
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"a", "b", "c"})
}

func (s *PublishedStorageSuite) TestFilelistPlusWorkaround(c *C) {
	s.storage.plusWorkaround = true
	s.prefixedStorage.plusWorkaround = true

	paths := []string{"a", "b", "c", "testa", "test/a+1", "test/a 1", "lala/a+b", "lala/a b", "lala/c"}
	for _, path := range paths {
		err := s.storage.bucket.Put(path, []byte("test"), "binary/octet-stream", "private")
		c.Check(err, IsNil)
	}

	list, err := s.storage.Filelist("")
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"a", "b", "c", "lala/a+b", "lala/c", "test/a+1", "testa"})

	list, err = s.storage.Filelist("test")
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"a+1"})

	list, err = s.storage.Filelist("test2")
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{})

	list, err = s.prefixedStorage.Filelist("")
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"a+b", "c"})
}

func (s *PublishedStorageSuite) TestRemove(c *C) {
	err := s.storage.bucket.Put("a/b", []byte("test"), "binary/octet-stream", "private")
	c.Check(err, IsNil)

	err = s.storage.Remove("a/b")
	c.Check(err, IsNil)

	_, err = s.storage.bucket.Get("a/b")
	c.Check(err, ErrorMatches, "The specified key does not exist.")
}

func (s *PublishedStorageSuite) TestRemovePlusWorkaround(c *C) {
	s.storage.plusWorkaround = true

	err := s.storage.bucket.Put("a/b+c", []byte("test"), "binary/octet-stream", "private")
	c.Check(err, IsNil)

	err = s.storage.bucket.Put("a/b", []byte("test"), "binary/octet-stream", "private")
	c.Check(err, IsNil)

	err = s.storage.Remove("a/b+c")
	c.Check(err, IsNil)

	_, err = s.storage.bucket.Get("a/b+c")
	c.Check(err, ErrorMatches, "The specified key does not exist.")

	_, err = s.storage.bucket.Get("a/b c")
	c.Check(err, ErrorMatches, "The specified key does not exist.")

	err = s.storage.Remove("a/b")
	c.Check(err, IsNil)

	_, err = s.storage.bucket.Get("a/b")
	c.Check(err, ErrorMatches, "The specified key does not exist.")

}

func (s *PublishedStorageSuite) TestRemoveDirs(c *C) {
	s.storage.plusWorkaround = true

	paths := []string{"a", "b", "c", "testa", "test/a+1", "test/a 1", "lala/a+b", "lala/a b", "lala/c"}
	for _, path := range paths {
		err := s.storage.bucket.Put(path, []byte("test"), "binary/octet-stream", "private")
		c.Check(err, IsNil)
	}

	err := s.storage.RemoveDirs("test", nil)
	c.Check(err, IsNil)

	list, err := s.storage.Filelist("")
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"a", "b", "c", "lala/a+b", "lala/c", "testa"})
}

func (s *PublishedStorageSuite) TestRemoveDirsPlusWorkaround(c *C) {
	paths := []string{"a", "b", "c", "testa", "test/a", "test/b", "lala/a", "lala/b", "lala/c"}
	for _, path := range paths {
		err := s.storage.bucket.Put(path, []byte("test"), "binary/octet-stream", "private")
		c.Check(err, IsNil)
	}

	err := s.storage.RemoveDirs("test", nil)
	c.Check(err, IsNil)

	list, err := s.storage.Filelist("")
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"a", "b", "c", "lala/a", "lala/b", "lala/c", "testa"})
}

func (s *PublishedStorageSuite) TestRenameFile(c *C) {
	c.Skip("copy not available in s3test")
}

func (s *PublishedStorageSuite) TestLinkFromPool(c *C) {
	root := c.MkDir()
	pool := files.NewPackagePool(root)

	sourcePath := filepath.Join(root, "pool/c1/df/mars-invaders_1.03.deb")
	err := os.MkdirAll(filepath.Dir(sourcePath), 0755)
	c.Assert(err, IsNil)

	err = ioutil.WriteFile(sourcePath, []byte("Contents"), 0644)
	c.Assert(err, IsNil)

	sourcePath2 := filepath.Join(root, "pool/e9/df/mars-invaders_1.03.deb")
	err = os.MkdirAll(filepath.Dir(sourcePath2), 0755)
	c.Assert(err, IsNil)

	err = ioutil.WriteFile(sourcePath2, []byte("Spam"), 0644)
	c.Assert(err, IsNil)

	// first link from pool
	err = s.storage.LinkFromPool(filepath.Join("", "pool", "main", "m/mars-invaders"), pool, sourcePath, "c1df1da7a1ce305a3b60af9d5733ac1d", false)
	c.Check(err, IsNil)

	data, err := s.storage.bucket.Get("pool/main/m/mars-invaders/mars-invaders_1.03.deb")
	c.Check(err, IsNil)
	c.Check(data, DeepEquals, []byte("Contents"))

	// duplicate link from pool
	err = s.storage.LinkFromPool(filepath.Join("", "pool", "main", "m/mars-invaders"), pool, sourcePath, "c1df1da7a1ce305a3b60af9d5733ac1d", false)
	c.Check(err, IsNil)

	data, err = s.storage.bucket.Get("pool/main/m/mars-invaders/mars-invaders_1.03.deb")
	c.Check(err, IsNil)
	c.Check(data, DeepEquals, []byte("Contents"))

	// link from pool with conflict
	err = s.storage.LinkFromPool(filepath.Join("", "pool", "main", "m/mars-invaders"), pool, sourcePath2, "e9dfd31cc505d51fc26975250750deab", false)
	c.Check(err, ErrorMatches, ".*file already exists and is different.*")

	data, err = s.storage.bucket.Get("pool/main/m/mars-invaders/mars-invaders_1.03.deb")
	c.Check(err, IsNil)
	c.Check(data, DeepEquals, []byte("Contents"))

	// link from pool with conflict and force
	err = s.storage.LinkFromPool(filepath.Join("", "pool", "main", "m/mars-invaders"), pool, sourcePath2, "e9dfd31cc505d51fc26975250750deab", true)
	c.Check(err, IsNil)

	data, err = s.storage.bucket.Get("pool/main/m/mars-invaders/mars-invaders_1.03.deb")
	c.Check(err, IsNil)
	c.Check(data, DeepEquals, []byte("Spam"))
}
