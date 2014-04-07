package deb

import (
	"errors"
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
	s.repo, _ = NewRemoteRepo("yandex", "http://mirror.yandex.ru/debian/", "squeeze", []string{"main"}, []string{}, false)
	s.repo.packageRefs = s.reflist
}

func (s *SnapshotSuite) TestNewSnapshotFromRepository(c *C) {
	snapshot, _ := NewSnapshotFromRepository("snap1", s.repo)
	c.Check(snapshot.Name, Equals, "snap1")
	c.Check(snapshot.NumPackages(), Equals, 3)
	c.Check(snapshot.RefList().Len(), Equals, 3)
	c.Check(snapshot.SourceKind, Equals, "repo")
	c.Check(snapshot.SourceIDs, DeepEquals, []string{s.repo.UUID})

	s.repo.packageRefs = nil
	_, err := NewSnapshotFromRepository("snap2", s.repo)
	c.Check(err, ErrorMatches, ".*not updated")
}

func (s *SnapshotSuite) TestNewSnapshotFromLocalRepo(c *C) {
	localRepo := NewLocalRepo("lala", "hoorah!")

	_, err := NewSnapshotFromLocalRepo("snap2", localRepo)
	c.Check(err, ErrorMatches, "local repo doesn't have packages")

	localRepo.UpdateRefList(s.reflist)
	snapshot, _ := NewSnapshotFromLocalRepo("snap1", localRepo)
	c.Check(snapshot.Name, Equals, "snap1")
	c.Check(snapshot.NumPackages(), Equals, 3)
	c.Check(snapshot.RefList().Len(), Equals, 3)
	c.Check(snapshot.SourceKind, Equals, "local")
	c.Check(snapshot.SourceIDs, DeepEquals, []string{localRepo.UUID})

}

func (s *SnapshotSuite) TestNewSnapshotFromPackageList(c *C) {
	snap, _ := NewSnapshotFromRepository("snap1", s.repo)

	snapshot := NewSnapshotFromPackageList("snap2", []*Snapshot{snap}, s.list, "Pulled")
	c.Check(snapshot.Name, Equals, "snap2")
	c.Check(snapshot.NumPackages(), Equals, 3)
	c.Check(snapshot.SourceKind, Equals, "snapshot")
	c.Check(snapshot.SourceIDs, DeepEquals, []string{snap.UUID})
}

func (s *SnapshotSuite) TestNewSnapshotFromRefList(c *C) {
	snap, _ := NewSnapshotFromRepository("snap1", s.repo)

	snapshot := NewSnapshotFromRefList("snap2", []*Snapshot{snap}, s.reflist, "Merged")
	c.Check(snapshot.Name, Equals, "snap2")
	c.Check(snapshot.NumPackages(), Equals, 3)
	c.Check(snapshot.SourceKind, Equals, "snapshot")
	c.Check(snapshot.SourceIDs, DeepEquals, []string{snap.UUID})
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
	lrepo1, lrepo2       *LocalRepo
	snapshot1, snapshot2 *Snapshot
	snapshot3, snapshot4 *Snapshot
	collection           *SnapshotCollection
}

var _ = Suite(&SnapshotCollectionSuite{})

func (s *SnapshotCollectionSuite) SetUpTest(c *C) {
	s.db, _ = database.OpenDB(c.MkDir())
	s.collection = NewSnapshotCollection(s.db)
	s.SetUpPackages()

	s.repo1, _ = NewRemoteRepo("yandex", "http://mirror.yandex.ru/debian/", "squeeze", []string{"main"}, []string{}, false)
	s.repo1.packageRefs = s.reflist
	s.snapshot1, _ = NewSnapshotFromRepository("snap1", s.repo1)

	s.repo2, _ = NewRemoteRepo("android", "http://mirror.yandex.ru/debian/", "lenny", []string{"main"}, []string{}, false)
	s.repo2.packageRefs = s.reflist
	s.snapshot2, _ = NewSnapshotFromRepository("snap2", s.repo2)

	s.lrepo1 = NewLocalRepo("local1", "")
	s.lrepo1.packageRefs = s.reflist
	s.snapshot3, _ = NewSnapshotFromLocalRepo("snap3", s.lrepo1)

	s.lrepo2 = NewLocalRepo("local2", "")
	s.lrepo2.packageRefs = s.reflist
	s.snapshot4, _ = NewSnapshotFromLocalRepo("snap4", s.lrepo2)
}

func (s *SnapshotCollectionSuite) TearDownTest(c *C) {
	s.db.Close()
}

func (s *SnapshotCollectionSuite) TestAddByNameByUUID(c *C) {
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

	snapshot, err = collection.ByUUID(s.snapshot1.UUID)
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

func (s *SnapshotCollectionSuite) TestForEachAndLen(c *C) {
	s.collection.Add(s.snapshot1)
	s.collection.Add(s.snapshot2)

	count := 0
	err := s.collection.ForEach(func(*Snapshot) error {
		count++
		return nil
	})
	c.Assert(count, Equals, 2)
	c.Assert(err, IsNil)

	c.Check(s.collection.Len(), Equals, 2)

	e := errors.New("d")
	err = s.collection.ForEach(func(*Snapshot) error {
		return e
	})
	c.Assert(err, Equals, e)
}

func (s *SnapshotCollectionSuite) TestFindByRemoteRepoSource(c *C) {
	c.Assert(s.collection.Add(s.snapshot1), IsNil)
	c.Assert(s.collection.Add(s.snapshot2), IsNil)

	c.Check(s.collection.ByRemoteRepoSource(s.repo1), DeepEquals, []*Snapshot{s.snapshot1})
	c.Check(s.collection.ByRemoteRepoSource(s.repo2), DeepEquals, []*Snapshot{s.snapshot2})

	repo3, _ := NewRemoteRepo("other", "http://mirror.yandex.ru/debian/", "lenny", []string{"main"}, []string{}, false)

	c.Check(s.collection.ByRemoteRepoSource(repo3), DeepEquals, []*Snapshot{})
}

func (s *SnapshotCollectionSuite) TestFindByLocalRepoSource(c *C) {
	c.Assert(s.collection.Add(s.snapshot1), IsNil)
	c.Assert(s.collection.Add(s.snapshot2), IsNil)
	c.Assert(s.collection.Add(s.snapshot3), IsNil)
	c.Assert(s.collection.Add(s.snapshot4), IsNil)

	c.Check(s.collection.ByLocalRepoSource(s.lrepo1), DeepEquals, []*Snapshot{s.snapshot3})
	c.Check(s.collection.ByLocalRepoSource(s.lrepo2), DeepEquals, []*Snapshot{s.snapshot4})

	lrepo3 := NewLocalRepo("other", "")

	c.Check(s.collection.ByLocalRepoSource(lrepo3), DeepEquals, []*Snapshot{})
}

func (s *SnapshotCollectionSuite) TestFindSnapshotSource(c *C) {
	snapshot3 := NewSnapshotFromRefList("snap3", []*Snapshot{s.snapshot1, s.snapshot2}, s.reflist, "desc1")
	snapshot4 := NewSnapshotFromRefList("snap4", []*Snapshot{s.snapshot1}, s.reflist, "desc2")
	snapshot5 := NewSnapshotFromRefList("snap5", []*Snapshot{snapshot3}, s.reflist, "desc3")

	c.Assert(s.collection.Add(s.snapshot1), IsNil)
	c.Assert(s.collection.Add(s.snapshot2), IsNil)
	c.Assert(s.collection.Add(snapshot3), IsNil)
	c.Assert(s.collection.Add(snapshot4), IsNil)
	c.Assert(s.collection.Add(snapshot5), IsNil)

	c.Check(s.collection.BySnapshotSource(s.snapshot1), DeepEquals, []*Snapshot{snapshot3, snapshot4})
	c.Check(s.collection.BySnapshotSource(s.snapshot2), DeepEquals, []*Snapshot{snapshot3})
	c.Check(s.collection.BySnapshotSource(snapshot5), DeepEquals, []*Snapshot{})
}

func (s *SnapshotCollectionSuite) TestDrop(c *C) {
	s.collection.Add(s.snapshot1)
	s.collection.Add(s.snapshot2)

	snap, _ := s.collection.ByUUID(s.snapshot1.UUID)
	c.Check(snap, Equals, s.snapshot1)

	err := s.collection.Drop(s.snapshot1)
	c.Check(err, IsNil)

	_, err = s.collection.ByUUID(s.snapshot1.UUID)
	c.Check(err, ErrorMatches, "snapshot .* not found")

	collection := NewSnapshotCollection(s.db)

	_, err = collection.ByUUID(s.snapshot1.UUID)
	c.Check(err, ErrorMatches, "snapshot .* not found")

	c.Check(func() { s.collection.Drop(s.snapshot1) }, Panics, "snapshot not found!")
}
