// Package debian implements Debian-specific repository handling
package debian

import (
	"bytes"
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/utils"
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
	Meta Stanza
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

// RefList returns package list for repo
func (repo *RemoteRepo) RefList() *PackageRefList {
	return repo.packageRefs
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

	sreader := NewControlFileReader(release)
	stanza, err := sreader.ReadStanza()
	if err != nil {
		return err
	}

	architectures := strings.Split(stanza["Architectures"], " ")
	if len(repo.Architectures) == 0 {
		repo.Architectures = architectures
	} else {
		err = utils.StringsIsSubset(repo.Architectures, architectures,
			fmt.Sprintf("architecture %%s not available in repo %s", repo))
		if err != nil {
			return err
		}
	}

	components := strings.Split(stanza["Components"], " ")
	if len(repo.Components) == 0 {
		repo.Components = components
	} else {
		err = utils.StringsIsSubset(repo.Components, components,
			fmt.Sprintf("component %%s not available in repo %s", repo))
		if err != nil {
			return err
		}
	}

	delete(stanza, "MD5Sum")
	delete(stanza, "SHA1")
	delete(stanza, "SHA256")
	repo.Meta = stanza

	return nil
}

// Download downloads all repo files
func (repo *RemoteRepo) Download(d utils.Downloader, packageCollection *PackageCollection, packageRepo *Repository) error {
	list := NewPackageList()

	fmt.Printf("Downloading & parsing package files...\n")

	// Download and parse all Release files
	for _, component := range repo.Components {
		for _, architecture := range repo.Architectures {
			packagesReader, packagesFile, err := utils.DownloadTryCompression(d, repo.BinaryURL(component, architecture).String())
			if err != nil {
				return err
			}
			defer packagesFile.Close()

			sreader := NewControlFileReader(packagesReader)

			for {
				stanza, err := sreader.ReadStanza()
				if err != nil {
					return err
				}
				if stanza == nil {
					break
				}

				p := NewPackageFromControlFile(stanza)

				list.Add(p)
			}
		}
	}

	fmt.Printf("Saving packages to database...\n")

	// Save package meta information to DB
	err := list.ForEach(func(p *Package) error {
		return packageCollection.Update(p)
	})

	if err != nil {
		return fmt.Errorf("unable to save packages to db: %s", err)
	}

	fmt.Printf("Building download queue...\n")

	// Build download queue
	queued := make(map[string]PackageDownloadTask, list.Len())
	count := 0
	downloadSize := int64(0)

	err = list.ForEach(func(p *Package) error {
		list, err := p.DownloadList(packageRepo)
		if err != nil {
			return err
		}

		for _, task := range list {
			key := task.RepoURI + "-" + task.DestinationPath
			_, found := queued[key]
			if !found {
				count++
				downloadSize += task.Size
				queued[key] = task
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("unable to build download queue: %s", err)
	}

	fmt.Printf("Download queue: %d items, %.2f GiB size\n", count, float64(downloadSize)/(1024.0*1024.0*1024.0))

	// Download all package files
	ch := make(chan error, len(queued))

	for _, task := range queued {
		d.Download(repo.PackageURL(task.RepoURI).String(), task.DestinationPath, ch)
	}

	// Wait for all downloads to finish
	errors := make([]string, 0)

	for count > 0 {
		err = <-ch
		if err != nil {
			errors = append(errors, err.Error())
		}
		count--
	}

	if len(errors) > 0 {
		return fmt.Errorf("download errors:\n  %s\n", strings.Join(errors, "\n  "))
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

// ByUUID looks up repository by uuid
func (collection *RemoteRepoCollection) ByUUID(uuid string) (*RemoteRepo, error) {
	for _, r := range collection.list {
		if r.UUID == uuid {
			return r, nil
		}
	}
	return nil, fmt.Errorf("mirror with uuid %s not found", uuid)
}

// ForEach runs method for each repository
func (collection *RemoteRepoCollection) ForEach(handler func(*RemoteRepo) error) error {
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
func (collection *RemoteRepoCollection) Len() int {
	return len(collection.list)
}

// Drop removes remote repo from collection
func (collection *RemoteRepoCollection) Drop(repo *RemoteRepo) error {
	repoPosition := -1

	for i, r := range collection.list {
		if r == repo {
			repoPosition = i
			break
		}
	}

	if repoPosition == -1 {
		panic("repo not found!")
	}

	collection.list[len(collection.list)-1], collection.list[repoPosition], collection.list =
		nil, collection.list[len(collection.list)-1], collection.list[:len(collection.list)-1]

	err := collection.db.Delete(repo.Key())
	if err != nil {
		return err
	}

	return collection.db.Delete(repo.RefKey())
}
