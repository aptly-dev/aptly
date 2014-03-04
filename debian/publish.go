package debian

import (
	"bufio"
	"bytes"
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/utils"
	"github.com/ugorji/go/codec"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// PublishedRepo is a published for http/ftp representation of snapshot as Debian repository
type PublishedRepo struct {
	// Internal unique ID
	UUID string
	// Prefix & distribution should be unique across all published repositories
	Prefix       string
	Distribution string
	Component    string
	// Architectures is a list of all architectures published
	Architectures []string
	// Snapshot as a source of publishing
	SnapshotUUID string

	snapshot *Snapshot
}

// NewPublishedRepo creates new published repository
func NewPublishedRepo(prefix string, distribution string, component string, architectures []string, snapshot *Snapshot) (*PublishedRepo, error) {
	prefix = filepath.Clean(prefix)
	if strings.HasPrefix(prefix, "/") {
		prefix = prefix[1:]
	}
	if strings.HasSuffix(prefix, "/") {
		prefix = prefix[:len(prefix)-1]
	}
	prefix = filepath.Clean(prefix)

	for _, component := range strings.Split(prefix, "/") {
		if component == ".." || component == "dists" || component == "pool" {
			return nil, fmt.Errorf("invalid prefix %s", prefix)
		}
	}

	return &PublishedRepo{
		UUID:          uuid.New(),
		Prefix:        prefix,
		Distribution:  distribution,
		Component:     component,
		Architectures: architectures,
		SnapshotUUID:  snapshot.UUID,
		snapshot:      snapshot,
	}, nil
}

// String returns human-readable represenation of PublishedRepo
func (p *PublishedRepo) String() string {
	var archs string

	if len(p.Architectures) > 0 {
		archs = fmt.Sprintf(" [%s]", strings.Join(p.Architectures, ", "))
	}

	return fmt.Sprintf("%s/%s (%s)%s publishes %s", p.Prefix, p.Distribution, p.Component, archs, p.snapshot.String())
}

// Key returns unique key identifying PublishedRepo
func (p *PublishedRepo) Key() []byte {
	return []byte("U" + p.Prefix + ">>" + p.Distribution)
}

// Encode does msgpack encoding of PublishedRepo
func (p *PublishedRepo) Encode() []byte {
	var buf bytes.Buffer

	encoder := codec.NewEncoder(&buf, &codec.MsgpackHandle{})
	encoder.Encode(p)

	return buf.Bytes()
}

// Decode decodes msgpack representation into PublishedRepo
func (p *PublishedRepo) Decode(input []byte) error {
	decoder := codec.NewDecoderBytes(input, &codec.MsgpackHandle{})
	return decoder.Decode(p)
}

// Publish publishes snapshot (repository) contents, links package files, generates Packages & Release files, signs them
func (p *PublishedRepo) Publish(packagePool aptly.PackagePool, publishedStorage aptly.PublishedStorage, packageCollection *PackageCollection, signer utils.Signer) error {
	err := publishedStorage.MkDir(filepath.Join(p.Prefix, "pool"))
	if err != nil {
		return err
	}
	basePath := filepath.Join(p.Prefix, "dists", p.Distribution)
	err = publishedStorage.MkDir(basePath)
	if err != nil {
		return err
	}

	// Load all packages
	list, err := NewPackageListFromRefList(p.snapshot.RefList(), packageCollection)
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	if list.Len() == 0 {
		return fmt.Errorf("snapshot is empty")
	}

	if len(p.Architectures) == 0 {
		p.Architectures = list.Architectures(true)
	}

	if len(p.Architectures) == 0 {
		return fmt.Errorf("unable to figure out list of architectures, please supply explicit list")
	}

	sort.Strings(p.Architectures)

	generatedFiles := map[string]utils.ChecksumInfo{}

	// For all architectures, generate release file
	for _, arch := range p.Architectures {
		var relativePath string
		if arch == "source" {
			relativePath = filepath.Join(p.Component, "source", "Sources")
		} else {
			relativePath = filepath.Join(p.Component, fmt.Sprintf("binary-%s", arch), "Packages")
		}
		err = publishedStorage.MkDir(filepath.Dir(filepath.Join(basePath, relativePath)))
		if err != nil {
			return err
		}

		packagesFile, err := publishedStorage.CreateFile(filepath.Join(basePath, relativePath))
		if err != nil {
			return fmt.Errorf("unable to creates Packages file: %s", err)
		}

		bufWriter := bufio.NewWriter(packagesFile)

		err = list.ForEach(func(pkg *Package) error {
			if pkg.MatchesArchitecture(arch) {
				err = pkg.LinkFromPool(publishedStorage, packagePool, p.Prefix, p.Component)
				if err != nil {
					return err
				}

				err = pkg.Stanza().WriteTo(bufWriter)
				if err != nil {
					return err
				}
				err = bufWriter.WriteByte('\n')
				if err != nil {
					return err
				}

				pkg.files = nil
				pkg.deps = nil
				pkg.extra = nil

			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("unable to process packages: %s", err)
		}

		err = bufWriter.Flush()
		if err != nil {
			return fmt.Errorf("unable to write Packages file: %s", err)
		}

		err = utils.CompressFile(packagesFile)
		if err != nil {
			return fmt.Errorf("unable to compress Packages files: %s", err)
		}

		packagesFile.Close()

		checksumInfo, err := publishedStorage.ChecksumsForFile(filepath.Join(basePath, relativePath))
		if err != nil {
			return fmt.Errorf("unable to collect checksums: %s", err)
		}
		generatedFiles[relativePath] = checksumInfo

		checksumInfo, err = publishedStorage.ChecksumsForFile(filepath.Join(basePath, relativePath+".gz"))
		if err != nil {
			return fmt.Errorf("unable to collect checksums: %s", err)
		}
		generatedFiles[relativePath+".gz"] = checksumInfo

		checksumInfo, err = publishedStorage.ChecksumsForFile(filepath.Join(basePath, relativePath+".bz2"))
		if err != nil {
			return fmt.Errorf("unable to collect checksums: %s", err)
		}
		generatedFiles[relativePath+".bz2"] = checksumInfo

	}

	release := make(Stanza)
	release["Origin"] = p.Prefix + " " + p.Distribution
	release["Label"] = p.Prefix + " " + p.Distribution
	release["Codename"] = p.Distribution
	release["Date"] = time.Now().UTC().Format("Mon, 2 Jan 2006 15:04:05 MST")
	release["Components"] = p.Component
	release["Architectures"] = strings.Join(utils.StrSlicesSubstract(p.Architectures, []string{"source"}), " ")
	release["Description"] = " Generated by aptly\n"
	release["MD5Sum"] = "\n"
	release["SHA1"] = "\n"
	release["SHA256"] = "\n"

	for path, info := range generatedFiles {
		release["MD5Sum"] += fmt.Sprintf(" %s %8d %s\n", info.MD5, info.Size, path)
		release["SHA1"] += fmt.Sprintf(" %s %8d %s\n", info.SHA1, info.Size, path)
		release["SHA256"] += fmt.Sprintf(" %s %8d %s\n", info.SHA256, info.Size, path)
	}

	releaseFile, err := publishedStorage.CreateFile(filepath.Join(basePath, "Release"))
	if err != nil {
		return fmt.Errorf("unable to create Release file: %s", err)
	}

	bufWriter := bufio.NewWriter(releaseFile)

	err = release.WriteTo(bufWriter)
	if err != nil {
		return fmt.Errorf("unable to create Release file: %s", err)
	}

	err = bufWriter.Flush()
	if err != nil {
		return fmt.Errorf("unable to create Release file: %s", err)
	}

	releaseFilename := releaseFile.Name()
	releaseFile.Close()

	if signer != nil {
		err = signer.DetachedSign(releaseFilename, releaseFilename+".gpg")
		if err != nil {
			return fmt.Errorf("unable to sign Release file: %s", err)
		}

		err = signer.ClearSign(releaseFilename, filepath.Join(filepath.Dir(releaseFilename), "InRelease"))
		if err != nil {
			return fmt.Errorf("unable to sign Release file: %s", err)
		}
	}

	return nil
}

// RemoveFiles removes files that were created by Publish
//
// It can remove prefix fully, and part of pool (for specific component)
func (p *PublishedRepo) RemoveFiles(publishedStorage aptly.PublishedStorage, removePrefix, removePoolComponent bool) error {
	if removePrefix {
		err := publishedStorage.RemoveDirs(filepath.Join(p.Prefix, "dists"))
		if err != nil {
			return err
		}

		return publishedStorage.RemoveDirs(filepath.Join(p.Prefix, "pool"))
	}

	err := publishedStorage.RemoveDirs(filepath.Join(p.Prefix, "dists", p.Distribution))
	if err != nil {
		return err
	}

	if removePoolComponent {
		err = publishedStorage.RemoveDirs(filepath.Join(p.Prefix, "pool", p.Component))
		if err != nil {
			return err
		}
	}
	return nil
}

// PublishedRepoCollection does listing, updating/adding/deleting of PublishedRepos
type PublishedRepoCollection struct {
	db   database.Storage
	list []*PublishedRepo
}

// NewPublishedRepoCollection loads PublishedRepos from DB and makes up collection
func NewPublishedRepoCollection(db database.Storage) *PublishedRepoCollection {
	result := &PublishedRepoCollection{
		db: db,
	}

	blobs := db.FetchByPrefix([]byte("U"))
	result.list = make([]*PublishedRepo, 0, len(blobs))

	for _, blob := range blobs {
		r := &PublishedRepo{}
		if err := r.Decode(blob); err != nil {
			log.Printf("Error decoding published repo: %s\n", err)
		} else {
			result.list = append(result.list, r)
		}
	}

	return result
}

// Add appends new repo to collection and saves it
func (collection *PublishedRepoCollection) Add(repo *PublishedRepo) error {
	if collection.CheckDuplicate(repo) != nil {
		return fmt.Errorf("published repo with prefix/distribution %s/%s already exists", repo.Prefix, repo.Distribution)
	}

	err := collection.Update(repo)
	if err != nil {
		return err
	}

	collection.list = append(collection.list, repo)
	return nil
}

// CheckDuplicate verifies that there's no published repo with the same name
func (collection *PublishedRepoCollection) CheckDuplicate(repo *PublishedRepo) *PublishedRepo {
	for _, r := range collection.list {
		if r.Prefix == repo.Prefix && r.Distribution == repo.Distribution {
			return r
		}
	}

	return nil
}

// Update stores updated information about repo in DB
func (collection *PublishedRepoCollection) Update(repo *PublishedRepo) error {
	err := collection.db.Put(repo.Key(), repo.Encode())
	if err != nil {
		return err
	}
	return nil
}

// LoadComplete loads additional information for remote repo
func (collection *PublishedRepoCollection) LoadComplete(repo *PublishedRepo, snapshotCollection *SnapshotCollection) error {
	snapshot, err := snapshotCollection.ByUUID(repo.SnapshotUUID)
	if err != nil {
		return err
	}

	repo.snapshot = snapshot
	return nil
}

// ByPrefixDistribution looks up repository by prefix & distribution
func (collection *PublishedRepoCollection) ByPrefixDistribution(prefix, distribution string) (*PublishedRepo, error) {
	for _, r := range collection.list {
		if r.Prefix == prefix && r.Distribution == distribution {
			return r, nil
		}
	}
	return nil, fmt.Errorf("published repo with prefix/distribution %s/%s not found", prefix, distribution)
}

// ByUUID looks up repository by uuid
func (collection *PublishedRepoCollection) ByUUID(uuid string) (*PublishedRepo, error) {
	for _, r := range collection.list {
		if r.UUID == uuid {
			return r, nil
		}
	}
	return nil, fmt.Errorf("published repo with uuid %s not found", uuid)
}

// BySnapshot looks up repository by snapshot source
func (collection *PublishedRepoCollection) BySnapshot(snapshot *Snapshot) []*PublishedRepo {
	result := make([]*PublishedRepo, 0)
	for _, r := range collection.list {
		if r.SnapshotUUID == snapshot.UUID {
			result = append(result, r)
		}
	}
	return result
}

// ForEach runs method for each repository
func (collection *PublishedRepoCollection) ForEach(handler func(*PublishedRepo) error) error {
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
func (collection *PublishedRepoCollection) Len() int {
	return len(collection.list)
}

// Remove removes published repository, cleaning up directories, files
func (collection *PublishedRepoCollection) Remove(publishedStorage aptly.PublishedStorage, prefix, distribution string) error {
	repo, err := collection.ByPrefixDistribution(prefix, distribution)
	if err != nil {
		return err
	}

	removePrefix := true
	removePoolComponent := true
	repoPosition := -1

	for i, r := range collection.list {
		if r == repo {
			repoPosition = i
			continue
		}
		if r.Prefix == repo.Prefix {
			removePrefix = false
			if r.Component == repo.Component {
				removePoolComponent = false
			}
		}
	}

	err = repo.RemoveFiles(publishedStorage, removePrefix, removePoolComponent)
	if err != nil {
		return err
	}

	collection.list[len(collection.list)-1], collection.list[repoPosition], collection.list =
		nil, collection.list[len(collection.list)-1], collection.list[:len(collection.list)-1]

	return collection.db.Delete(repo.Key())
}
