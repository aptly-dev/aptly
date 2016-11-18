package deb

import (
	"bytes"
	"errors"
	"fmt"
	"log"

	"github.com/aptly-dev/aptly/database"
	"github.com/pborman/uuid"
	"github.com/ugorji/go/codec"
)

// LocalRepo is a collection of packages created locally
type LocalRepo struct {
	// Permanent internal ID
	UUID string `codec:"UUID" json:"-"`
	// User-assigned name
	Name string
	// Comment
	Comment string
	// DefaultDistribution
	DefaultDistribution string `codec:",omitempty"`
	// DefaultComponent
	DefaultComponent string `codec:",omitempty"`
	// Uploaders configuration
	Uploaders *Uploaders `codec:"Uploaders,omitempty" json:"-"`
	// "Snapshot" of current list of packages
	packageRefs *PackageRefList
}

// NewLocalRepo creates new instance of Debian local repository
func NewLocalRepo(name string, comment string) *LocalRepo {
	return &LocalRepo{
		UUID:    uuid.New(),
		Name:    name,
		Comment: comment,
	}
}

// String interface
func (repo *LocalRepo) String() string {
	if repo.Comment != "" {
		return fmt.Sprintf("[%s]: %s", repo.Name, repo.Comment)
	}
	return fmt.Sprintf("[%s]", repo.Name)
}

// NumPackages return number of packages in local repo
func (repo *LocalRepo) NumPackages() int {
	if repo.packageRefs == nil {
		return 0
	}
	return repo.packageRefs.Len()
}

// RefList returns package list for repo
func (repo *LocalRepo) RefList() *PackageRefList {
	return repo.packageRefs
}

// UpdateRefList changes package list for local repo
func (repo *LocalRepo) UpdateRefList(reflist *PackageRefList) {
	repo.packageRefs = reflist
}

// Encode does msgpack encoding of LocalRepo
func (repo *LocalRepo) Encode() []byte {
	var buf bytes.Buffer

	encoder := codec.NewEncoder(&buf, &codec.MsgpackHandle{})
	encoder.Encode(repo)

	return buf.Bytes()
}

// Decode decodes msgpack representation into LocalRepo
func (repo *LocalRepo) Decode(input []byte) error {
	decoder := codec.NewDecoderBytes(input, &codec.MsgpackHandle{})
	return decoder.Decode(repo)
}

// Key is a unique id in DB
func (repo *LocalRepo) Key() []byte {
	return []byte("L" + repo.UUID)
}

// RefKey is a unique id for package reference list
func (repo *LocalRepo) RefKey() []byte {
	return []byte("E" + repo.UUID)
}

// LocalRepoCollection does listing, updating/adding/deleting of LocalRepos
type LocalRepoCollection struct {
	db    database.Storage
	cache map[string]*LocalRepo
}

// NewLocalRepoCollection loads LocalRepos from DB and makes up collection
func NewLocalRepoCollection(db database.Storage) *LocalRepoCollection {
	return &LocalRepoCollection{
		db:    db,
		cache: make(map[string]*LocalRepo),
	}
}

func (collection *LocalRepoCollection) search(filter func(*LocalRepo) bool, unique bool) []*LocalRepo {
	result := []*LocalRepo(nil)
	for _, r := range collection.cache {
		if filter(r) {
			result = append(result, r)
		}
	}

	if unique && len(result) > 0 {
		return result
	}

	collection.db.ProcessByPrefix([]byte("L"), func(key, blob []byte) error {
		r := &LocalRepo{}
		if err := r.Decode(blob); err != nil {
			log.Printf("Error decoding local repo: %s\n", err)
			return nil
		}

		if filter(r) {
			if _, exists := collection.cache[r.UUID]; !exists {
				collection.cache[r.UUID] = r
				result = append(result, r)
				if unique {
					return errors.New("abort")
				}
			}
		}

		return nil
	})

	return result
}

// Add appends new repo to collection and saves it
func (collection *LocalRepoCollection) Add(repo *LocalRepo) error {
	_, err := collection.ByName(repo.Name)

	if err == nil {
		return fmt.Errorf("local repo with name %s already exists", repo.Name)
	}

	err = collection.Update(repo)
	if err != nil {
		return err
	}

	collection.cache[repo.UUID] = repo
	return nil
}

// Update stores updated information about repo in DB
func (collection *LocalRepoCollection) Update(repo *LocalRepo) error {
	batch := collection.db.CreateBatch()
	err := batch.Put(repo.Key(), repo.Encode())
	if err != nil {
		return err
	}
	if repo.packageRefs != nil {
		err = batch.Put(repo.RefKey(), repo.packageRefs.Encode())
		if err != nil {
			return err
		}
	}
	return batch.Write()
}

// LoadComplete loads additional information for local repo
func (collection *LocalRepoCollection) LoadComplete(repo *LocalRepo) error {
	encoded, err := collection.db.Get(repo.RefKey())
	if err == database.ErrNotFound {
		return nil
	}
	if err != nil {
		return err
	}

	repo.packageRefs = &PackageRefList{}
	return repo.packageRefs.Decode(encoded)
}

// ByName looks up repository by name
func (collection *LocalRepoCollection) ByName(name string) (*LocalRepo, error) {
	result := collection.search(func(r *LocalRepo) bool { return r.Name == name }, true)
	if len(result) == 0 {
		return nil, fmt.Errorf("local repo with name %s not found", name)
	}

	return result[0], nil
}

// ByUUID looks up repository by uuid
func (collection *LocalRepoCollection) ByUUID(uuid string) (*LocalRepo, error) {
	if r, ok := collection.cache[uuid]; ok {
		return r, nil
	}

	key := (&LocalRepo{UUID: uuid}).Key()

	value, err := collection.db.Get(key)
	if err == database.ErrNotFound {
		return nil, fmt.Errorf("local repo with uuid %s not found", uuid)
	}

	if err != nil {
		return nil, err
	}

	r := &LocalRepo{}
	err = r.Decode(value)

	if err == nil {
		collection.cache[r.UUID] = r
	}

	return r, err
}

// ForEach runs method for each repository
func (collection *LocalRepoCollection) ForEach(handler func(*LocalRepo) error) error {
	return collection.db.ProcessByPrefix([]byte("L"), func(key, blob []byte) error {
		r := &LocalRepo{}
		if err := r.Decode(blob); err != nil {
			log.Printf("Error decoding repo: %s\n", err)
			return nil
		}

		return handler(r)
	})
}

// Len returns number of remote repos
func (collection *LocalRepoCollection) Len() int {
	return len(collection.db.KeysByPrefix([]byte("L")))
}

// Drop removes remote repo from collection
func (collection *LocalRepoCollection) Drop(repo *LocalRepo) error {
	transaction, err := collection.db.OpenTransaction()
	if err != nil {
		return err
	}
	defer transaction.Discard()

	delete(collection.cache, repo.UUID)

	batch := collection.db.CreateBatch()
	err = batch.Delete(repo.Key())
	if err != nil {
		return err
	}

	err = batch.Delete(repo.RefKey())
	if err != nil {
		return err
	}

	return batch.Write()
}
