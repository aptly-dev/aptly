package debian

import (
	"io/ioutil"
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

func (s *RepositorySuite) TestRelativePoolPath(c *C) {
	path, err := s.repo.RelativePoolPath("a/b/package.deb", "91b1a1480b90b9e269ca44d897b12575")
	c.Assert(err, IsNil)
	c.Assert(path, Equals, "91/b1/package.deb")

	_, err = s.repo.RelativePoolPath("/", "91b1a1480b90b9e269ca44d897b12575")
	c.Assert(err, ErrorMatches, ".*is invalid")
	_, err = s.repo.RelativePoolPath("", "91b1a1480b90b9e269ca44d897b12575")
	c.Assert(err, ErrorMatches, ".*is invalid")
	_, err = s.repo.RelativePoolPath("a/b/package.deb", "9")
	c.Assert(err, ErrorMatches, ".*MD5 is missing")
}

func (s *RepositorySuite) TestPoolPath(c *C) {
	path, err := s.repo.PoolPath("a/b/package.deb", "91b1a1480b90b9e269ca44d897b12575")
	c.Assert(err, IsNil)
	c.Assert(path, Equals, filepath.Join(s.repo.RootPath, "pool", "91/b1/package.deb"))

	_, err = s.repo.PoolPath("/", "91b1a1480b90b9e269ca44d897b12575")
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

func (s *RepositorySuite) TestRemoveDirs(c *C) {
	err := s.repo.MkDir("ppa/dists/squeeze/")
	c.Assert(err, IsNil)

	file, err := s.repo.CreateFile("ppa/dists/squeeze/Release")
	c.Assert(err, IsNil)
	defer file.Close()

	err = s.repo.RemoveDirs("ppa/dists/")

	_, err = os.Stat(filepath.Join(s.repo.RootPath, "public/ppa/dists/squeeze/Release"))
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)
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

		err = ioutil.WriteFile(t.sourcePath, []byte("Contents"), 0644)
		c.Assert(err, IsNil)

		path, err := s.repo.LinkFromPool(t.prefix, t.component, t.sourcePath, t.poolDirectory)
		c.Assert(err, IsNil)
		c.Assert(path, Equals, t.expectedFilename)

		st, err := os.Stat(filepath.Join(s.repo.RootPath, "public", t.prefix, t.expectedFilename))
		c.Assert(err, IsNil)

		info := st.Sys().(*syscall.Stat_t)
		c.Check(int(info.Nlink), Equals, 2)
	}
}

func (s *RepositorySuite) TestPoolFilepathList(c *C) {
	list, err := s.repo.PoolFilepathList(nil)
	c.Check(err, IsNil)
	c.Check(list, IsNil)

	os.MkdirAll(filepath.Join(s.repo.RootPath, "pool", "bd", "0b"), 0755)
	os.MkdirAll(filepath.Join(s.repo.RootPath, "pool", "bd", "0a"), 0755)
	os.MkdirAll(filepath.Join(s.repo.RootPath, "pool", "ae", "0c"), 0755)

	list, err = s.repo.PoolFilepathList(nil)
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{})

	ioutil.WriteFile(filepath.Join(s.repo.RootPath, "pool", "ae", "0c", "1.deb"), nil, 0644)
	ioutil.WriteFile(filepath.Join(s.repo.RootPath, "pool", "ae", "0c", "2.deb"), nil, 0644)
	ioutil.WriteFile(filepath.Join(s.repo.RootPath, "pool", "bd", "0a", "3.deb"), nil, 0644)
	ioutil.WriteFile(filepath.Join(s.repo.RootPath, "pool", "bd", "0b", "4.deb"), nil, 0644)

	list, err = s.repo.PoolFilepathList(nil)
	c.Check(err, IsNil)
	c.Check(list, DeepEquals, []string{"ae/0c/1.deb", "ae/0c/2.deb", "bd/0a/3.deb", "bd/0b/4.deb"})
}
