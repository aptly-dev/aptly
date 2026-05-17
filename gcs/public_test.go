package gcs

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"

	"cloud.google.com/go/storage"
	"github.com/fsouza/fake-gcs-server/fakestorage"
	. "gopkg.in/check.v1"

	"github.com/aptly-dev/aptly/files"
	"github.com/aptly-dev/aptly/utils"
)

type PublishedStorageSuite struct {
	srv                      *fakestorage.Server
	prevEmulatorHost         string
	prevEmulatorHostSet      bool
	storage, prefixedStorage *PublishedStorage
	noSuchBucketStorage      *PublishedStorage
}

var _ = Suite(&PublishedStorageSuite{})

func (s *PublishedStorageSuite) SetUpTest(c *C) {
	var err error
	s.srv, err = fakestorage.NewServerWithOptions(fakestorage.Options{
		Scheme: "http",
		Host:   "127.0.0.1",
	})
	c.Assert(err, IsNil)
	c.Assert(s.srv, NotNil)

	s.srv.CreateBucketWithOpts(fakestorage.CreateBucketOpts{Name: "test"})

	// The cloud.google.com/go/storage client honors STORAGE_EMULATOR_HOST and
	// will route all requests (including media uploads) to the fake server.
	s.prevEmulatorHost, s.prevEmulatorHostSet = os.LookupEnv("STORAGE_EMULATOR_HOST")
	c.Assert(os.Setenv("STORAGE_EMULATOR_HOST", s.srv.URL()), IsNil)

	s.storage, err = NewPublishedStorage("test", "", "", "", "", "", "", "", "", false, false)
	c.Assert(err, IsNil)
	s.prefixedStorage, err = NewPublishedStorage("test", "lala", "", "", "", "", "", "", "", false, false)
	c.Assert(err, IsNil)
	s.noSuchBucketStorage, err = NewPublishedStorage("no-bucket", "", "", "", "", "", "", "", "", false, false)
	c.Assert(err, IsNil)
}

func (s *PublishedStorageSuite) TearDownTest(c *C) {
	if s.prevEmulatorHostSet {
		_ = os.Setenv("STORAGE_EMULATOR_HOST", s.prevEmulatorHost)
	} else {
		_ = os.Unsetenv("STORAGE_EMULATOR_HOST")
	}
	s.srv.Stop()
}

func (s *PublishedStorageSuite) GetFile(c *C, path string) []byte {
	r, err := s.storage.bucket.Object(path).NewReader(context.TODO())
	c.Assert(err, IsNil)
	defer func() { _ = r.Close() }()

	body, err := io.ReadAll(r)
	c.Assert(err, IsNil)
	return body
}

func (s *PublishedStorageSuite) AssertNoFile(c *C, path string) {
	_, err := s.storage.bucket.Object(path).Attrs(context.TODO())
	c.Assert(errors.Is(err, storage.ErrObjectNotExist), Equals, true)
}

func (s *PublishedStorageSuite) PutFile(c *C, path string, data []byte) {
	w := s.storage.bucket.Object(path).NewWriter(context.TODO())
	_, err := w.Write(data)
	c.Assert(err, IsNil)
	c.Assert(w.Close(), IsNil)
}

func (s *PublishedStorageSuite) TestString(c *C) {
	c.Check(s.storage.String(), Equals, "GCS: test/")
	c.Check(s.prefixedStorage.String(), Equals, "GCS: test/lala")
}

func (s *PublishedStorageSuite) TestMkDir(c *C) {
	c.Check(s.storage.MkDir("anything"), IsNil)
}

func (s *PublishedStorageSuite) TestApplyACLNoOpModes(c *C) {
	for _, acl := range []string{"", "none", "private"} {
		st := &PublishedStorage{acl: acl}
		c.Check(st.applyACL(nil), IsNil)
	}
}

func (s *PublishedStorageSuite) TestApplyACLUnsupported(c *C) {
	st := &PublishedStorage{acl: "bucket-owner-full-control"}
	err := st.applyACL(nil)
	c.Assert(err, NotNil)
	c.Check(err, ErrorMatches, "unsupported GCS ACL value: bucket-owner-full-control")
}

func (s *PublishedStorageSuite) TestPutFile(c *C) {
	dir := c.MkDir()
	err := os.WriteFile(filepath.Join(dir, "a"), []byte("welcome to gcs!"), 0644)
	c.Assert(err, IsNil)

	err = s.storage.PutFile("a/b.txt", filepath.Join(dir, "a"))
	c.Check(err, IsNil)
	c.Check(s.GetFile(c, "a/b.txt"), DeepEquals, []byte("welcome to gcs!"))

	err = s.prefixedStorage.PutFile("a/b.txt", filepath.Join(dir, "a"))
	c.Check(err, IsNil)
	c.Check(s.GetFile(c, "lala/a/b.txt"), DeepEquals, []byte("welcome to gcs!"))
}

func (s *PublishedStorageSuite) TestPutFileMissingSource(c *C) {
	err := s.storage.PutFile("a/b.txt", filepath.Join(c.MkDir(), "does-not-exist"))
	c.Check(err, ErrorMatches, ".*no such file or directory.*")
}

func (s *PublishedStorageSuite) TestFilelist(c *C) {
	paths := []string{"a", "b", "c", "testa", "test/a", "test/b", "lala/a", "lala/b", "lala/c"}
	for _, path := range paths {
		s.PutFile(c, path, []byte("test"))
	}

	list, err := s.storage.Filelist("")
	c.Check(err, IsNil)
	sort.Strings(list)
	c.Check(list, DeepEquals, []string{"a", "b", "c", "lala/a", "lala/b", "lala/c", "test/a", "test/b", "testa"})

	list, err = s.storage.Filelist("test")
	c.Check(err, IsNil)
	sort.Strings(list)
	c.Check(list, DeepEquals, []string{"a", "b"})

	list, err = s.storage.Filelist("test2")
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{})

	list, err = s.prefixedStorage.Filelist("")
	c.Check(err, IsNil)
	sort.Strings(list)
	c.Check(list, DeepEquals, []string{"a", "b", "c"})
}

func (s *PublishedStorageSuite) TestRemove(c *C) {
	s.PutFile(c, "a/b", []byte("test"))

	err := s.storage.Remove("a/b")
	c.Check(err, IsNil)

	s.AssertNoFile(c, "a/b")

	s.PutFile(c, "lala/xyz", []byte("test"))

	err = s.prefixedStorage.Remove("xyz")
	c.Check(err, IsNil)

	s.AssertNoFile(c, "lala/xyz")
}

func (s *PublishedStorageSuite) TestRemoveMissing(c *C) {
	c.Check(s.storage.Remove("does/not/exist"), IsNil)
}

func (s *PublishedStorageSuite) TestRemoveDirs(c *C) {
	paths := []string{"a", "b", "c", "testa", "test/a", "test/b", "lala/a", "lala/b", "lala/c"}
	for _, path := range paths {
		s.PutFile(c, path, []byte("test"))
	}

	err := s.storage.RemoveDirs("test", nil)
	c.Check(err, IsNil)

	list, err := s.storage.Filelist("")
	c.Check(err, IsNil)
	sort.Strings(list)
	c.Check(list, DeepEquals, []string{"a", "b", "c", "lala/a", "lala/b", "lala/c", "testa"})
}

func (s *PublishedStorageSuite) TestRenameFile(c *C) {
	s.PutFile(c, "src", []byte("payload"))

	err := s.storage.RenameFile("src", "dst")
	c.Check(err, IsNil)

	c.Check(s.GetFile(c, "dst"), DeepEquals, []byte("payload"))
	s.AssertNoFile(c, "src")
}

func (s *PublishedStorageSuite) TestSymLink(c *C) {
	s.PutFile(c, "a/b", []byte("test"))

	err := s.storage.SymLink("a/b", "a/b.link")
	c.Check(err, IsNil)

	link, err := s.storage.ReadLink("a/b.link")
	c.Check(err, IsNil)
	c.Check(link, Equals, "a/b")

	c.Check(s.GetFile(c, "a/b.link"), DeepEquals, []byte("test"))
}

func (s *PublishedStorageSuite) TestHardLink(c *C) {
	s.PutFile(c, "a/b", []byte("test"))

	err := s.storage.HardLink("a/b", "a/b.hard")
	c.Check(err, IsNil)

	link, err := s.storage.ReadLink("a/b.hard")
	c.Check(err, IsNil)
	c.Check(link, Equals, "a/b")
}

func (s *PublishedStorageSuite) TestFileExists(c *C) {
	s.PutFile(c, "a/b", []byte("test"))

	exists, err := s.storage.FileExists("a/b")
	c.Check(err, IsNil)
	c.Check(exists, Equals, true)

	exists, err = s.storage.FileExists("a/b.invalid")
	c.Check(err, IsNil)
	c.Check(exists, Equals, false)
}

func (s *PublishedStorageSuite) TestObjectPath(c *C) {
	st := &PublishedStorage{prefix: "root"}
	c.Check(st.objectPath("dists/stable/Release"), Equals, filepath.Join("root", "dists/stable/Release"))
}

func (s *PublishedStorageSuite) TestLinkFromPool(c *C) {
	root := c.MkDir()
	pool := files.NewPackagePool(root, false)
	cs := files.NewMockChecksumStorage()

	tmpFile1 := filepath.Join(c.MkDir(), "mars-invaders_1.03.deb")
	c.Assert(os.WriteFile(tmpFile1, []byte("Contents"), 0644), IsNil)
	cksum1 := utils.ChecksumInfo{MD5: "c1df1da7a1ce305a3b60af9d5733ac1d"}

	tmpFile2 := filepath.Join(c.MkDir(), "mars-invaders_1.03.deb")
	c.Assert(os.WriteFile(tmpFile2, []byte("Spam"), 0644), IsNil)
	cksum2 := utils.ChecksumInfo{MD5: "e9dfd31cc505d51fc26975250750deab"}

	tmpFile3 := filepath.Join(c.MkDir(), "netboot/boot.img.gz")
	_ = os.MkdirAll(filepath.Dir(tmpFile3), 0777)
	c.Assert(os.WriteFile(tmpFile3, []byte("Contents"), 0644), IsNil)
	cksum3 := utils.ChecksumInfo{MD5: "c1df1da7a1ce305a3b60af9d5733ac1d"}

	src1, err := pool.Import(tmpFile1, "mars-invaders_1.03.deb", &cksum1, true, cs)
	c.Assert(err, IsNil)
	src2, err := pool.Import(tmpFile2, "mars-invaders_1.03.deb", &cksum2, true, cs)
	c.Assert(err, IsNil)
	src3, err := pool.Import(tmpFile3, "netboot/boot.img.gz", &cksum3, true, cs)
	c.Assert(err, IsNil)

	// first link from pool
	err = s.storage.LinkFromPool("", filepath.Join("pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, src1, cksum1, false)
	c.Check(err, IsNil)
	c.Check(s.GetFile(c, "pool/main/m/mars-invaders/mars-invaders_1.03.deb"), DeepEquals, []byte("Contents"))

	// duplicate link from pool (same MD5 → no-op)
	err = s.storage.LinkFromPool("", filepath.Join("pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, src1, cksum1, false)
	c.Check(err, IsNil)
	c.Check(s.GetFile(c, "pool/main/m/mars-invaders/mars-invaders_1.03.deb"), DeepEquals, []byte("Contents"))

	// link from pool with conflict, no force
	err = s.storage.LinkFromPool("", filepath.Join("pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, src2, cksum2, false)
	c.Check(err, ErrorMatches, ".*file already exists and is different.*")
	c.Check(s.GetFile(c, "pool/main/m/mars-invaders/mars-invaders_1.03.deb"), DeepEquals, []byte("Contents"))

	// link from pool with conflict and force
	err = s.storage.LinkFromPool("", filepath.Join("pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, src2, cksum2, true)
	c.Check(err, IsNil)
	c.Check(s.GetFile(c, "pool/main/m/mars-invaders/mars-invaders_1.03.deb"), DeepEquals, []byte("Spam"))

	// for prefixed storage:
	err = s.prefixedStorage.LinkFromPool("", filepath.Join("pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, src1, cksum1, false)
	c.Check(err, IsNil)

	// 2nd link from pool, providing wrong path for source file:
	// should hit the path cache and skip upload (which would otherwise fail).
	err = s.prefixedStorage.LinkFromPool("", filepath.Join("pool", "main", "m/mars-invaders"), "mars-invaders_1.03.deb", pool, "wrong-looks-like-pathcache-doesnt-work", cksum1, false)
	c.Check(err, IsNil)
	c.Check(s.GetFile(c, "lala/pool/main/m/mars-invaders/mars-invaders_1.03.deb"), DeepEquals, []byte("Contents"))

	// link from pool with nested file name
	err = s.storage.LinkFromPool("", "dists/jessie/non-free/installer-i386/current/images", "netboot/boot.img.gz", pool, src3, cksum3, false)
	c.Check(err, IsNil)
	c.Check(s.GetFile(c, "dists/jessie/non-free/installer-i386/current/images/netboot/boot.img.gz"), DeepEquals, []byte("Contents"))
}

func (s *PublishedStorageSuite) TestLinkFromPoolMissingMD5(c *C) {
	publishedPrefix := "repo"
	publishedRelPath := "pool/main/a/aptly"
	fileName := "pkg.deb"
	relPath := filepath.Join(filepath.Join(publishedPrefix, publishedRelPath), fileName)

	st := &PublishedStorage{pathCache: map[string]string{relPath: "0123456789abcdef0123456789abcdef"}}

	err := st.LinkFromPool(publishedPrefix, publishedRelPath, fileName, nil, "", utils.ChecksumInfo{}, false)
	c.Assert(err, NotNil)
	c.Check(err, ErrorMatches, "unable to compare object, MD5 checksum missing")
}

func (s *PublishedStorageSuite) TestLinkFromPoolDifferentMD5NoForce(c *C) {
	publishedPrefix := "repo"
	publishedRelPath := "pool/main/a/aptly"
	fileName := "pkg.deb"
	relPath := filepath.Join(filepath.Join(publishedPrefix, publishedRelPath), fileName)

	st := &PublishedStorage{pathCache: map[string]string{relPath: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}

	err := st.LinkFromPool(publishedPrefix, publishedRelPath, fileName, nil, "", utils.ChecksumInfo{MD5: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}, false)
	c.Assert(err, NotNil)
	c.Check(err, ErrorMatches, ".*file already exists and is different.*")
}

func (s *PublishedStorageSuite) TestLinkFromPoolSameMD5NoUpload(c *C) {
	publishedPrefix := "repo"
	publishedRelPath := "pool/main/a/aptly"
	fileName := "pkg.deb"
	relPath := filepath.Join(filepath.Join(publishedPrefix, publishedRelPath), fileName)

	st := &PublishedStorage{pathCache: map[string]string{relPath: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}

	err := st.LinkFromPool(publishedPrefix, publishedRelPath, fileName, nil, "", utils.ChecksumInfo{MD5: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}, false)
	c.Check(err, IsNil)
}

// putWithMetadata uploads an object with arbitrary metadata directly via the
// storage client, bypassing the production putFile path. Used to seed objects
// that exercise getMD5 / metadata-handling branches.
func (s *PublishedStorageSuite) putWithMetadata(c *C, path string, data []byte, metadata map[string]string) {
	w := s.storage.bucket.Object(path).NewWriter(context.TODO())
	w.Metadata = metadata
	_, err := w.Write(data)
	c.Assert(err, IsNil)
	c.Assert(w.Close(), IsNil)
}

// TestLinkFromPoolShortCachedMD5 exercises the LinkFromPool branch where the
// path cache holds a non-32-char checksum (so getMD5 must be called to fetch
// the real MD5 from object attrs), and along the way covers getMD5 plus the
// Md5-metadata branch in internalFilelist.
func (s *PublishedStorageSuite) TestLinkFromPoolShortCachedMD5(c *C) {
	root := c.MkDir()
	pool := files.NewPackagePool(root, false)
	cs := files.NewMockChecksumStorage()

	tmpFile := filepath.Join(c.MkDir(), "mars-invaders_1.03.deb")
	c.Assert(os.WriteFile(tmpFile, []byte("Contents"), 0644), IsNil)
	cksum := utils.ChecksumInfo{MD5: "c1df1da7a1ce305a3b60af9d5733ac1d"}

	src, err := pool.Import(tmpFile, "mars-invaders_1.03.deb", &cksum, true, cs)
	c.Assert(err, IsNil)

	// Seed the bucket with a short Md5 metadata value so internalFilelist
	// returns it (covering the metadata branch) and LinkFromPool then has to
	// re-fetch via getMD5 because len != 32.
	relPath := filepath.Join("pool/main/m/mars-invaders", "mars-invaders_1.03.deb")
	s.putWithMetadata(c, relPath, []byte("Contents"), map[string]string{"Md5": "short"})

	// force=true so the conflict (the seeded "short" md5 will never match
	// sourceMD5 once getMD5 normalises) resolves into a fresh upload rather
	// than an error — what matters here is that we traverse the getMD5 +
	// short-cache + internalFilelist md5-metadata branches.
	err = s.storage.LinkFromPool("", "pool/main/m/mars-invaders", "mars-invaders_1.03.deb", pool, src, cksum, true)
	c.Check(err, IsNil)
}

// TestLinkFromPoolMissingSource covers the source-pool open error path.
func (s *PublishedStorageSuite) TestLinkFromPoolMissingSource(c *C) {
	pool := files.NewPackagePool(c.MkDir(), false)

	err := s.storage.LinkFromPool("", "pool/x", "y.deb", pool, "non-existent-pool-key", utils.ChecksumInfo{MD5: "33333333333333333333333333333333"}, false)
	c.Check(err, ErrorMatches, ".*no such file or directory.*")
}

// TestPutFilePublicReadACL covers the applyACL public-read branch and the
// ACL().Set call against the fake server.
func (s *PublishedStorageSuite) TestPutFilePublicReadACL(c *C) {
	st, err := NewPublishedStorage("test", "", "", "", "", "", "public-read", "", "", false, false)
	c.Assert(err, IsNil)

	dir := c.MkDir()
	src := filepath.Join(dir, "f")
	c.Assert(os.WriteFile(src, []byte("hello"), 0644), IsNil)

	c.Check(st.PutFile("a/b.txt", src), IsNil)
	c.Check(s.GetFile(c, "a/b.txt"), DeepEquals, []byte("hello"))
}

// TestPutFileUnsupportedACL covers the default (error) branch of applyACL
// when invoked from the production putFile flow.
func (s *PublishedStorageSuite) TestPutFileUnsupportedACL(c *C) {
	st, err := NewPublishedStorage("test", "", "", "", "", "", "bucket-owner-full-control", "", "", false, false)
	c.Assert(err, IsNil)

	dir := c.MkDir()
	src := filepath.Join(dir, "f")
	c.Assert(os.WriteFile(src, []byte("hello"), 0644), IsNil)

	err = st.PutFile("a/b.txt", src)
	c.Assert(err, NotNil)
	c.Check(err, ErrorMatches, ".*unsupported GCS ACL value.*")
}

// TestPutFileWithStorageClass covers the storageClass branch in putFile.
func (s *PublishedStorageSuite) TestPutFileWithStorageClass(c *C) {
	st, err := NewPublishedStorage("test", "", "", "", "", "", "", "NEARLINE", "", false, false)
	c.Assert(err, IsNil)

	dir := c.MkDir()
	src := filepath.Join(dir, "f")
	c.Assert(os.WriteFile(src, []byte("hi"), 0644), IsNil)

	c.Check(st.PutFile("a/b.txt", src), IsNil)

	attrs, err := s.storage.bucket.Object("a/b.txt").Attrs(context.TODO())
	c.Assert(err, IsNil)
	c.Check(attrs.StorageClass, Equals, "NEARLINE")
}

// TestRemoveDirsNoSuchBucket covers the internalFilelist error path inside
// RemoveDirs (and the iterator error branch in internalFilelist itself).
func (s *PublishedStorageSuite) TestRemoveDirsNoSuchBucket(c *C) {
	err := s.noSuchBucketStorage.RemoveDirs("a/b", nil)
	c.Check(err, ErrorMatches, ".*error listing under prefix.*")
}

// TestFilelistNoSuchBucket also covers the iterator error path.
func (s *PublishedStorageSuite) TestFilelistNoSuchBucket(c *C) {
	_, err := s.noSuchBucketStorage.Filelist("")
	c.Check(err, ErrorMatches, ".*error listing under prefix.*")
}

// TestRemoveCacheEviction verifies that a successful Remove evicts the entry
// from pathCache (covers the delete(g.pathCache, ...) line).
func (s *PublishedStorageSuite) TestRemoveCacheEviction(c *C) {
	s.PutFile(c, "a/b", []byte("test"))

	s.storage.pathCache = map[string]string{"a/b": "deadbeefdeadbeefdeadbeefdeadbeef"}
	c.Check(s.storage.Remove("a/b"), IsNil)
	_, present := s.storage.pathCache["a/b"]
	c.Check(present, Equals, false)
}

// TestDebugMode exercises the if-debug log branches across the main verbs.
func (s *PublishedStorageSuite) TestDebugMode(c *C) {
	st, err := NewPublishedStorage("test", "", "", "", "", "", "", "", "", false, true)
	c.Assert(err, IsNil)

	dir := c.MkDir()
	src := filepath.Join(dir, "f")
	c.Assert(os.WriteFile(src, []byte("dbg"), 0644), IsNil)

	c.Check(st.PutFile("d/a", src), IsNil)
	c.Check(st.RenameFile("d/a", "d/b"), IsNil)
	c.Check(st.SymLink("d/b", "d/b.link"), IsNil)
	c.Check(st.HardLink("d/b", "d/b.hard"), IsNil)
	c.Check(st.Remove("d/b"), IsNil)
	c.Check(st.RemoveDirs("d", nil), IsNil)

	pool := files.NewPackagePool(c.MkDir(), false)
	cs := files.NewMockChecksumStorage()
	tmp := filepath.Join(c.MkDir(), "x.deb")
	c.Assert(os.WriteFile(tmp, []byte("Contents"), 0644), IsNil)
	cksum := utils.ChecksumInfo{MD5: "c1df1da7a1ce305a3b60af9d5733ac1d"}
	srcKey, err := pool.Import(tmp, "x.deb", &cksum, true, cs)
	c.Assert(err, IsNil)
	c.Check(st.LinkFromPool("", "pool/x", "x.deb", pool, srcKey, cksum, false), IsNil)
}

// TestObjectHandleWithEncryptionKey covers the encryptionKey branch in
// objectHandle. fsouza doesn't enforce CSEK headers but we just need to walk
// the code path.
func (s *PublishedStorageSuite) TestObjectHandleWithEncryptionKey(c *C) {
	st := &PublishedStorage{
		bucket:        s.storage.bucket,
		bucketName:    "test",
		encryptionKey: "0123456789abcdef0123456789abcdef",
	}
	c.Check(st.objectHandle("a/b"), NotNil)
}

// TestReadLinkMissing covers the Attrs-error return in ReadLink.
func (s *PublishedStorageSuite) TestReadLinkMissing(c *C) {
	_, err := s.storage.ReadLink("does/not/exist")
	c.Check(err, ErrorMatches, ".*object doesn't exist.*")
}

// TestFileExistsNoSuchBucket exercises FileExists' wrapped-googleapi-404 path.
func (s *PublishedStorageSuite) TestFileExistsNoSuchBucket(c *C) {
	exists, err := s.noSuchBucketStorage.FileExists("a/b")
	c.Check(err, IsNil)
	c.Check(exists, Equals, false)
}

// TestNewPublishedStorageWithEndpoint exercises the endpoint-injection branch
// in NewPublishedStorage (the production knob, separate from the env-var path).
func (s *PublishedStorageSuite) TestNewPublishedStorageWithEndpoint(c *C) {
	saved := os.Getenv("STORAGE_EMULATOR_HOST")
	c.Assert(os.Unsetenv("STORAGE_EMULATOR_HOST"), IsNil)
	defer func() { _ = os.Setenv("STORAGE_EMULATOR_HOST", saved) }()

	st, err := NewPublishedStorage("test", "", "", "", "", s.srv.URL()+"/storage/v1/", "", "", "", false, false)
	c.Assert(err, IsNil)

	_, err = st.Filelist("")
	c.Check(err, IsNil)
}

// TestNewPublishedStorageWithProject covers the project!="" → WithQuotaProject
// branch. WithQuotaProject is incompatible with WithEndpoint, so this test
// relies on the STORAGE_EMULATOR_HOST env var (still set from SetUpTest) for
// fake-server routing.
func (s *PublishedStorageSuite) TestNewPublishedStorageWithProject(c *C) {
	st, err := NewPublishedStorage("test", "", "", "", "fake-project", "", "", "", "", false, false)
	c.Assert(err, IsNil)

	_, err = st.Filelist("")
	c.Check(err, IsNil)
}
