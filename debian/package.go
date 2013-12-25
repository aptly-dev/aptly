package debian

import (
	"bytes"
	"fmt"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/utils"
	"github.com/ugorji/go/codec"
	"os"
	"strconv"
	"strings"
)

// Package is single instance of Debian package
//
// TODO: support source & binary
type Package struct {
	Name         string
	Version      string
	Filename     string
	Filesize     int64
	Architecture string
	Source       string
	// Various dependencies
	Depends    []string
	PreDepends []string
	Suggests   []string
	Recommends []string
	// Hashsums of package contents
	HashMD5    string
	HashSHA1   string
	HashSHA256 string
	// Extra information from stanza
	Extra Stanza
}

func parseDependencies(input Stanza, key string) []string {
	value, ok := input[key]
	if !ok {
		return nil
	}

	delete(input, key)

	return strings.Split(value, ", ")
}

// NewPackageFromControlFile creates Package from parsed Debian control file
func NewPackageFromControlFile(input Stanza) *Package {
	result := &Package{
		Name:         input["Package"],
		Version:      input["Version"],
		Filename:     input["Filename"],
		Architecture: input["Architecture"],
		Source:       input["Source"],
		HashMD5:      input["MD5sum"],
		HashSHA1:     input["SHA1"],
		HashSHA256:   input["SHA256"],
	}

	delete(input, "Package")
	delete(input, "Version")
	delete(input, "Filename")
	delete(input, "Architecture")
	delete(input, "Source")
	delete(input, "MD5sum")
	delete(input, "SHA1")
	delete(input, "SHA256")

	result.Filesize, _ = strconv.ParseInt(input["Size"], 10, 64)
	delete(input, "Size")

	result.Depends = parseDependencies(input, "Depends")
	result.PreDepends = parseDependencies(input, "Pre-Depends")
	result.Suggests = parseDependencies(input, "Suggests")
	result.Recommends = parseDependencies(input, "Recommends")

	result.Extra = input

	return result
}

// Key returns unique key identifying package
func (p *Package) Key() []byte {
	return []byte("P" + p.Name + " " + p.Version + " " + p.Architecture)
}

// Encode does msgpack encoding of Package
func (p *Package) Encode() []byte {
	var buf bytes.Buffer

	encoder := codec.NewEncoder(&buf, &codec.MsgpackHandle{})
	encoder.Encode(p)

	return buf.Bytes()
}

// Decode decodes msgpack representation into Package
func (p *Package) Decode(input []byte) error {
	decoder := codec.NewDecoderBytes(input, &codec.MsgpackHandle{})
	return decoder.Decode(p)
}

// String creates readable representation
func (p *Package) String() string {
	return fmt.Sprintf("%s-%s_%s", p.Name, p.Version, p.Architecture)
}

// Stanza creates original stanza from package
func (p *Package) Stanza() (result Stanza) {
	result = p.Extra.Copy()
	result["Package"] = p.Name
	result["Version"] = p.Version
	result["Filename"] = p.Filename
	result["Architecture"] = p.Architecture
	result["Source"] = p.Source

	if p.HashMD5 != "" {
		result["MD5sum"] = p.HashMD5
	}
	if p.HashSHA1 != "" {
		result["SHA1"] = p.HashSHA1
	}
	if p.HashSHA256 != "" {
		result["SHA256"] = p.HashSHA256
	}

	if p.Depends != nil {
		result["Depends"] = strings.Join(p.Depends, ", ")
	}
	if p.PreDepends != nil {
		result["Pre-Depends"] = strings.Join(p.PreDepends, ", ")
	}
	if p.Suggests != nil {
		result["Suggests"] = strings.Join(p.Suggests, ", ")
	}
	if p.Recommends != nil {
		result["Recommends"] = strings.Join(p.Recommends, ", ")
	}

	result["Size"] = fmt.Sprintf("%d", p.Filesize)

	return
}

// Equals compares two packages to be identical
func (p *Package) Equals(p2 *Package) bool {
	return p.Name == p2.Name && p.Version == p2.Version && p.Filename == p2.Filename &&
		p.Architecture == p2.Architecture && utils.StrSlicesEqual(p.Depends, p2.Depends) &&
		utils.StrSlicesEqual(p.PreDepends, p2.PreDepends) && utils.StrSlicesEqual(p.Suggests, p2.Suggests) &&
		utils.StrSlicesEqual(p.Recommends, p2.Recommends) && utils.StrMapsEqual(p.Extra, p2.Extra) &&
		p.Filesize == p2.Filesize && p.HashMD5 == p2.HashMD5 && p.HashSHA1 == p2.HashSHA1 &&
		p.HashSHA256 == p2.HashSHA256 && p.Source == p2.Source
}

// VerifyFile verifies integrity and existence of local files for the package
func (p *Package) VerifyFile(filepath string) bool {
	st, err := os.Stat(filepath)
	if err != nil {
		return false
	}
	return st.Size() == p.Filesize
}

// PackageCollection does management of packages in DB
type PackageCollection struct {
	db database.Storage
}

// NewPackageCollection creates new PackageCollection and binds it to database
func NewPackageCollection(db database.Storage) *PackageCollection {
	return &PackageCollection{
		db: db,
	}
}

// ByKey find package in DB by its key
func (collection *PackageCollection) ByKey(key []byte) (*Package, error) {
	encoded, err := collection.db.Get(key)
	if err != nil {
		return nil, err
	}

	p := &Package{}
	err = p.Decode(encoded)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Update adds or updates information about package in DB
func (collection *PackageCollection) Update(p *Package) error {
	return collection.db.Put(p.Key(), p.Encode())
}
