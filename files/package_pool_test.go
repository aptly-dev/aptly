package files

import (
	"io/ioutil"
	. "launchpad.net/gocheck"
	"os"
	"path/filepath"
	"runtime"
)

type PackagePoolSuite struct {
	pool *PackagePool
}

var _ = Suite(&PackagePoolSuite{})

func (s *PackagePoolSuite) SetUpTest(c *C) {
	s.pool = NewPackagePool(c.MkDir())

}

func (s *PackagePoolSuite) TestRelativePath(c *C) {
	path, err := s.pool.RelativePath("a/b/package.deb", "91b1a1480b90b9e269ca44d897b12575")
	c.Assert(err, IsNil)
	c.Assert(path, Equals, "91/b1/package.deb")

	_, err = s.pool.RelativePath("/", "91b1a1480b90b9e269ca44d897b12575")
	c.Assert(err, ErrorMatches, ".*is invalid")
	_, err = s.pool.RelativePath("", "91b1a1480b90b9e269ca44d897b12575")
	c.Assert(err, ErrorMatches, ".*is invalid")
	_, err = s.pool.RelativePath("a/b/package.deb", "9")
	c.Assert(err, ErrorMatches, ".*MD5 is missing")
}

func (s *PackagePoolSuite) TestPath(c *C) {
	path, err := s.pool.Path("a/b/package.deb", "91b1a1480b90b9e269ca44d897b12575")
	c.Assert(err, IsNil)
	c.Assert(path, Equals, filepath.Join(s.pool.rootPath, "91/b1/package.deb"))

	_, err = s.pool.Path("/", "91b1a1480b90b9e269ca44d897b12575")
	c.Assert(err, ErrorMatches, ".*is invalid")
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
	_, _File, _, _ := runtime.Caller(0)
	debFile := filepath.Join(filepath.Dir(_File), "../system/files/libboost-program-options-dev_1.49.0.1_i386.deb")

	err := s.pool.Import(debFile, "91b1a1480b90b9e269ca44d897b12575")
	c.Check(err, IsNil)

	info, err := os.Stat(filepath.Join(s.pool.rootPath, "91", "b1", "libboost-program-options-dev_1.49.0.1_i386.deb"))
	c.Check(err, IsNil)
	c.Check(info.Size(), Equals, int64(2738))

	// double import, should be ok
	err = s.pool.Import(debFile, "91b1a1480b90b9e269ca44d897b12575")
	c.Check(err, IsNil)
}

func (s *PackagePoolSuite) TestImportNotExist(c *C) {
	err := s.pool.Import("no-such-file", "91b1a1480b90b9e269ca44d897b12575")
	c.Check(err, ErrorMatches, ".*no such file or directory")
}

func (s *PackagePoolSuite) TestImportOverwrite(c *C) {
	_, _File, _, _ := runtime.Caller(0)
	debFile := filepath.Join(filepath.Dir(_File), "../system/files/libboost-program-options-dev_1.49.0.1_i386.deb")

	os.MkdirAll(filepath.Join(s.pool.rootPath, "91", "b1"), 0755)
	ioutil.WriteFile(filepath.Join(s.pool.rootPath, "91", "b1", "libboost-program-options-dev_1.49.0.1_i386.deb"), []byte("1"), 0644)

	err := s.pool.Import(debFile, "91b1a1480b90b9e269ca44d897b12575")
	c.Check(err, ErrorMatches, "unable to import into pool.*")
}
