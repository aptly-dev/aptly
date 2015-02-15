package deb

import (
	"github.com/smira/aptly/database"
	"sync"
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
}

// NewCollectionFactory creates new factory
func NewCollectionFactory(db database.Storage) *CollectionFactory {
	return &CollectionFactory{Mutex: &sync.Mutex{}, db: db}
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

// Flush removes all references to collections, so that memory could be reclaimed
func (factory *CollectionFactory) Flush() {
	factory.Lock()
	defer factory.Unlock()

	factory.localRepos = nil
	factory.snapshots = nil
	factory.remoteRepos = nil
	factory.publishedRepos = nil
	factory.packages = nil
}
