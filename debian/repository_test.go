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

func (s *RepositorySuite) TestPublicPath(c *C) {
	c.Assert(s.repo.PublicPath(), Equals, filepath.Join(s.repo.RootPath, "public"))
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
		prefix           string
		component        string
		sourcePath       string
		poolDirectory    string
		expectedFilename string
	}{
		{ // package name regular
			prefix:           "",
			component:        "main",
			sourcePath:       "pool/01/ae/mars-invaders_1.03.deb",
			poolDirectory:    "m/mars-invaders",
			expectedFilename: "pool/main/m/mars-invaders/mars-invaders_1.03.deb",
		},
		{ // lib-like filename
			prefix:           "",
			component:        "main",
			sourcePath:       "pool/01/ae/libmars-invaders_1.03.deb",
			poolDirectory:    "libm/libmars-invaders",
			expectedFilename: "pool/main/libm/libmars-invaders/libmars-invaders_1.03.deb",
		},
		{ // duplicate link, shouldn't panic
			prefix:           "",
			component:        "main",
			sourcePath:       "pool/01/ae/mars-invaders_1.03.deb",
			poolDirectory:    "m/mars-invaders",
			expectedFilename: "pool/main/m/mars-invaders/mars-invaders_1.03.deb",
		},
		{ // prefix & component
			prefix:           "ppa",
			component:        "contrib",
			sourcePath:       "pool/01/ae/libmars-invaders_1.04.deb",
			poolDirectory:    "libm/libmars-invaders",
			expectedFilename: "pool/contrib/libm/libmars-invaders/libmars-invaders_1.04.deb",
		},
	}

	for _, t := range tests {
		t.sourcePath = filepath.Join(s.repo.RootPath, t.sourcePath)

		err := os.MkdirAll(filepath.Dir(t.sourcePath), 0755)
		c.Assert(err, IsNil)

		file, err := os.Create(t.sourcePath)
		c.Assert(err, IsNil)

		file.Write([]byte("Contents"))
		file.Close()

		path, err := s.repo.LinkFromPool(t.prefix, t.component, t.sourcePath, t.poolDirectory)
		c.Assert(err, IsNil)
		c.Assert(path, Equals, t.expectedFilename)

		st, err := os.Stat(filepath.Join(s.repo.RootPath, "public", t.prefix, t.expectedFilename))
		c.Assert(err, IsNil)

		info := st.Sys().(*syscall.Stat_t)
		c.Check(int(info.Nlink), Equals, 2)
	}
}
