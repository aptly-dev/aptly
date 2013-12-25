package debian

import (
	. "launchpad.net/gocheck"
	"os"
	"path/filepath"
	"syscall"
)

type RepositorySuite struct {
	repo *Repository
}

var _ = Suite(&RepositorySuite{})

func (s *RepositorySuite) SetUpTest(c *C) {
	s.repo = NewRepository(c.MkDir())
}

func (s *RepositorySuite) TestPoolPath(c *C) {
	path, err := s.repo.PoolPath("a/b/package.deb", "91b1a1480b90b9e269ca44d897b12575")
	c.Assert(err, IsNil)
	c.Assert(path, Equals, filepath.Join(s.repo.RootPath, "pool", "91/b1/package.deb"))

	_, err = s.repo.PoolPath("/", "91b1a1480b90b9e269ca44d897b12575")
	c.Assert(err, ErrorMatches, ".*is invalid")
	_, err = s.repo.PoolPath("", "91b1a1480b90b9e269ca44d897b12575")
	c.Assert(err, ErrorMatches, ".*is invalid")
}

func (s *RepositorySuite) TestMkDir(c *C) {
	err := s.repo.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	_, err = os.Stat(filepath.Join(s.repo.RootPath, "public/ppa/dists/squeeze/"))
	c.Assert(err, IsNil)
}

func (s *RepositorySuite) TestCreateFile(c *C) {
	err := s.repo.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	file, err := s.repo.CreateFile("ppa/dists/squeeze/Release")
	c.Assert(err, IsNil)
	defer file.Close()

	_, err = os.Stat(filepath.Join(s.repo.RootPath, "public/ppa/dists/squeeze/Release"))
	c.Assert(err, IsNil)
}

func (s *RepositorySuite) TestLinkFromPool(c *C) {
	tests := []struct {
		packageFilename  string
		MD5              string
		expectedFilename string
	}{
		{
			packageFilename:  "pool/m/mars-invaders_1.03.deb",
			MD5:              "91b1a1480b90b9e269ca44d897b12575",
			expectedFilename: "public/pool/main/m/mars-invaders_1.03.deb",
		},
		{
			packageFilename:  "pool/libm/libmars-invaders_1.03.deb",
			MD5:              "12c2a1480b90b9e269ca44d897b12575",
			expectedFilename: "public/pool/main/libm/libmars-invaders_1.03.deb",
		},
	}

	for _, t := range tests {
		poolPath, err := s.repo.PoolPath(t.packageFilename, t.MD5)
		c.Assert(err, IsNil)

		err = os.MkdirAll(filepath.Dir(poolPath), 0755)
		c.Assert(err, IsNil)

		file, err := os.Create(poolPath)
		c.Assert(err, IsNil)

		file.Write([]byte("Contents"))
		file.Close()

		err = s.repo.LinkFromPool("", "main", t.packageFilename, t.MD5)
		c.Assert(err, IsNil)

		st, err := os.Stat(filepath.Join(s.repo.RootPath, t.expectedFilename))
		c.Assert(err, IsNil)

		info := st.Sys().(*syscall.Stat_t)
		c.Check(int(info.Nlink), Equals, 2)
	}
}
