// Package debian implements Debian-specific repository handling
package debian

import (
	"bytes"
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/utils"
	debc "github.com/smira/godebiancontrol"
	"github.com/ugorji/go/codec"
	"log"
	"net/url"
	"strings"
	"time"
)

// RemoteRepo represents remote (fetchable) Debian repository.
//
// Repostitory could be filtered when fetching by components, architectures
// TODO: support flat format
type RemoteRepo struct {
	// Permanent internal ID
	UUID string
	// User-assigned name
	Name string
	// Root of Debian archive, URL
	ArchiveRoot string
	// Distribution name, e.g. squeeze
	Distribution string
	// List of components to fetch, if empty, then fetch all components
	Components []string
	// List of architectures to fetch, if empty, then fetch all architectures
	Architectures []string
	// Meta-information about repository
	Meta debc.Paragraph
	// Last update date
	LastDownloadDate time.Time
	// "Snapshot" of current list of packages
	packageRefs *PackageRefList
	// Parsed archived root
	archiveRootURL *url.URL
}

// NewRemoteRepo creates new instance of Debian remote repository with specified params
func NewRemoteRepo(name string, archiveRoot string, distribution string, components []string, architectures []string) (*RemoteRepo, error) {
	result := &RemoteRepo{
		UUID:          uuid.New(),
		Name:          name,
		ArchiveRoot:   archiveRoot,
		Distribution:  distribution,
		Components:    components,
		Architectures: architectures,
	}

	err := result.prepare()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (repo *RemoteRepo) prepare() error {
	var err error
	repo.archiveRootURL, err = url.Parse(repo.ArchiveRoot)
	return err
}

// String interface
func (repo *RemoteRepo) String() string {
	return fmt.Sprintf("[%s]: %s %s", repo.Name, repo.ArchiveRoot, repo.Distribution)
}

// NumPackages return number of packages retrived from remore repo
func (repo *RemoteRepo) NumPackages() int {
	if repo.packageRefs == nil {
		return 0
	}
	return repo.packageRefs.Len()
}

// ReleaseURL returns URL to Release file in repo root
// TODO: InRelease, Release.gz, Release.bz2 handling
func (repo *RemoteRepo) ReleaseURL() *url.URL {
	path := &url.URL{Path: fmt.Sprintf("dists/%s/Release", repo.Distribution)}
	return repo.archiveRootURL.ResolveReference(path)
}

// BinaryURL returns URL of Packages file for given component and
// architecture
func (repo *RemoteRepo) BinaryURL(component string, architecture string) *url.URL {
	path := &url.URL{Path: fmt.Sprintf("dists/%s/%s/binary-%s/Packages", repo.Distribution, component, architecture)}
	return repo.archiveRootURL.ResolveReference(path)
}

// PackageURL returns URL of package file relative to repository root
// architecture
func (repo *RemoteRepo) PackageURL(filename string) *url.URL {
	path := &url.URL{Path: filename}
	return repo.archiveRootURL.ResolveReference(path)
}

// Fetch updates information about repository
func (repo *RemoteRepo) Fetch(d utils.Downloader) error {
	// Download release file to temporary URL
	release, err := utils.DownloadTemp(d, repo.ReleaseURL().String())
	if err != nil {
		return err
	}
	defer release.Close()

	paras, err := debc.Parse(release)
	if err != nil {
		return err
	}

	if len(paras) != 1 {
		return fmt.Errorf("wrong number of parts in Release file")
	}

	para := paras[0]

	architectures := strings.Split(para["Architectures"], " ")
	if len(repo.Architectures) == 0 {
		repo.Architectures = architectures
	} else {
		err = utils.StringsIsSubset(repo.Architectures, architectures,
			fmt.Sprintf("architecture %%s not available in repo %s", repo))
		if err != nil {
			return err
		}
	}

	components := strings.Split(para["Components"], " ")
	if len(repo.Components) == 0 {
		repo.Components = components
	} else {
		err = utils.StringsIsSubset(repo.Components, components,
			fmt.Sprintf("component %%s not available in repo %s", repo))
		if err != nil {
			return err
		}
	}

	delete(para, "MD5Sum")
	delete(para, "SHA1")
	delete(para, "SHA256")
	repo.Meta = para

	return nil
}

// Download downloads all repo files
func (repo *RemoteRepo) Download(d utils.Downloader, packageCollection *PackageCollection, packageRepo *Repository) error {
	list := NewPackageList()

	// Download and parse all Release files
	for _, component := range repo.Components {
		for _, architecture := range repo.Architectures {
			packagesReader, packagesFile, err := utils.DownloadTryCompression(d, repo.BinaryURL(component, architecture).String())
			if err != nil {
				return err
			}
			defer packagesFile.Close()

			paras, err := debc.Parse(packagesReader)
			if err != nil {
				return err
			}

			for _, para := range paras {
				p := NewPackageFromControlFile(para)

				list.Add(p)
			}
		}
	}

	// Save package meta information to DB
	list.ForEach(func(p *Package) {
		packageCollection.Update(p)
	})

	// Download all package files
	ch := make(chan error, list.Len())
	count := 0

	list.ForEach(func(p *Package) {
		poolPath, err := packageRepo.PoolPath(p.Filename)
		if err == nil {
			if !p.VerifyFile(poolPath) {
				d.Download(repo.PackageURL(p.Filename).String(), poolPath, ch)
				count++
			}
		}
	})

	// Wait for all downloads to finish
	// TODO: report errors
	for count > 0 {
		_ = <-ch
		count--
	}

	repo.LastDownloadDate = time.Now()
	repo.packageRefs = NewPackageRefListFromPackageList(list)

	return nil
}

// Encode does msgpack encoding of RemoteRepo
func (repo *RemoteRepo) Encode() []byte {
	var buf bytes.Buffer

	encoder := codec.NewEncoder(&buf, &codec.MsgpackHandle{})
	encoder.Encode(repo)

	return buf.Bytes()
}

// Decode decodes msgpack representation into RemoteRepo
func (repo *RemoteRepo) Decode(input []byte) error {
	decoder := codec.NewDecoderBytes(input, &codec.MsgpackHandle{})
	err := decoder.Decode(repo)
	if err != nil {
		return err
	}
	return repo.prepare()
}

// Key is a unique id in DB
func (repo *RemoteRepo) Key() []byte {
	return []byte("R" + repo.UUID)
}

// RefKey is a unique id for package reference list
func (repo *RemoteRepo) RefKey() []byte {
	return []byte("E" + repo.UUID)
}

// RemoteRepoCollection does listing, updating/adding/deleting of RemoteRepos
type RemoteRepoCollection struct {
	db   database.Storage
	list []*RemoteRepo
}

// NewRemoteRepoCollection loads RemoteRepos from DB and makes up collection
func NewRemoteRepoCollection(db database.Storage) *RemoteRepoCollection {
	result := &RemoteRepoCollection{
		db: db,
	}

	blobs := db.FetchByPrefix([]byte("R"))
	result.list = make([]*RemoteRepo, 0, len(blobs))

	for _, blob := range blobs {
		r := &RemoteRepo{}
		if err := r.Decode(blob); err != nil {
			log.Printf("Error decoding mirror: %s\n", err)
		} else {
			result.list = append(result.list, r)
		}
	}

	return result
}

// Add appends new repo to collection and saves it
func (collection *RemoteRepoCollection) Add(repo *RemoteRepo) error {
	for _, r := range collection.list {
		if r.Name == repo.Name {
			return fmt.Errorf("mirror with name %s already exists", repo.Name)
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
func (collection *RemoteRepoCollection) Update(repo *RemoteRepo) error {
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

// LoadComplete loads additional information for remote repo
func (collection *RemoteRepoCollection) LoadComplete(repo *RemoteRepo) error {
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
func (collection *RemoteRepoCollection) ByName(name string) (*RemoteRepo, error) {
	for _, r := range collection.list {
		if r.Name == name {
			return r, nil
		}
	}
	return nil, fmt.Errorf("mirror with name %s not found", name)
}

// ForEach runs method for each repository
func (collection *RemoteRepoCollection) ForEach(handler func(*RemoteRepo)) {
	for _, r := range collection.list {
		handler(r)
	}
}
