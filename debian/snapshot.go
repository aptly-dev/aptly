package debian

import (
	"bytes"
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/smira/aptly/database"
	"github.com/ugorji/go/codec"
	"log"
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

// String returns string representation of snapshot
func (s *Snapshot) String() string {
	return fmt.Sprintf("[%s]: %s", s.Name, s.Description)
}

// NumPackages returns number of packages in snapshot
func (s *Snapshot) NumPackages() int {
	return s.packageRefs.Len()
}

// Key is a unique id in DB
func (s *Snapshot) Key() []byte {
	return []byte("S" + s.UUID)
}

// RefKey is a unique id for package reference list
func (s *Snapshot) RefKey() []byte {
	return []byte("E" + s.UUID)
}

// Encode does msgpack encoding of Snapshot
func (s *Snapshot) Encode() []byte {
	var buf bytes.Buffer

	encoder := codec.NewEncoder(&buf, &codec.MsgpackHandle{})
	encoder.Encode(s)

	return buf.Bytes()
}

// Decode decodes msgpack representation into Snapshot
func (s *Snapshot) Decode(input []byte) error {
	decoder := codec.NewDecoderBytes(input, &codec.MsgpackHandle{})
	return decoder.Decode(s)
}

// SnapshotCollection does listing, updating/adding/deleting of Snapshots
type SnapshotCollection struct {
	db   database.Storage
	list []*Snapshot
}

// NewSnapshotCollection loads Snapshots from DB and makes up collection
func NewSnapshotCollection(db database.Storage) *SnapshotCollection {
	result := &SnapshotCollection{
		db: db,
	}

	blobs := db.FetchByPrefix([]byte("S"))
	result.list = make([]*Snapshot, 0, len(blobs))

	for _, blob := range blobs {
		s := &Snapshot{}
		if err := s.Decode(blob); err != nil {
			log.Printf("Error decoding snapshot: %s\n", err)
		} else {
			result.list = append(result.list, s)
		}
	}

	return result
}

// Add appends new repo to collection and saves it
func (collection *SnapshotCollection) Add(snapshot *Snapshot) error {
	for _, s := range collection.list {
		if s.Name == snapshot.Name {
			return fmt.Errorf("snapshot with name %s already exists", snapshot.Name)
		}
	}

	err := collection.Update(snapshot)
	if err != nil {
		return err
	}

	collection.list = append(collection.list, snapshot)
	return nil
}

// Update stores updated information about repo in DB
func (collection *SnapshotCollection) Update(snapshot *Snapshot) error {
	err := collection.db.Put(snapshot.Key(), snapshot.Encode())
	if err != nil {
		return err
	}
	return collection.db.Put(snapshot.RefKey(), snapshot.packageRefs.Encode())
}

// LoadComplete loads additional information about snapshot
func (collection *SnapshotCollection) LoadComplete(snapshot *Snapshot) error {
	encoded, err := collection.db.Get(snapshot.RefKey())
	if err != nil {
		return err
	}

	snapshot.packageRefs = &PackageRefList{}
	return snapshot.packageRefs.Decode(encoded)
}

// ByName looks up snapshot by name
func (collection *SnapshotCollection) ByName(name string) (*Snapshot, error) {
	for _, s := range collection.list {
		if s.Name == name {
			return s, nil
		}
	}
	return nil, fmt.Errorf("snapshot with name %s not found", name)
}

// ForEach runs method for each snapshot
func (collection *SnapshotCollection) ForEach(handler func(*Snapshot)) {
	for _, s := range collection.list {
		handler(s)
	}
}
