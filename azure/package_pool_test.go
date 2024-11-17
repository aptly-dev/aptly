package azure

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/files"
	"github.com/aptly-dev/aptly/utils"

	. "gopkg.in/check.v1"
)

type PackagePoolSuite struct {
	accountName, accountKey, endpoint string
	pool, prefixedPool                *PackagePool
	debFile                           string
	cs                                aptly.ChecksumStorage
}

var _ = Suite(&PackagePoolSuite{})

func (s *PackagePoolSuite) SetUpSuite(c *C) {
	s.accountName = os.Getenv("AZURE_STORAGE_ACCOUNT")
	if s.accountName == "" {
		println("Please set the the following two environment variables to run the Azure storage tests.")
		println("  1. AZURE_STORAGE_ACCOUNT")
		println("  2. AZURE_STORAGE_ACCESS_KEY")
		c.Skip("AZURE_STORAGE_ACCOUNT not set.")
	}
	s.accountKey = os.Getenv("AZURE_STORAGE_ACCESS_KEY")
	if s.accountKey == "" {
		println("Please set the the following two environment variables to run the Azure storage tests.")
		println("  1. AZURE_STORAGE_ACCOUNT")
		println("  2. AZURE_STORAGE_ACCESS_KEY")
		c.Skip("AZURE_STORAGE_ACCESS_KEY not set.")
	}
	s.endpoint = os.Getenv("AZURE_STORAGE_ENDPOINT")
}

func (s *PackagePoolSuite) SetUpTest(c *C) {
	container := randContainer()
	prefix := "lala"

	var err error

	s.pool, err = NewPackagePool(s.accountName, s.accountKey, container, "", s.endpoint)
	c.Assert(err, IsNil)
	cnt := s.pool.az.container
	_, err = cnt.Create(context.Background(), azblob.Metadata{}, azblob.PublicAccessContainer)
	c.Assert(err, IsNil)

	s.prefixedPool, err = NewPackagePool(s.accountName, s.accountKey, container, prefix, s.endpoint)
	c.Assert(err, IsNil)

	_, _File, _, _ := runtime.Caller(0)
	s.debFile = filepath.Join(filepath.Dir(_File), "../system/files/libboost-program-options-dev_1.49.0.1_i386.deb")
	s.cs = files.NewMockChecksumStorage()
}

func (s *PackagePoolSuite) TestFilepathList(c *C) {
	list, err := s.pool.FilepathList(nil)
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{})

	s.pool.Import(s.debFile, "a.deb", &utils.ChecksumInfo{}, false, s.cs)
	s.pool.Import(s.debFile, "b.deb", &utils.ChecksumInfo{}, false, s.cs)

	list, err = s.pool.FilepathList(nil)
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{
		"c7/6b/4bd12fd92e4dfe1b55b18a67a669_a.deb",
		"c7/6b/4bd12fd92e4dfe1b55b18a67a669_b.deb",
	})
}

func (s *PackagePoolSuite) TestRemove(c *C) {
	s.pool.Import(s.debFile, "a.deb", &utils.ChecksumInfo{}, false, s.cs)
	s.pool.Import(s.debFile, "b.deb", &utils.ChecksumInfo{}, false, s.cs)

	size, err := s.pool.Remove("c7/6b/4bd12fd92e4dfe1b55b18a67a669_a.deb")
	c.Check(err, IsNil)
	c.Check(size, Equals, int64(2738))

	_, err = s.pool.Remove("c7/6b/4bd12fd92e4dfe1b55b18a67a669_a.deb")
	c.Check(err, ErrorMatches, "(.|\n)*BlobNotFound(.|\n)*")

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
	// checksum storage is filled with new checksum
	c.Check(s.cs.(*files.MockChecksumStorage).Store[path].SHA256, Equals, "c76b4bd12fd92e4dfe1b55b18a67a669d92f62985d6a96c8a21d96120982cf12")

	// double import, should be ok
	checksum = utils.ChecksumInfo{}
	path, err = s.pool.Import(s.debFile, filepath.Base(s.debFile), &checksum, false, s.cs)
	c.Check(err, IsNil)
	c.Check(path, Equals, "c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb")
	// checksum is filled back based on checksum storage
	c.Check(checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// clear checksum storage, and do double-import
	delete(s.cs.(*files.MockChecksumStorage).Store, path)
	checksum = utils.ChecksumInfo{}
	path, err = s.pool.Import(s.debFile, filepath.Base(s.debFile), &checksum, false, s.cs)
	c.Check(err, IsNil)
	c.Check(path, Equals, "c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb")
	// checksum is filled back based on re-calculation of file in the pool
	c.Check(checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// import under new name, but with path-relevant checksums already filled in
	checksum = utils.ChecksumInfo{SHA256: checksum.SHA256}
	path, err = s.pool.Import(s.debFile, "other.deb", &checksum, false, s.cs)
	c.Check(err, IsNil)
	c.Check(path, Equals, "c7/6b/4bd12fd92e4dfe1b55b18a67a669_other.deb")
	// checksum is filled back based on re-calculation of source file
	c.Check(checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")
}

func (s *PackagePoolSuite) TestVerify(c *C) {
	// file doesn't exist yet
	ppath, exists, err := s.pool.Verify("", filepath.Base(s.debFile), &utils.ChecksumInfo{}, s.cs)
	c.Check(ppath, Equals, "")
	c.Check(err, IsNil)
	c.Check(exists, Equals, false)

	// import file
	checksum := utils.ChecksumInfo{}
	path, err := s.pool.Import(s.debFile, filepath.Base(s.debFile), &checksum, false, s.cs)
	c.Check(err, IsNil)
	c.Check(path, Equals, "c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb")

	// check existence
	ppath, exists, err = s.pool.Verify("", filepath.Base(s.debFile), &checksum, s.cs)
	c.Check(ppath, Equals, ppath)
	c.Check(err, IsNil)
	c.Check(exists, Equals, true)
	c.Check(checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// check existence with fixed path
	checksum = utils.ChecksumInfo{Size: checksum.Size}
	ppath, exists, err = s.pool.Verify(path, filepath.Base(s.debFile), &checksum, s.cs)
	c.Check(ppath, Equals, path)
	c.Check(err, IsNil)
	c.Check(exists, Equals, true)
	c.Check(checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// check existence, but with checksums missing (that aren't needed to find the path)
	checksum.SHA512 = ""
	ppath, exists, err = s.pool.Verify("", filepath.Base(s.debFile), &checksum, s.cs)
	c.Check(ppath, Equals, path)
	c.Check(err, IsNil)
	c.Check(exists, Equals, true)
	// checksum is filled back based on checksum storage
	c.Check(checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// check existence, with missing checksum info but correct path and size available
	checksum = utils.ChecksumInfo{Size: checksum.Size}
	ppath, exists, err = s.pool.Verify(path, filepath.Base(s.debFile), &checksum, s.cs)
	c.Check(ppath, Equals, path)
	c.Check(err, IsNil)
	c.Check(exists, Equals, true)
	// checksum is filled back based on checksum storage
	c.Check(checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// check existence, with wrong checksum info but correct path and size available
	ppath, exists, err = s.pool.Verify(path, filepath.Base(s.debFile), &utils.ChecksumInfo{
		SHA256: "abc",
		Size:   checksum.Size,
	}, s.cs)
	c.Check(ppath, Equals, "")
	c.Check(err, IsNil)
	c.Check(exists, Equals, false)

	// check existence, with missing checksums (that aren't needed to find the path)
	// and no info in checksum storage
	delete(s.cs.(*files.MockChecksumStorage).Store, path)
	checksum.SHA512 = ""
	ppath, exists, err = s.pool.Verify("", filepath.Base(s.debFile), &checksum, s.cs)
	c.Check(ppath, Equals, path)
	c.Check(err, IsNil)
	c.Check(exists, Equals, true)
	// checksum is filled back based on re-calculation
	c.Check(checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// check existence, with wrong size
	checksum = utils.ChecksumInfo{Size: 13455}
	ppath, exists, err = s.pool.Verify(path, filepath.Base(s.debFile), &checksum, s.cs)
	c.Check(ppath, Equals, "")
	c.Check(err, IsNil)
	c.Check(exists, Equals, false)

	// check existence, with empty checksum info
	ppath, exists, err = s.pool.Verify("", filepath.Base(s.debFile), &utils.ChecksumInfo{}, s.cs)
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
	c.Check(err, ErrorMatches, "(.|\n)*BlobNotFound(.|\n)*")
}

func (s *PackagePoolSuite) TestOpen(c *C) {
	path, err := s.pool.Import(s.debFile, filepath.Base(s.debFile), &utils.ChecksumInfo{}, false, s.cs)
	c.Check(err, IsNil)

	f, err := s.pool.Open(path)
	c.Assert(err, IsNil)
	contents, err := ioutil.ReadAll(f)
	c.Assert(err, IsNil)
	c.Check(len(contents), Equals, 2738)
	c.Check(f.Close(), IsNil)

	_, err = s.pool.Open("do/es/ntexist")
	c.Check(err, ErrorMatches, "(.|\n)*BlobNotFound(.|\n)*")
}
