package debian

import (
	"bufio"
	"bytes"
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/utils"
	"github.com/ugorji/go/codec"
	"log"
	"path/filepath"
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
func NewPublishedRepo(prefix string, distribution string, component string, architectures []string, snapshot *Snapshot) *PublishedRepo {
	return &PublishedRepo{
		UUID:          uuid.New(),
		Prefix:        prefix,
		Distribution:  distribution,
		Component:     component,
		Architectures: architectures,
		SnapshotUUID:  snapshot.UUID,
		snapshot:      snapshot,
	}
}

// String returns human-readable represenation of PublishedRepo
func (p *PublishedRepo) String() string {
	var prefix, archs string

	if p.Prefix != "" {
		prefix = p.Prefix
	} else {
		prefix = "."
	}

	if len(p.Architectures) > 0 {
		archs = fmt.Sprintf(" [%s]", strings.Join(p.Architectures, ", "))
	}

	return fmt.Sprintf("%s/%s (%s)%s publishes %s", prefix, p.Distribution, p.Component, archs, p.snapshot.String())
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
func (p *PublishedRepo) Publish(repo *Repository, packageCollection *PackageCollection, signer utils.Signer) error {
	err := repo.MkDir(filepath.Join(p.Prefix, "pool"))
	if err != nil {
		return err
	}
	basePath := filepath.Join(p.Prefix, "dists", p.Distribution)
	err = repo.MkDir(basePath)
	if err != nil {
		return err
	}

	// Load all packages
	list, err := NewPackageListFromRefList(p.snapshot.RefList(), packageCollection)
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	if list.Len() == 0 {
		return fmt.Errorf("repository is empty, can't publish")
	}

	if len(p.Architectures) == 0 {
		p.Architectures = list.Architectures()
	}

	if len(p.Architectures) == 0 {
		return fmt.Errorf("unable to figure out list of architectures, please supply explicit list")
	}

	generatedFiles := map[string]*utils.ChecksumInfo{}

	// For all architectures, generate release file
	for _, arch := range p.Architectures {
		relativePath := filepath.Join(p.Component, fmt.Sprintf("binary-%s", arch), "Packages")
		err = repo.MkDir(filepath.Dir(filepath.Join(basePath, relativePath)))
		if err != nil {
			return err
		}

		packagesFile, err := repo.CreateFile(filepath.Join(basePath, relativePath))
		if err != nil {
			return fmt.Errorf("unable to creates Packages file: %s", err)
		}

		bufWriter := bufio.NewWriter(packagesFile)

		err = list.ForEach(func(pkg *Package) error {
			if pkg.MatchesArchitecture(arch) {
				err = pkg.LinkFromPool(repo, p.Prefix, p.Component)
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

			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("unable to creates process packages: %s", err)
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

		checksumInfo, err := repo.ChecksumsForFile(filepath.Join(basePath, relativePath))
		if err != nil {
			return fmt.Errorf("unable to collect checksums: %s", err)
		}
		generatedFiles[relativePath] = checksumInfo

		checksumInfo, err = repo.ChecksumsForFile(filepath.Join(basePath, relativePath+".gz"))
		if err != nil {
			return fmt.Errorf("unable to collect checksums: %s", err)
		}
		generatedFiles[relativePath+".gz"] = checksumInfo

		checksumInfo, err = repo.ChecksumsForFile(filepath.Join(basePath, relativePath+".bz2"))
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
	release["Architectures"] = strings.Join(p.Architectures, " ")
	release["Description"] = "Generated by aptly\n"
	release["MD5Sum"] = "\n"
	release["SHA1"] = "\n"
	release["SHA256"] = "\n"

	for path, info := range generatedFiles {
		release["MD5Sum"] += fmt.Sprintf(" %s %8d %s\n", info.MD5, info.Size, path)
		release["SHA1"] += fmt.Sprintf(" %s %8d %s\n", info.SHA1, info.Size, path)
		release["SHA256"] += fmt.Sprintf(" %s %8d %s\n", info.SHA256, info.Size, path)
	}

	releaseFile, err := repo.CreateFile(filepath.Join(basePath, "Release"))
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

	err = signer.DetachedSign(releaseFilename, releaseFilename+".gpg")
	if err != nil {
		return fmt.Errorf("unable to sign Release file: %s", err)
	}

	err = signer.ClearSign(releaseFilename, filepath.Join(filepath.Dir(releaseFilename), "InRelease"))
	if err != nil {
		return fmt.Errorf("unable to sign Release file: %s", err)
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
