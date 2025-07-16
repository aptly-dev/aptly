package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

	s.storage, err = NewPublishedStorage("aa", "bb", "", "test-1", s.srv.URL(), "test", "", "", "", "", false, true, false, false, false, 0, 0)
	c.Assert(err, IsNil)
	s.prefixedStorage, err = NewPublishedStorage("aa", "bb", "", "test-1", s.srv.URL(), "test", "", "lala", "", "", false, true, false, false, false, 0, 0)
	c.Assert(err, IsNil)
	s.noSuchBucketStorage, err = NewPublishedStorage("aa", "bb", "", "test-1", s.srv.URL(), "no-bucket", "", "", "", "", false, true, false, false, false, 0, 0)
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
		Bucket: aws.String("test"),
		Key:    aws.String(path),
	})
	c.Assert(err, IsNil)
	defer resp.Body.Close()

	contents, err := io.ReadAll(resp.Body)
	c.Assert(err, IsNil)

	return contents
}

func (s *PublishedStorageSuite) GetFileWithBucket(c *C, bucket, path string) []byte {
	resp, err := s.storage.s3.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	})
	c.Assert(err, IsNil)
	defer resp.Body.Close()

	contents, err := io.ReadAll(resp.Body)
	c.Assert(err, IsNil)

	return contents
}

func (s *PublishedStorageSuite) checkGetRequestsEqual(c *C, prefix string, expectedRequests []string) {
	requests := []string{}
	for _, r := range s.srv.Requests {
		if r.Method == "GET" && strings.Contains(r.RequestURI, prefix) {
			requests = append(requests, r.RequestURI)
		}
	}
	c.Check(requests, DeepEquals, expectedRequests)
}

func (s *PublishedStorageSuite) TestNoSuchBucketCreateAndPutFile(c *C) {
	err := s.noSuchBucketStorage.PutFile("a/b.txt", "/dev/null")
	c.Check(err, NotNil)
}

func (s *PublishedStorageSuite) TestNoSuchBucketRemoveDirs(c *C) {
	err := s.noSuchBucketStorage.RemoveDirs("a/b", nil)
	c.Check(err, IsNil)
}

func (s *PublishedStorageSuite) TestPutFile(c *C) {
	dir := c.MkDir()
	err := os.WriteFile(filepath.Join(dir, "a"), []byte("welcome to s3!"), 0644)
	c.Assert(err, IsNil)

	err = s.storage.PutFile("a/b.txt", filepath.Join(dir, "a"))
	c.Check(err, IsNil)

	c.Check(s.GetFile(c, "a/b.txt"), DeepEquals, []byte("welcome to s3!"))

	err = s.prefixedStorage.PutFile("a/b.txt", filepath.Join(dir, "a"))
	c.Check(err, IsNil)

	c.Check(s.GetFile(c, "lala/a/b.txt"), DeepEquals, []byte("welcome to s3!"))
}

func (s *PublishedStorageSuite) TestPutFileWithPlusWorkaround(c *C) {
	storage, err := NewPublishedStorage("aa", "bb", "", "test-1", s.srv.URL(), "test", "", "lala", "", "", true, true, false, false, false, 0, 0)
	c.Assert(err, IsNil)

	dir := c.MkDir()
	err = os.WriteFile(filepath.Join(dir, "a"), []byte("welcome to s3!"), 0644)
	c.Assert(err, IsNil)

	err = storage.PutFile("a+/b+.txt", filepath.Join(dir, "a"))
	c.Check(err, IsNil)

	c.Check(s.GetFile(c, "lala/a+/b+.txt"), DeepEquals, []byte("welcome to s3!"))
	c.Check(s.GetFile(c, "lala/a /b .txt"), DeepEquals, []byte("welcome to s3!"))
}

func (s *PublishedStorageSuite) TestFilelist(c *C) {
	paths := []string{"a", "b", "c", "testa", "test/a", "test/b", "lala/a", "lala/b", "lala/c"}
	for _, path := range paths {
		err := s.storage.PutFile(path, "/dev/null")
		c.Check(err, IsNil)
	}

	list, err := s.storage.Filelist("")
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, paths)

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

func (s *PublishedStorageSuite) TestFilelistPagination(c *C) {
	for i := 0; i < 2030; i++ {
		err := s.storage.PutFile(strings.Repeat("la", i%23), "/dev/null")
		c.Check(err, IsNil)
	}

	list, err := s.storage.Filelist("")
	c.Check(err, IsNil)
	c.Check(len(list), Equals, 23)
}

func (s *PublishedStorageSuite) TestFilelistWithPlusWorkaround(c *C) {
	storage, err := NewPublishedStorage("aa", "bb", "", "test-1", s.srv.URL(), "test", "", "lala", "", "", true, true, false, false, false, 0, 0)
	c.Assert(err, IsNil)

	paths := []string{"a", "b", "c", "test+a", "test/a+", "test/b", "lala/a+", "lala/b", "lala/c+"}
	for _, path := range paths {
		err := storage.PutFile(path, "/dev/null")
		c.Check(err, IsNil)
	}

	list, err := storage.Filelist("")
	c.Check(err, IsNil)
	sort.Strings(list)
	c.Check(list, DeepEquals, []string{"a", "b", "c", "lala/a+", "lala/b", "lala/c+", "test+a", "test/a+", "test/b"})

	list, err = storage.Filelist("test")
	c.Check(err, IsNil)
	sort.Strings(list)
	c.Check(list, DeepEquals, []string{"a+", "b"})
}

func (s *PublishedStorageSuite) TestRemove(c *C) {
	err := s.storage.PutFile("a/b.txt", "/dev/null")
	c.Check(err, IsNil)

	err = s.storage.Remove("a/b.txt")
	c.Check(err, IsNil)

	_, err = s.storage.s3.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String("test"),
		Key:    aws.String("a/b.txt"),
	})
	c.Check(err, NotNil)

	// double remove
	err = s.storage.Remove("a/b.txt")
	c.Check(err, IsNil)
}

func (s *PublishedStorageSuite) TestRemoveWithPlusWorkaround(c *C) {
	storage, err := NewPublishedStorage("aa", "bb", "", "test-1", s.srv.URL(), "test", "", "lala", "", "", true, true, false, false, false, 0, 0)
	c.Assert(err, IsNil)

	err = storage.PutFile("a+/b+.txt", "/dev/null")
	c.Check(err, IsNil)

	err = storage.Remove("a+/b+.txt")
	c.Check(err, IsNil)

	_, err = s.storage.s3.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String("test"),
		Key:    aws.String("lala/a+/b+.txt"),
	})
	c.Check(err, NotNil)

	_, err = s.storage.s3.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String("test"),
		Key:    aws.String("lala/a /b .txt"),
	})
	c.Check(err, NotNil)
}

func (s *PublishedStorageSuite) TestRemoveDirs(c *C) {
	paths := []string{"a", "b", "c", "testa", "test/a", "test/b", "test/c/d", "test/c/e", "lala/a", "lala/b", "lala/c"}
	for _, path := range paths {
		err := s.storage.PutFile(path, "/dev/null")
		c.Check(err, IsNil)
	}

	err := s.storage.RemoveDirs("test", nil)
	c.Check(err, IsNil)

	list, err := s.storage.Filelist("")
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"a", "b", "c", "lala/a", "lala/b", "lala/c", "testa"})
}

func (s *PublishedStorageSuite) TestRemoveDirsWithPlusWorkaround(c *C) {
	storage, err := NewPublishedStorage("aa", "bb", "", "test-1", s.srv.URL(), "test", "", "lala", "", "", true, true, false, false, false, 0, 0)
	c.Assert(err, IsNil)

	paths := []string{"a", "b", "c", "test+a", "test/a+", "test/b", "test/c/d+", "test/c/e", "lala/a", "lala/b+", "lala/c"}
	for _, path := range paths {
		err := storage.PutFile(path, "/dev/null")
		c.Check(err, IsNil)
	}

	err = storage.RemoveDirs("test", nil)
	c.Check(err, IsNil)

	list, err := storage.Filelist("")
	c.Check(err, IsNil)
	sort.Strings(list)
	c.Check(list, DeepEquals, []string{"a", "b", "c", "lala/a", "lala/b+", "lala/c", "test+a"})
}

func (s *PublishedStorageSuite) TestRenameFile(c *C) {
	err := s.storage.PutFile("a/b", "/dev/null")
	c.Check(err, IsNil)

	err = s.storage.RenameFile("a/b", "c/d")
	c.Check(err, IsNil)

	list, err := s.storage.Filelist("")
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"c/d"})
}

func (s *PublishedStorageSuite) TestLinkFromPool(c *C) {
	root := c.MkDir()
	pool := files.NewPackagePool(root, false)
	cs := files.NewMockChecksumStorage()

	tmpFile1 := filepath.Join(c.MkDir(), "mars-invaders_1.03.deb")
	err := os.WriteFile(tmpFile1, []byte("Contents"), 0644)
	c.Assert(err, IsNil)
	cksum1 := utils.ChecksumInfo{MD5: "c1df1da7a1ce305a3b60af9d5733ac1d"}

	src1, err := pool.Import(tmpFile1, "mars-invaders_1.03.deb", &cksum1, true, cs)
	c.Assert(err, IsNil)

	tmpFile2 := filepath.Join(c.MkDir(), "mars-invaders_1.03.deb")
	err = os.WriteFile(tmpFile2, []byte("Spam"), 0644)
	c.Assert(err, IsNil)
	cksum2 := utils.ChecksumInfo{MD5: "07563b64442662f7b7e6e5afe5bb55d7"}

	src2, err := pool.Import(tmpFile2, "mars-invaders_1.03.deb", &cksum2, true, cs)
	c.Assert(err, IsNil)

	tmpFile3 := filepath.Join(c.MkDir(), "boot.img.gz")
	err = os.WriteFile(tmpFile3, []byte("Contents"), 0644)
	c.Assert(err, IsNil)
	cksum3 := utils.ChecksumInfo{MD5: "c1df1da7a1ce305a3b60af9d5733ac1d"}

	src3, err := pool.Import(tmpFile3, "boot.img.gz", &cksum3, true, cs)
	c.Assert(err, IsNil)

	// first link from pool
	err = s.storage.LinkFromPool("", filepath.Join("pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, src1, cksum1, false)
	c.Check(err, IsNil)

	c.Check(s.GetFile(c, "pool/main/m/mars-invaders/mars-invaders_1.03.deb"), DeepEquals, []byte("Contents"))

	// duplicate link from pool
	err = s.storage.LinkFromPool("", filepath.Join("pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, src1, cksum1, false)
	c.Check(err, IsNil)

	c.Check(s.GetFile(c, "pool/main/m/mars-invaders/mars-invaders_1.03.deb"), DeepEquals, []byte("Contents"))

	// link from pool with conflict
	err = s.storage.LinkFromPool("", filepath.Join("pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, src2, cksum2, false)
	c.Check(err, ErrorMatches, "error putting file to .*: file already exists and is different: .*")

	c.Check(s.GetFile(c, "pool/main/m/mars-invaders/mars-invaders_1.03.deb"), DeepEquals, []byte("Contents"))

	// link from pool with conflict and force
	err = s.storage.LinkFromPool("", filepath.Join("pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, src2, cksum2, true)
	c.Check(err, IsNil)

	c.Check(s.GetFile(c, "pool/main/m/mars-invaders/mars-invaders_1.03.deb"), DeepEquals, []byte("Spam"))

	// for prefixed storage:
	// first link from pool
	err = s.prefixedStorage.LinkFromPool("", filepath.Join("pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, src1, cksum1, false)
	c.Check(err, IsNil)

	// 2nd link from pool, providing wrong path for source file
	//
	// this test should check that file already exists in S3 and skip upload (which would fail if not skipped)
	err = s.prefixedStorage.LinkFromPool("", filepath.Join("pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, "wrong-looks-like-pathcache-doesnt-work", cksum1, false)
	c.Check(err, IsNil)

	c.Check(s.GetFile(c, "lala/pool/main/m/mars-invaders/mars-invaders_1.03.deb"), DeepEquals, []byte("Contents"))

	// link from pool with nested file name
	err = s.storage.LinkFromPool("", "dists/jessie/non-free/installer-i386/current/images", "netboot/boot.img.gz", pool, src3, cksum3, false)
	c.Check(err, IsNil)

	c.Check(s.GetFile(c, "dists/jessie/non-free/installer-i386/current/images/netboot/boot.img.gz"), DeepEquals, []byte("Contents"))
}

func (s *PublishedStorageSuite) TestLinkFromPoolCache(c *C) {
	root := c.MkDir()
	pool := files.NewPackagePool(root, false)
	cs := files.NewMockChecksumStorage()

	tmpFile1 := filepath.Join(c.MkDir(), "mars-invaders_1.03.deb")
	err := os.WriteFile(tmpFile1, []byte("Contents"), 0644)
	c.Assert(err, IsNil)
	cksum1 := utils.ChecksumInfo{MD5: "c1df1da7a1ce305a3b60af9d5733ac1d"}

	src1, err := pool.Import(tmpFile1, "mars-invaders_1.03.deb", &cksum1, true, cs)
	c.Assert(err, IsNil)

	// Publish two packages at the same publish prefix
	err = s.storage.LinkFromPool("", filepath.Join("pool", "a"), "mars-invaders_1.03.deb", pool, src1, cksum1, false)
	c.Check(err, IsNil)

	err = s.storage.LinkFromPool("", filepath.Join("pool", "b"), "mars-invaders_1.03.deb", pool, src1, cksum1, false)
	c.Check(err, IsNil)

	// Check only one listing request was done to the server
	s.checkGetRequestsEqual(c, "/test?", []string{"/test?encryption=", "/test?encryption=", "/test?list-type=2&max-keys=1000&prefix=pool%2F"})

	s.srv.Requests = nil
	// Publish two packages at a different prefix
	err = s.storage.LinkFromPool("publish-prefix", filepath.Join("pool", "a"), "mars-invaders_1.03.deb", pool, src1, cksum1, false)
	c.Check(err, IsNil)

	err = s.storage.LinkFromPool("publish-prefix", filepath.Join("pool", "b"), "mars-invaders_1.03.deb", pool, src1, cksum1, false)
	c.Check(err, IsNil)

	// Check no listing request was done to the server (pathCache is used)
	s.checkGetRequestsEqual(c, "/test?", []string{})

	s.srv.Requests = nil
	// Publish two packages at a prefixed storage
	err = s.prefixedStorage.LinkFromPool("", filepath.Join("pool", "a"), "mars-invaders_1.03.deb", pool, src1, cksum1, false)
	c.Check(err, IsNil)

	err = s.prefixedStorage.LinkFromPool("", filepath.Join("pool", "b"), "mars-invaders_1.03.deb", pool, src1, cksum1, false)
	c.Check(err, IsNil)

	// Check only one listing request was done to the server
	s.checkGetRequestsEqual(c, "/test?", []string{
		"/test?list-type=2&max-keys=1000&prefix=lala%2Fpool%2F",
	})

	// Publish two packages at a prefixed storage plus a publish prefix.
	s.srv.Requests = nil
	err = s.prefixedStorage.LinkFromPool("publish-prefix", filepath.Join("pool", "a"), "mars-invaders_1.03.deb", pool, src1, cksum1, false)
	c.Check(err, IsNil)

	err = s.prefixedStorage.LinkFromPool("publish-prefix", filepath.Join("pool", "b"), "mars-invaders_1.03.deb", pool, src1, cksum1, false)
	c.Check(err, IsNil)

	// Check no listing request was done to the server (pathCache is used)
	s.checkGetRequestsEqual(c, "/test?", []string{})
}

func (s *PublishedStorageSuite) TestConcurrentUploads(c *C) {
	// Create storage with concurrent uploads enabled (3 workers, default queue size)
	concurrentStorage, err := NewPublishedStorage("aa", "bb", "", "test-1", s.srv.URL(), "test", "", "concurrent", "", "", false, true, false, false, false, 3, 0)
	c.Assert(err, IsNil)

	// Create test files
	tmpDir := c.MkDir()
	files := []string{"file1.txt", "file2.txt", "file3.txt", "file4.txt", "file5.txt"}
	for _, name := range files {
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte("test content: "+name), 0644)
		c.Assert(err, IsNil)
	}

	// Upload files concurrently
	for _, name := range files {
		err := concurrentStorage.PutFile(name, filepath.Join(tmpDir, name))
		c.Assert(err, IsNil)
	}

	// Flush to ensure all uploads complete
	err = concurrentStorage.Flush()
	c.Assert(err, IsNil)

	// Verify all files exist
	for _, name := range files {
		exists, err := concurrentStorage.FileExists(name)
		c.Assert(err, IsNil)
		c.Check(exists, Equals, true)
	}
}

func (s *PublishedStorageSuite) TestConcurrentUploadsWithCustomQueueSize(c *C) {
	// Create storage with concurrent uploads and custom queue size (2 workers, 5x queue size)
	concurrentStorage, err := NewPublishedStorage("aa", "bb", "", "test-1", s.srv.URL(), "test", "", "concurrent-custom", "", "", false, true, false, false, false, 2, 5)
	c.Assert(err, IsNil)

	// Create test files
	tmpDir := c.MkDir()
	// Create more files than workers * queue size (2 * 5 = 10)
	fileCount := 12
	var files []string
	for i := 0; i < fileCount; i++ {
		name := fmt.Sprintf("file%d.txt", i)
		files = append(files, name)
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte(fmt.Sprintf("content %d", i)), 0644)
		c.Assert(err, IsNil)
	}

	// Upload files concurrently
	for _, name := range files {
		err := concurrentStorage.PutFile(name, filepath.Join(tmpDir, name))
		c.Assert(err, IsNil)
	}

	// Flush to ensure all uploads complete
	err = concurrentStorage.Flush()
	c.Assert(err, IsNil)

	// Verify all files exist
	for _, name := range files {
		exists, err := concurrentStorage.FileExists(name)
		c.Assert(err, IsNil)
		c.Check(exists, Equals, true)
	}
}
