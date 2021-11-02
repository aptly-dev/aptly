package files

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/utils"

	. "gopkg.in/check.v1"
)

type PackagePoolSuite struct {
	pool     *PackagePool
	checksum utils.ChecksumInfo
	debFile  string
	cs       aptly.ChecksumStorage
}

var _ = Suite(&PackagePoolSuite{})

func (s *PackagePoolSuite) SetUpTest(c *C) {
	s.pool = NewPackagePool(c.MkDir(), true)
	s.checksum = utils.ChecksumInfo{
		MD5: "0035d7822b2f8f0ec4013f270fd650c2",
	}
	_, _File, _, _ := runtime.Caller(0)
	s.debFile = filepath.Join(filepath.Dir(_File), "../system/files/libboost-program-options-dev_1.49.0.1_i386.deb")
	s.cs = NewMockChecksumStorage()
}

func (s *PackagePoolSuite) TestLegacyPath(c *C) {
	path, err := s.pool.LegacyPath("a/b/package.deb", &s.checksum)
	c.Assert(err, IsNil)
	c.Assert(path, Equals, "00/35/package.deb")

	_, err = s.pool.LegacyPath("/", &s.checksum)
	c.Assert(err, ErrorMatches, ".*is invalid")
	_, err = s.pool.LegacyPath("", &s.checksum)
	c.Assert(err, ErrorMatches, ".*is invalid")
	_, err = s.pool.LegacyPath("a/b/package.deb", &utils.ChecksumInfo{MD5: "9"})
	c.Assert(err, ErrorMatches, ".*MD5 is missing")
}

func (s *PackagePoolSuite) TestFilepathList(c *C) {
	list, err := s.pool.FilepathList(nil)
	c.Check(err, IsNil)
	c.Check(list, IsNil)

	os.MkdirAll(filepath.Join(s.pool.rootPath, "bd", "0b"), 0755)
	os.MkdirAll(filepath.Join(s.pool.rootPath, "bd", "0a"), 0755)
	os.MkdirAll(filepath.Join(s.pool.rootPath, "ae", "0c"), 0755)

	list, err = s.pool.FilepathList(nil)
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{})

	ioutil.WriteFile(filepath.Join(s.pool.rootPath, "ae", "0c", "1.deb"), nil, 0644)
	ioutil.WriteFile(filepath.Join(s.pool.rootPath, "ae", "0c", "2.deb"), nil, 0644)
	ioutil.WriteFile(filepath.Join(s.pool.rootPath, "bd", "0a", "3.deb"), nil, 0644)
	ioutil.WriteFile(filepath.Join(s.pool.rootPath, "bd", "0b", "4.deb"), nil, 0644)

	list, err = s.pool.FilepathList(nil)
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"ae/0c/1.deb", "ae/0c/2.deb", "bd/0a/3.deb", "bd/0b/4.deb"})
}

func (s *PackagePoolSuite) TestRemove(c *C) {
	os.MkdirAll(filepath.Join(s.pool.rootPath, "bd", "0b"), 0755)
	os.MkdirAll(filepath.Join(s.pool.rootPath, "bd", "0a"), 0755)
	os.MkdirAll(filepath.Join(s.pool.rootPath, "ae", "0c"), 0755)

	ioutil.WriteFile(filepath.Join(s.pool.rootPath, "ae", "0c", "1.deb"), []byte("1"), 0644)
	ioutil.WriteFile(filepath.Join(s.pool.rootPath, "ae", "0c", "2.deb"), []byte("22"), 0644)
	ioutil.WriteFile(filepath.Join(s.pool.rootPath, "bd", "0a", "3.deb"), []byte("333"), 0644)
	ioutil.WriteFile(filepath.Join(s.pool.rootPath, "bd", "0b", "4.deb"), []byte("4444"), 0644)

	size, err := s.pool.Remove("ae/0c/2.deb")
	c.Check(err, IsNil)
	c.Check(size, Equals, int64(2))

	_, err = s.pool.Remove("ae/0c/2.deb")
	c.Check(err, ErrorMatches, ".*no such file or directory")

	list, err := s.pool.FilepathList(nil)
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"ae/0c/1.deb", "bd/0a/3.deb", "bd/0b/4.deb"})
}

func isSameDevice(s *PackagePoolSuite) bool {
	poolPath, _ := s.pool.buildPoolPath(filepath.Base(s.debFile), &s.checksum)
	fullPoolPath := filepath.Join(s.pool.rootPath, poolPath)
	poolDir := filepath.Dir(fullPoolPath)
	poolDirInfo, _ := os.Stat(poolDir)

	source, _ := os.Open(s.debFile)
	sourceInfo, _ := source.Stat()
	defer source.Close()

	return poolDirInfo.Sys().(*syscall.Stat_t).Dev == sourceInfo.Sys().(*syscall.Stat_t).Dev
}

func (s *PackagePoolSuite) TestImportOk(c *C) {
	path, err := s.pool.Import(s.debFile, filepath.Base(s.debFile), &s.checksum, false, s.cs)
	c.Check(err, IsNil)
	c.Check(path, Equals, "c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb")
	// SHA256 should be automatically calculated
	c.Check(s.checksum.SHA256, Equals, "c76b4bd12fd92e4dfe1b55b18a67a669d92f62985d6a96c8a21d96120982cf12")
	// checksum storage is filled with new checksum
	c.Check(s.cs.(*mockChecksumStorage).store[path].SHA256, Equals, "c76b4bd12fd92e4dfe1b55b18a67a669d92f62985d6a96c8a21d96120982cf12")

	info, err := s.pool.Stat(path)
	c.Assert(err, IsNil)
	c.Check(info.Size(), Equals, int64(2738))
	// /tmp may be on different devices, so hardlinks are not used
	if isSameDevice(s) {
		c.Check(info.Sys().(*syscall.Stat_t).Nlink > 1, Equals, true)
	} else {
		c.Check(info.Sys().(*syscall.Stat_t).Nlink, Equals, uint64(1))
	}

	// import as different name
	path, err = s.pool.Import(s.debFile, "some.deb", &s.checksum, false, s.cs)
	c.Check(err, IsNil)
	c.Check(path, Equals, "c7/6b/4bd12fd92e4dfe1b55b18a67a669_some.deb")
	// checksum storage is filled with new checksum
	c.Check(s.cs.(*mockChecksumStorage).store[path].SHA256, Equals, "c76b4bd12fd92e4dfe1b55b18a67a669d92f62985d6a96c8a21d96120982cf12")

	// double import, should be ok
	s.checksum.SHA512 = "" // clear checksum
	path, err = s.pool.Import(s.debFile, filepath.Base(s.debFile), &s.checksum, false, s.cs)
	c.Check(err, IsNil)
	c.Check(path, Equals, "c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb")
	// checksum is filled back based on checksum storage
	c.Check(s.checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// clear checksum storage, and do double-import
	delete(s.cs.(*mockChecksumStorage).store, path)
	s.checksum.SHA512 = "" // clear checksum
	path, err = s.pool.Import(s.debFile, filepath.Base(s.debFile), &s.checksum, false, s.cs)
	c.Check(err, IsNil)
	c.Check(path, Equals, "c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb")
	// checksum is filled back based on re-calculation of file in the pool
	c.Check(s.checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// import under new name, but with checksums already filled in
	s.checksum.SHA512 = "" // clear checksum
	path, err = s.pool.Import(s.debFile, "other.deb", &s.checksum, false, s.cs)
	c.Check(err, IsNil)
	c.Check(path, Equals, "c7/6b/4bd12fd92e4dfe1b55b18a67a669_other.deb")
	// checksum is filled back based on re-calculation of source file
	c.Check(s.checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")
}

func (s *PackagePoolSuite) TestImportLegacy(c *C) {
	os.MkdirAll(filepath.Join(s.pool.rootPath, "00", "35"), 0755)
	err := utils.CopyFile(s.debFile, filepath.Join(s.pool.rootPath, "00", "35", "libboost-program-options-dev_1.49.0.1_i386.deb"))
	c.Assert(err, IsNil)

	s.checksum.Size = 2738
	var path string
	path, err = s.pool.Import(s.debFile, filepath.Base(s.debFile), &s.checksum, false, s.cs)
	c.Check(err, IsNil)
	c.Check(path, Equals, "00/35/libboost-program-options-dev_1.49.0.1_i386.deb")
	// checksum is filled back based on checksum storage
	c.Check(s.checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")
}

func (s *PackagePoolSuite) TestVerifyLegacy(c *C) {
	s.checksum.Size = 2738
	// file doesn't exist yet
	path, exists, err := s.pool.Verify("", filepath.Base(s.debFile), &s.checksum, s.cs)
	c.Check(path, Equals, "")
	c.Check(err, IsNil)
	c.Check(exists, Equals, false)

	os.MkdirAll(filepath.Join(s.pool.rootPath, "00", "35"), 0755)
	err = utils.CopyFile(s.debFile, filepath.Join(s.pool.rootPath, "00", "35", "libboost-program-options-dev_1.49.0.1_i386.deb"))
	c.Assert(err, IsNil)

	// check existence (and fills back checksum)
	path, exists, err = s.pool.Verify("", filepath.Base(s.debFile), &s.checksum, s.cs)
	c.Check(path, Equals, "00/35/libboost-program-options-dev_1.49.0.1_i386.deb")
	c.Check(err, IsNil)
	c.Check(exists, Equals, true)
	c.Check(s.checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")
}

func (s *PackagePoolSuite) TestVerify(c *C) {
	// file doesn't exist yet
	ppath, exists, err := s.pool.Verify("", filepath.Base(s.debFile), &s.checksum, s.cs)
	c.Check(ppath, Equals, "")
	c.Check(err, IsNil)
	c.Check(exists, Equals, false)

	// import file
	path, err := s.pool.Import(s.debFile, filepath.Base(s.debFile), &s.checksum, false, s.cs)
	c.Check(err, IsNil)
	c.Check(path, Equals, "c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb")

	// check existence
	ppath, exists, err = s.pool.Verify("", filepath.Base(s.debFile), &s.checksum, s.cs)
	c.Check(ppath, Equals, ppath)
	c.Check(err, IsNil)
	c.Check(exists, Equals, true)
	c.Check(s.checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// check existence with fixed path
	ppath, exists, err = s.pool.Verify(path, filepath.Base(s.debFile), &s.checksum, s.cs)
	c.Check(ppath, Equals, path)
	c.Check(err, IsNil)
	c.Check(exists, Equals, true)
	c.Check(s.checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// check existence, but with missing checksum
	s.checksum.SHA512 = ""
	ppath, exists, err = s.pool.Verify("", filepath.Base(s.debFile), &s.checksum, s.cs)
	c.Check(ppath, Equals, path)
	c.Check(err, IsNil)
	c.Check(exists, Equals, true)
	// checksum is filled back based on checksum storage
	c.Check(s.checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// check existence, with missing checksum info but correct path and size available
	ck := utils.ChecksumInfo{
		Size: s.checksum.Size,
	}
	ppath, exists, err = s.pool.Verify(path, filepath.Base(s.debFile), &ck, s.cs)
	c.Check(ppath, Equals, path)
	c.Check(err, IsNil)
	c.Check(exists, Equals, true)
	// checksum is filled back based on checksum storage
	c.Check(ck.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// check existence, with wrong checksum info but correct path and size available
	ck.SHA256 = "abc"
	ppath, exists, err = s.pool.Verify(path, filepath.Base(s.debFile), &ck, s.cs)
	c.Check(ppath, Equals, "")
	c.Check(err, IsNil)
	c.Check(exists, Equals, false)

	// check existence, with missing checksum and no info in checksum storage
	delete(s.cs.(*mockChecksumStorage).store, path)
	s.checksum.SHA512 = ""
	ppath, exists, err = s.pool.Verify("", filepath.Base(s.debFile), &s.checksum, s.cs)
	c.Check(ppath, Equals, path)
	c.Check(err, IsNil)
	c.Check(exists, Equals, true)
	// checksum is filled back based on re-calculation
	c.Check(s.checksum.SHA512, Equals, "d7302241373da972aa9b9e71d2fd769b31a38f71182aa71bc0d69d090d452c69bb74b8612c002ccf8a89c279ced84ac27177c8b92d20f00023b3d268e6cec69c")

	// check existence, with wrong size
	s.checksum.Size = 13455
	ppath, exists, err = s.pool.Verify("", filepath.Base(s.debFile), &s.checksum, s.cs)
	c.Check(ppath, Equals, "")
	c.Check(err, IsNil)
	c.Check(exists, Equals, false)

	// check existence, with empty checksum info
	ppath, exists, err = s.pool.Verify("", filepath.Base(s.debFile), &utils.ChecksumInfo{}, s.cs)
	c.Check(ppath, Equals, "")
	c.Check(err, IsNil)
	c.Check(exists, Equals, false)
}

func (s *PackagePoolSuite) TestImportMove(c *C) {
	tmpDir := c.MkDir()
	tmpPath := filepath.Join(tmpDir, filepath.Base(s.debFile))

	dst, err := os.Create(tmpPath)
	c.Assert(err, IsNil)

	src, err := os.Open(s.debFile)
	c.Assert(err, IsNil)

	_, err = io.Copy(dst, src)
	c.Assert(err, IsNil)

	c.Assert(dst.Close(), IsNil)
	c.Assert(src.Close(), IsNil)

	path, err := s.pool.Import(tmpPath, filepath.Base(tmpPath), &s.checksum, true, s.cs)
	c.Check(err, IsNil)
	c.Check(path, Equals, "c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb")

	info, err := s.pool.Stat(path)
	c.Assert(err, IsNil)
	c.Check(info.Size(), Equals, int64(2738))
	c.Check(int(info.Sys().(*syscall.Stat_t).Nlink), Equals, 1)
}

func (s *PackagePoolSuite) TestImportNotExist(c *C) {
	_, err := s.pool.Import("no-such-file", "a.deb", &s.checksum, false, s.cs)
	c.Check(err, ErrorMatches, ".*no such file or directory")
}

func (s *PackagePoolSuite) TestImportOverwrite(c *C) {
	os.MkdirAll(filepath.Join(s.pool.rootPath, "c7", "6b"), 0755)
	ioutil.WriteFile(filepath.Join(s.pool.rootPath, "c7", "6b", "4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb"), []byte("1"), 0644)

	_, err := s.pool.Import(s.debFile, filepath.Base(s.debFile), &s.checksum, false, s.cs)
	c.Check(err, ErrorMatches, "unable to import into pool.*")
}

func (s *PackagePoolSuite) TestStat(c *C) {
	path, err := s.pool.Import(s.debFile, filepath.Base(s.debFile), &s.checksum, false, s.cs)
	c.Check(err, IsNil)

	info, err := s.pool.Stat(path)
	c.Assert(err, IsNil)
	c.Check(info.Size(), Equals, int64(2738))

	_, err = s.pool.Stat("do/es/ntexist")
	c.Assert(os.IsNotExist(err), Equals, true)
}

func (s *PackagePoolSuite) TestOpen(c *C) {
	path, err := s.pool.Import(s.debFile, filepath.Base(s.debFile), &s.checksum, false, s.cs)
	c.Check(err, IsNil)

	f, err := s.pool.Open(path)
	c.Assert(err, IsNil)
	contents, err := ioutil.ReadAll(f)
	c.Assert(err, IsNil)
	c.Check(len(contents), Equals, 2738)
	c.Check(f.Close(), IsNil)

	_, err = s.pool.Open("do/es/ntexist")
	c.Assert(os.IsNotExist(err), Equals, true)
}

func (s *PackagePoolSuite) TestLink(c *C) {
	path, err := s.pool.Import(s.debFile, filepath.Base(s.debFile), &s.checksum, false, s.cs)
	c.Check(err, IsNil)

	tmpDir := c.MkDir()
	dstPath := filepath.Join(tmpDir, filepath.Base(s.debFile))
	c.Check(s.pool.Link(path, dstPath), IsNil)

	info, err := os.Stat(dstPath)
	c.Assert(err, IsNil)
	c.Check(info.Size(), Equals, int64(2738))
	if isSameDevice(s) {
		c.Check(info.Sys().(*syscall.Stat_t).Nlink > 2, Equals, true)
	} else {
		c.Check(info.Sys().(*syscall.Stat_t).Nlink, Equals, uint64(2))
	}
}

func (s *PackagePoolSuite) TestSymlink(c *C) {
	path, err := s.pool.Import(s.debFile, filepath.Base(s.debFile), &s.checksum, false, s.cs)
	c.Check(err, IsNil)

	tmpDir := c.MkDir()
	dstPath := filepath.Join(tmpDir, filepath.Base(s.debFile))
	c.Check(s.pool.Symlink(path, dstPath), IsNil)

	info, err := os.Stat(dstPath)
	c.Assert(err, IsNil)
	c.Check(info.Size(), Equals, int64(2738))
	if isSameDevice(s) {
		c.Check(info.Sys().(*syscall.Stat_t).Nlink > 2, Equals, true)
	} else {
		c.Check(info.Sys().(*syscall.Stat_t).Nlink, Equals, uint64(1))
	}

	info, err = os.Lstat(dstPath)
	c.Assert(err, IsNil)
	c.Check(int(info.Sys().(*syscall.Stat_t).Mode&syscall.S_IFMT), Equals, int(syscall.S_IFLNK))
}

func (s *PackagePoolSuite) TestGenerateRandomPath(c *C) {
	path, err := s.pool.GenerateTempPath("a.deb")
	c.Check(err, IsNil)

	c.Check(path, Matches, ".+/[0-9a-f][0-9a-f]/[0-9a-f][0-9a-f]/[0-9a-f-]+a\\.deb")
}
