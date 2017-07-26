package deb

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/http"
	"github.com/smira/aptly/pgp"
	"github.com/smira/aptly/utils"
	"github.com/smira/go-uuid/uuid"
	"github.com/ugorji/go/codec"
)

// RemoteRepo statuses
const (
	MirrorIdle = iota
	MirrorUpdating
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
	// Meta-information about repository
	Meta Stanza
	// Last update date
	LastDownloadDate time.Time
	// Checksums for release files
	ReleaseFiles map[string]utils.ChecksumInfo
	// Filter for packages
	Filter string
	// Status marks state of repository (being updated, no action)
	Status int
	// WorkerPID is PID of the process modifying the mirror (if any)
	WorkerPID int
	// FilterWithDeps to include dependencies from filter query
	FilterWithDeps bool
	// SkipComponentCheck skips component list verification
	SkipComponentCheck bool
	// SkipArchitectureCheck skips architecture list verification
	SkipArchitectureCheck bool
	// Should we download sources?
	DownloadSources bool
	// Should we download .udebs?
	DownloadUdebs bool
	// "Snapshot" of current list of packages
	packageRefs *PackageRefList
	// Parsed archived root
	archiveRootURL *url.URL
	// Current list of packages (filled while updating mirror)
	packageList *PackageList
}

// NewRemoteRepo creates new instance of Debian remote repository with specified params
func NewRemoteRepo(name string, archiveRoot string, distribution string, components []string,
	architectures []string, downloadSources bool, downloadUdebs bool) (*RemoteRepo, error) {
	result := &RemoteRepo{
		UUID:            uuid.New(),
		Name:            name,
		ArchiveRoot:     archiveRoot,
		Distribution:    distribution,
		Components:      components,
		Architectures:   architectures,
		DownloadSources: downloadSources,
		DownloadUdebs:   downloadUdebs,
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
		if result.DownloadUdebs {
			return nil, fmt.Errorf("debian-installer udebs aren't supported for flat repos")
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
		srcFlag += " [src]"
	}
	if repo.DownloadUdebs {
		srcFlag += " [udeb]"
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

// NumPackages return number of packages retrieved from remote repo
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

// MarkAsUpdating puts current PID and sets status to updating
func (repo *RemoteRepo) MarkAsUpdating() {
	repo.Status = MirrorUpdating
	repo.WorkerPID = os.Getpid()
}

// MarkAsIdle clears updating flag
func (repo *RemoteRepo) MarkAsIdle() {
	repo.Status = MirrorIdle
	repo.WorkerPID = 0
}

// CheckLock returns error if mirror is being updated by another process
func (repo *RemoteRepo) CheckLock() error {
	if repo.Status == MirrorIdle || repo.WorkerPID == 0 {
		return nil
	}

	p, err := os.FindProcess(repo.WorkerPID)
	if err != nil {
		return nil
	}

	err = p.Signal(syscall.Signal(0))
	if err == nil {
		return fmt.Errorf("mirror is locked by update operation, PID %d", repo.WorkerPID)
	}

	return nil
}

// IndexesRootURL builds URL for various indexes
func (repo *RemoteRepo) IndexesRootURL() *url.URL {
	var path *url.URL

	if !repo.IsFlat() {
		path = &url.URL{Path: fmt.Sprintf("dists/%s/", repo.Distribution)}
	} else {
		path = &url.URL{Path: repo.Distribution}
	}

	return repo.archiveRootURL.ResolveReference(path)
}

// ReleaseURL returns URL to Release* files in repo root
func (repo *RemoteRepo) ReleaseURL(name string) *url.URL {
	return repo.IndexesRootURL().ResolveReference(&url.URL{Path: name})
}

// FlatBinaryPath returns path to Packages files for flat repo
func (repo *RemoteRepo) FlatBinaryPath() string {
	return "Packages"
}

// FlatSourcesPath returns path to Sources files for flat repo
func (repo *RemoteRepo) FlatSourcesPath() string {
	return "Sources"
}

// BinaryPath returns path to Packages files for given component and
// architecture
func (repo *RemoteRepo) BinaryPath(component string, architecture string) string {
	return fmt.Sprintf("%s/binary-%s/Packages", component, architecture)
}

// SourcesPath returns path to Sources files for given component
func (repo *RemoteRepo) SourcesPath(component string) string {
	return fmt.Sprintf("%s/source/Sources", component)
}

// UdebPath returns path of Packages files for given component and
// architecture
func (repo *RemoteRepo) UdebPath(component string, architecture string) string {
	return fmt.Sprintf("%s/debian-installer/binary-%s/Packages", component, architecture)
}

// PackageURL returns URL of package file relative to repository root
// architecture
func (repo *RemoteRepo) PackageURL(filename string) *url.URL {
	path := &url.URL{Path: filename}
	return repo.archiveRootURL.ResolveReference(path)
}

// Fetch updates information about repository
func (repo *RemoteRepo) Fetch(d aptly.Downloader, verifier pgp.Verifier) error {
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

		_, err = verifier.VerifyClearsigned(inrelease, true)
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

		err = verifier.VerifyDetachedSignature(releasesig, release, true)
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
	stanza, err := sreader.ReadStanza(true)
	if err != nil {
		return err
	}

	if !repo.IsFlat() {
		architectures := strings.Split(stanza["Architectures"], " ")
		sort.Strings(architectures)
		// "source" architecture is never present, despite Release file claims
		architectures = utils.StrSlicesSubstract(architectures, []string{ArchitectureSource})
		if len(repo.Architectures) == 0 {
			repo.Architectures = architectures
		} else if !repo.SkipArchitectureCheck {
			err = utils.StringsIsSubset(repo.Architectures, architectures,
				fmt.Sprintf("architecture %%s not available in repo %s, use -force-architectures to override", repo))
			if err != nil {
				return err
			}
		}

		components := strings.Split(stanza["Components"], " ")
		if strings.Contains(repo.Distribution, "/") {
			distributionLast := path.Base(repo.Distribution) + "/"
			for i := range components {
				components[i] = strings.TrimPrefix(components[i], distributionLast)
			}
		}
		if len(repo.Components) == 0 {
			repo.Components = components
		} else if !repo.SkipComponentCheck {
			err = utils.StringsIsSubset(repo.Components, components,
				fmt.Sprintf("component %%s not available in repo %s, use -force-components to override", repo))
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

	delete(stanza, "SHA512")

	repo.Meta = stanza

	return nil
}

// DownloadPackageIndexes downloads & parses package index files
func (repo *RemoteRepo) DownloadPackageIndexes(progress aptly.Progress, d aptly.Downloader, collectionFactory *CollectionFactory,
	ignoreMismatch bool, maxTries int) error {
	if repo.packageList != nil {
		panic("packageList != nil")
	}
	repo.packageList = NewPackageList()

	// Download and parse all Packages & Source files
	packagesPaths := [][]string{}

	if repo.IsFlat() {
		packagesPaths = append(packagesPaths, []string{repo.FlatBinaryPath(), PackageTypeBinary})
		if repo.DownloadSources {
			packagesPaths = append(packagesPaths, []string{repo.FlatSourcesPath(), PackageTypeSource})
		}
	} else {
		for _, component := range repo.Components {
			for _, architecture := range repo.Architectures {
				packagesPaths = append(packagesPaths, []string{repo.BinaryPath(component, architecture), PackageTypeBinary})
				if repo.DownloadUdebs {
					packagesPaths = append(packagesPaths, []string{repo.UdebPath(component, architecture), PackageTypeUdeb})
				}
			}
			if repo.DownloadSources {
				packagesPaths = append(packagesPaths, []string{repo.SourcesPath(component), PackageTypeSource})
			}
		}
	}

	for _, info := range packagesPaths {
		path, kind := info[0], info[1]
		packagesReader, packagesFile, err := http.DownloadTryCompression(d, repo.IndexesRootURL(), path, repo.ReleaseFiles, ignoreMismatch, maxTries)
		if err != nil {
			return err
		}
		defer packagesFile.Close()

		stat, _ := packagesFile.Stat()
		progress.InitBar(stat.Size(), true)

		sreader := NewControlFileReader(packagesReader)

		for {
			stanza, err := sreader.ReadStanza(false)
			if err != nil {
				return err
			}
			if stanza == nil {
				break
			}

			off, _ := packagesFile.Seek(0, 1)
			progress.SetBar(int(off))

			var p *Package

			if kind == PackageTypeBinary {
				p = NewPackageFromControlFile(stanza)
			} else if kind == PackageTypeUdeb {
				p = NewUdebPackageFromControlFile(stanza)
			} else if kind == PackageTypeSource {
				p, err = NewSourcePackageFromControlFile(stanza)
				if err != nil {
					return err
				}
			}
			err = repo.packageList.Add(p)
			if err != nil {
				if _, ok := err.(*PackageConflictError); ok {
					progress.ColoredPrintf("@y[!]@| @!skipping package %s: duplicate in packages index@|", p)
				} else {
					return err
				}
			}
		}

		progress.ShutdownBar()
	}

	return nil
}

// ApplyFilter applies filtering to already built PackageList
func (repo *RemoteRepo) ApplyFilter(dependencyOptions int, filterQuery PackageQuery, progress aptly.Progress) (oldLen, newLen int, err error) {
	repo.packageList.PrepareIndex()

	emptyList := NewPackageList()
	emptyList.PrepareIndex()

	oldLen = repo.packageList.Len()
	repo.packageList, err = repo.packageList.FilterWithProgress([]PackageQuery{filterQuery}, repo.FilterWithDeps, emptyList, dependencyOptions, repo.Architectures, progress)
	if repo.packageList != nil {
		newLen = repo.packageList.Len()
	}
	return
}

// BuildDownloadQueue builds queue, discards current PackageList
func (repo *RemoteRepo) BuildDownloadQueue(packagePool aptly.PackagePool, packageCollection *PackageCollection, checksumStorage aptly.ChecksumStorage, skipExistingPackages bool) (queue []PackageDownloadTask, downloadSize int64, err error) {
	queue = make([]PackageDownloadTask, 0, repo.packageList.Len())
	seen := make(map[string]int, repo.packageList.Len())

	err = repo.packageList.ForEach(func(p *Package) error {
		if repo.packageRefs != nil && skipExistingPackages {
			if repo.packageRefs.Has(p) {
				// skip this package, but load checksums/files from package in DB
				var prevP *Package
				prevP, err = packageCollection.ByKey(p.Key(""))
				if err != nil {
					return err
				}

				p.UpdateFiles(prevP.Files())
				return nil
			}
		}

		list, err2 := p.DownloadList(packagePool, checksumStorage)
		if err2 != nil {
			return err2
		}

		for _, task := range list {
			key := task.File.DownloadURL()
			idx, found := seen[key]
			if !found {
				queue = append(queue, task)
				downloadSize += task.File.Checksums.Size
				seen[key] = len(queue) - 1
			} else {
				// hook up the task to duplicate entry already on the list
				queue[idx].Additional = append(queue[idx].Additional, task)
			}
		}

		return nil
	})
	if err != nil {
		return
	}

	return
}

// FinalizeDownload swaps for final value of package refs
func (repo *RemoteRepo) FinalizeDownload(collectionFactory *CollectionFactory, progress aptly.Progress) error {
	repo.LastDownloadDate = time.Now()

	if progress != nil {
		progress.InitBar(int64(repo.packageList.Len()), true)
	}

	var i int

	// update all the packages in collection
	err := repo.packageList.ForEach(func(p *Package) error {
		i++
		if progress != nil {
			progress.SetBar(i)
		}
		// download process might have updated checksums
		p.UpdateFiles(p.Files())
		return collectionFactory.PackageCollection().Update(p)
	})

	repo.packageRefs = NewPackageRefListFromPackageList(repo.packageList)

	if progress != nil {
		progress.ShutdownBar()
	}

	repo.packageList = nil

	return err
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
	*sync.RWMutex
	db   database.Storage
	list []*RemoteRepo
}

// NewRemoteRepoCollection loads RemoteRepos from DB and makes up collection
func NewRemoteRepoCollection(db database.Storage) *RemoteRepoCollection {
	result := &RemoteRepoCollection{
		RWMutex: &sync.RWMutex{},
		db:      db,
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
