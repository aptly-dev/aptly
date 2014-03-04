package debian

import (
	"bytes"
	"fmt"
	"github.com/smira/aptly/database"
	"github.com/ugorji/go/codec"
	"path/filepath"
)

// PackageCollection does management of packages in DB
type PackageCollection struct {
	db           database.Storage
	encodeBuffer bytes.Buffer
}

// NewPackageCollection creates new PackageCollection and binds it to database
func NewPackageCollection(db database.Storage) *PackageCollection {
	return &PackageCollection{
		db: db,
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

		decoder := codec.NewDecoderBytes(encoded, &codec.MsgpackHandle{})
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
		err = collection.internalUpdate(p)
		if err != nil {
			return nil, err
		}
	} else {
		decoder := codec.NewDecoderBytes(encoded[2:], &codec.MsgpackHandle{})
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

	decoder := codec.NewDecoderBytes(encoded, &codec.MsgpackHandle{})
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

	decoder := codec.NewDecoderBytes(encoded, &codec.MsgpackHandle{})
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

	decoder := codec.NewDecoderBytes(encoded, &codec.MsgpackHandle{})
	err = decoder.Decode(files)
	if err != nil {
		panic("unable to decode files")
	}

	return files
}

// Update adds or updates information about package in DB checking for conficts first
func (collection *PackageCollection) Update(p *Package) error {
	existing, err := collection.ByKey(p.Key(""))
	if err == nil {
		// if .Files is different, consider to be conflict
		if p.FilesHash != existing.FilesHash {
			return fmt.Errorf("unable to save: %s, conflict with existing packge", p)
		}
		// ok, .Files are the same, but maybe some meta-data is different, proceed to saving
	} else {
		if err != database.ErrNotFound {
			return err
		}
		// ok, package doesn't exist yet
	}

	return collection.internalUpdate(p)
}

// internalUpdate updates information in DB about package and offloaded fields
func (collection *PackageCollection) internalUpdate(p *Package) error {
	encoder := codec.NewEncoder(&collection.encodeBuffer, &codec.MsgpackHandle{})

	collection.encodeBuffer.Reset()
	collection.encodeBuffer.WriteByte(0xc1)
	collection.encodeBuffer.WriteByte(0x1)
	err := encoder.Encode(p)
	if err != nil {
		return err
	}

	err = collection.db.Put(p.Key(""), collection.encodeBuffer.Bytes())
	if err != nil {
		return err
	}

	// Encode offloaded fields one by one
	if p.files != nil {
		collection.encodeBuffer.Reset()
		err = encoder.Encode(*p.files)
		if err != nil {
			return err
		}

		err = collection.db.Put(p.Key("xF"), collection.encodeBuffer.Bytes())
		if err != nil {
			return err
		}
	}

	if p.deps != nil {
		collection.encodeBuffer.Reset()
		err = encoder.Encode(*p.deps)
		if err != nil {
			return err
		}

		err = collection.db.Put(p.Key("xD"), collection.encodeBuffer.Bytes())
		if err != nil {
			return err
		}

		p.deps = nil
	}

	if p.extra != nil {
		collection.encodeBuffer.Reset()
		err = encoder.Encode(*p.extra)
		if err != nil {
			return err
		}

		err = collection.db.Put(p.Key("xE"), collection.encodeBuffer.Bytes())
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
func (collection *PackageCollection) DeleteByKey(key []byte) error {
	for _, key := range [][]byte{key, append([]byte("xF"), key...), append([]byte("xD"), key...), append([]byte("xE"), key...)} {
		err := collection.db.Delete(key)
		if err != nil {
			return err
		}
	}
	return nil
}
