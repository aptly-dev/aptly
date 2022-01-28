package deb

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/aptly-dev/aptly/database"
	"github.com/aptly-dev/aptly/utils"
	"github.com/pborman/uuid"
	"github.com/ugorji/go/codec"
)

// Snapshot is immutable state of repository: list of packages
type Snapshot struct {
	// Persisten internal ID
	UUID string `codec:"UUID" json:"-"`
	// Human-readable name
	Name string
	// Date of creation
	CreatedAt time.Time

	// Source: kind + ID
	SourceKind string   `codec:"SourceKind"`
	SourceIDs  []string `codec:"SourceIDs" json:"-"`
	// Sources
	Snapshots   []*Snapshot   `codec:"-" json:",omitempty"`
	RemoteRepos []*RemoteRepo `codec:"-" json:",omitempty"`
	LocalRepos  []*LocalRepo  `codec:"-" json:",omitempty"`
	Packages    []string      `codec:"-" json:",omitempty"`

	// Description of how snapshot was created
	Description string

	Origin               string
	NotAutomatic         string
	ButAutomaticUpgrades string

	packageRefs *PackageRefList
}

// NewSnapshotFromRepository creates snapshot from current state of repository
func NewSnapshotFromRepository(name string, repo *RemoteRepo) (*Snapshot, error) {
	if repo.packageRefs == nil {
		return nil, errors.New("mirror not updated")
	}

	return &Snapshot{
		UUID:                 uuid.New(),
		Name:                 name,
		CreatedAt:            time.Now(),
		SourceKind:           SourceRemoteRepo,
		SourceIDs:            []string{repo.UUID},
		Description:          fmt.Sprintf("Snapshot from mirror %s", repo),
		Origin:               repo.Meta["Origin"],
		NotAutomatic:         repo.Meta["NotAutomatic"],
		ButAutomaticUpgrades: repo.Meta["ButAutomaticUpgrades"],
		packageRefs:          repo.packageRefs,
	}, nil
}

// NewSnapshotFromLocalRepo creates snapshot from current state of local repository
func NewSnapshotFromLocalRepo(name string, repo *LocalRepo) (*Snapshot, error) {
	snap := &Snapshot{
		UUID:        uuid.New(),
		Name:        name,
		CreatedAt:   time.Now(),
		SourceKind:  SourceLocalRepo,
		SourceIDs:   []string{repo.UUID},
		Description: fmt.Sprintf("Snapshot from local repo %s", repo),
		packageRefs: repo.packageRefs,
	}

	if snap.packageRefs == nil {
		snap.packageRefs = NewPackageRefList()
	}

	return snap, nil
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

// ResourceKey is a unique identifier of the resource
// this snapshot uses. Instead of uuid it uses name
// which needs to be unique as well.
func (s *Snapshot) ResourceKey() []byte {
	return []byte("S" + s.Name)
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
		} else if strings.Contains(err.Error(), "invalid length of bytes for decoding time") {
			// DB created by old codec version, time.Time is not builtin type.
			// https://github.com/ugorji/go-codec/issues/269
			decoder := codec.NewDecoderBytes(input, &codec.MsgpackHandle{
				// only can be configured in Deprecated BasicHandle struct
				BasicHandle: codec.BasicHandle{ // nolint: staticcheck
					TimeNotBuiltin: true,
				},
			})
			if err = decoder.Decode(s); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

// SnapshotCollection does listing, updating/adding/deleting of Snapshots
type SnapshotCollection struct {
	db    database.Storage
	cache map[string]*Snapshot
}

// NewSnapshotCollection loads Snapshots from DB and makes up collection
func NewSnapshotCollection(db database.Storage) *SnapshotCollection {
	return &SnapshotCollection{
		db:    db,
		cache: map[string]*Snapshot{},
	}
}

// Add appends new repo to collection and saves it
func (collection *SnapshotCollection) Add(snapshot *Snapshot) error {
	_, err := collection.ByName(snapshot.Name)
	if err == nil {
		return fmt.Errorf("snapshot with name %s already exists", snapshot.Name)
	}

	err = collection.Update(snapshot)
	if err != nil {
		return err
	}

	collection.cache[snapshot.UUID] = snapshot
	return nil
}

// Update stores updated information about snapshot in DB
func (collection *SnapshotCollection) Update(snapshot *Snapshot) error {
	batch := collection.db.CreateBatch()

	batch.Put(snapshot.Key(), snapshot.Encode())
	if snapshot.packageRefs != nil {
		batch.Put(snapshot.RefKey(), snapshot.packageRefs.Encode())
	}

	return batch.Write()
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

func (collection *SnapshotCollection) search(filter func(*Snapshot) bool, unique bool) []*Snapshot {
	result := []*Snapshot(nil)
	for _, s := range collection.cache {
		if filter(s) {
			result = append(result, s)
		}
	}

	if unique && len(result) > 0 {
		return result
	}

	collection.db.ProcessByPrefix([]byte("S"), func(key, blob []byte) error {
		s := &Snapshot{}
		if err := s.Decode(blob); err != nil {
			log.Printf("Error decoding snapshot: %s\n", err)
			return nil
		}

		if filter(s) {
			if _, exists := collection.cache[s.UUID]; !exists {
				collection.cache[s.UUID] = s
				result = append(result, s)
				if unique {
					return errors.New("abort")
				}
			}
		}

		return nil
	})

	return result
}

// ByName looks up snapshot by name
func (collection *SnapshotCollection) ByName(name string) (*Snapshot, error) {
	result := collection.search(func(s *Snapshot) bool { return s.Name == name }, true)
	if len(result) > 0 {
		return result[0], nil
	}

	return nil, fmt.Errorf("snapshot with name %s not found", name)
}

// ByUUID looks up snapshot by UUID
func (collection *SnapshotCollection) ByUUID(uuid string) (*Snapshot, error) {
	if s, ok := collection.cache[uuid]; ok {
		return s, nil
	}

	key := (&Snapshot{UUID: uuid}).Key()

	value, err := collection.db.Get(key)
	if err == database.ErrNotFound {
		return nil, fmt.Errorf("snapshot with uuid %s not found", uuid)
	}
	if err != nil {
		return nil, err
	}

	s := &Snapshot{}
	err = s.Decode(value)

	if err == nil {
		collection.cache[s.UUID] = s
	}

	return s, err
}

// ByRemoteRepoSource looks up snapshots that have specified RemoteRepo as a source
func (collection *SnapshotCollection) ByRemoteRepoSource(repo *RemoteRepo) []*Snapshot {
	return collection.search(func(s *Snapshot) bool {
		return s.SourceKind == SourceRemoteRepo && utils.StrSliceHasItem(s.SourceIDs, repo.UUID)
	}, false)
}

// ByLocalRepoSource looks up snapshots that have specified LocalRepo as a source
func (collection *SnapshotCollection) ByLocalRepoSource(repo *LocalRepo) []*Snapshot {
	return collection.search(func(s *Snapshot) bool {
		return s.SourceKind == SourceLocalRepo && utils.StrSliceHasItem(s.SourceIDs, repo.UUID)
	}, false)
}

// BySnapshotSource looks up snapshots that have specified snapshot as a source
func (collection *SnapshotCollection) BySnapshotSource(snapshot *Snapshot) []*Snapshot {
	return collection.search(func(s *Snapshot) bool {
		return s.SourceKind == "snapshot" && utils.StrSliceHasItem(s.SourceIDs, snapshot.UUID)
	}, false)
}

// ForEach runs method for each snapshot
func (collection *SnapshotCollection) ForEach(handler func(*Snapshot) error) error {
	return collection.db.ProcessByPrefix([]byte("S"), func(key, blob []byte) error {
		s := &Snapshot{}
		if err := s.Decode(blob); err != nil {
			log.Printf("Error decoding snapshot: %s\n", err)
			return nil
		}

		return handler(s)
	})
}

// ForEachSorted runs method for each snapshot following some sort order
func (collection *SnapshotCollection) ForEachSorted(sortMethod string, handler func(*Snapshot) error) error {
	blobs := collection.db.FetchByPrefix([]byte("S"))
	list := make([]*Snapshot, 0, len(blobs))

	for _, blob := range blobs {
		s := &Snapshot{}
		if err := s.Decode(blob); err != nil {
			log.Printf("Error decoding snapshot: %s\n", err)
		} else {
			list = append(list, s)
		}
	}

	sorter, err := newSnapshotSorter(sortMethod, list)
	if err != nil {
		return err
	}

	for _, s := range sorter.list {
		err = handler(s)
		if err != nil {
			return err
		}
	}

	return nil
}

// Len returns number of snapshots in collection
// ForEach runs method for each snapshot
func (collection *SnapshotCollection) Len() int {
	return len(collection.db.KeysByPrefix([]byte("S")))
}

// Drop removes snapshot from collection
func (collection *SnapshotCollection) Drop(snapshot *Snapshot) error {
	if _, err := collection.db.Get(snapshot.Key()); err != nil {
		if err == database.ErrNotFound {
			return errors.New("snapshot not found")
		}

		return err
	}

	delete(collection.cache, snapshot.UUID)

	batch := collection.db.CreateBatch()
	batch.Delete(snapshot.Key())
	batch.Delete(snapshot.RefKey())
	return batch.Write()
}

// Snapshot sorting methods
const (
	SortName = iota
	SortTime
)

type snapshotSorter struct {
	list       []*Snapshot
	sortMethod int
}

func newSnapshotSorter(sortMethod string, list []*Snapshot) (*snapshotSorter, error) {
	s := &snapshotSorter{list: list}

	switch sortMethod {
	case "time", "Time":
		s.sortMethod = SortTime
	case "name", "Name":
		s.sortMethod = SortName
	default:
		return nil, fmt.Errorf("sorting method \"%s\" unknown", sortMethod)
	}

	sort.Sort(s)

	return s, nil
}

func (s *snapshotSorter) Swap(i, j int) {
	s.list[i], s.list[j] = s.list[j], s.list[i]
}

func (s *snapshotSorter) Less(i, j int) bool {
	switch s.sortMethod {
	case SortName:
		return s.list[i].Name < s.list[j].Name
	case SortTime:
		return s.list[i].CreatedAt.Before(s.list[j].CreatedAt)
	}
	panic("unknown sort method")
}

func (s *snapshotSorter) Len() int {
	return len(s.list)
}
