package debian

import (
	"bytes"
	"fmt"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/utils"
	"github.com/ugorji/go/codec"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// PackageFile is a single file entry in package
type PackageFile struct {
	Filename  string
	Checksums utils.ChecksumInfo
}

// Verify that package file is present and correct
func (f *PackageFile) Verify(packageRepo *Repository) (bool, error) {
	poolPath, err := packageRepo.PoolPath(f.Filename, f.Checksums.MD5)
	if err != nil {
		return false, err
	}

	st, err := os.Stat(poolPath)
	if err != nil {
		return false, nil
	}

	// verify size
	// TODO: verify checksum if configured
	return st.Size() == f.Checksums.Size, nil
}

// Package is single instance of Debian package
type Package struct {
	Name         string
	Version      string
	Architecture string
	Source       string
	Provides     []string
	// Various dependencies
	Depends    []string
	PreDepends []string
	Suggests   []string
	Recommends []string
	// Files in package
	Files []PackageFile
	// Extra information from stanza
	Extra Stanza
}

func parseDependencies(input Stanza, key string) []string {
	value, ok := input[key]
	if !ok {
		return nil
	}

	delete(input, key)

	result := strings.Split(value, ",")
	for i := range result {
		result[i] = strings.TrimSpace(result[i])
	}
	return result
}

// NewPackageFromControlFile creates Package from parsed Debian control file
func NewPackageFromControlFile(input Stanza) *Package {
	result := &Package{
		Name:         input["Package"],
		Version:      input["Version"],
		Architecture: input["Architecture"],
		Source:       input["Source"],
		Files:        make([]PackageFile, 0, 1),
	}

	delete(input, "Package")
	delete(input, "Version")
	delete(input, "Architecture")
	delete(input, "Source")

	filesize, _ := strconv.ParseInt(input["Size"], 10, 64)

	result.Files = append(result.Files, PackageFile{
		Filename: input["Filename"],
		Checksums: utils.ChecksumInfo{
			Size:   filesize,
			MD5:    input["MD5sum"],
			SHA1:   input["SHA1"],
			SHA256: input["SHA256"],
		},
	})

	delete(input, "Filename")
	delete(input, "MD5sum")
	delete(input, "SHA1")
	delete(input, "SHA256")
	delete(input, "Size")

	result.Depends = parseDependencies(input, "Depends")
	result.PreDepends = parseDependencies(input, "Pre-Depends")
	result.Suggests = parseDependencies(input, "Suggests")
	result.Recommends = parseDependencies(input, "Recommends")
	result.Provides = parseDependencies(input, "Provides")

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

// MatchesArchitecture checks whether packages matches specified architecture
func (p *Package) MatchesArchitecture(arch string) bool {
	if p.Architecture == "all" {
		return true
	}

	return p.Architecture == arch
}

// GetDependencies compiles list of dependenices by flags from options
func (p *Package) GetDependencies(options int) (dependencies []string) {
	dependencies = make([]string, 0, 30)
	dependencies = append(dependencies, p.Depends...)
	dependencies = append(dependencies, p.PreDepends...)

	if options&DepFollowRecommends == DepFollowRecommends {
		dependencies = append(dependencies, p.Recommends...)
	}

	if options&DepFollowSuggests == DepFollowSuggests {
		dependencies = append(dependencies, p.Suggests...)
	}

	return
}

// Stanza creates original stanza from package
func (p *Package) Stanza() (result Stanza) {
	result = p.Extra.Copy()
	result["Package"] = p.Name
	result["Version"] = p.Version
	result["Filename"] = p.Files[0].Filename
	result["Architecture"] = p.Architecture
	result["Source"] = p.Source

	if p.Files[0].Checksums.MD5 != "" {
		result["MD5sum"] = p.Files[0].Checksums.MD5
	}
	if p.Files[0].Checksums.SHA1 != "" {
		result["SHA1"] = p.Files[0].Checksums.SHA1
	}
	if p.Files[0].Checksums.SHA256 != "" {
		result["SHA256"] = p.Files[0].Checksums.SHA256
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
	if p.Provides != nil {
		result["Provides"] = strings.Join(p.Provides, ", ")
	}

	result["Size"] = fmt.Sprintf("%d", p.Files[0].Checksums.Size)

	return
}

// Equals compares two packages to be identical
func (p *Package) Equals(p2 *Package) bool {
	if len(p.Files) != len(p2.Files) {
		return false
	}

	for i, f := range p.Files {
		if p2.Files[i] != f {
			return false
		}
	}

	return p.Name == p2.Name && p.Version == p2.Version &&
		p.Architecture == p2.Architecture && utils.StrSlicesEqual(p.Depends, p2.Depends) &&
		utils.StrSlicesEqual(p.PreDepends, p2.PreDepends) && utils.StrSlicesEqual(p.Suggests, p2.Suggests) &&
		utils.StrSlicesEqual(p.Recommends, p2.Recommends) && utils.StrMapsEqual(p.Extra, p2.Extra) &&
		p.Source == p2.Source && utils.StrSlicesEqual(p.Provides, p2.Provides)
}

// LinkFromPool links package file from pool to dist's pool location
func (p *Package) LinkFromPool(packageRepo *Repository, prefix string, component string) error {
	poolDir, err := p.PoolDirectory()
	if err != nil {
		return err
	}

	for i, f := range p.Files {
		sourcePath, err := packageRepo.PoolPath(f.Filename, f.Checksums.MD5)
		if err != nil {
			return err
		}

		relPath, err := packageRepo.LinkFromPool(prefix, component, sourcePath, poolDir)
		if err != nil {
			return err
		}

		p.Files[i].Filename = relPath
	}

	return nil
}

// PoolDirectory returns directory in package pool for this package files
func (p *Package) PoolDirectory() (string, error) {
	source := p.Source
	if source == "" {
		source = p.Name
	}

	if len(source) < 2 {
		return "", fmt.Errorf("package source %s too short", source)
	}

	var subdir string
	if strings.HasPrefix(source, "lib") {
		subdir = source[:4]
	} else {
		subdir = source[:1]

	}

	return filepath.Join(subdir, source), nil
}

// DownloadList returns list of missing package files for download in format
// [[srcpath, dstpath]]
func (p *Package) DownloadList(packageRepo *Repository) (result [][]string, err error) {
	result = make([][]string, 0, 1)

	for _, f := range p.Files {
		poolPath, err := packageRepo.PoolPath(f.Filename, f.Checksums.MD5)
		if err != nil {
			return nil, err
		}

		verified, err := f.Verify(packageRepo)
		if err != nil {
			return nil, err
		}

		if !verified {
			result = append(result, []string{f.Filename, poolPath})
		}
	}

	return result, nil
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
