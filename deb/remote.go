package deb

import (
	"bytes"
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/http"
	"github.com/smira/aptly/utils"
	"github.com/ugorji/go/codec"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// RemoteRepo represents remote (fetchable) Debian repository.
//
// Repostitory could be filtered when fetching by components, architectures
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
	// Should we download sources?
	DownloadSources bool
	// Meta-information about repository
	Meta Stanza
	// Last update date
	LastDownloadDate time.Time
	// Checksums for release files
	ReleaseFiles map[string]utils.ChecksumInfo
	// Filter for packages
	Filter string
	// FilterWithDeps to include dependencies from filter query
	FilterWithDeps bool
	// "Snapshot" of current list of packages
	packageRefs *PackageRefList
	// Parsed archived root
	archiveRootURL *url.URL
}

// NewRemoteRepo creates new instance of Debian remote repository with specified params
func NewRemoteRepo(name string, archiveRoot string, distribution string, components []string,
	architectures []string, downloadSources bool) (*RemoteRepo, error) {
	result := &RemoteRepo{
		UUID:            uuid.New(),
		Name:            name,
		ArchiveRoot:     archiveRoot,
		Distribution:    distribution,
		Components:      components,
		Architectures:   architectures,
		DownloadSources: downloadSources,
	}

	err := result.prepare()
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(result.Distribution, "/") || strings.HasPrefix(result.Distribution, ".") {
		// flat repo
		if !strings.HasPrefix(result.Distribution, ".") {
			result.Distribution = "./" + result.Distribution
		}
		result.Architectures = nil
		if len(result.Components) > 0 {
			return nil, fmt.Errorf("components aren't supported for flat repos")
		}
		result.Components = nil
	}

	return result, nil
}

func (repo *RemoteRepo) prepare() error {
	var err error

	// Add final / to URL
	if !strings.HasSuffix(repo.ArchiveRoot, "/") {
		repo.ArchiveRoot = repo.ArchiveRoot + "/"
	}

	repo.archiveRootURL, err = url.Parse(repo.ArchiveRoot)
	return err
}

// String interface
func (repo *RemoteRepo) String() string {
	srcFlag := ""
	if repo.DownloadSources {
		srcFlag = " [src]"
	}
	distribution := repo.Distribution
	if distribution == "" {
		distribution = "./"
	}
	return fmt.Sprintf("[%s]: %s %s%s", repo.Name, repo.ArchiveRoot, distribution, srcFlag)
}

// IsFlat determines if repository is flat
func (repo *RemoteRepo) IsFlat() bool {
	// aptly < 0.5.1 had Distribution = "" for flat repos
	// aptly >= 0.5.1 had Distribution = "./[path]/" for flat repos
	return repo.Distribution == "" || (strings.HasPrefix(repo.Distribution, ".") && strings.HasSuffix(repo.Distribution, "/"))
}

// NumPackages return number of packages retrived from remote repo
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

// ReleaseURL returns URL to Release* files in repo root
func (repo *RemoteRepo) ReleaseURL(name string) *url.URL {
	var path *url.URL

	if !repo.IsFlat() {
		path = &url.URL{Path: fmt.Sprintf("dists/%s/%s", repo.Distribution, name)}
	} else {
		path = &url.URL{Path: filepath.Join(repo.Distribution, name)}
	}

	return repo.archiveRootURL.ResolveReference(path)
}

// FlatBinaryURL returns URL to Packages files for flat repo
func (repo *RemoteRepo) FlatBinaryURL() *url.URL {
	path := &url.URL{Path: filepath.Join(repo.Distribution, "Packages")}
	return repo.archiveRootURL.ResolveReference(path)
}

// FlatSourcesURL returns URL to Sources files for flat repo
func (repo *RemoteRepo) FlatSourcesURL() *url.URL {
	path := &url.URL{Path: filepath.Join(repo.Distribution, "Sources")}
	return repo.archiveRootURL.ResolveReference(path)
}

// BinaryURL returns URL of Packages files for given component and
// architecture
func (repo *RemoteRepo) BinaryURL(component string, architecture string) *url.URL {
	path := &url.URL{Path: fmt.Sprintf("dists/%s/%s/binary-%s/Packages", repo.Distribution, component, architecture)}
	return repo.archiveRootURL.ResolveReference(path)
}

// SourcesURL returns URL of Sources files for given component
func (repo *RemoteRepo) SourcesURL(component string) *url.URL {
	path := &url.URL{Path: fmt.Sprintf("dists/%s/%s/source/Sources", repo.Distribution, component)}
	return repo.archiveRootURL.ResolveReference(path)
}

// PackageURL returns URL of package file relative to repository root
// architecture
func (repo *RemoteRepo) PackageURL(filename string) *url.URL {
	path := &url.URL{Path: filename}
	return repo.archiveRootURL.ResolveReference(path)
}

// Fetch updates information about repository
func (repo *RemoteRepo) Fetch(d aptly.Downloader, verifier utils.Verifier) error {
	var (
		release, inrelease, releasesig *os.File
		err                            error
	)

	if verifier == nil {
		// 0. Just download release file to temporary URL
		release, err = http.DownloadTemp(d, repo.ReleaseURL("Release").String())
		if err != nil {
			return err
		}
	} else {
		// 1. try InRelease file
		inrelease, err = http.DownloadTemp(d, repo.ReleaseURL("InRelease").String())
		if err != nil {
			goto splitsignature
		}
		defer inrelease.Close()

		err = verifier.VerifyClearsigned(inrelease)
		if err != nil {
			goto splitsignature
		}

		inrelease.Seek(0, 0)

		release, err = verifier.ExtractClearsigned(inrelease)
		if err != nil {
			goto splitsignature
		}

		goto ok

	splitsignature:
		// 2. try Release + Release.gpg
		release, err = http.DownloadTemp(d, repo.ReleaseURL("Release").String())
		if err != nil {
			return err
		}

		releasesig, err = http.DownloadTemp(d, repo.ReleaseURL("Release.gpg").String())
		if err != nil {
			return err
		}

		err = verifier.VerifyDetachedSignature(releasesig, release)
		if err != nil {
			return err
		}

		_, err = release.Seek(0, 0)
		if err != nil {
			return err
		}
	}
ok:

	defer release.Close()

	sreader := NewControlFileReader(release)
	stanza, err := sreader.ReadStanza()
	if err != nil {
		return err
	}

	if !repo.IsFlat() {
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
		for i := range components {
			components[i] = path.Base(components[i])
		}
		if len(repo.Components) == 0 {
			repo.Components = components
		} else {
			err = utils.StringsIsSubset(repo.Components, components,
				fmt.Sprintf("component %%s not available in repo %s", repo))
			if err != nil {
				return err
			}
		}
	}

	repo.ReleaseFiles = make(map[string]utils.ChecksumInfo)

	parseSums := func(field string, setter func(sum *utils.ChecksumInfo, data string)) error {
		for _, line := range strings.Split(stanza[field], "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.Fields(line)

			if len(parts) != 3 {
				return fmt.Errorf("unparseable hash sum line: %#v", line)
			}

			var size int64
			size, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return fmt.Errorf("unable to parse size: %s", err)
			}

			sum := repo.ReleaseFiles[parts[2]]

			sum.Size = size
			setter(&sum, parts[0])

			repo.ReleaseFiles[parts[2]] = sum
		}

		delete(stanza, field)

		return nil
	}

	err = parseSums("MD5Sum", func(sum *utils.ChecksumInfo, data string) { sum.MD5 = data })
	if err != nil {
		return err
	}

	err = parseSums("SHA1", func(sum *utils.ChecksumInfo, data string) { sum.SHA1 = data })
	if err != nil {
		return err
	}

	err = parseSums("SHA256", func(sum *utils.ChecksumInfo, data string) { sum.SHA256 = data })
	if err != nil {
		return err
	}

	repo.Meta = stanza

	return nil
}

// Download downloads all repo files
func (repo *RemoteRepo) Download(progress aptly.Progress, d aptly.Downloader, collectionFactory *CollectionFactory,
	packagePool aptly.PackagePool, ignoreMismatch bool, dependencyOptions int, filterQuery PackageQuery) error {
	list := NewPackageList()

	progress.Printf("Downloading & parsing package files...\n")

	// Download and parse all Packages & Source files
	packagesURLs := [][]string{}

	if repo.IsFlat() {
		packagesURLs = append(packagesURLs, []string{repo.FlatBinaryURL().String(), "binary"})
		if repo.DownloadSources {
			packagesURLs = append(packagesURLs, []string{repo.FlatSourcesURL().String(), "source"})
		}
	} else {
		for _, component := range repo.Components {
			for _, architecture := range repo.Architectures {
				packagesURLs = append(packagesURLs, []string{repo.BinaryURL(component, architecture).String(), "binary"})
			}
			if repo.DownloadSources {
				packagesURLs = append(packagesURLs, []string{repo.SourcesURL(component).String(), "source"})
			}
		}
	}

	for _, info := range packagesURLs {
		url, kind := info[0], info[1]
		packagesReader, packagesFile, err := http.DownloadTryCompression(d, url, repo.ReleaseFiles, ignoreMismatch)
		if err != nil {
			return err
		}
		defer packagesFile.Close()

		stat, _ := packagesFile.Stat()
		progress.InitBar(stat.Size(), true)

		sreader := NewControlFileReader(packagesReader)

		for {
			stanza, err := sreader.ReadStanza()
			if err != nil {
				return err
			}
			if stanza == nil {
				break
			}

			off, _ := packagesFile.Seek(0, 1)
			progress.SetBar(int(off))

			var p *Package

			if kind == "binary" {
				p = NewPackageFromControlFile(stanza)
			} else if kind == "source" {
				p, err = NewSourcePackageFromControlFile(stanza)
				if err != nil {
					return err
				}
			}
			err = list.Add(p)
			if err != nil {
				return err
			}

			err = collectionFactory.PackageCollection().Update(p)
			if err != nil {
				return err
			}
		}

		progress.ShutdownBar()
	}

	var err error

	if repo.Filter != "" {
		progress.Printf("Applying filter...\n")

		list.PrepareIndex()

		emptyList := NewPackageList()
		emptyList.PrepareIndex()

		origPackages := list.Len()
		list, err = list.Filter([]PackageQuery{filterQuery}, repo.FilterWithDeps, emptyList, dependencyOptions, repo.Architectures)
		if err != nil {
			return err
		}

		progress.Printf("Packages filtered: %d -> %d.\n", origPackages, list.Len())
	}

	progress.Printf("Building download queue...\n")

	// Build download queue
	queued := make(map[string]PackageDownloadTask, list.Len())
	count := 0
	downloadSize := int64(0)

	err = list.ForEach(func(p *Package) error {
		list, err2 := p.DownloadList(packagePool)
		if err2 != nil {
			return err2
		}
		p.files = nil

		for _, task := range list {
			key := task.RepoURI + "-" + task.DestinationPath
			_, found := queued[key]
			if !found {
				count++
				downloadSize += task.Checksums.Size
				queued[key] = task
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to build download queue: %s", err)
	}

	repo.packageRefs = NewPackageRefListFromPackageList(list)
	// free up package list, we don't need it after this point
	list = nil

	progress.Printf("Download queue: %d items (%s)\n", count, utils.HumanBytes(downloadSize))

	progress.InitBar(downloadSize, true)

	// Download all package files
	ch := make(chan error, len(queued))

	for _, task := range queued {
		d.DownloadWithChecksum(repo.PackageURL(task.RepoURI).String(), task.DestinationPath, ch, task.Checksums, ignoreMismatch)
	}

	// We don't need queued after this point
	queued = nil

	// Wait for all downloads to finish
	errors := make([]string, 0)

	for count > 0 {
		err = <-ch
		if err != nil {
			errors = append(errors, err.Error())
		}
		count--
	}

	progress.ShutdownBar()

	if len(errors) > 0 {
		return fmt.Errorf("download errors:\n  %s\n", strings.Join(errors, "\n  "))
	}

	repo.LastDownloadDate = time.Now()

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
		if strings.HasPrefix(err.Error(), "codec.decoder: readContainerLen: Unrecognized descriptor byte: hex: 80") {
			// probably it is broken DB from go < 1.2, try decoding w/o time.Time
			var repo11 struct {
				UUID             string
				Name             string
				ArchiveRoot      string
				Distribution     string
				Components       []string
				Architectures    []string
				DownloadSources  bool
				Meta             Stanza
				LastDownloadDate []byte
				ReleaseFiles     map[string]utils.ChecksumInfo
				Filter           string
				FilterWithDeps   bool
			}

			decoder = codec.NewDecoderBytes(input, &codec.MsgpackHandle{})
			err2 := decoder.Decode(&repo11)
			if err2 != nil {
				return err
			}

			repo.UUID = repo11.UUID
			repo.Name = repo11.Name
			repo.ArchiveRoot = repo11.ArchiveRoot
			repo.Distribution = repo11.Distribution
			repo.Components = repo11.Components
			repo.Architectures = repo11.Architectures
			repo.DownloadSources = repo11.DownloadSources
			repo.Meta = repo11.Meta
			repo.ReleaseFiles = repo11.ReleaseFiles
			repo.Filter = repo11.Filter
			repo.FilterWithDeps = repo11.FilterWithDeps
		} else {
			return err
		}
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
