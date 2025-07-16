package deb

import (
	"github.com/aptly-dev/aptly/database"
	"github.com/aptly-dev/aptly/database/goleveldb"
	. "gopkg.in/check.v1"
)

type CollectionsSuite struct {
	db      database.Storage
	factory *CollectionFactory
}

var _ = Suite(&CollectionsSuite{})

func (s *CollectionsSuite) SetUpTest(c *C) {
	s.db, _ = goleveldb.NewOpenDB(c.MkDir())
	s.factory = NewCollectionFactory(s.db)
}

func (s *CollectionsSuite) TearDownTest(c *C) {
	s.db.Close()
}

func (s *CollectionsSuite) TestNewCollectionFactory(c *C) {
	factory := NewCollectionFactory(s.db)
	c.Check(factory, NotNil)
	c.Check(factory.db, Equals, s.db)
	c.Check(factory.Mutex, NotNil)
}

func (s *CollectionsSuite) TestTemporaryDB(c *C) {
	tempDB, err := s.factory.TemporaryDB()
	c.Check(err, IsNil)
	c.Check(tempDB, NotNil)

	// Clean up
	tempDB.Close()
	tempDB.Drop()
}

func (s *CollectionsSuite) TestPackageCollection(c *C) {
	// First call creates the collection
	collection1 := s.factory.PackageCollection()
	c.Check(collection1, NotNil)

	// Second call returns the same instance
	collection2 := s.factory.PackageCollection()
	c.Check(collection2, Equals, collection1)
}

func (s *CollectionsSuite) TestRemoteRepoCollection(c *C) {
	// First call creates the collection
	collection1 := s.factory.RemoteRepoCollection()
	c.Check(collection1, NotNil)

	// Second call returns the same instance
	collection2 := s.factory.RemoteRepoCollection()
	c.Check(collection2, Equals, collection1)
}

func (s *CollectionsSuite) TestSnapshotCollection(c *C) {
	// First call creates the collection
	collection1 := s.factory.SnapshotCollection()
	c.Check(collection1, NotNil)

	// Second call returns the same instance
	collection2 := s.factory.SnapshotCollection()
	c.Check(collection2, Equals, collection1)
}

func (s *CollectionsSuite) TestLocalRepoCollection(c *C) {
	// First call creates the collection
	collection1 := s.factory.LocalRepoCollection()
	c.Check(collection1, NotNil)

	// Second call returns the same instance
	collection2 := s.factory.LocalRepoCollection()
	c.Check(collection2, Equals, collection1)
}

func (s *CollectionsSuite) TestPublishedRepoCollection(c *C) {
	// First call creates the collection
	collection1 := s.factory.PublishedRepoCollection()
	c.Check(collection1, NotNil)

	// Second call returns the same instance
	collection2 := s.factory.PublishedRepoCollection()
	c.Check(collection2, Equals, collection1)
}

func (s *CollectionsSuite) TestChecksumCollectionWithNilDB(c *C) {
	// First call with nil DB creates the collection
	collection1 := s.factory.ChecksumCollection(nil)
	c.Check(collection1, NotNil)

	// Second call with nil DB returns the same instance
	collection2 := s.factory.ChecksumCollection(nil)
	c.Check(collection2, Equals, collection1)
}

func (s *CollectionsSuite) TestChecksumCollectionWithDB(c *C) {
	// Create temporary DB
	tempDB, err := s.factory.TemporaryDB()
	c.Check(err, IsNil)
	defer tempDB.Close()
	defer tempDB.Drop()

	// Call with specific DB creates new collection
	collection1 := s.factory.ChecksumCollection(tempDB)
	c.Check(collection1, NotNil)

	// Call with different DB creates different collection
	collection2 := s.factory.ChecksumCollection(s.db)
	c.Check(collection2, NotNil)
	c.Check(collection2, Not(Equals), collection1)
}

func (s *CollectionsSuite) TestFlush(c *C) {
	// Create all collections
	packages := s.factory.PackageCollection()
	remoteRepos := s.factory.RemoteRepoCollection()
	snapshots := s.factory.SnapshotCollection()
	localRepos := s.factory.LocalRepoCollection()
	publishedRepos := s.factory.PublishedRepoCollection()
	checksums := s.factory.ChecksumCollection(nil)

	c.Check(packages, NotNil)
	c.Check(remoteRepos, NotNil)
	c.Check(snapshots, NotNil)
	c.Check(localRepos, NotNil)
	c.Check(publishedRepos, NotNil)
	c.Check(checksums, NotNil)

	// Flush all collections
	s.factory.Flush()

	// After flush, new calls should create new instances
	newPackages := s.factory.PackageCollection()
	newRemoteRepos := s.factory.RemoteRepoCollection()
	newSnapshots := s.factory.SnapshotCollection()
	newLocalRepos := s.factory.LocalRepoCollection()
	newPublishedRepos := s.factory.PublishedRepoCollection()
	newChecksums := s.factory.ChecksumCollection(nil)

	c.Check(newPackages, Not(Equals), packages)
	c.Check(newRemoteRepos, Not(Equals), remoteRepos)
	c.Check(newSnapshots, Not(Equals), snapshots)
	c.Check(newLocalRepos, Not(Equals), localRepos)
	c.Check(newPublishedRepos, Not(Equals), publishedRepos)
	c.Check(newChecksums, Not(Equals), checksums)
}

func (s *CollectionsSuite) TestConcurrentAccess(c *C) {
	// Test that concurrent access to collections works properly
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			// Each goroutine should get the same instances
			packages := s.factory.PackageCollection()
			remoteRepos := s.factory.RemoteRepoCollection()
			snapshots := s.factory.SnapshotCollection()
			localRepos := s.factory.LocalRepoCollection()
			publishedRepos := s.factory.PublishedRepoCollection()
			checksums := s.factory.ChecksumCollection(nil)

			c.Check(packages, NotNil)
			c.Check(remoteRepos, NotNil)
			c.Check(snapshots, NotNil)
			c.Check(localRepos, NotNil)
			c.Check(publishedRepos, NotNil)
			c.Check(checksums, NotNil)

			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify that all collections are still accessible
	packages := s.factory.PackageCollection()
	c.Check(packages, NotNil)
}

func (s *CollectionsSuite) TestFlushAndRecreate(c *C) {
	// Create collections, use them, flush, then recreate
	originalPackages := s.factory.PackageCollection()
	c.Check(originalPackages, NotNil)

	// Add a package to test that it exists
	pkg := NewPackageFromControlFile(packageStanza.Copy())
	err := originalPackages.Update(pkg)
	c.Check(err, IsNil)

	// Flush
	s.factory.Flush()

	// Get new collection
	newPackages := s.factory.PackageCollection()
	c.Check(newPackages, NotNil)
	c.Check(newPackages, Not(Equals), originalPackages)

	// The package should still exist in the database
	retrievedPkg, err := newPackages.ByKey(pkg.Key(""))
	c.Check(err, IsNil)
	c.Check(retrievedPkg.Name, Equals, pkg.Name)
}
