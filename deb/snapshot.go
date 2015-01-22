package deb

import (
	"bytes"
	"code.google.com/p/go-uuid/uuid"
	"errors"
	"fmt"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/utils"
	"github.com/ugorji/go/codec"
	"log"
	"sort"
	"strings"
	"sync"
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
func NewSnapshotFromRepository(name string, repo *RemoteRepo) (*Snapshot, error) {
	if repo.packageRefs == nil {
		return nil, errors.New("mirror not updated")
	}

	return &Snapshot{
		UUID:        uuid.New(),
		Name:        name,
		CreatedAt:   time.Now(),
		SourceKind:  "repo",
		SourceIDs:   []string{repo.UUID},
		Description: fmt.Sprintf("Snapshot from mirror %s", repo),
		packageRefs: repo.packageRefs,
	}, nil
}

// NewSnapshotFromLocalRepo creates snapshot from current state of local repository
func NewSnapshotFromLocalRepo(name string, repo *LocalRepo) (*Snapshot, error) {
	if repo.packageRefs == nil {
		return nil, errors.New("local repo doesn't have packages")
	}

	return &Snapshot{
		UUID:        uuid.New(),
		Name:        name,
		CreatedAt:   time.Now(),
		SourceKind:  "local",
		SourceIDs:   []string{repo.UUID},
		Description: fmt.Sprintf("Snapshot from local repo %s", repo),
		packageRefs: repo.packageRefs,
	}, nil
}

// NewSnapshotFromPackageList creates snapshot from PackageList
func NewSnapshotFromPackageList(name string, sources []*Snapshot, list *PackageList, description string) *Snapshot {
	return NewSnapshotFromRefList(name, sources, NewPackageRefListFromPackageList(list), description)
}

// NewSnapshotFromRefList creates snapshot from PackageRefList
func NewSnapshotFromRefList(name string, sources []*Snapshot, list *PackageRefList, description string) *Snapshot {
	sourceUUIDs := make([]string, len(sources))
	for i := range sources {
		sourceUUIDs[i] = sources[i].UUID
	}

	return &Snapshot{
		UUID:        uuid.New(),
		Name:        name,
		CreatedAt:   time.Now(),
		SourceKind:  "snapshot",
		SourceIDs:   sourceUUIDs,
		Description: description,
		packageRefs: list,
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

// RefList returns list of package refs in snapshot
func (s *Snapshot) RefList() *PackageRefList {
	return s.packageRefs
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
	err := decoder.Decode(s)
	if err != nil {
		if strings.HasPrefix(err.Error(), "codec.decoder: readContainerLen: Unrecognized descriptor byte: hex: 80") {
			// probably it is broken DB from go < 1.2, try decoding w/o time.Time
			var snapshot11 struct {
				UUID      string
				Name      string
				CreatedAt []byte

				SourceKind  string
				SourceIDs   []string
				Description string
			}

			decoder = codec.NewDecoderBytes(input, &codec.MsgpackHandle{})
			err2 := decoder.Decode(&snapshot11)
			if err2 != nil {
				return err
			}

			s.UUID = snapshot11.UUID
			s.Name = snapshot11.Name
			s.SourceKind = snapshot11.SourceKind
			s.SourceIDs = snapshot11.SourceIDs
			s.Description = snapshot11.Description
		} else {
			return err
		}
	}
	return nil
}

// SnapshotCollection does listing, updating/adding/deleting of Snapshots
type SnapshotCollection struct {
	*sync.RWMutex
	db   database.Storage
	list []*Snapshot
}

// NewSnapshotCollection loads Snapshots from DB and makes up collection
func NewSnapshotCollection(db database.Storage) *SnapshotCollection {
	result := &SnapshotCollection{
		RWMutex: &sync.RWMutex{},
		db:      db,
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
	if snapshot.packageRefs != nil {
		return collection.db.Put(snapshot.RefKey(), snapshot.packageRefs.Encode())
	}
	return nil
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

// ByUUID looks up snapshot by UUID
func (collection *SnapshotCollection) ByUUID(uuid string) (*Snapshot, error) {
	for _, s := range collection.list {
		if s.UUID == uuid {
			return s, nil
		}
	}
	return nil, fmt.Errorf("snapshot with uuid %s not found", uuid)
}

// ByRemoteRepoSource looks up snapshots that have specified RemoteRepo as a source
func (collection *SnapshotCollection) ByRemoteRepoSource(repo *RemoteRepo) []*Snapshot {
	result := make([]*Snapshot, 0)

	for _, s := range collection.list {
		if s.SourceKind == "repo" && utils.StrSliceHasItem(s.SourceIDs, repo.UUID) {
			result = append(result, s)
		}
	}
	return result
}

// ByLocalRepoSource looks up snapshots that have specified LocalRepo as a source
func (collection *SnapshotCollection) ByLocalRepoSource(repo *LocalRepo) []*Snapshot {
	result := make([]*Snapshot, 0)

	for _, s := range collection.list {
		if s.SourceKind == "local" && utils.StrSliceHasItem(s.SourceIDs, repo.UUID) {
			result = append(result, s)
		}
	}
	return result
}

// BySnapshotSource looks up snapshots that have specified snapshot as a source
func (collection *SnapshotCollection) BySnapshotSource(snapshot *Snapshot) []*Snapshot {
	result := make([]*Snapshot, 0)

	for _, s := range collection.list {
		if s.SourceKind == "snapshot" && utils.StrSliceHasItem(s.SourceIDs, snapshot.UUID) {
			result = append(result, s)
		}
	}
	return result
}

// ForEach runs method for each snapshot
func (collection *SnapshotCollection) ForEach(handler func(*Snapshot) error) error {
	var err error
	for _, s := range collection.list {
		err = handler(s)
		if err != nil {
			return err
		}
	}
	return err
}

// Len returns number of snapshots in collection
// ForEach runs method for each snapshot
func (collection *SnapshotCollection) Len() int {
	return len(collection.list)
}

// Drop removes snapshot from collection
func (collection *SnapshotCollection) Drop(snapshot *Snapshot) error {
	snapshotPosition := -1

	for i, s := range collection.list {
		if s == snapshot {
			snapshotPosition = i
			break
		}
	}

	if snapshotPosition == -1 {
		panic("snapshot not found!")
	}

	collection.list[len(collection.list)-1], collection.list[snapshotPosition], collection.list =
		nil, collection.list[len(collection.list)-1], collection.list[:len(collection.list)-1]

	err := collection.db.Delete(snapshot.Key())
	if err != nil {
		return err
	}

	return collection.db.Delete(snapshot.RefKey())
}

// Snapshot sorting methods
const (
	SortName = iota
	SortTime
)

type snapshotListToSort struct {
	list       []*Snapshot
	sortMethod int
}

func parseSortMethod(sortMethod string) (int, error) {
	switch sortMethod {
	case "time", "Time":
		return SortTime, nil
	case "name", "Name":
		return SortName, nil
	}

	return -1, fmt.Errorf("sorting method \"%s\" unknown", sortMethod)
}

func (s snapshotListToSort) Swap(i, j int) {
	s.list[i], s.list[j] = s.list[j], s.list[i]
}

func (s snapshotListToSort) Less(i, j int) bool {
	switch s.sortMethod {
	case SortName:
		return s.list[i].Name < s.list[j].Name
	case SortTime:
		return s.list[i].CreatedAt.Before(s.list[j].CreatedAt)
	}
	panic("unknown sort method")
}

func (s snapshotListToSort) Len() int {
	return len(s.list)
}

func (collection *SnapshotCollection) Sort(sortMethodString string) error {
	var err error
	snapshotsToSort := &snapshotListToSort{}
	snapshotsToSort.list = collection.list
	snapshotsToSort.sortMethod, err = parseSortMethod(sortMethodString)
	if err != nil {
		return err
	}

	sort.Sort(snapshotsToSort)
	collection.list = snapshotsToSort.list

    return err
}
