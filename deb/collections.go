package deb

import (
	"sync"

	"github.com/smira/aptly/database"
)

// CollectionFactory is a single place to generate all desired collections
type CollectionFactory struct {
	*sync.Mutex
	db             database.Storage
	packages       *PackageCollection
	remoteRepos    *RemoteRepoCollection
	snapshots      *SnapshotCollection
	localRepos     *LocalRepoCollection
	publishedRepos *PublishedRepoCollection
	checksums      *ChecksumCollection
}

// NewCollectionFactory creates new factory
func NewCollectionFactory(db database.Storage) *CollectionFactory {
	return &CollectionFactory{Mutex: &sync.Mutex{}, db: db}
}

// TemporaryDB creates new temporary DB
//
// DB should be closed/droped after being used
func (factory *CollectionFactory) TemporaryDB() (database.Storage, error) {
	return factory.db.CreateTemporary()
}

// PackageCollection returns (or creates) new PackageCollection
func (factory *CollectionFactory) PackageCollection() *PackageCollection {
	factory.Lock()
	defer factory.Unlock()

	if factory.packages == nil {
		factory.packages = NewPackageCollection(factory.db)
	}

	return factory.packages
}

// RemoteRepoCollection returns (or creates) new RemoteRepoCollection
func (factory *CollectionFactory) RemoteRepoCollection() *RemoteRepoCollection {
	factory.Lock()
	defer factory.Unlock()

	if factory.remoteRepos == nil {
		factory.remoteRepos = NewRemoteRepoCollection(factory.db)
	}

	return factory.remoteRepos
}

// SnapshotCollection returns (or creates) new SnapshotCollection
func (factory *CollectionFactory) SnapshotCollection() *SnapshotCollection {
	factory.Lock()
	defer factory.Unlock()

	if factory.snapshots == nil {
		factory.snapshots = NewSnapshotCollection(factory.db)
	}

	return factory.snapshots
}

// LocalRepoCollection returns (or creates) new LocalRepoCollection
func (factory *CollectionFactory) LocalRepoCollection() *LocalRepoCollection {
	factory.Lock()
	defer factory.Unlock()

	if factory.localRepos == nil {
		factory.localRepos = NewLocalRepoCollection(factory.db)
	}

	return factory.localRepos
}

// PublishedRepoCollection returns (or creates) new PublishedRepoCollection
func (factory *CollectionFactory) PublishedRepoCollection() *PublishedRepoCollection {
	factory.Lock()
	defer factory.Unlock()

	if factory.publishedRepos == nil {
		factory.publishedRepos = NewPublishedRepoCollection(factory.db)
	}

	return factory.publishedRepos
}

// ChecksumCollection returns (or creates) new ChecksumCollection
func (factory *CollectionFactory) ChecksumCollection() *ChecksumCollection {
	factory.Lock()
	defer factory.Unlock()

	if factory.checksums == nil {
		factory.checksums = NewChecksumCollection(factory.db)
	}

	return factory.checksums
}

// Flush removes all references to collections, so that memory could be reclaimed
func (factory *CollectionFactory) Flush() {
	factory.Lock()
	defer factory.Unlock()

	factory.localRepos = nil
	factory.snapshots = nil
	factory.remoteRepos = nil
	factory.publishedRepos = nil
	factory.packages = nil
	factory.checksums = nil
}
