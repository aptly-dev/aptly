package deb

import (
	"bytes"
	gocontext "context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/database"
	"github.com/aptly-dev/aptly/http"
	"github.com/aptly-dev/aptly/pgp"
	"github.com/aptly-dev/aptly/utils"
	"github.com/pborman/uuid"
	"github.com/ugorji/go/codec"
)

// RemoteRepo statuses
const (
	MirrorIdle = iota
	MirrorUpdating
)

// RemoteRepo represents remote (fetchable) Debian repository.
//
// Repository could be filtered when fetching by components, architectures
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
	ReleaseFiles map[string]utils.ChecksumInfo `json:"-"` // exclude from json output
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
	// Should we download installer files?
	DownloadInstaller bool
	// Packages for json output
	Packages []string `codec:"-" json:",omitempty"`
	// "Snapshot" of current list of packages
	packageRefs *PackageRefList
	// Parsed archived root
	archiveRootURL *url.URL
	// Current list of packages (filled while updating mirror)
	packageList *PackageList
}

// NewRemoteRepo creates new instance of Debian remote repository with specified params
func NewRemoteRepo(name string, archiveRoot string, distribution string, components []string,
	architectures []string, downloadSources bool, downloadUdebs bool, downloadInstaller bool) (*RemoteRepo, error) {
	result := &RemoteRepo{
		UUID:              uuid.New(),
		Name:              name,
		ArchiveRoot:       archiveRoot,
		Distribution:      distribution,
		Components:        components,
		Architectures:     architectures,
		DownloadSources:   downloadSources,
		DownloadUdebs:     downloadUdebs,
		DownloadInstaller: downloadInstaller,
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

// SetArchiveRoot of remote repo
func (repo *RemoteRepo) SetArchiveRoot(archiveRoot string) {
	repo.ArchiveRoot = archiveRoot
	repo.prepare()
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
	if repo.DownloadInstaller {
		srcFlag += " [installer]"
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

// PackageList returns package list for repo
func (repo *RemoteRepo) PackageList() *PackageList {
	return repo.packageList
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

// InstallerPath returns path of Packages files for given component and
// architecture
func (repo *RemoteRepo) InstallerPath(component string, architecture string) string {
	return fmt.Sprintf("%s/installer-%s/current/images/SHA256SUMS", component, architecture)
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
		release, err = http.DownloadTemp(gocontext.TODO(), d, repo.ReleaseURL("Release").String())
		if err != nil {
			return err
		}
	} else {
		// 1. try InRelease file
		inrelease, err = http.DownloadTemp(gocontext.TODO(), d, repo.ReleaseURL("InRelease").String())
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
		release, err = http.DownloadTemp(gocontext.TODO(), d, repo.ReleaseURL("Release").String())
		if err != nil {
			return err
		}

		releasesig, err = http.DownloadTemp(gocontext.TODO(), d, repo.ReleaseURL("Release.gpg").String())
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

	sreader := NewControlFileReader(release, true, false)
	stanza, err := sreader.ReadStanza()
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

	err = parseSums("SHA512", func(sum *utils.ChecksumInfo, data string) { sum.SHA512 = data })
	if err != nil {
		return err
	}

	repo.Meta = stanza

	return nil
}

// DownloadPackageIndexes downloads & parses package index files
func (repo *RemoteRepo) DownloadPackageIndexes(progress aptly.Progress, d aptly.Downloader, verifier pgp.Verifier, collectionFactory *CollectionFactory,
	ignoreMismatch bool) error {
	if repo.packageList != nil {
		panic("packageList != nil")
	}
	repo.packageList = NewPackageList()

	// Download and parse all Packages & Source files
	packagesPaths := [][]string{}

	if repo.IsFlat() {
		packagesPaths = append(packagesPaths, []string{repo.FlatBinaryPath(), PackageTypeBinary, "", ""})
		if repo.DownloadSources {
			packagesPaths = append(packagesPaths, []string{repo.FlatSourcesPath(), PackageTypeSource, "", ""})
		}
	} else {
		for _, component := range repo.Components {
			for _, architecture := range repo.Architectures {
				packagesPaths = append(packagesPaths, []string{repo.BinaryPath(component, architecture), PackageTypeBinary, component, architecture})
				if repo.DownloadUdebs {
					packagesPaths = append(packagesPaths, []string{repo.UdebPath(component, architecture), PackageTypeUdeb, component, architecture})
				}
				if repo.DownloadInstaller {
					packagesPaths = append(packagesPaths, []string{repo.InstallerPath(component, architecture), PackageTypeInstaller, component, architecture})
				}
			}
			if repo.DownloadSources {
				packagesPaths = append(packagesPaths, []string{repo.SourcesPath(component), PackageTypeSource, component, "source"})
			}
		}
	}

	for _, info := range packagesPaths {
		path, kind, component, architecture := info[0], info[1], info[2], info[3]
		packagesReader, packagesFile, err := http.DownloadTryCompression(gocontext.TODO(), d, repo.IndexesRootURL(), path, repo.ReleaseFiles, ignoreMismatch)

		isInstaller := kind == PackageTypeInstaller
		if err != nil {
			if _, ok := err.(*http.NoCandidateFoundError); isInstaller && ok {
				// checking if gpg file is only needed when checksums matches are required.
				// otherwise there actually has been no candidate found and we can continue
				if ignoreMismatch {
					continue
				}

				// some repos do not have installer hashsum file listed in release file but provide a separate gpg file
				hashsumPath := repo.IndexesRootURL().ResolveReference(&url.URL{Path: path}).String()
				packagesFile, err = http.DownloadTemp(gocontext.TODO(), d, hashsumPath)
				if err != nil {
					if herr, ok := err.(*http.Error); ok && (herr.Code == 404 || herr.Code == 403) {
						// installer files are not available in all components and architectures
						// so ignore it if not found
						continue
					}

					return err
				}

				if verifier != nil {
					hashsumGpgPath := repo.IndexesRootURL().ResolveReference(&url.URL{Path: path + ".gpg"}).String()
					var filesig *os.File
					filesig, err = http.DownloadTemp(gocontext.TODO(), d, hashsumGpgPath)
					if err != nil {
						return err
					}

					err = verifier.VerifyDetachedSignature(filesig, packagesFile, false)
					if err != nil {
						return err
					}

					_, err = packagesFile.Seek(0, 0)
				}

				packagesReader = packagesFile
			}

			if err != nil {
				return err
			}
		}
		defer packagesFile.Close()

		if progress != nil {
			stat, _ := packagesFile.Stat()
			progress.InitBar(stat.Size(), true)
		}

		sreader := NewControlFileReader(packagesReader, false, isInstaller)

		for {
			stanza, err := sreader.ReadStanza()
			if err != nil {
				return err
			}
			if stanza == nil {
				break
			}

			if progress != nil {
				off, _ := packagesFile.Seek(0, 1)
				progress.SetBar(int(off))
			}

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
			} else if kind == PackageTypeInstaller {
				p, err = NewInstallerPackageFromControlFile(stanza, repo, component, architecture, d)
				if err != nil {
					return err
				}
			}
			err = repo.packageList.Add(p)
			if err != nil {
				if _, ok := err.(*PackageConflictError); ok {
					if progress != nil {
						progress.ColoredPrintf("@y[!]@| @!skipping package %s: duplicate in packages index@|", p)
					}
				} else if err != nil {
					return err
				}
			}
		}

		if progress != nil {
			progress.ShutdownBar()
		}
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
	transaction, err := collectionFactory.PackageCollection().db.OpenTransaction()
	if err != nil {
		return err
	}
	defer transaction.Discard()

	repo.LastDownloadDate = time.Now()

	if progress != nil {
		progress.InitBar(int64(repo.packageList.Len()), false)
	}

	var i int

	// update all the packages in collection
	err = repo.packageList.ForEach(func(p *Package) error {
		i++
		if progress != nil {
			progress.SetBar(i)
		}
		// download process might have updated checksums
		p.UpdateFiles(p.Files())
		return collectionFactory.PackageCollection().UpdateInTransaction(p, transaction)
	})

	if err == nil {
		repo.packageRefs = NewPackageRefListFromPackageList(repo.packageList)
		repo.packageList = nil
	}

	if progress != nil {
		progress.ShutdownBar()
	}

	if err != nil {
		return err
	}
	return transaction.Commit()
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
			var repo11 struct { // nolint: maligned
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
		} else if strings.Contains(err.Error(), "invalid length of bytes for decoding time") {
			// DB created by old codec version, time.Time is not builtin type.
			// https://github.com/ugorji/go-codec/issues/269
			decoder := codec.NewDecoderBytes(input, &codec.MsgpackHandle{
				// only can be configured in Deprecated BasicHandle struct
				BasicHandle: codec.BasicHandle{ // nolint: staticcheck
					TimeNotBuiltin: true,
				},
			})
			if err = decoder.Decode(repo); err != nil {
				return err
			}
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
	db    database.Storage
	cache map[string]*RemoteRepo
}

// NewRemoteRepoCollection loads RemoteRepos from DB and makes up collection
func NewRemoteRepoCollection(db database.Storage) *RemoteRepoCollection {
	return &RemoteRepoCollection{
		db:    db,
		cache: make(map[string]*RemoteRepo),
	}
}

func (collection *RemoteRepoCollection) search(filter func(*RemoteRepo) bool, unique bool) []*RemoteRepo {
	result := []*RemoteRepo(nil)
	for _, r := range collection.cache {
		if filter(r) {
			result = append(result, r)
		}
	}

	if unique && len(result) > 0 {
		return result
	}

	collection.db.ProcessByPrefix([]byte("R"), func(key, blob []byte) error {
		r := &RemoteRepo{}
		if err := r.Decode(blob); err != nil {
			log.Printf("Error decoding remote repo: %s\n", err)
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
func (collection *RemoteRepoCollection) Add(repo *RemoteRepo) error {
	_, err := collection.ByName(repo.Name)

	if err == nil {
		return fmt.Errorf("mirror with name %s already exists", repo.Name)
	}

	err = collection.Update(repo)
	if err != nil {
		return err
	}

	collection.cache[repo.UUID] = repo
	return nil
}

// Update stores updated information about repo in DB
func (collection *RemoteRepoCollection) Update(repo *RemoteRepo) error {
	batch := collection.db.CreateBatch()

	batch.Put(repo.Key(), repo.Encode())
	if repo.packageRefs != nil {
		batch.Put(repo.RefKey(), repo.packageRefs.Encode())
	}
	return batch.Write()
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
	result := collection.search(func(r *RemoteRepo) bool { return r.Name == name }, true)
	if len(result) == 0 {
		return nil, fmt.Errorf("mirror with name %s not found", name)
	}

	return result[0], nil
}

// ByUUID looks up repository by uuid
func (collection *RemoteRepoCollection) ByUUID(uuid string) (*RemoteRepo, error) {
	if r, ok := collection.cache[uuid]; ok {
		return r, nil
	}

	key := (&RemoteRepo{UUID: uuid}).Key()

	value, err := collection.db.Get(key)
	if err == database.ErrNotFound {
		return nil, fmt.Errorf("mirror with uuid %s not found", uuid)
	}
	if err != nil {
		return nil, err
	}

	r := &RemoteRepo{}
	err = r.Decode(value)

	if err == nil {
		collection.cache[r.UUID] = r
	}

	return r, err
}

// ForEach runs method for each repository
func (collection *RemoteRepoCollection) ForEach(handler func(*RemoteRepo) error) error {
	return collection.db.ProcessByPrefix([]byte("R"), func(key, blob []byte) error {
		r := &RemoteRepo{}
		if err := r.Decode(blob); err != nil {
			log.Printf("Error decoding mirror: %s\n", err)
			return nil
		}

		return handler(r)
	})
}

// Len returns number of remote repos
func (collection *RemoteRepoCollection) Len() int {
	return len(collection.db.KeysByPrefix([]byte("R")))
}

// Drop removes remote repo from collection
func (collection *RemoteRepoCollection) Drop(repo *RemoteRepo) error {
	transaction, err := collection.db.OpenTransaction()
	if err != nil {
		return err
	}
	defer transaction.Discard()

	if _, err = transaction.Get(repo.Key()); err != nil {
		if err == database.ErrNotFound {
			return errors.New("repo not found")
		}

		return err
	}

	delete(collection.cache, repo.UUID)

	batch := collection.db.CreateBatch()
	batch.Delete(repo.Key())
	batch.Delete(repo.RefKey())
	return batch.Write()
}
