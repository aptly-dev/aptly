package debian

import (
	"github.com/smira/aptly/database"
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
	snapshot, _ := NewSnapshotFromRepository("snap1", s.repo)
	c.Check(snapshot.Name, Equals, "snap1")
	c.Check(snapshot.NumPackages(), Equals, 3)

	s.repo.packageRefs = nil
	_, err := NewSnapshotFromRepository("snap2", s.repo)
	c.Check(err, ErrorMatches, ".*not updated")
}

func (s *SnapshotSuite) TestKey(c *C) {
	snapshot, _ := NewSnapshotFromRepository("snap1", s.repo)
	c.Assert(len(snapshot.Key()), Equals, 37)
	c.Assert(snapshot.Key()[0], Equals, byte('S'))
}

func (s *SnapshotSuite) TestRefKey(c *C) {
	snapshot, _ := NewSnapshotFromRepository("snap1", s.repo)
	c.Assert(len(snapshot.RefKey()), Equals, 37)
	c.Assert(snapshot.RefKey()[0], Equals, byte('E'))
	c.Assert(snapshot.RefKey()[1:], DeepEquals, snapshot.Key()[1:])
}

func (s *SnapshotSuite) TestEncodeDecode(c *C) {
	snapshot, _ := NewSnapshotFromRepository("snap1", s.repo)
	s.repo.packageRefs = s.reflist

	snapshot2 := &Snapshot{}
	c.Assert(snapshot2.Decode(snapshot.Encode()), IsNil)
	c.Assert(snapshot2.Name, Equals, snapshot.Name)
	c.Assert(snapshot2.packageRefs, IsNil)
}

type SnapshotCollectionSuite struct {
	PackageListMixinSuite
	db                   database.Storage
	repo1, repo2         *RemoteRepo
	snapshot1, snapshot2 *Snapshot
	collection           *SnapshotCollection
}

var _ = Suite(&SnapshotCollectionSuite{})

func (s *SnapshotCollectionSuite) SetUpTest(c *C) {
	s.db, _ = database.OpenDB(c.MkDir())
	s.collection = NewSnapshotCollection(s.db)
	s.SetUpPackages()

	s.repo1, _ = NewRemoteRepo("yandex", "http://mirror.yandex.ru/debian/", "squeeze", []string{"main"}, []string{})
	s.repo1.packageRefs = s.reflist
	s.snapshot1, _ = NewSnapshotFromRepository("snap1", s.repo1)

	s.repo2, _ = NewRemoteRepo("android", "http://mirror.yandex.ru/debian/", "lenny", []string{"main"}, []string{})
	s.repo2.packageRefs = s.reflist
	s.snapshot2, _ = NewSnapshotFromRepository("snap2", s.repo2)
}

func (s *SnapshotCollectionSuite) TearDownTest(c *C) {
	s.db.Close()
}

func (s *SnapshotCollectionSuite) TestAddByName(c *C) {
	snapshot, err := s.collection.ByName("snap1")
	c.Assert(err, ErrorMatches, "*.not found")

	c.Assert(s.collection.Add(s.snapshot1), IsNil)
	c.Assert(s.collection.Add(s.snapshot1), ErrorMatches, ".*already exists")

	c.Assert(s.collection.Add(s.snapshot2), IsNil)

	snapshot, err = s.collection.ByName("snap1")
	c.Assert(err, IsNil)
	c.Assert(snapshot.String(), Equals, s.snapshot1.String())

	collection := NewSnapshotCollection(s.db)
	snapshot, err = collection.ByName("snap1")
	c.Assert(err, IsNil)
	c.Assert(snapshot.String(), Equals, s.snapshot1.String())
}

func (s *SnapshotCollectionSuite) TestUpdateLoadComplete(c *C) {
	c.Assert(s.collection.Update(s.snapshot1), IsNil)

	collection := NewSnapshotCollection(s.db)
	snapshot, err := collection.ByName("snap1")
	c.Assert(err, IsNil)
	c.Assert(snapshot.packageRefs, IsNil)

	c.Assert(s.collection.LoadComplete(snapshot), IsNil)
	c.Assert(snapshot.NumPackages(), Equals, 3)
}

func (s *SnapshotCollectionSuite) TestForEach(c *C) {
	s.collection.Add(s.snapshot1)
	s.collection.Add(s.snapshot2)

	count := 0
	s.collection.ForEach(func(*Snapshot) { count++ })
	c.Assert(count, Equals, 2)
}
