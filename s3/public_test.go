package s3

import (
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3/s3test"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"path/filepath"
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
	s.storage, err = NewPublishedStorageRaw(auth, aws.Region{Name: "test-1", S3Endpoint: s.srv.URL(), S3LocationConstraint: true}, "test", "", "")
	c.Assert(err, IsNil)

	s.prefixedStorage, err = NewPublishedStorageRaw(auth, aws.Region{Name: "test-1", S3Endpoint: s.srv.URL(), S3LocationConstraint: true}, "test", "", "lala")
	c.Assert(err, IsNil)

	err = s.storage.s3.Bucket("test").PutBucket("private")
	c.Assert(err, IsNil)
}

func (s *PublishedStorageSuite) TearDownTest(c *C) {
	s.srv.Quit()
}

func (s *PublishedStorageSuite) TestNewPublishedStorage(c *C) {
	stor, err := NewPublishedStorage("aa", "bbb", "", "", "", "")
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

func (s *PublishedStorageSuite) TestRemove(c *C) {
	err := s.storage.bucket.Put("a/b", []byte("test"), "binary/octet-stream", "private")
	c.Check(err, IsNil)

	err = s.storage.Remove("a/b")
	c.Check(err, IsNil)

	_, err = s.storage.bucket.Get("a/b")
	c.Check(err, ErrorMatches, "The specified key does not exist.")
}

func (s *PublishedStorageSuite) TestRemoveDirs(c *C) {
	c.Skip("multiple-delete not available in s3test")

	paths := []string{"a", "b", "c", "testa", "test/a", "test/b", "lala/a", "lala/b", "lala/c"}
	for _, path := range paths {
		err := s.storage.bucket.Put(path, []byte("test"), "binary/octet-stream", "private")
		c.Check(err, IsNil)
	}

	err := s.storage.RemoveDirs("test", nil)
	c.Check(err, IsNil)

	list, err := s.storage.Filelist("")
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"a", "b", "c", "lala/a", "lala/b", "lala/c", "test/a", "test/b", "testa"})

}
