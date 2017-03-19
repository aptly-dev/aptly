package deb

import (
	"github.com/smira/aptly/files"
	"github.com/smira/aptly/utils"
	"os"
	"path/filepath"

	. "gopkg.in/check.v1"
)

type PackageFilesSuite struct {
	files PackageFiles
}

var _ = Suite(&PackageFilesSuite{})

func (s *PackageFilesSuite) SetUpTest(c *C) {
	s.files = PackageFiles{PackageFile{
		Filename:     "alien-arena-common_7.40-2_i386.deb",
		downloadPath: "pool/contrib/a/alien-arena",
		Checksums: utils.ChecksumInfo{
			Size:   187518,
			MD5:    "1e8cba92c41420aa7baa8a5718d67122",
			SHA1:   "46955e48cad27410a83740a21d766ce362364024",
			SHA256: "eb4afb9885cba6dc70cccd05b910b2dbccc02c5900578be5e99f0d3dbf9d76a5",
		}}}
}

func (s *PackageFilesSuite) TestVerify(c *C) {
	packagePool := files.NewPackagePool(c.MkDir())
	poolPath, _ := packagePool.Path(s.files[0].Filename, s.files[0].Checksums.MD5)

	result, err := s.files[0].Verify(packagePool)
	c.Check(err, IsNil)
	c.Check(result, Equals, false)

	err = os.MkdirAll(filepath.Dir(poolPath), 0755)
	c.Assert(err, IsNil)

	file, err := os.Create(poolPath)
	c.Assert(err, IsNil)
	file.WriteString("abcde")
	file.Close()

	result, err = s.files[0].Verify(packagePool)
	c.Check(err, IsNil)
	c.Check(result, Equals, false)

	s.files[0].Checksums.Size = 5
	result, err = s.files[0].Verify(packagePool)
	c.Check(err, IsNil)
	c.Check(result, Equals, true)
}

func (s *PackageFilesSuite) TestDownloadURL(c *C) {
	c.Check(s.files[0].DownloadURL(), Equals, "pool/contrib/a/alien-arena/alien-arena-common_7.40-2_i386.deb")
}

func (s *PackageFilesSuite) TestHash(c *C) {
	c.Check(s.files.Hash(), Equals, uint64(0xc8901eedd79ac51b))
}

func (s *PackageFilesSuite) TestSelectChecksum(c *C) {
	c.Check(s.files[0].SelectChecksum(""), Equals, "1e8cba92c41420aa7baa8a5718d67122")
	c.Check(s.files[0].SelectChecksum("MD5"), Equals, "1e8cba92c41420aa7baa8a5718d67122")
	c.Check(s.files[0].SelectChecksum("SHA1"), Equals, "46955e48cad27410a83740a21d766ce362364024")
	c.Check(s.files[0].SelectChecksum("SHA256"), Equals, "eb4afb9885cba6dc70cccd05b910b2dbccc02c5900578be5e99f0d3dbf9d76a5")

	// We check about the empty string since the package file does not have a SHA512 set
	c.Check(s.files[0].SelectChecksum("SHA512"), Equals, "")
}
