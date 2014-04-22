package deb

import (
	"errors"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/files"
	. "launchpad.net/gocheck"
	"os"
	"path/filepath"
)

type pathExistsChecker struct {
	*CheckerInfo
}

var PathExists = &pathExistsChecker{
	&CheckerInfo{Name: "PathExists", Params: []string{"path"}},
}

func (checker *pathExistsChecker) Check(params []interface{}, names []string) (result bool, error string) {
	_, err := os.Stat(params[0].(string))
	return err == nil, ""
}

type NullSigner struct{}

func (n *NullSigner) Init() error {
	return nil
}

func (n *NullSigner) SetKey(keyRef string) {
}

func (n *NullSigner) SetKeyRing(keyring, secretKeyring string) {
}

func (n *NullSigner) DetachedSign(source string, destination string) error {
	return nil
}

func (n *NullSigner) ClearSign(source string, destination string) error {
	return nil
}

type PublishedRepoSuite struct {
	PackageListMixinSuite
	repo, repo2       *PublishedRepo
	root              string
	publishedStorage  aptly.PublishedStorage
	packagePool       aptly.PackagePool
	localRepo         *LocalRepo
	snapshot          *Snapshot
	db                database.Storage
	factory           *CollectionFactory
	packageCollection *PackageCollection
}

var _ = Suite(&PublishedRepoSuite{})

func (s *PublishedRepoSuite) SetUpTest(c *C) {
	s.SetUpPackages()

	s.db, _ = database.OpenDB(c.MkDir())
	s.factory = NewCollectionFactory(s.db)

	s.root = c.MkDir()
	s.publishedStorage = files.NewPublishedStorage(s.root)
	s.packagePool = files.NewPackagePool(s.root)

	repo, _ := NewRemoteRepo("yandex", "http://mirror.yandex.ru/debian/", "squeeze", []string{"main"}, []string{}, false)
	repo.packageRefs = s.reflist
	s.factory.RemoteRepoCollection().Add(repo)

	s.localRepo = NewLocalRepo("local1", "comment1")
	s.localRepo.packageRefs = s.reflist
	s.factory.LocalRepoCollection().Add(s.localRepo)

	s.snapshot, _ = NewSnapshotFromRepository("snap", repo)
	s.factory.SnapshotCollection().Add(s.snapshot)

	s.packageCollection = s.factory.PackageCollection()
	s.packageCollection.Update(s.p1)
	s.packageCollection.Update(s.p2)
	s.packageCollection.Update(s.p3)

	s.repo, _ = NewPublishedRepo("ppa", "squeeze", "main", nil, s.snapshot, s.factory)

	s.repo2, _ = NewPublishedRepo("ppa", "maverick", "main", nil, s.localRepo, s.factory)

	poolPath, _ := s.packagePool.Path(s.p1.Files()[0].Filename, s.p1.Files()[0].Checksums.MD5)
	err := os.MkdirAll(filepath.Dir(poolPath), 0755)
	f, err := os.Create(poolPath)
	c.Assert(err, IsNil)
	f.Close()
}

func (s *PublishedRepoSuite) TearDownTest(c *C) {
	s.db.Close()
}

func (s *PublishedRepoSuite) TestNewPublishedRepo(c *C) {
	c.Check(s.repo.snapshot, Equals, s.snapshot)
	c.Check(s.repo.SourceKind, Equals, "snapshot")
	c.Check(s.repo.SourceUUID, Equals, s.snapshot.UUID)

	c.Check(s.repo2.localRepo, Equals, s.localRepo)
	c.Check(s.repo2.SourceKind, Equals, "local")
	c.Check(s.repo2.SourceUUID, Equals, s.localRepo.UUID)
	c.Check(s.repo2.packageRefs.Len(), Equals, 3)

	c.Check(s.repo.RefList().Len(), Equals, 3)
	c.Check(s.repo2.RefList().Len(), Equals, 3)
}

func (s *PublishedRepoSuite) TestPrefixNormalization(c *C) {

	for _, t := range []struct {
		prefix        string
		expected      string
		errorExpected string
	}{
		{
			prefix:   "ppa",
			expected: "ppa",
		},
		{
			prefix:   "",
			expected: ".",
		},
		{
			prefix:   "/",
			expected: ".",
		},
		{
			prefix:   "//",
			expected: ".",
		},
		{
			prefix:   "//ppa/",
			expected: "ppa",
		},
		{
			prefix:   "ppa/..",
			expected: ".",
		},
		{
			prefix:   "ppa/ubuntu/",
			expected: "ppa/ubuntu",
		},
		{
			prefix:   "ppa/../ubuntu/",
			expected: "ubuntu",
		},
		{
			prefix:        "../ppa/",
			errorExpected: "invalid prefix .*",
		},
		{
			prefix:        "../ppa/../ppa/",
			errorExpected: "invalid prefix .*",
		},
		{
			prefix:        "ppa/dists",
			errorExpected: "invalid prefix .*",
		},
		{
			prefix:        "ppa/pool",
			errorExpected: "invalid prefix .*",
		},
	} {
		repo, err := NewPublishedRepo(t.prefix, "squeeze", "main", nil, s.snapshot, s.factory)
		if t.errorExpected != "" {
			c.Check(err, ErrorMatches, t.errorExpected)
		} else {
			c.Check(repo.Prefix, Equals, t.expected)
		}
	}
}

func (s *PublishedRepoSuite) TestDistributionComponentGuessing(c *C) {
	repo, err := NewPublishedRepo("ppa", "", "", nil, s.snapshot, s.factory)
	c.Check(err, IsNil)
	c.Check(repo.Distribution, Equals, "squeeze")
	c.Check(repo.Component, Equals, "main")

	repo, err = NewPublishedRepo("ppa", "wheezy", "", nil, s.snapshot, s.factory)
	c.Check(err, IsNil)
	c.Check(repo.Distribution, Equals, "wheezy")
	c.Check(repo.Component, Equals, "main")

	repo, err = NewPublishedRepo("ppa", "", "non-free", nil, s.snapshot, s.factory)
	c.Check(err, IsNil)
	c.Check(repo.Distribution, Equals, "squeeze")
	c.Check(repo.Component, Equals, "non-free")

	repo, err = NewPublishedRepo("ppa", "squeeze", "", nil, s.localRepo, s.factory)
	c.Check(err, IsNil)
	c.Check(repo.Distribution, Equals, "squeeze")
	c.Check(repo.Component, Equals, "main")

	repo, err = NewPublishedRepo("ppa", "", "main", nil, s.localRepo, s.factory)
	c.Check(err, ErrorMatches, "unable to guess distribution name, please specify explicitly")

	s.localRepo.DefaultDistribution = "precise"
	s.localRepo.DefaultComponent = "contrib"
	s.factory.LocalRepoCollection().Update(s.localRepo)

	repo, err = NewPublishedRepo("ppa", "", "", nil, s.localRepo, s.factory)
	c.Check(err, IsNil)
	c.Check(repo.Distribution, Equals, "precise")
	c.Check(repo.Component, Equals, "contrib")
}

func (s *PublishedRepoSuite) TestPublish(c *C) {
	err := s.repo.Publish(s.packagePool, s.publishedStorage, s.factory, &NullSigner{}, nil)
	c.Assert(err, IsNil)

	c.Check(s.repo.Architectures, DeepEquals, []string{"i386"})

	rf, err := os.Open(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/squeeze/Release"))
	c.Assert(err, IsNil)

	cfr := NewControlFileReader(rf)
	st, err := cfr.ReadStanza()
	c.Assert(err, IsNil)

	c.Check(st["Origin"], Equals, "ppa squeeze")
	c.Check(st["Components"], Equals, "main")
	c.Check(st["Architectures"], Equals, "i386")

	pf, err := os.Open(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/squeeze/main/binary-i386/Packages"))
	c.Assert(err, IsNil)

	cfr = NewControlFileReader(pf)

	for i := 0; i < 3; i++ {
		st, err = cfr.ReadStanza()
		c.Assert(err, IsNil)

		c.Check(st["Filename"], Equals, "pool/main/a/alien-arena/alien-arena-common_7.40-2_i386.deb")
	}

	st, err = cfr.ReadStanza()
	c.Assert(err, IsNil)
	c.Assert(st, IsNil)

	_, err = os.Stat(filepath.Join(s.publishedStorage.PublicPath(), "ppa/pool/main/a/alien-arena/alien-arena-common_7.40-2_i386.deb"))
	c.Assert(err, IsNil)
}

func (s *PublishedRepoSuite) TestPublishNoSigner(c *C) {
	err := s.repo.Publish(s.packagePool, s.publishedStorage, s.factory, nil, nil)
	c.Assert(err, IsNil)

	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/squeeze/Release"), PathExists)
}

func (s *PublishedRepoSuite) TestPublishLocalRepo(c *C) {
	err := s.repo2.Publish(s.packagePool, s.publishedStorage, s.factory, nil, nil)
	c.Assert(err, IsNil)

	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/maverick/Release"), PathExists)
}

func (s *PublishedRepoSuite) TestString(c *C) {
	c.Check(s.repo.String(), Equals,
		"ppa/squeeze (main) [] publishes [snap]: Snapshot from mirror [yandex]: http://mirror.yandex.ru/debian/ squeeze")
	c.Check(s.repo2.String(), Equals,
		"ppa/maverick (main) [] publishes [local1]: comment1")
	repo, _ := NewPublishedRepo("", "squeeze", "main", []string{"s390"}, s.snapshot, s.factory)
	c.Check(repo.String(), Equals,
		"./squeeze (main) [s390] publishes [snap]: Snapshot from mirror [yandex]: http://mirror.yandex.ru/debian/ squeeze")
	repo, _ = NewPublishedRepo("", "squeeze", "main", []string{"i386", "amd64"}, s.snapshot, s.factory)
	c.Check(repo.String(), Equals,
		"./squeeze (main) [i386, amd64] publishes [snap]: Snapshot from mirror [yandex]: http://mirror.yandex.ru/debian/ squeeze")
	repo.Origin = "myorigin"
	c.Check(repo.String(), Equals,
		"./squeeze (main, origin: myorigin) [i386, amd64] publishes [snap]: Snapshot from mirror [yandex]: http://mirror.yandex.ru/debian/ squeeze")
	repo.Label = "mylabel"
	c.Check(repo.String(), Equals,
		"./squeeze (main, origin: myorigin, label: mylabel) [i386, amd64] publishes [snap]: Snapshot from mirror [yandex]: http://mirror.yandex.ru/debian/ squeeze")
}

func (s *PublishedRepoSuite) TestKey(c *C) {
	c.Check(s.repo.Key(), DeepEquals, []byte("Uppa>>squeeze"))
}

func (s *PublishedRepoSuite) TestRefKey(c *C) {
	c.Check(s.repo.RefKey(), DeepEquals, []byte("E"+s.repo.UUID))
}

func (s *PublishedRepoSuite) TestEncodeDecode(c *C) {
	encoded := s.repo.Encode()
	repo := &PublishedRepo{}
	err := repo.Decode(encoded)

	s.repo.snapshot = nil
	c.Assert(err, IsNil)
	c.Assert(repo, DeepEquals, s.repo)

	encoded2 := s.repo2.Encode()
	repo2 := &PublishedRepo{}
	err = repo2.Decode(encoded2)

	s.repo2.localRepo = nil
	s.repo2.packageRefs = nil
	c.Assert(err, IsNil)
	c.Assert(repo2, DeepEquals, s.repo2)
}

type PublishedRepoCollectionSuite struct {
	PackageListMixinSuite
	db                         database.Storage
	factory                    *CollectionFactory
	snapshotCollection         *SnapshotCollection
	collection                 *PublishedRepoCollection
	snap1, snap2               *Snapshot
	localRepo                  *LocalRepo
	repo1, repo2, repo3, repo4 *PublishedRepo
}

var _ = Suite(&PublishedRepoCollectionSuite{})

func (s *PublishedRepoCollectionSuite) SetUpTest(c *C) {
	s.db, _ = database.OpenDB(c.MkDir())
	s.factory = NewCollectionFactory(s.db)

	s.snapshotCollection = s.factory.SnapshotCollection()

	s.snap1 = NewSnapshotFromPackageList("snap1", []*Snapshot{}, NewPackageList(), "desc1")
	s.snap2 = NewSnapshotFromPackageList("snap2", []*Snapshot{}, NewPackageList(), "desc2")

	s.snapshotCollection.Add(s.snap1)
	s.snapshotCollection.Add(s.snap2)

	s.localRepo = NewLocalRepo("local1", "comment1")
	s.factory.LocalRepoCollection().Add(s.localRepo)

	s.repo1, _ = NewPublishedRepo("ppa", "anaconda", "main", []string{}, s.snap1, s.factory)
	s.repo2, _ = NewPublishedRepo("", "anaconda", "main", []string{}, s.snap2, s.factory)
	s.repo3, _ = NewPublishedRepo("ppa", "anaconda", "main", []string{}, s.snap2, s.factory)
	s.repo4, _ = NewPublishedRepo("ppa", "precise", "main", []string{}, s.localRepo, s.factory)

	s.collection = s.factory.PublishedRepoCollection()
}

func (s *PublishedRepoCollectionSuite) TearDownTest(c *C) {
	s.db.Close()
}

func (s *PublishedRepoCollectionSuite) TestAddByPrefixDistribution(c *C) {
	r, err := s.collection.ByPrefixDistribution("ppa", "anaconda")
	c.Assert(err, ErrorMatches, "*.not found")

	c.Assert(s.collection.Add(s.repo1), IsNil)
	c.Assert(s.collection.Add(s.repo1), ErrorMatches, ".*already exists")
	c.Assert(s.collection.CheckDuplicate(s.repo2), IsNil)
	c.Assert(s.collection.Add(s.repo2), IsNil)
	c.Assert(s.collection.Add(s.repo3), ErrorMatches, ".*already exists")
	c.Assert(s.collection.CheckDuplicate(s.repo3), Equals, s.repo1)
	c.Assert(s.collection.Add(s.repo4), IsNil)

	r, err = s.collection.ByPrefixDistribution("ppa", "anaconda")
	c.Assert(err, IsNil)

	err = s.collection.LoadComplete(r, s.factory)
	c.Assert(err, IsNil)
	c.Assert(r.String(), Equals, s.repo1.String())

	collection := NewPublishedRepoCollection(s.db)
	r, err = collection.ByPrefixDistribution("ppa", "anaconda")
	c.Assert(err, IsNil)

	err = s.collection.LoadComplete(r, s.factory)
	c.Assert(err, IsNil)
	c.Assert(r.String(), Equals, s.repo1.String())
}

func (s *PublishedRepoCollectionSuite) TestByUUID(c *C) {
	r, err := s.collection.ByUUID(s.repo1.UUID)
	c.Assert(err, ErrorMatches, "*.not found")

	c.Assert(s.collection.Add(s.repo1), IsNil)

	r, err = s.collection.ByUUID(s.repo1.UUID)
	c.Assert(err, IsNil)

	err = s.collection.LoadComplete(r, s.factory)
	c.Assert(err, IsNil)
	c.Assert(r.String(), Equals, s.repo1.String())
}

func (s *PublishedRepoCollectionSuite) TestUpdateLoadComplete(c *C) {
	c.Assert(s.collection.Update(s.repo1), IsNil)
	c.Assert(s.collection.Update(s.repo4), IsNil)

	collection := NewPublishedRepoCollection(s.db)
	r, err := collection.ByPrefixDistribution("ppa", "anaconda")
	c.Assert(err, IsNil)
	c.Assert(r.snapshot, IsNil)
	c.Assert(s.collection.LoadComplete(r, s.factory), IsNil)
	c.Assert(r.snapshot.UUID, Equals, s.repo1.snapshot.UUID)
	c.Assert(r.RefList().Len(), Equals, 0)

	r, err = collection.ByPrefixDistribution("ppa", "precise")
	c.Assert(err, IsNil)
	c.Assert(r.localRepo, IsNil)
	c.Assert(s.collection.LoadComplete(r, s.factory), IsNil)
	c.Assert(r.localRepo.UUID, Equals, s.repo4.localRepo.UUID)
	c.Assert(r.packageRefs.Len(), Equals, 0)
	c.Assert(r.RefList().Len(), Equals, 0)
}

func (s *PublishedRepoCollectionSuite) TestForEachAndLen(c *C) {
	s.collection.Add(s.repo1)

	count := 0
	err := s.collection.ForEach(func(*PublishedRepo) error {
		count++
		return nil
	})
	c.Assert(count, Equals, 1)
	c.Assert(err, IsNil)

	c.Check(s.collection.Len(), Equals, 1)

	e := errors.New("c")

	err = s.collection.ForEach(func(*PublishedRepo) error {
		return e
	})
	c.Assert(err, Equals, e)
}

func (s *PublishedRepoCollectionSuite) TestBySnapshot(c *C) {
	c.Check(s.collection.Add(s.repo1), IsNil)
	c.Check(s.collection.Add(s.repo2), IsNil)

	c.Check(s.collection.BySnapshot(s.snap1), DeepEquals, []*PublishedRepo{s.repo1})
	c.Check(s.collection.BySnapshot(s.snap2), DeepEquals, []*PublishedRepo{s.repo2})
}

func (s *PublishedRepoCollectionSuite) TestByLocalRepo(c *C) {
	c.Check(s.collection.Add(s.repo1), IsNil)
	c.Check(s.collection.Add(s.repo4), IsNil)

	c.Check(s.collection.ByLocalRepo(s.localRepo), DeepEquals, []*PublishedRepo{s.repo4})
}

type PublishedRepoRemoveSuite struct {
	PackageListMixinSuite
	db                         database.Storage
	factory                    *CollectionFactory
	snapshotCollection         *SnapshotCollection
	collection                 *PublishedRepoCollection
	root                       string
	publishedStorage           aptly.PublishedStorage
	snap1                      *Snapshot
	repo1, repo2, repo3, repo4 *PublishedRepo
}

var _ = Suite(&PublishedRepoRemoveSuite{})

func (s *PublishedRepoRemoveSuite) SetUpTest(c *C) {
	s.db, _ = database.OpenDB(c.MkDir())
	s.factory = NewCollectionFactory(s.db)

	s.snapshotCollection = s.factory.SnapshotCollection()

	s.snap1 = NewSnapshotFromPackageList("snap1", []*Snapshot{}, NewPackageList(), "desc1")

	s.snapshotCollection.Add(s.snap1)

	s.repo1, _ = NewPublishedRepo("ppa", "anaconda", "main", []string{}, s.snap1, s.factory)
	s.repo2, _ = NewPublishedRepo("", "anaconda", "main", []string{}, s.snap1, s.factory)
	s.repo3, _ = NewPublishedRepo("ppa", "meduza", "main", []string{}, s.snap1, s.factory)
	s.repo4, _ = NewPublishedRepo("ppa", "osminog", "contrib", []string{}, s.snap1, s.factory)

	s.collection = s.factory.PublishedRepoCollection()
	s.collection.Add(s.repo1)
	s.collection.Add(s.repo2)
	s.collection.Add(s.repo3)
	s.collection.Add(s.repo4)

	s.root = c.MkDir()
	s.publishedStorage = files.NewPublishedStorage(s.root)
	s.publishedStorage.MkDir("ppa/dists/anaconda")
	s.publishedStorage.MkDir("ppa/dists/meduza")
	s.publishedStorage.MkDir("ppa/dists/osminog")
	s.publishedStorage.MkDir("ppa/pool/main")
	s.publishedStorage.MkDir("ppa/pool/contrib")
	s.publishedStorage.MkDir("dists/anaconda")
	s.publishedStorage.MkDir("pool/main")
}

func (s *PublishedRepoRemoveSuite) TearDownTest(c *C) {
	s.db.Close()
}

func (s *PublishedRepoRemoveSuite) TestRemoveFilesOnlyDist(c *C) {
	s.repo1.RemoveFiles(s.publishedStorage, false, false, nil)

	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/anaconda"), Not(PathExists))
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/meduza"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/osminog"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/pool/main"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/pool/contrib"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "dists/anaconda"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "pool/main"), PathExists)
}

func (s *PublishedRepoRemoveSuite) TestRemoveFilesWithPool(c *C) {
	s.repo1.RemoveFiles(s.publishedStorage, false, true, nil)

	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/anaconda"), Not(PathExists))
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/meduza"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/osminog"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/pool/main"), Not(PathExists))
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/pool/contrib"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "dists/anaconda"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "pool/main"), PathExists)
}

func (s *PublishedRepoRemoveSuite) TestRemoveFilesWithPrefix(c *C) {
	s.repo1.RemoveFiles(s.publishedStorage, true, true, nil)

	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/anaconda"), Not(PathExists))
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/meduza"), Not(PathExists))
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/osminog"), Not(PathExists))
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/pool/main"), Not(PathExists))
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/pool/contrib"), Not(PathExists))
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "dists/anaconda"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "pool/main"), PathExists)
}

func (s *PublishedRepoRemoveSuite) TestRemoveFilesWithPrefixRoot(c *C) {
	s.repo2.RemoveFiles(s.publishedStorage, true, true, nil)

	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/anaconda"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/meduza"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/pool/main"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/pool/contrib"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "dists/anaconda"), Not(PathExists))
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "pool/main"), Not(PathExists))
}

func (s *PublishedRepoRemoveSuite) TestRemoveRepo1and2(c *C) {
	err := s.collection.Remove(s.publishedStorage, "ppa", "anaconda", s.factory, nil)
	c.Check(err, IsNil)

	_, err = s.collection.ByPrefixDistribution("ppa", "anaconda")
	c.Check(err, ErrorMatches, ".*not found")

	collection := NewPublishedRepoCollection(s.db)
	_, err = collection.ByPrefixDistribution("ppa", "anaconda")
	c.Check(err, ErrorMatches, ".*not found")

	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/anaconda"), Not(PathExists))
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/meduza"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/osminog"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/pool/main"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/pool/contrib"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "dists/anaconda"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "pool/main"), PathExists)

	err = s.collection.Remove(s.publishedStorage, "ppa", "anaconda", s.factory, nil)
	c.Check(err, ErrorMatches, ".*not found")

	err = s.collection.Remove(s.publishedStorage, "ppa", "meduza", s.factory, nil)
	c.Check(err, IsNil)

	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/anaconda"), Not(PathExists))
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/meduza"), Not(PathExists))
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/osminog"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/pool/main"), Not(PathExists))
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/pool/contrib"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "dists/anaconda"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "pool/main"), PathExists)
}

func (s *PublishedRepoRemoveSuite) TestRemoveRepo3(c *C) {
	err := s.collection.Remove(s.publishedStorage, ".", "anaconda", s.factory, nil)
	c.Check(err, IsNil)

	_, err = s.collection.ByPrefixDistribution(".", "anaconda")
	c.Check(err, ErrorMatches, ".*not found")

	collection := NewPublishedRepoCollection(s.db)
	_, err = collection.ByPrefixDistribution(".", "anaconda")
	c.Check(err, ErrorMatches, ".*not found")

	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/anaconda"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/meduza"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/dists/osminog"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/pool/main"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "ppa/pool/contrib"), PathExists)
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "dists/"), Not(PathExists))
	c.Check(filepath.Join(s.publishedStorage.PublicPath(), "pool/"), Not(PathExists))
}
