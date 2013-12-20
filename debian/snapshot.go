package debian

import (
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"time"
)

// Snapshot is immutable state of repository: list of packages
type Snapshot struct {
	// Persisten internal ID
	UUID string
	// Human-readable name
	Name string
	// Date of creation
	CreatedAt time.Time

	// Source: kind + ID
	SourceKind string
	SourceIDs  []string
	// Description of how snapshot was created
	Description string

	packageRefs *PackageRefList
}

// NewSnapshotFromRepository creates snapshot from current state of repository
func NewSnapshotFromRepository(name string, repo *RemoteRepo) *Snapshot {
	if repo.packageRefs == nil {
		panic("repo.packageRefs == nil")
	}

	return &Snapshot{
		UUID:        uuid.New(),
		Name:        name,
		CreatedAt:   time.Now(),
		SourceKind:  "repo",
		SourceIDs:   []string{repo.UUID},
		Description: fmt.Sprintf("Snapshot from mirror %s", repo),
		packageRefs: repo.packageRefs,
	}
}

// NumPackages returns number of packages in snapshot
func (s *Snapshot) NumPackages() int {
	return s.packageRefs.Len()
}

// Key is a unique id in DB
func (s *Snapshot) Key() []byte {
	return []byte("S" + s.UUID)
}
