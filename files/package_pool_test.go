package files

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/smira/aptly/utils"

	. "gopkg.in/check.v1"
)

type PackagePoolSuite struct {
	pool     *PackagePool
	checksum utils.ChecksumInfo
	debFile  string
}

var _ = Suite(&PackagePoolSuite{})

func (s *PackagePoolSuite) SetUpTest(c *C) {
	s.pool = NewPackagePool(c.MkDir())
	s.checksum = utils.ChecksumInfo{
		MD5: "91b1a1480b90b9e269ca44d897b12575",
	}
	_, _File, _, _ := runtime.Caller(0)
	s.debFile = filepath.Join(filepath.Dir(_File), "../system/files/libboost-program-options-dev_1.49.0.1_i386.deb")
}

func (s *PackagePoolSuite) TestLegacyPath(c *C) {
	path, err := s.pool.LegacyPath("a/b/package.deb", &s.checksum)
	c.Assert(err, IsNil)
	c.Assert(path, Equals, "91/b1/package.deb")

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

func (s *PackagePoolSuite) TestImportOk(c *C) {
	path, err := s.pool.Import(s.debFile, filepath.Base(s.debFile), &s.checksum, false)
	c.Check(err, IsNil)
	c.Check(path, Equals, "91/b1/libboost-program-options-dev_1.49.0.1_i386.deb")

	info, err := s.pool.Stat(path)
	c.Assert(err, IsNil)
	c.Check(info.Size(), Equals, int64(2738))
	c.Check(info.Sys().(*syscall.Stat_t).Nlink > 1, Equals, true)

	// import as different name
	path, err = s.pool.Import(s.debFile, "some.deb", &s.checksum, false)
	c.Check(err, IsNil)
	c.Check(path, Equals, "91/b1/some.deb")

	// double import, should be ok
	path, err = s.pool.Import(s.debFile, filepath.Base(s.debFile), &s.checksum, false)
	c.Check(err, IsNil)
	c.Check(path, Equals, "91/b1/libboost-program-options-dev_1.49.0.1_i386.deb")
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

	path, err := s.pool.Import(tmpPath, filepath.Base(tmpPath), &s.checksum, true)
	c.Check(err, IsNil)
	c.Check(path, Equals, "91/b1/libboost-program-options-dev_1.49.0.1_i386.deb")

	info, err := s.pool.Stat(path)
	c.Assert(err, IsNil)
	c.Check(info.Size(), Equals, int64(2738))
	c.Check(info.Sys().(*syscall.Stat_t).Nlink, Equals, uint16(1))
}

func (s *PackagePoolSuite) TestImportNotExist(c *C) {
	_, err := s.pool.Import("no-such-file", "a.deb", &s.checksum, false)
	c.Check(err, ErrorMatches, ".*no such file or directory")
}

func (s *PackagePoolSuite) TestImportOverwrite(c *C) {
	os.MkdirAll(filepath.Join(s.pool.rootPath, "91", "b1"), 0755)
	ioutil.WriteFile(filepath.Join(s.pool.rootPath, "91", "b1", "libboost-program-options-dev_1.49.0.1_i386.deb"), []byte("1"), 0644)

	_, err := s.pool.Import(s.debFile, filepath.Base(s.debFile), &s.checksum, false)
	c.Check(err, ErrorMatches, "unable to import into pool.*")
}

func (s *PackagePoolSuite) TestStat(c *C) {
	path, err := s.pool.Import(s.debFile, filepath.Base(s.debFile), &s.checksum, false)
	c.Check(err, IsNil)

	info, err := s.pool.Stat(path)
	c.Assert(err, IsNil)
	c.Check(info.Size(), Equals, int64(2738))

	_, err = s.pool.Stat("do/es/ntexist")
	c.Assert(os.IsNotExist(err), Equals, true)
}

func (s *PackagePoolSuite) TestOpen(c *C) {
	path, err := s.pool.Import(s.debFile, filepath.Base(s.debFile), &s.checksum, false)
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
	path, err := s.pool.Import(s.debFile, filepath.Base(s.debFile), &s.checksum, false)
	c.Check(err, IsNil)

	tmpDir := c.MkDir()
	dstPath := filepath.Join(tmpDir, filepath.Base(s.debFile))
	c.Check(s.pool.Link(path, dstPath), IsNil)

	info, err := os.Stat(dstPath)
	c.Assert(err, IsNil)
	c.Check(info.Size(), Equals, int64(2738))
	c.Check(info.Sys().(*syscall.Stat_t).Nlink > 2, Equals, true)
}

func (s *PackagePoolSuite) TestGenerateRandomPath(c *C) {
	path, err := s.pool.GenerateTempPath("a.deb")
	c.Check(err, IsNil)

	c.Check(path, Matches, ".+/[0-9a-f][0-9a-f]/[0-9a-f][0-9a-f]/[0-9a-f-]+a\\.deb")
}
