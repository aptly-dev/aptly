package s3

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	. "gopkg.in/check.v1"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/aptly-dev/aptly/files"
	"github.com/aptly-dev/aptly/utils"
)

type PublishedStorageSuite struct {
	srv                      *Server
	storage, prefixedStorage *PublishedStorage
	noSuchBucketStorage      *PublishedStorage
}

var _ = Suite(&PublishedStorageSuite{})

func (s *PublishedStorageSuite) SetUpTest(c *C) {
	var err error
	s.srv, err = NewServer(&Config{})
	c.Assert(err, IsNil)
	c.Assert(s.srv, NotNil)

	s.storage, err = NewPublishedStorage("aa", "bb", "", "test-1", s.srv.URL(), "test", "", "", "", "", false, true, false, false, false)
	c.Assert(err, IsNil)
	s.prefixedStorage, err = NewPublishedStorage("aa", "bb", "", "test-1", s.srv.URL(), "test", "", "lala", "", "", false, true, false, false, false)
	c.Assert(err, IsNil)
	s.noSuchBucketStorage, err = NewPublishedStorage("aa", "bb", "", "test-1", s.srv.URL(), "no-bucket", "", "", "", "", false, true, false, false, false)
	c.Assert(err, IsNil)

	_, err = s.storage.s3.CreateBucket(context.TODO(), &s3.CreateBucketInput{
		Bucket: aws.String("test"),
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: "test-1",
		}})
	c.Assert(err, IsNil)
}

func (s *PublishedStorageSuite) TearDownTest(c *C) {
	s.srv.Quit()
}

func (s *PublishedStorageSuite) GetFile(c *C, path string) []byte {
	resp, err := s.storage.s3.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.storage.bucket),
		Key:    aws.String(path),
	})
	c.Assert(err, IsNil)

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	c.Assert(err, IsNil)

	return body
}

func (s *PublishedStorageSuite) AssertNoFile(c *C, path string) {
	_, err := s.storage.s3.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(s.storage.bucket),
		Key:    aws.String(path),
	})
	c.Assert(err, ErrorMatches, ".*StatusCode: 404.*")
}

func (s *PublishedStorageSuite) PutFile(c *C, path string, data []byte) {
	_, err := s.storage.s3.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(s.storage.bucket),
		Key:         aws.String(path),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("binary/octet-stream"),
		ACL:         types.ObjectCannedACLPrivate,
	})
	c.Assert(err, IsNil)
}

func (s *PublishedStorageSuite) TestPutFile(c *C) {
	dir := c.MkDir()
	err := ioutil.WriteFile(filepath.Join(dir, "a"), []byte("welcome to s3!"), 0644)
	c.Assert(err, IsNil)

	err = s.storage.PutFile("a/b.txt", filepath.Join(dir, "a"))
	c.Check(err, IsNil)

	c.Check(s.GetFile(c, "a/b.txt"), DeepEquals, []byte("welcome to s3!"))

	err = s.prefixedStorage.PutFile("a/b.txt", filepath.Join(dir, "a"))
	c.Check(err, IsNil)

	c.Check(s.GetFile(c, "lala/a/b.txt"), DeepEquals, []byte("welcome to s3!"))
}

func (s *PublishedStorageSuite) TestPutFilePlusWorkaround(c *C) {
	s.storage.plusWorkaround = true

	dir := c.MkDir()
	err := ioutil.WriteFile(filepath.Join(dir, "a"), []byte("welcome to s3!"), 0644)
	c.Assert(err, IsNil)

	err = s.storage.PutFile("a/b+c.txt", filepath.Join(dir, "a"))
	c.Check(err, IsNil)

	c.Check(s.GetFile(c, "a/b+c.txt"), DeepEquals, []byte("welcome to s3!"))

	c.Check(s.GetFile(c, "a/b c.txt"), DeepEquals, []byte("welcome to s3!"))
}

func (s *PublishedStorageSuite) TestFilelist(c *C) {
	paths := []string{"a", "b", "c", "testa", "test/a", "test/b", "lala/a", "lala/b", "lala/c"}
	for _, path := range paths {
		s.PutFile(c, path, []byte("test"))
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
		s.PutFile(c, path, []byte("test"))
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
	s.PutFile(c, "a/b", []byte("test"))

	err := s.storage.Remove("a/b")
	c.Check(err, IsNil)

	s.AssertNoFile(c, "a/b")

	s.PutFile(c, "lala/xyz", []byte("test"))

	errp := s.prefixedStorage.Remove("xyz")
	c.Check(errp, IsNil)

	s.AssertNoFile(c, "lala/xyz")
}

func (s *PublishedStorageSuite) TestRemoveNoSuchBucket(c *C) {
	err := s.noSuchBucketStorage.Remove("a/b")
	c.Check(err, IsNil)
}

func (s *PublishedStorageSuite) TestRemovePlusWorkaround(c *C) {
	s.storage.plusWorkaround = true

	s.PutFile(c, "a/b+c", []byte("test"))
	s.PutFile(c, "a/b", []byte("test"))

	err := s.storage.Remove("a/b+c")
	c.Check(err, IsNil)

	s.AssertNoFile(c, "a/b+c")
	s.AssertNoFile(c, "a/b c")

	err = s.storage.Remove("a/b")
	c.Check(err, IsNil)

	s.AssertNoFile(c, "a/b")
}

func (s *PublishedStorageSuite) TestRemoveDirs(c *C) {
	s.storage.plusWorkaround = true

	paths := []string{"a", "b", "c", "testa", "test/a+1", "test/a 1", "lala/a+b", "lala/a b", "lala/c"}
	for _, path := range paths {
		s.PutFile(c, path, []byte("test"))
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
		s.PutFile(c, path, []byte("test"))
	}

	err := s.storage.RemoveDirs("test", nil)
	c.Check(err, IsNil)

	list, err := s.storage.Filelist("")
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"a", "b", "c", "lala/a", "lala/b", "lala/c", "testa"})
}

func (s *PublishedStorageSuite) TestRemoveDirsNoSuchBucket(c *C) {
	err := s.noSuchBucketStorage.RemoveDirs("a/b", nil)
	c.Check(err, ErrorMatches, ".*StatusCode: 404.*")
}

func (s *PublishedStorageSuite) TestRenameFile(c *C) {
	c.Skip("copy not available in s3test")
}

func (s *PublishedStorageSuite) TestLinkFromPool(c *C) {
	root := c.MkDir()
	pool := files.NewPackagePool(root, false)
	cs := files.NewMockChecksumStorage()

	tmpFile1 := filepath.Join(c.MkDir(), "mars-invaders_1.03.deb")
	err := ioutil.WriteFile(tmpFile1, []byte("Contents"), 0644)
	c.Assert(err, IsNil)
	cksum1 := utils.ChecksumInfo{MD5: "c1df1da7a1ce305a3b60af9d5733ac1d"}

	tmpFile2 := filepath.Join(c.MkDir(), "mars-invaders_1.03.deb")
	err = ioutil.WriteFile(tmpFile2, []byte("Spam"), 0644)
	c.Assert(err, IsNil)
	cksum2 := utils.ChecksumInfo{MD5: "e9dfd31cc505d51fc26975250750deab"}

	tmpFile3 := filepath.Join(c.MkDir(), "netboot/boot.img.gz")
	os.MkdirAll(filepath.Dir(tmpFile3), 0777)
	err = ioutil.WriteFile(tmpFile3, []byte("Contents"), 0644)
	c.Assert(err, IsNil)
	cksum3 := utils.ChecksumInfo{MD5: "c1df1da7a1ce305a3b60af9d5733ac1d"}

	src1, err := pool.Import(tmpFile1, "mars-invaders_1.03.deb", &cksum1, true, cs)
	c.Assert(err, IsNil)
	src2, err := pool.Import(tmpFile2, "mars-invaders_1.03.deb", &cksum2, true, cs)
	c.Assert(err, IsNil)
	src3, err := pool.Import(tmpFile3, "netboot/boot.img.gz", &cksum3, true, cs)
	c.Assert(err, IsNil)

	// first link from pool
	err = s.storage.LinkFromPool(filepath.Join("", "pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, src1, cksum1, false)
	c.Check(err, IsNil)

	c.Check(s.GetFile(c, "pool/main/m/mars-invaders/mars-invaders_1.03.deb"), DeepEquals, []byte("Contents"))

	// duplicate link from pool
	err = s.storage.LinkFromPool(filepath.Join("", "pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, src1, cksum1, false)
	c.Check(err, IsNil)

	c.Check(s.GetFile(c, "pool/main/m/mars-invaders/mars-invaders_1.03.deb"), DeepEquals, []byte("Contents"))

	// link from pool with conflict
	err = s.storage.LinkFromPool(filepath.Join("", "pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, src2, cksum2, false)
	c.Check(err, ErrorMatches, ".*file already exists and is different.*")

	c.Check(s.GetFile(c, "pool/main/m/mars-invaders/mars-invaders_1.03.deb"), DeepEquals, []byte("Contents"))

	// link from pool with conflict and force
	err = s.storage.LinkFromPool(filepath.Join("", "pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, src2, cksum2, true)
	c.Check(err, IsNil)

	c.Check(s.GetFile(c, "pool/main/m/mars-invaders/mars-invaders_1.03.deb"), DeepEquals, []byte("Spam"))

	// for prefixed storage:
	// first link from pool
	err = s.prefixedStorage.LinkFromPool(filepath.Join("", "pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, src1, cksum1, false)
	c.Check(err, IsNil)

	// 2nd link from pool, providing wrong path for source file
	//
	// this test should check that file already exists in S3 and skip upload (which would fail if not skipped)
	s.prefixedStorage.pathCache = nil
	err = s.prefixedStorage.LinkFromPool(filepath.Join("", "pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, "wrong-looks-like-pathcache-doesnt-work", cksum1, false)
	c.Check(err, IsNil)

	c.Check(s.GetFile(c, "lala/pool/main/m/mars-invaders/mars-invaders_1.03.deb"), DeepEquals, []byte("Contents"))

	// link from pool with nested file name
	err = s.storage.LinkFromPool("dists/jessie/non-free/installer-i386/current/images", "netboot/boot.img.gz", pool, src3, cksum3, false)
	c.Check(err, IsNil)

	c.Check(s.GetFile(c, "dists/jessie/non-free/installer-i386/current/images/netboot/boot.img.gz"), DeepEquals, []byte("Contents"))
}

func (s *PublishedStorageSuite) TestSymLink(c *C) {
	s.PutFile(c, "a/b", []byte("test"))

	err := s.storage.SymLink("a/b", "a/b.link")
	c.Check(err, IsNil)

	var link string
	link, err = s.storage.ReadLink("a/b.link")
	c.Check(err, IsNil)
	c.Check(link, Equals, "a/b")

	c.Skip("copy not available in s3test")
}

func (s *PublishedStorageSuite) TestFileExists(c *C) {
	s.PutFile(c, "a/b", []byte("test"))

	exists, err := s.storage.FileExists("a/b")
	c.Check(err, IsNil)
	c.Check(exists, Equals, true)

	exists, _ = s.storage.FileExists("a/b.invalid")
	// Comment out as there is an error in s3test implementation
	// c.Check(err, IsNil)
	c.Check(exists, Equals, false)
}
