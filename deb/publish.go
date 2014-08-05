package deb

import (
	"bufio"
	"bytes"
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/utils"
	"github.com/ugorji/go/codec"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type repoSourceItem struct {
	// Pointer to snapshot if SourceKind == "snapshot"
	snapshot *Snapshot
	// Pointer to local repo if SourceKind == "local"
	localRepo *LocalRepo
	// Package references is SourceKind == "local"
	packageRefs *PackageRefList
}

// PublishedRepo is a published for http/ftp representation of snapshot as Debian repository
type PublishedRepo struct {
	// Internal unique ID
	UUID string
	// Storage & Prefix & distribution should be unique across all published repositories
	Storage      string
	Prefix       string
	Distribution string
	Origin       string
	Label        string
	// Architectures is a list of all architectures published
	Architectures []string
	// SourceKind is "local"/"repo"
	SourceKind string

	// Map of sources by each component: component name -> source UUID
	Sources map[string]string

	// Legacy fields for compatibily with old published repositories (< 0.6)
	Component string
	// SourceUUID is UUID of either snapshot or local repo
	SourceUUID string `codec:"SnapshotUUID"`

	// Map of component to source items
	sourceItems map[string]repoSourceItem

	// True if repo is being re-published
	rePublishing bool
}

// walkUpTree goes from source in the tree of source snapshots/mirrors/local repos
// gathering information about declared components and distributions
func walkUpTree(source interface{}, collectionFactory *CollectionFactory) (rootDistributions []string, rootComponents []string) {
	var (
		head    interface{}
		current = []interface{}{source}
	)

	rootComponents = []string{}
	rootDistributions = []string{}

	// walk up the tree from current source up to roots (local or remote repos)
	// and collect information about distribution and components
	for len(current) > 0 {
		head, current = current[0], current[1:]

		if snapshot, ok := head.(*Snapshot); ok {
			for _, uuid := range snapshot.SourceIDs {
				if snapshot.SourceKind == "repo" {
					remoteRepo, err := collectionFactory.RemoteRepoCollection().ByUUID(uuid)
					if err != nil {
						continue
					}
					current = append(current, remoteRepo)
				} else if snapshot.SourceKind == "local" {
					localRepo, err := collectionFactory.LocalRepoCollection().ByUUID(uuid)
					if err != nil {
						continue
					}
					current = append(current, localRepo)
				} else if snapshot.SourceKind == "snapshot" {
					snap, err := collectionFactory.SnapshotCollection().ByUUID(uuid)
					if err != nil {
						continue
					}
					current = append(current, snap)
				}
			}
		} else if localRepo, ok := head.(*LocalRepo); ok {
			if localRepo.DefaultDistribution != "" {
				rootDistributions = append(rootDistributions, localRepo.DefaultDistribution)
			}
			if localRepo.DefaultComponent != "" {
				rootComponents = append(rootComponents, localRepo.DefaultComponent)
			}
		} else if remoteRepo, ok := head.(*RemoteRepo); ok {
			if remoteRepo.Distribution != "" {
				rootDistributions = append(rootDistributions, remoteRepo.Distribution)
			}
			rootComponents = append(rootComponents, remoteRepo.Components...)
		} else {
			panic("unknown type")
		}
	}

	return
}

// NewPublishedRepo creates new published repository
//
// storage is PublishedStorage name
// prefix specifies publishing prefix
// distribution and architectures are user-defined properties
// components & sources are lists of component to source mapping (*Snapshot or *LocalRepo)
func NewPublishedRepo(storage, prefix, distribution string, architectures []string,
	components []string, sources []interface{}, collectionFactory *CollectionFactory) (*PublishedRepo, error) {
	result := &PublishedRepo{
		UUID:          uuid.New(),
		Storage:       storage,
		Architectures: architectures,
		Sources:       make(map[string]string),
		sourceItems:   make(map[string]repoSourceItem),
	}

	if len(sources) == 0 {
		panic("publish with empty sources")
	}

	if len(sources) != len(components) {
		panic("sources and components should be equal in size")
	}

	var (
		discoveredDistributions = []string{}
		source                  interface{}
		component               string
		snapshot                *Snapshot
		localRepo               *LocalRepo
		ok                      bool
	)

	// get first source
	source = sources[0]

	// figure out source kind
	snapshot, ok = source.(*Snapshot)
	if ok {
		result.SourceKind = "snapshot"
	} else {
		localRepo, ok = source.(*LocalRepo)
		if ok {
			result.SourceKind = "local"
		} else {
			panic("unknown source kind")
		}
	}

	for i := range sources {
		component, source = components[i], sources[i]
		if distribution == "" || component == "" {
			rootDistributions, rootComponents := walkUpTree(source, collectionFactory)
			if distribution == "" {
				discoveredDistributions = append(discoveredDistributions, rootDistributions...)
			}
			if component == "" {
				sort.Strings(rootComponents)
				if len(rootComponents) > 0 && rootComponents[0] == rootComponents[len(rootComponents)-1] {
					component = rootComponents[0]
				} else if len(sources) == 1 {
					// only if going from one source, assume default component "main"
					component = "main"
				} else {
					return nil, fmt.Errorf("unable to figure out component name for %s", source)
				}
			}
		}

		_, exists := result.Sources[component]
		if exists {
			return nil, fmt.Errorf("duplicate component name: %s", component)
		}

		if result.SourceKind == "snapshot" {
			snapshot = source.(*Snapshot)
			result.Sources[component] = snapshot.UUID
			result.sourceItems[component] = repoSourceItem{snapshot: snapshot}
		} else if result.SourceKind == "local" {
			localRepo = source.(*LocalRepo)
			result.Sources[component] = localRepo.UUID
			result.sourceItems[component] = repoSourceItem{localRepo: localRepo, packageRefs: localRepo.RefList()}
		}
	}

	// clean & verify prefix
	prefix = filepath.Clean(prefix)
	if strings.HasPrefix(prefix, "/") {
		prefix = prefix[1:]
	}
	if strings.HasSuffix(prefix, "/") {
		prefix = prefix[:len(prefix)-1]
	}
	prefix = filepath.Clean(prefix)

	for _, part := range strings.Split(prefix, "/") {
		if part == ".." || part == "dists" || part == "pool" {
			return nil, fmt.Errorf("invalid prefix %s", prefix)
		}
	}

	result.Prefix = prefix

	// guessing distribution
	if distribution == "" {
		sort.Strings(discoveredDistributions)
		if len(discoveredDistributions) > 0 && discoveredDistributions[0] == discoveredDistributions[len(discoveredDistributions)-1] {
			distribution = discoveredDistributions[0]
		} else {
			return nil, fmt.Errorf("unable to guess distribution name, please specify explicitly")
		}
	}

	result.Distribution = distribution

	return result, nil
}

// String returns human-readable represenation of PublishedRepo
func (p *PublishedRepo) String() string {
	var sources = []string{}

	for _, component := range p.Components() {
		var source string

		item := p.sourceItems[component]
		if item.snapshot != nil {
			source = item.snapshot.String()
		} else if item.localRepo != nil {
			source = item.localRepo.String()
		} else {
			panic("no snapshot/localRepo")
		}

		sources = append(sources, fmt.Sprintf("{%s: %s}", component, source))
	}

	var extra string

	if p.Origin != "" {
		extra += fmt.Sprintf("origin: %s", p.Origin)
	}

	if p.Label != "" {
		if extra != "" {
			extra += ", "
		}
		extra += fmt.Sprintf("label: %s", p.Label)
	}

	if extra != "" {
		extra = " (" + extra + ")"
	}

	return fmt.Sprintf("%s/%s%s [%s] publishes %s", p.StoragePrefix(), p.Distribution, extra, strings.Join(p.Architectures, ", "),
		strings.Join(sources, ", "))
}

// StoragePrefix returns combined storage & prefix for the repo
func (p *PublishedRepo) StoragePrefix() string {
	result := p.Prefix
	if p.Storage != "" {
		result = p.Storage + ":" + p.Prefix
	}
	return result
}

// Key returns unique key identifying PublishedRepo
func (p *PublishedRepo) Key() []byte {
	return []byte("U" + p.StoragePrefix() + ">>" + p.Distribution)
}

// RefKey is a unique id for package reference list
func (p *PublishedRepo) RefKey(component string) []byte {
	return []byte("E" + p.UUID + component)
}

// RefList returns list of package refs in local repo
func (p *PublishedRepo) RefList(component string) *PackageRefList {
	item := p.sourceItems[component]
	if p.SourceKind == "local" {
		return item.packageRefs
	}
	if p.SourceKind == "snapshot" {
		return item.snapshot.RefList()
	}
	panic("unknown source")
}

// Components returns sorted list of published repo components
func (p *PublishedRepo) Components() []string {
	result := make([]string, 0, len(p.Sources))
	for component := range p.Sources {
		result = append(result, component)
	}

	sort.Strings(result)
	return result
}

// UpdateLocalRepo updates content from local repo in component
func (p *PublishedRepo) UpdateLocalRepo(component string) {
	if p.SourceKind != "local" {
		panic("not local repo publish")
	}

	item := p.sourceItems[component]
	item.packageRefs = item.localRepo.RefList()
	p.sourceItems[component] = item

	p.rePublishing = true
}

// UpdateSnapshot switches snapshot for component
func (p *PublishedRepo) UpdateSnapshot(component string, snapshot *Snapshot) {
	if p.SourceKind != "snapshot" {
		panic("not snapshot publish")
	}

	item := p.sourceItems[component]
	item.snapshot = snapshot
	p.sourceItems[component] = item

	p.Sources[component] = snapshot.UUID
	p.rePublishing = true
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
	err := decoder.Decode(p)
	if err != nil {
		return err
	}

	// old PublishedRepo were publishing only snapshots
	if p.SourceKind == "" {
		p.SourceKind = "snapshot"
	}

	// <0.6 aptly used single SourceUUID + Component instead of Sources
	if p.Component != "" && p.SourceUUID != "" && len(p.Sources) == 0 {
		p.Sources = map[string]string{p.Component: p.SourceUUID}
		p.Component = ""
		p.SourceUUID = ""
	}

	return nil
}

// GetOrigin returns default or manual Origin:
func (p *PublishedRepo) GetOrigin() string {
	if p.Origin == "" {
		return p.Prefix + " " + p.Distribution
	}
	return p.Origin
}

// GetLabel returns default or manual Label:
func (p *PublishedRepo) GetLabel() string {
	if p.Label == "" {
		return p.Prefix + " " + p.Distribution
	}
	return p.Label
}

// Publish publishes snapshot (repository) contents, links package files, generates Packages & Release files, signs them
func (p *PublishedRepo) Publish(packagePool aptly.PackagePool, publishedStorageProvider aptly.PublishedStorageProvider,
	collectionFactory *CollectionFactory, signer utils.Signer, progress aptly.Progress, forceOverwrite bool) error {
	publishedStorage := publishedStorageProvider.GetPublishedStorage(p.Storage)

	err := publishedStorage.MkDir(filepath.Join(p.Prefix, "pool"))
	if err != nil {
		return err
	}
	basePath := filepath.Join(p.Prefix, "dists", p.Distribution)
	err = publishedStorage.MkDir(basePath)
	if err != nil {
		return err
	}

	if progress != nil {
		progress.Printf("Loading packages...\n")
	}

	lists := map[string]*PackageList{}

	for component := range p.sourceItems {
		// Load all packages
		lists[component], err = NewPackageListFromRefList(p.RefList(component), collectionFactory.PackageCollection(), progress)
		if err != nil {
			return fmt.Errorf("unable to load packages: %s", err)
		}
	}

	if !p.rePublishing {
		if len(p.Architectures) == 0 {
			for _, list := range lists {
				p.Architectures = append(p.Architectures, list.Architectures(true)...)
			}
		}

		if len(p.Architectures) == 0 {
			return fmt.Errorf("unable to figure out list of architectures, please supply explicit list")
		}

		sort.Strings(p.Architectures)
		p.Architectures = utils.StrSliceDeduplicate(p.Architectures)
	}

	var suffix string
	if p.rePublishing {
		suffix = ".tmp"
	}

	generatedFiles := map[string]utils.ChecksumInfo{}
	renameMap := map[string]string{}

	if progress != nil {
		progress.Printf("Generating metadata files and linking package files...\n")
	}

	var tempDir string
	tempDir, err = ioutil.TempDir(os.TempDir(), "aptly")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	for component, list := range lists {
		var relativePath string

		// For all architectures, generate packages/sources files
		for _, arch := range p.Architectures {
			if progress != nil {
				progress.InitBar(int64(list.Len()), false)
			}

			if arch == "source" {
				relativePath = filepath.Join(component, "source", "Sources")
			} else {
				relativePath = filepath.Join(component, fmt.Sprintf("binary-%s", arch), "Packages")
			}
			err = publishedStorage.MkDir(filepath.Dir(filepath.Join(basePath, relativePath)))
			if err != nil {
				return err
			}

			var packagesFile *os.File

			packagesFileName := filepath.Join(tempDir, fmt.Sprintf("pkgs_%s_%s", component, arch))
			packagesFile, err = os.Create(packagesFileName)
			if err != nil {
				return fmt.Errorf("unable to create temporary Packages file: %s", err)
			}

			bufWriter := bufio.NewWriter(packagesFile)

			err = list.ForEach(func(pkg *Package) error {
				if progress != nil {
					progress.AddBar(1)
				}
				if pkg.MatchesArchitecture(arch) {
					err = pkg.LinkFromPool(publishedStorage, packagePool, p.Prefix, component, forceOverwrite)
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

			for _, ext := range []string{"", ".gz", ".bz2"} {
				var checksumInfo utils.ChecksumInfo

				checksumInfo, err = utils.ChecksumsForFile(packagesFileName + ext)
				if err != nil {
					return fmt.Errorf("unable to collect checksums: %s", err)
				}
				generatedFiles[relativePath+ext] = checksumInfo

				err = publishedStorage.PutFile(filepath.Join(basePath, relativePath+suffix+ext), packagesFileName+ext)
				if err != nil {
					return fmt.Errorf("unable to publish file: %s", err)
				}

				if suffix != "" {
					renameMap[filepath.Join(basePath, relativePath+suffix+ext)] = filepath.Join(basePath, relativePath+ext)
				}
			}

			if progress != nil {
				progress.ShutdownBar()
			}
		}

		// For all architectures, generate Release files
		for _, arch := range p.Architectures {
			release := make(Stanza)
			release["Archive"] = p.Distribution
			release["Architecture"] = arch
			release["Component"] = component
			release["Origin"] = p.GetOrigin()
			release["Label"] = p.GetLabel()

			if arch == "source" {
				relativePath = filepath.Join(component, "source", "Release")
			} else {
				relativePath = filepath.Join(component, fmt.Sprintf("binary-%s", arch), "Release")
			}

			var file *os.File

			fileName := filepath.Join(tempDir, fmt.Sprintf("release_%s_%s", component, arch))
			file, err = os.Create(fileName)
			if err != nil {
				return fmt.Errorf("unable to create temporary Release file: %s", err)
			}

			bufWriter := bufio.NewWriter(file)

			err = release.WriteTo(bufWriter)
			if err != nil {
				return fmt.Errorf("unable to create Release file: %s", err)
			}

			err = bufWriter.Flush()
			if err != nil {
				return fmt.Errorf("unable to create Release file: %s", err)
			}

			file.Close()

			var checksumInfo utils.ChecksumInfo
			checksumInfo, err = utils.ChecksumsForFile(fileName)
			if err != nil {
				return fmt.Errorf("unable to collect checksums: %s", err)
			}
			generatedFiles[relativePath] = checksumInfo

			err = publishedStorage.PutFile(filepath.Join(basePath, relativePath+suffix), fileName)
			if err != nil {
				file.Close()
				return fmt.Errorf("unable to publish file: %s", err)
			}

			if suffix != "" {
				renameMap[filepath.Join(basePath, relativePath+suffix)] = filepath.Join(basePath, relativePath)
			}

		}
	}

	release := make(Stanza)
	release["Origin"] = p.GetOrigin()
	release["Label"] = p.GetLabel()
	release["Codename"] = p.Distribution
	release["Date"] = time.Now().UTC().Format("Mon, 2 Jan 2006 15:04:05 MST")
	release["Architectures"] = strings.Join(utils.StrSlicesSubstract(p.Architectures, []string{"source"}), " ")
	release["Description"] = " Generated by aptly\n"
	release["MD5Sum"] = "\n"
	release["SHA1"] = "\n"
	release["SHA256"] = "\n"

	release["Components"] = strings.Join(p.Components(), " ")

	for path, info := range generatedFiles {
		release["MD5Sum"] += fmt.Sprintf(" %s %8d %s\n", info.MD5, info.Size, path)
		release["SHA1"] += fmt.Sprintf(" %s %8d %s\n", info.SHA1, info.Size, path)
		release["SHA256"] += fmt.Sprintf(" %s %8d %s\n", info.SHA256, info.Size, path)
	}

	var releaseFile *os.File
	releaseFilename := filepath.Join(tempDir, "Release")
	releaseFile, err = os.Create(releaseFilename)
	if err != nil {
		return fmt.Errorf("unable to create temporary Release file: %s", err)
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

	releaseFile.Close()

	if suffix != "" {
		renameMap[filepath.Join(basePath, "Release"+suffix)] = filepath.Join(basePath, "Release")
	}

	err = publishedStorage.PutFile(filepath.Join(basePath, "Release"+suffix), releaseFilename)
	if err != nil {
		return fmt.Errorf("unable to publish file: %s", err)
	}

	// Signing files might output to console, so flush progress writer first
	if progress != nil {
		progress.Flush()
	}

	if signer != nil {
		err = signer.DetachedSign(releaseFilename, releaseFilename+".gpg")
		if err != nil {
			return fmt.Errorf("unable to sign Release file: %s", err)
		}

		err = signer.ClearSign(releaseFilename, filepath.Join(filepath.Dir(releaseFilename), "InRelease"+suffix))
		if err != nil {
			return fmt.Errorf("unable to sign Release file: %s", err)
		}

		if suffix != "" {
			renameMap[filepath.Join(basePath, "Release"+suffix+".gpg")] = filepath.Join(basePath, "Release.gpg")
			renameMap[filepath.Join(basePath, "InRelease"+suffix)] = filepath.Join(basePath, "InRelease")
		}

		err = publishedStorage.PutFile(filepath.Join(basePath, "Release"+suffix+".gpg"), releaseFilename+".gpg")
		if err != nil {
			return fmt.Errorf("unable to publish file: %s", err)
		}

		err = publishedStorage.PutFile(filepath.Join(basePath, "InRelease"+suffix),
			filepath.Join(filepath.Dir(releaseFilename), "InRelease"+suffix))
		if err != nil {
			return fmt.Errorf("unable to publish file: %s", err)
		}
	}

	for oldName, newName := range renameMap {
		err = publishedStorage.RenameFile(oldName, newName)
		if err != nil {
			return fmt.Errorf("unable to rename: %s", err)
		}
	}

	return nil
}

// RemoveFiles removes files that were created by Publish
//
// It can remove prefix fully, and part of pool (for specific component)
func (p *PublishedRepo) RemoveFiles(publishedStorageProvider aptly.PublishedStorageProvider, removePrefix bool,
	removePoolComponents []string, progress aptly.Progress) error {
	publishedStorage := publishedStorageProvider.GetPublishedStorage(p.Storage)

	// I. Easy: remove whole prefix (meta+packages)
	if removePrefix {
		err := publishedStorage.RemoveDirs(filepath.Join(p.Prefix, "dists"), progress)
		if err != nil {
			return err
		}

		return publishedStorage.RemoveDirs(filepath.Join(p.Prefix, "pool"), progress)
	}

	// II. Medium: remove metadata, it can't be shared as prefix/distribution as unique
	err := publishedStorage.RemoveDirs(filepath.Join(p.Prefix, "dists", p.Distribution), progress)
	if err != nil {
		return err
	}

	// III. Complex: there are no other publishes with the same prefix + component
	for _, component := range removePoolComponents {
		err = publishedStorage.RemoveDirs(filepath.Join(p.Prefix, "pool", component), progress)
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
		return fmt.Errorf("published repo with storage/prefix/distribution %s/%s/%s already exists", repo.Storage, repo.Prefix, repo.Distribution)
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
		if r.Prefix == repo.Prefix && r.Distribution == repo.Distribution && r.Storage == repo.Storage {
			return r
		}
	}

	return nil
}

// Update stores updated information about repo in DB
func (collection *PublishedRepoCollection) Update(repo *PublishedRepo) (err error) {
	err = collection.db.Put(repo.Key(), repo.Encode())
	if err != nil {
		return
	}

	if repo.SourceKind == "local" {
		for component, item := range repo.sourceItems {
			err = collection.db.Put(repo.RefKey(component), item.packageRefs.Encode())
			if err != nil {
				return
			}
		}
	}
	return
}

// LoadComplete loads additional information for remote repo
func (collection *PublishedRepoCollection) LoadComplete(repo *PublishedRepo, collectionFactory *CollectionFactory) (err error) {
	repo.sourceItems = make(map[string]repoSourceItem)

	if repo.SourceKind == "snapshot" {
		for component, sourceUUID := range repo.Sources {
			item := repoSourceItem{}

			item.snapshot, err = collectionFactory.SnapshotCollection().ByUUID(sourceUUID)
			if err != nil {
				return
			}
			err = collectionFactory.SnapshotCollection().LoadComplete(item.snapshot)
			if err != nil {
				return
			}

			repo.sourceItems[component] = item
		}
	} else if repo.SourceKind == "local" {
		for component, sourceUUID := range repo.Sources {
			item := repoSourceItem{}

			item.localRepo, err = collectionFactory.LocalRepoCollection().ByUUID(sourceUUID)
			if err != nil {
				return
			}
			err = collectionFactory.LocalRepoCollection().LoadComplete(item.localRepo)
			if err != nil {
				return
			}

			var encoded []byte
			encoded, err = collection.db.Get(repo.RefKey(component))
			if err != nil {
				// < 0.6 saving w/o component name
				if err == database.ErrNotFound && len(repo.Sources) == 1 {
					encoded, err = collection.db.Get(repo.RefKey(""))
				}

				if err != nil {
					return
				}
			}

			item.packageRefs = &PackageRefList{}
			err = item.packageRefs.Decode(encoded)
			if err != nil {
				return
			}

			repo.sourceItems[component] = item
		}
	} else {
		panic("unknown SourceKind")
	}

	return
}

// ByStoragePrefixDistribution looks up repository by storage, prefix & distribution
func (collection *PublishedRepoCollection) ByStoragePrefixDistribution(storage, prefix, distribution string) (*PublishedRepo, error) {
	for _, r := range collection.list {
		if r.Prefix == prefix && r.Distribution == distribution && r.Storage == storage {
			return r, nil
		}
	}
	if storage != "" {
		storage += ":"
	}
	return nil, fmt.Errorf("published repo with storage:prefix/distribution %s%s/%s not found", storage, prefix, distribution)
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
		if r.SourceKind == "snapshot" {
			if r.SourceUUID == snapshot.UUID {
				result = append(result, r)
			}

			for _, sourceUUID := range r.Sources {
				if sourceUUID == snapshot.UUID {
					result = append(result, r)
					break
				}
			}
		}
	}
	return result
}

// ByLocalRepo looks up repository by local repo source
func (collection *PublishedRepoCollection) ByLocalRepo(repo *LocalRepo) []*PublishedRepo {
	result := make([]*PublishedRepo, 0)
	for _, r := range collection.list {
		if r.SourceKind == "local" {
			if r.SourceUUID == repo.UUID {
				result = append(result, r)
			}

			for _, sourceUUID := range r.Sources {
				if sourceUUID == repo.UUID {
					result = append(result, r)
					break
				}
			}
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

// CleanupPrefixComponentFiles removes all unreferenced files in published storage under prefix/component pair
func (collection *PublishedRepoCollection) CleanupPrefixComponentFiles(prefix string, components []string,
	publishedStorage aptly.PublishedStorage, collectionFactory *CollectionFactory, progress aptly.Progress) error {

	var err error
	referencedFiles := map[string][]string{}

	if progress != nil {
		progress.Printf("Cleaning up prefix %#v components %s...\n", prefix, strings.Join(components, ", "))
	}

	for _, r := range collection.list {
		if r.Prefix == prefix {
			matches := false

			repoComponents := r.Components()

			for _, component := range components {
				if utils.StrSliceHasItem(repoComponents, component) {
					matches = true
					break
				}
			}

			if !matches {
				continue
			}

			err = collection.LoadComplete(r, collectionFactory)
			if err != nil {
				return err
			}

			for _, component := range components {
				if utils.StrSliceHasItem(repoComponents, component) {
					packageList, err := NewPackageListFromRefList(r.RefList(component), collectionFactory.PackageCollection(), progress)
					if err != nil {
						return err
					}

					packageList.ForEach(func(p *Package) error {
						poolDir, err := p.PoolDirectory()
						if err != nil {
							return err
						}

						for _, f := range p.Files() {
							referencedFiles[component] = append(referencedFiles[component], filepath.Join(poolDir, f.Filename))
						}

						return nil
					})
				}
			}
		}
	}

	for _, component := range components {
		sort.Strings(referencedFiles[component])

		rootPath := filepath.Join(prefix, "pool", component)
		existingFiles, err := publishedStorage.Filelist(rootPath)
		if err != nil {
			return err
		}

		sort.Strings(existingFiles)

		filesToDelete := utils.StrSlicesSubstract(existingFiles, referencedFiles[component])

		for _, file := range filesToDelete {
			err = publishedStorage.Remove(filepath.Join(rootPath, file))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Remove removes published repository, cleaning up directories, files
func (collection *PublishedRepoCollection) Remove(publishedStorageProvider aptly.PublishedStorageProvider,
	storage, prefix, distribution string, collectionFactory *CollectionFactory, progress aptly.Progress) error {
	repo, err := collection.ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		return err
	}

	removePrefix := true
	removePoolComponents := repo.Components()
	cleanComponents := []string{}
	repoPosition := -1

	for i, r := range collection.list {
		if r == repo {
			repoPosition = i
			continue
		}
		if r.Storage == repo.Storage && r.Prefix == repo.Prefix {
			removePrefix = false

			rComponents := r.Components()
			for _, component := range rComponents {
				if utils.StrSliceHasItem(removePoolComponents, component) {
					removePoolComponents = utils.StrSlicesSubstract(removePoolComponents, []string{component})
					cleanComponents = append(cleanComponents, component)
				}
			}
		}
	}

	err = repo.RemoveFiles(publishedStorageProvider, removePrefix, removePoolComponents, progress)
	if err != nil {
		return err
	}

	collection.list[len(collection.list)-1], collection.list[repoPosition], collection.list =
		nil, collection.list[len(collection.list)-1], collection.list[:len(collection.list)-1]

	if len(cleanComponents) > 0 {
		err = collection.CleanupPrefixComponentFiles(repo.Prefix, cleanComponents,
			publishedStorageProvider.GetPublishedStorage(storage), collectionFactory, progress)
		if err != nil {
			return err
		}
	}

	err = collection.db.Delete(repo.Key())
	if err != nil {
		return err
	}

	for _, component := range repo.Components() {
		err = collection.db.Delete(repo.RefKey(component))
		if err != nil {
			return err
		}
	}

	return nil
}
