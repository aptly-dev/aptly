package debian

import (
	. "launchpad.net/gocheck"
	"path/filepath"
)

type RepositorySuite struct {
	repo *Repository
}

var _ = Suite(&RepositorySuite{})

func (s *RepositorySuite) SetUpTest(c *C) {
	s.repo = NewRepository(c.MkDir())
}

func (s *RepositorySuite) TestPoolPath(c *C) {
	path, err := s.repo.PoolPath("a/b/package.deb")
	c.Assert(err, IsNil)
	c.Assert(path, Equals, filepath.Join(s.repo.RootPath, "pool", "a/b/package.deb"))

	path, err = s.repo.PoolPath("pool/a/b/package.deb")
	c.Assert(err, IsNil)
	c.Assert(path, Equals, filepath.Join(s.repo.RootPath, "pool", "a/b/package.deb"))

	_, err = s.repo.PoolPath("/dev/stdin")
	c.Assert(err, ErrorMatches, "absolute filename.*")

	_, err = s.repo.PoolPath("../../../etc/passwd")
	c.Assert(err, ErrorMatches, ".*starts with dot")

	_, err = s.repo.PoolPath("pool/a/../../../etc/passwd")
	c.Assert(err, ErrorMatches, ".*starts with dot")

	path, err = s.repo.PoolPath("./etc/passwd")
	c.Assert(err, IsNil)
	c.Assert(path, Equals, filepath.Join(s.repo.RootPath, "pool", "etc/passwd"))
}
