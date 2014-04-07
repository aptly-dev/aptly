package deb

import (
	"github.com/smira/aptly/database"
)

// CollectionFactory is a single place to generate all desired collections
type CollectionFactory struct {
	db             database.Storage
	packages       *PackageCollection
	remoteRepos    *RemoteRepoCollection
	snapshots      *SnapshotCollection
	localRepos     *LocalRepoCollection
	publishedRepos *PublishedRepoCollection
}

// NewCollectionFactory creates new factory
func NewCollectionFactory(db database.Storage) *CollectionFactory {
	return &CollectionFactory{db: db}
}

// PackageCollection returns (or creates) new PackageCollection
func (factory *CollectionFactory) PackageCollection() *PackageCollection {
	if factory.packages == nil {
		factory.packages = NewPackageCollection(factory.db)
	}

	return factory.packages
}

// RemoteRepoCollection returns (or creates) new RemoteRepoCollection
func (factory *CollectionFactory) RemoteRepoCollection() *RemoteRepoCollection {
	if factory.remoteRepos == nil {
		factory.remoteRepos = NewRemoteRepoCollection(factory.db)
	}

	return factory.remoteRepos
}

// SnapshotCollection returns (or creates) new SnapshotCollection
func (factory *CollectionFactory) SnapshotCollection() *SnapshotCollection {
	if factory.snapshots == nil {
		factory.snapshots = NewSnapshotCollection(factory.db)
	}

	return factory.snapshots
}

// LocalRepoCollection returns (or creates) new LocalRepoCollection
func (factory *CollectionFactory) LocalRepoCollection() *LocalRepoCollection {
	if factory.localRepos == nil {
		factory.localRepos = NewLocalRepoCollection(factory.db)
	}

	return factory.localRepos
}

// PublishedRepoCollection returns (or creates) new PublishedRepoCollection
func (factory *CollectionFactory) PublishedRepoCollection() *PublishedRepoCollection {
	if factory.publishedRepos == nil {
		factory.publishedRepos = NewPublishedRepoCollection(factory.db)
	}

	return factory.publishedRepos
}
