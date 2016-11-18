package deb

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/database"
	"github.com/ugorji/go/codec"
)

// PackageCollection does management of packages in DB
type PackageCollection struct {
	db          database.Storage
	codecHandle *codec.MsgpackHandle
}

// Verify interface
var (
	_ PackageCatalog = &PackageCollection{}
)

// NewPackageCollection creates new PackageCollection and binds it to database
func NewPackageCollection(db database.Storage) *PackageCollection {
	return &PackageCollection{
		db:          db,
		codecHandle: &codec.MsgpackHandle{},
	}
}

// oldPackage is Package struct for aptly < 0.4 with all fields in one struct
// It is used to decode old aptly DBs
type oldPackage struct {
	IsSource           bool
	Name               string
	Version            string
	Architecture       string
	SourceArchitecture string
	Source             string
	Provides           []string
	Depends            []string
	BuildDepends       []string
	BuildDependsInDep  []string
	PreDepends         []string
	Suggests           []string
	Recommends         []string
	Files              []PackageFile
	Extra              Stanza
}

// ByKey find package in DB by its key
func (collection *PackageCollection) ByKey(key []byte) (*Package, error) {
	encoded, err := collection.db.Get(key)
	if err != nil {
		return nil, err
	}

	p := &Package{}

	if len(encoded) > 2 && (encoded[0] != 0xc1 || encoded[1] != 0x1) {
		oldp := &oldPackage{}

		decoder := codec.NewDecoderBytes(encoded, collection.codecHandle)
		err = decoder.Decode(oldp)
		if err != nil {
			return nil, err
		}

		p.Name = oldp.Name
		p.Version = oldp.Version
		p.Architecture = oldp.Architecture
		p.IsSource = oldp.IsSource
		p.SourceArchitecture = oldp.SourceArchitecture
		p.Source = oldp.Source
		p.Provides = oldp.Provides

		p.deps = &PackageDependencies{
			Depends:           oldp.Depends,
			BuildDepends:      oldp.BuildDepends,
			BuildDependsInDep: oldp.BuildDependsInDep,
			PreDepends:        oldp.PreDepends,
			Suggests:          oldp.Suggests,
			Recommends:        oldp.Recommends,
		}

		p.extra = &oldp.Extra
		for i := range oldp.Files {
			oldp.Files[i].Filename = filepath.Base(oldp.Files[i].Filename)
		}
		p.UpdateFiles(PackageFiles(oldp.Files))

		// Save in new format
		err = collection.Update(p)
		if err != nil {
			return nil, err
		}
	} else {
		decoder := codec.NewDecoderBytes(encoded[2:], collection.codecHandle)
		err = decoder.Decode(p)
		if err != nil {
			return nil, err
		}
	}

	p.collection = collection

	return p, nil
}

// loadExtra loads Stanza with all the xtra information about the package
func (collection *PackageCollection) loadExtra(p *Package) *Stanza {
	encoded, err := collection.db.Get(p.Key("xE"))
	if err != nil {
		panic("unable to load extra")
	}

	stanza := &Stanza{}

	decoder := codec.NewDecoderBytes(encoded, collection.codecHandle)
	err = decoder.Decode(stanza)
	if err != nil {
		panic("unable to decode extra")
	}

	return stanza
}

// loadDependencies loads dependencies for the package
func (collection *PackageCollection) loadDependencies(p *Package) *PackageDependencies {
	encoded, err := collection.db.Get(p.Key("xD"))
	if err != nil {
		panic(fmt.Sprintf("unable to load deps: %s, %s", p, err))
	}

	deps := &PackageDependencies{}

	decoder := codec.NewDecoderBytes(encoded, collection.codecHandle)
	err = decoder.Decode(deps)
	if err != nil {
		panic("unable to decode deps")
	}

	return deps
}

// loadFiles loads additional PackageFiles record
func (collection *PackageCollection) loadFiles(p *Package) *PackageFiles {
	encoded, err := collection.db.Get(p.Key("xF"))
	if err != nil {
		panic("unable to load files")
	}

	files := &PackageFiles{}

	decoder := codec.NewDecoderBytes(encoded, collection.codecHandle)
	err = decoder.Decode(files)
	if err != nil {
		panic("unable to decode files")
	}

	return files
}

// loadContents loads or calculates and saves package contents
func (collection *PackageCollection) loadContents(p *Package, packagePool aptly.PackagePool, progress aptly.Progress) []string {
	encoded, err := collection.db.Get(p.Key("xC"))
	if err == nil {
		contents := []string{}

		decoder := codec.NewDecoderBytes(encoded, collection.codecHandle)
		err = decoder.Decode(&contents)
		if err != nil {
			panic("unable to decode contents")
		}

		return contents
	}

	if err != database.ErrNotFound {
		panic("unable to load contents")
	}

	contents, err := p.CalculateContents(packagePool, progress)
	if err != nil {
		// failed to acquire contents, don't persist it
		return contents
	}

	var buf bytes.Buffer
	err = codec.NewEncoder(&buf, collection.codecHandle).Encode(contents)
	if err != nil {
		panic("unable to encode contents")
	}

	err = collection.db.Put(p.Key("xC"), buf.Bytes())
	if err != nil {
		panic("unable to save contents")
	}

	return contents
}

// Update adds or updates information about package in DB
func (collection *PackageCollection) Update(p *Package) error {
	transaction, err := collection.db.OpenTransaction()
	if err != nil {
		return err
	}
	defer transaction.Discard()

	if err = collection.UpdateInTransaction(p, transaction); err != nil {
		return err
	}

	return transaction.Commit()
}

// UpdateInTransaction updates/creates package info in the context of the outer transaction
func (collection *PackageCollection) UpdateInTransaction(p *Package, transaction database.Transaction) error {
	var encodeBuffer bytes.Buffer

	encoder := codec.NewEncoder(&encodeBuffer, collection.codecHandle)

	encodeBuffer.Reset()
	encodeBuffer.WriteByte(0xc1)
	encodeBuffer.WriteByte(0x1)
	if err := encoder.Encode(p); err != nil {
		return err
	}

	err := transaction.Put(p.Key(""), encodeBuffer.Bytes())
	if err != nil {
		return err
	}

	// Encode offloaded fields one by one
	if p.files != nil {
		encodeBuffer.Reset()
		err = encoder.Encode(*p.files)
		if err != nil {
			return err
		}

		err = transaction.Put(p.Key("xF"), encodeBuffer.Bytes())
		if err != nil {
			return err
		}
	}

	if p.deps != nil {
		encodeBuffer.Reset()
		err = encoder.Encode(*p.deps)
		if err != nil {
			return err
		}

		err = transaction.Put(p.Key("xD"), encodeBuffer.Bytes())
		if err != nil {
			return err
		}

		p.deps = nil
	}

	if p.extra != nil {
		encodeBuffer.Reset()
		err = encoder.Encode(*p.extra)
		if err != nil {
			return err
		}

		err = transaction.Put(p.Key("xE"), encodeBuffer.Bytes())
		if err != nil {
			return err
		}

		p.extra = nil
	}

	p.collection = collection
	return nil
}

// AllPackageRefs returns list of all packages as PackageRefList
func (collection *PackageCollection) AllPackageRefs() *PackageRefList {
	return &PackageRefList{Refs: collection.db.KeysByPrefix([]byte("P"))}
}

// DeleteByKey deletes package in DB by key
func (collection *PackageCollection) DeleteByKey(key []byte, dbw database.Writer) error {
	for _, key := range [][]byte{key, append([]byte("xF"), key...), append([]byte("xD"), key...), append([]byte("xE"), key...)} {
		err := dbw.Delete(key)
		if err != nil {
			return err
		}
	}
	return nil
}

// Scan does full scan on all the packages
func (collection *PackageCollection) Scan(q PackageQuery) (result *PackageList) {
	result = NewPackageListWithDuplicates(true, 0)

	for _, key := range collection.db.KeysByPrefix([]byte("P")) {
		pkg, err := collection.ByKey(key)
		if err != nil {
			panic(fmt.Sprintf("unable to load package: %s", err))
		}

		if q.Matches(pkg) {
			result.Add(pkg)
		}
	}

	return
}

// Search is not implemented
func (collection *PackageCollection) Search(dep Dependency, allMatches bool) (searchResults []*Package) {
	panic("Not implemented")
}

// SearchSupported returns false
func (collection *PackageCollection) SearchSupported() bool {
	return false
}

// SearchByKey finds package by exact key
func (collection *PackageCollection) SearchByKey(arch, name, version string) (result *PackageList) {
	result = NewPackageListWithDuplicates(true, 0)

	for _, key := range collection.db.KeysByPrefix([]byte(fmt.Sprintf("P%s %s %s", arch, name, version))) {
		pkg, err := collection.ByKey(key)
		if err != nil {
			panic(fmt.Sprintf("unable to load package: %s", err))
		}

		if pkg.Architecture == arch && pkg.Name == name && pkg.Version == version {
			result.Add(pkg)
		}
	}

	return
}
