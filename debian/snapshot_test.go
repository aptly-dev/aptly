package debian

import (
	. "launchpad.net/gocheck"
)

type SnapshotSuite struct {
	PackageListMixinSuite
	repo *RemoteRepo
}

var _ = Suite(&SnapshotSuite{})

func (s *SnapshotSuite) SetUpTest(c *C) {
	s.SetUpPackages()
	s.repo, _ = NewRemoteRepo("yandex", "http://mirror.yandex.ru/debian/", "squeeze", []string{"main"}, []string{})
	s.repo.packageRefs = s.reflist
}

func (s *SnapshotSuite) TestNewSnapshotFromRepository(c *C) {
	snapshot := NewSnapshotFromRepository("snap1", s.repo)
	c.Check(snapshot.Name, Equals, "snap1")
	c.Check(snapshot.NumPackages(), Equals, 3)

	s.repo.packageRefs = nil
	c.Check(func() { NewSnapshotFromRepository("snap2", s.repo) }, PanicMatches, "repo.packageRefs == nil")
}

func (s *SnapshotSuite) TestKey(c *C) {
	snapshot := NewSnapshotFromRepository("snap1", s.repo)
	c.Assert(len(snapshot.Key()), Equals, 37)
	c.Assert(snapshot.Key()[0], Equals, byte('S'))
}
