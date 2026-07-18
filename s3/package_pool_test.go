package s3

import (
	"context"
	"io"
	"path/filepath"
	"runtime"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/files"
	"github.com/aptly-dev/aptly/utils"

	. "gopkg.in/check.v1"
)

type PackagePoolSuite struct {
	srv                *Server
	pool, prefixedPool *PackagePool
	debFile            string
	cs                 aptly.ChecksumStorage
}

var _ = Suite(&PackagePoolSuite{})

func (s *PackagePoolSuite) SetUpTest(c *C) {
	var err error
	s.srv, err = NewServer(&Config{})
	c.Assert(err, IsNil)
	c.Assert(s.srv, NotNil)

	// accessKey, secretKey, sessionToken, bucket, prefix, defaultACL,
	// storageClass, encryptionMethod, region, endpoint, forceVirtualHostedStyle, debug
	s.pool, err = NewPackagePool("aa", "bb", "", "pool-test", "", "", "", "", "test-1", s.srv.URL(), false, false)
	c.Assert(err, IsNil)

	s.prefixedPool, err = NewPackagePool("aa", "bb", "", "pool-test", "lala", "", "", "", "test-1", s.srv.URL(), false, false)
	c.Assert(err, IsNil)

	_, err = s.pool.s3.CreateBucket(context.TODO(), &s3.CreateBucketInput{
		Bucket: aws.String("pool-test"),
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: "test-1",
		}})
	c.Assert(err, IsNil)

	_, _File, _, _ := runtime.Caller(0)
	s.debFile = filepath.Join(filepath.Dir(_File), "../system/files/libboost-program-options-dev_1.49.0.1_i386.deb")
	s.cs = files.NewMockChecksumStorage()
}

func (s *PackagePoolSuite) TearDownTest(c *C) {
	s.srv.Quit()
}

func (s *PackagePoolSuite) TestFilepathList(c *C) {
	list, err := s.pool.FilepathList(nil)
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{})

	_, _ = s.pool.Import(s.debFile, "a.deb", &utils.ChecksumInfo{}, false, s.cs)
	_, _ = s.pool.Import(s.debFile, "b.deb", &utils.ChecksumInfo{}, false, s.cs)

	list, err = s.pool.FilepathList(nil)
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{
		"c7/6b/4bd12fd92e4dfe1b55b18a67a669_a.deb",
		"c7/6b/4bd12fd92e4dfe1b55b18a67a669_b.deb",
	})
}

func (s *PackagePoolSuite) TestRemove(c *C) {
	_, _ = s.pool.Import(s.debFile, "a.deb", &utils.ChecksumInfo{}, false, s.cs)
	_, _ = s.pool.Import(s.debFile, "b.deb", &utils.ChecksumInfo{}, false, s.cs)

	size, err := s.pool.Remove("c7/6b/4bd12fd92e4dfe1b55b18a67a669_a.deb")
	c.Check(err, IsNil)
	c.Check(size, Equals, int64(2738))

	_, err = s.pool.Remove("c7/6b/4bd12fd92e4dfe1b55b18a67a669_a.deb")
	c.Check(err, ErrorMatches, "(.|\n)*not found(.|\n)*")

	list, err := s.pool.FilepathList(nil)
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"c7/6b/4bd12fd92e4dfe1b55b18a67a669_b.deb"})
}

func (s *PackagePoolSuite) TestImportOk(c *C) {
	var checksum utils.ChecksumInfo
	path, err := s.pool.Import(s.debFile, filepath.Base(s.debFile), &checksum, false, s.cs)
	c.Check(err, IsNil)
	c.Check(path, Equals, "c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb")
	// SHA256 should be automatically calculated
	c.Check(checksum.SHA256, Equals, "c76b4bd12fd92e4dfe1b55b18a67a669d92f62985d6a96c8a21d96120982cf12")
	// checksum storage is filled with new checksum
	c.Check(s.cs.(*files.MockChecksumStorage).Store[path].SHA256, Equals, "c76b4bd12fd92e4dfe1b55b18a67a669d92f62985d6a96c8a21d96120982cf12")

	size, err := s.pool.Size(path)
	c.Assert(err, IsNil)
	c.Check(size, Equals, int64(2738))

	// import as different name
	checksum = utils.ChecksumInfo{}
	path, err = s.pool.Import(s.debFile, "some.deb", &checksum, false, s.cs)
	c.Check(err, IsNil)
	c.Check(path, Equals, "c7/6b/4bd12fd92e4dfe1b55b18a67a669_some.deb")
	c.Check(s.cs.(*files.MockChecksumStorage).Store[path].SHA256, Equals, "c76b4bd12fd92e4dfe1b55b18a67a669d92f62985d6a96c8a21d96120982cf12")

	// double import, should be ok (dedup: no re-upload)
	checksum = utils.ChecksumInfo{}
	path, err = s.pool.Import(s.debFile, filepath.Base(s.debFile), &checksum, false, s.cs)
	c.Check(err, IsNil)
	c.Check(path, Equals, "c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb")
	// checksum is filled back from checksum storage (cache level 1)
	c.Check(checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// clear checksum storage, then double-import again: checksums must be
	// recomputed by downloading the object back from S3 (cache level 2)
	delete(s.cs.(*files.MockChecksumStorage).Store, path)
	checksum = utils.ChecksumInfo{}
	path, err = s.pool.Import(s.debFile, filepath.Base(s.debFile), &checksum, false, s.cs)
	c.Check(err, IsNil)
	c.Check(path, Equals, "c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb")
	c.Check(checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// import under a new name, with only the path-relevant checksum filled in.
	// SHA1 is missing here, so Import must top the checksums back up from the
	// source file before writing them to the checksum storage.
	checksum = utils.ChecksumInfo{SHA256: checksum.SHA256}
	path, err = s.pool.Import(s.debFile, "other.deb", &checksum, false, s.cs)
	c.Check(err, IsNil)
	c.Check(path, Equals, "c7/6b/4bd12fd92e4dfe1b55b18a67a669_other.deb")
	c.Check(checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")
}

func (s *PackagePoolSuite) TestVerify(c *C) {
	// pool is empty: a missing object is NOT an error, just not-found
	ppath, exists, err := s.pool.Verify("", filepath.Base(s.debFile), &utils.ChecksumInfo{}, s.cs)
	c.Check(ppath, Equals, "")
	c.Check(err, IsNil)
	c.Check(exists, Equals, false)

	// import file
	checksum := utils.ChecksumInfo{}
	path, err := s.pool.Import(s.debFile, filepath.Base(s.debFile), &checksum, false, s.cs)
	c.Check(err, IsNil)
	c.Check(path, Equals, "c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb")

	// check existence, path derived from checksums
	ppath, exists, err = s.pool.Verify("", filepath.Base(s.debFile), &checksum, s.cs)
	c.Check(ppath, Equals, path)
	c.Check(err, IsNil)
	c.Check(exists, Equals, true)
	c.Check(checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// check existence with an explicit pool path
	checksum = utils.ChecksumInfo{Size: checksum.Size}
	ppath, exists, err = s.pool.Verify(path, filepath.Base(s.debFile), &checksum, s.cs)
	c.Check(ppath, Equals, path)
	c.Check(err, IsNil)
	c.Check(exists, Equals, true)
	c.Check(checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// checksums missing that aren't needed to find the path: filled back from storage
	checksum.SHA512 = ""
	ppath, exists, err = s.pool.Verify("", filepath.Base(s.debFile), &checksum, s.cs)
	c.Check(ppath, Equals, path)
	c.Check(err, IsNil)
	c.Check(exists, Equals, true)
	c.Check(checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// wrong checksum, but correct path and size: the object is there but it is
	// the wrong file, so not-found (and NOT an error)
	ppath, exists, err = s.pool.Verify(path, filepath.Base(s.debFile), &utils.ChecksumInfo{
		SHA256: "abc",
		Size:   checksum.Size,
	}, s.cs)
	c.Check(ppath, Equals, "")
	c.Check(err, IsNil)
	c.Check(exists, Equals, false)

	// nothing in checksum storage: recomputed by downloading from S3
	delete(s.cs.(*files.MockChecksumStorage).Store, path)
	checksum.SHA512 = ""
	ppath, exists, err = s.pool.Verify("", filepath.Base(s.debFile), &checksum, s.cs)
	c.Check(ppath, Equals, path)
	c.Check(err, IsNil)
	c.Check(exists, Equals, true)
	c.Check(checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// wrong size: not-found
	checksum = utils.ChecksumInfo{Size: 13455}
	ppath, exists, err = s.pool.Verify(path, filepath.Base(s.debFile), &checksum, s.cs)
	c.Check(ppath, Equals, "")
	c.Check(err, IsNil)
	c.Check(exists, Equals, false)

	// empty checksum info and no pool path: nothing to look for
	ppath, exists, err = s.pool.Verify("", filepath.Base(s.debFile), &utils.ChecksumInfo{}, s.cs)
	c.Check(ppath, Equals, "")
	c.Check(err, IsNil)
	c.Check(exists, Equals, false)

	// object genuinely not in the pool at all: a 404 must surface as
	// not-found, never as an error (deliberate deviation from azure.PackagePool)
	ppath, exists, err = s.pool.Verify("", "missing.deb", &utils.ChecksumInfo{
		SHA256: "0000000000000000000000000000000000000000000000000000000000000000",
		Size:   2738,
	}, files.NewMockChecksumStorage())
	c.Check(ppath, Equals, "")
	c.Check(err, IsNil)
	c.Check(exists, Equals, false)
}

func (s *PackagePoolSuite) TestImportNotExist(c *C) {
	_, err := s.pool.Import("no-such-file", "a.deb", &utils.ChecksumInfo{}, false, s.cs)
	c.Check(err, ErrorMatches, ".*no such file or directory")
}

func (s *PackagePoolSuite) TestSize(c *C) {
	path, err := s.pool.Import(s.debFile, filepath.Base(s.debFile), &utils.ChecksumInfo{}, false, s.cs)
	c.Check(err, IsNil)

	size, err := s.pool.Size(path)
	c.Assert(err, IsNil)
	c.Check(size, Equals, int64(2738))

	_, err = s.pool.Size("do/es/ntexist")
	c.Check(err, ErrorMatches, "(.|\n)*not found(.|\n)*")
}

func (s *PackagePoolSuite) TestOpen(c *C) {
	path, err := s.pool.Import(s.debFile, filepath.Base(s.debFile), &utils.ChecksumInfo{}, false, s.cs)
	c.Check(err, IsNil)

	f, err := s.pool.Open(path)
	c.Assert(err, IsNil)
	contents, err := io.ReadAll(f)
	c.Assert(err, IsNil)
	c.Check(len(contents), Equals, 2738)
	c.Check(f.Close(), IsNil)

	_, err = s.pool.Open("do/es/ntexist")
	c.Check(err, ErrorMatches, "(.|\n)*error downloading(.|\n)*")
}

func (s *PackagePoolSuite) TestLegacyPath(c *C) {
	_, err := s.pool.LegacyPath("a.deb", &utils.ChecksumInfo{MD5: "abcdef00"})
	c.Check(err, NotNil)
}

func (s *PackagePoolSuite) TestPrefixedPool(c *C) {
	path, err := s.prefixedPool.Import(s.debFile, "a.deb", &utils.ChecksumInfo{}, false, s.cs)
	c.Check(err, IsNil)

	// the returned pool path is prefix-relative
	c.Check(path, Equals, "c7/6b/4bd12fd92e4dfe1b55b18a67a669_a.deb")

	list, err := s.prefixedPool.FilepathList(nil)
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"c7/6b/4bd12fd92e4dfe1b55b18a67a669_a.deb"})

	size, err := s.prefixedPool.Size(path)
	c.Assert(err, IsNil)
	c.Check(size, Equals, int64(2738))

	resp, err := s.pool.s3.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String("pool-test"),
		Key:    aws.String("lala/c7/6b/4bd12fd92e4dfe1b55b18a67a669_a.deb"),
	})
	c.Assert(err, IsNil)
	_ = resp.Body.Close()

	removed, err := s.prefixedPool.Remove(path)
	c.Check(err, IsNil)
	c.Check(removed, Equals, int64(2738))
}
