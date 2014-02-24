package debian

import (
	"bytes"
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/smira/aptly/database"
	"github.com/ugorji/go/codec"
	"log"
)

// LocalRepo is a collection of packages created locally
type LocalRepo struct {
	// Permanent internal ID
	UUID string
	// User-assigned name
	Name string
	// Comment
	Comment string
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
	db   database.Storage
	list []*LocalRepo
}

// NewLocalRepoCollection loads LocalRepos from DB and makes up collection
func NewLocalRepoCollection(db database.Storage) *LocalRepoCollection {
	result := &LocalRepoCollection{
		db: db,
	}

	blobs := db.FetchByPrefix([]byte("L"))
	result.list = make([]*LocalRepo, 0, len(blobs))

	for _, blob := range blobs {
		r := &LocalRepo{}
		if err := r.Decode(blob); err != nil {
			log.Printf("Error decoding mirror: %s\n", err)
		} else {
			result.list = append(result.list, r)
		}
	}

	return result
}

// Add appends new repo to collection and saves it
func (collection *LocalRepoCollection) Add(repo *LocalRepo) error {
	for _, r := range collection.list {
		if r.Name == repo.Name {
			return fmt.Errorf("local repo with name %s already exists", repo.Name)
		}
	}

	err := collection.Update(repo)
	if err != nil {
		return err
	}

	collection.list = append(collection.list, repo)
	return nil
}

// Update stores updated information about repo in DB
func (collection *LocalRepoCollection) Update(repo *LocalRepo) error {
	err := collection.db.Put(repo.Key(), repo.Encode())
	if err != nil {
		return err
	}
	if repo.packageRefs != nil {
		err = collection.db.Put(repo.RefKey(), repo.packageRefs.Encode())
		if err != nil {
			return err
		}
	}
	return nil
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
	for _, r := range collection.list {
		if r.Name == name {
			return r, nil
		}
	}
	return nil, fmt.Errorf("local repo with name %s not found", name)
}

// ByUUID looks up repository by uuid
func (collection *LocalRepoCollection) ByUUID(uuid string) (*LocalRepo, error) {
	for _, r := range collection.list {
		if r.UUID == uuid {
			return r, nil
		}
	}
	return nil, fmt.Errorf("local repo with uuid %s not found", uuid)
}

// ForEach runs method for each repository
func (collection *LocalRepoCollection) ForEach(handler func(*LocalRepo) error) error {
	var err error
	for _, r := range collection.list {
		err = handler(r)
		if err != nil {
			return err
		}
	}
	return err
}

// Len returns number of remote repos
func (collection *LocalRepoCollection) Len() int {
	return len(collection.list)
}

// Drop removes remote repo from collection
func (collection *LocalRepoCollection) Drop(repo *LocalRepo) error {
	repoPosition := -1

	for i, r := range collection.list {
		if r == repo {
			repoPosition = i
			break
		}
	}

	if repoPosition == -1 {
		panic("local repo not found!")
	}

	collection.list[len(collection.list)-1], collection.list[repoPosition], collection.list =
		nil, collection.list[len(collection.list)-1], collection.list[:len(collection.list)-1]

	err := collection.db.Delete(repo.Key())
	if err != nil {
		return err
	}

	return collection.db.Delete(repo.RefKey())
}
