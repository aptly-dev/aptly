package debian

import (
	"bytes"
	"fmt"
	"github.com/smira/aptly/aptly"
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
	// Filename is name of file for the package (without directory)
	Filename string
	// Hashes for the file
	Checksums utils.ChecksumInfo
	// Temporary field used while downloading, stored relative path on the mirror
	downloadPath string
}

// Verify that package file is present and correct
func (f *PackageFile) Verify(packagePool aptly.PackagePool) (bool, error) {
	poolPath, err := packagePool.Path(f.Filename, f.Checksums.MD5)
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

// DownloadURL return relative URL to package download location
func (f *PackageFile) DownloadURL() string {
	return filepath.Join(f.downloadPath, f.Filename)
}

// Package is single instance of Debian package
type Package struct {
	// Is this source package
	IsSource bool
	// Basic package properties
	Name         string
	Version      string
	Architecture string
	// If this source package, this field holds "real" architecture value,
	// while Architecture would be equal to "source"
	SourceArchitecture string
	// For binary package, name of source package
	Source string
	// Various dependencies
	Provides          []string
	Depends           []string
	BuildDepends      []string
	BuildDependsInDep []string
	PreDepends        []string
	Suggests          []string
	Recommends        []string
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
		Filename:     filepath.Base(input["Filename"]),
		downloadPath: filepath.Dir(input["Filename"]),
		Checksums: utils.ChecksumInfo{
			Size:   filesize,
			MD5:    strings.TrimSpace(input["MD5sum"]),
			SHA1:   strings.TrimSpace(input["SHA1"]),
			SHA256: strings.TrimSpace(input["SHA256"]),
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

// NewSourcePackageFromControlFile creates Package from parsed Debian control file for source package
func NewSourcePackageFromControlFile(input Stanza) (*Package, error) {
	result := &Package{
		IsSource:           true,
		Name:               input["Package"],
		Version:            input["Version"],
		Architecture:       "source",
		SourceArchitecture: input["Architecture"],
	}

	delete(input, "Package")
	delete(input, "Version")
	delete(input, "Architecture")

	parseSums := func(field string, setter func(sum *utils.ChecksumInfo, data string)) error {
		for _, line := range strings.Split(input[field], "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.Fields(line)

			if len(parts) != 3 {
				return fmt.Errorf("unparseable hash sum line: %#v", line)
			}

			size, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return fmt.Errorf("unable to parse size: %s", err)
			}

			filename := filepath.Base(parts[2])

			found := false
			pos := 0
			for i, file := range result.Files {
				if file.Filename == filename {
					found = true
					pos = i
					break
				}
			}

			if !found {
				result.Files = append(result.Files, PackageFile{Filename: filename, downloadPath: input["Directory"]})
				pos = len(result.Files) - 1
			}

			result.Files[pos].Checksums.Size = size
			setter(&result.Files[pos].Checksums, parts[0])
		}

		delete(input, field)

		return nil
	}

	err := parseSums("Files", func(sum *utils.ChecksumInfo, data string) { sum.MD5 = data })
	if err != nil {
		return nil, err
	}
	err = parseSums("Checksums-Sha1", func(sum *utils.ChecksumInfo, data string) { sum.SHA1 = data })
	if err != nil {
		return nil, err
	}
	err = parseSums("Checksums-Sha256", func(sum *utils.ChecksumInfo, data string) { sum.SHA256 = data })
	if err != nil {
		return nil, err
	}

	result.BuildDepends = parseDependencies(input, "Build-Depends")
	result.BuildDependsInDep = parseDependencies(input, "Build-Depends-Indep")

	result.Extra = input

	return result, nil
}

// Key returns unique key identifying package
func (p *Package) Key() []byte {
	return []byte("P" + p.Architecture + " " + p.Name + " " + p.Version)
}

// Internal buffer reused by all Package.Encode operations
var encodeBuf bytes.Buffer

// Encode does msgpack encoding of Package, []byte should be copied, as buffer would
// be used for the next call to Encode
func (p *Package) Encode() []byte {
	encodeBuf.Reset()

	encoder := codec.NewEncoder(&encodeBuf, &codec.MsgpackHandle{})
	encoder.Encode(p)

	return encodeBuf.Bytes()
}

// Decode decodes msgpack representation into Package
func (p *Package) Decode(input []byte) error {
	decoder := codec.NewDecoderBytes(input, &codec.MsgpackHandle{})
	err := decoder.Decode(p)
	if err != nil {
		return err
	}

	for i := range p.Files {
		p.Files[i].Filename = filepath.Base(p.Files[i].Filename)
	}

	return nil
}

// String creates readable representation
func (p *Package) String() string {
	return fmt.Sprintf("%s-%s_%s", p.Name, p.Version, p.Architecture)
}

// MatchesArchitecture checks whether packages matches specified architecture
func (p *Package) MatchesArchitecture(arch string) bool {
	if p.Architecture == "all" && arch != "source" {
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

	if options&DepFollowBuild == DepFollowBuild {
		dependencies = append(dependencies, p.BuildDepends...)
		dependencies = append(dependencies, p.BuildDependsInDep...)
	}

	if options&DepFollowSource == DepFollowSource {
		source := p.Source
		if source == "" {
			source = p.Name
		}
		if strings.Index(source, ")") != -1 {
			dependencies = append(dependencies, fmt.Sprintf("%s {source}", source))
		} else {
			dependencies = append(dependencies, fmt.Sprintf("%s (= %s) {source}", source, p.Version))
		}
	}

	return
}

// Stanza creates original stanza from package
func (p *Package) Stanza() (result Stanza) {
	result = p.Extra.Copy()
	result["Package"] = p.Name
	result["Version"] = p.Version

	if p.IsSource {
		result["Architecture"] = p.SourceArchitecture
	} else {
		result["Architecture"] = p.Architecture
		result["Source"] = p.Source
	}

	if p.IsSource {
		md5, sha1, sha256 := make([]string, 0), make([]string, 0), make([]string, 0)

		for _, f := range p.Files {
			if f.Checksums.MD5 != "" {
				md5 = append(md5, fmt.Sprintf(" %s %d %s\n", f.Checksums.MD5, f.Checksums.Size, f.Filename))
			}
			if f.Checksums.SHA1 != "" {
				sha1 = append(sha1, fmt.Sprintf(" %s %d %s\n", f.Checksums.SHA1, f.Checksums.Size, f.Filename))
			}
			if f.Checksums.SHA256 != "" {
				sha256 = append(sha256, fmt.Sprintf(" %s %d %s\n", f.Checksums.SHA256, f.Checksums.Size, f.Filename))
			}
		}

		result["Files"] = strings.Join(md5, "")
		result["Checksums-Sha1"] = strings.Join(sha1, "")
		result["Checksums-Sha256"] = strings.Join(sha256, "")
	} else {
		result["Filename"] = p.Files[0].DownloadURL()
		if p.Files[0].Checksums.MD5 != "" {
			result["MD5sum"] = p.Files[0].Checksums.MD5
		}
		if p.Files[0].Checksums.SHA1 != "" {
			result["SHA1"] = " " + p.Files[0].Checksums.SHA1
		}
		if p.Files[0].Checksums.SHA256 != "" {
			result["SHA256"] = " " + p.Files[0].Checksums.SHA256
		}
		result["Size"] = fmt.Sprintf("%d", p.Files[0].Checksums.Size)
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
	if p.BuildDepends != nil {
		result["Build-Depends"] = strings.Join(p.BuildDepends, ", ")
	}
	if p.BuildDependsInDep != nil {
		result["Build-Depends-Indep"] = strings.Join(p.BuildDependsInDep, ", ")
	}

	return
}

// Equals compares two packages to be identical
func (p *Package) Equals(p2 *Package) bool {
	if len(p.Files) != len(p2.Files) {
		return false
	}

	for _, f := range p.Files {
		found := false
		for _, f2 := range p2.Files {
			if f2.Filename == f.Filename {
				found = true
				if f.Checksums != f2.Checksums {
					return false
				} else {
					break
				}
			}

		}
		if !found {
			return false
		}
	}

	return p.Name == p2.Name && p.Version == p2.Version && p.SourceArchitecture == p2.SourceArchitecture &&
		p.Architecture == p2.Architecture && utils.StrSlicesEqual(p.Depends, p2.Depends) &&
		utils.StrSlicesEqual(p.PreDepends, p2.PreDepends) && utils.StrSlicesEqual(p.Suggests, p2.Suggests) &&
		utils.StrSlicesEqual(p.Recommends, p2.Recommends) && utils.StrMapsEqual(p.Extra, p2.Extra) &&
		p.Source == p2.Source && utils.StrSlicesEqual(p.Provides, p2.Provides) && utils.StrSlicesEqual(p.BuildDepends, p2.BuildDepends) &&
		utils.StrSlicesEqual(p.BuildDependsInDep, p2.BuildDependsInDep) && p.IsSource == p2.IsSource
}

// LinkFromPool links package file from pool to dist's pool location
func (p *Package) LinkFromPool(publishedStorage aptly.PublishedStorage, packagePool aptly.PackagePool, prefix string, component string) error {
	poolDir, err := p.PoolDirectory()
	if err != nil {
		return err
	}

	for i, f := range p.Files {
		sourcePath, err := packagePool.Path(f.Filename, f.Checksums.MD5)
		if err != nil {
			return err
		}

		relPath, err := publishedStorage.LinkFromPool(prefix, component, poolDir, packagePool, sourcePath)
		if err != nil {
			return err
		}

		dir := filepath.Dir(relPath)
		if p.IsSource {
			p.Extra["Directory"] = dir
		} else {
			p.Files[i].downloadPath = dir
		}
	}

	return nil
}

// PoolDirectory returns directory in package pool of published repository for this package files
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

// PackageDownloadTask is a element of download queue for the package
type PackageDownloadTask struct {
	RepoURI         string
	DestinationPath string
	Checksums       utils.ChecksumInfo
}

// DownloadList returns list of missing package files for download in format
// [[srcpath, dstpath]]
func (p *Package) DownloadList(packagePool aptly.PackagePool) (result []PackageDownloadTask, err error) {
	result = make([]PackageDownloadTask, 0, 1)

	for _, f := range p.Files {
		poolPath, err := packagePool.Path(f.Filename, f.Checksums.MD5)
		if err != nil {
			return nil, err
		}

		verified, err := f.Verify(packagePool)
		if err != nil {
			return nil, err
		}

		if !verified {
			result = append(result, PackageDownloadTask{RepoURI: f.DownloadURL(), DestinationPath: poolPath, Checksums: f.Checksums})
		}
	}

	return result, nil
}

// VerifyFiles verifies that all package files have neen correctly downloaded
func (p *Package) VerifyFiles(packagePool aptly.PackagePool) (result bool, err error) {
	result = true

	for _, f := range p.Files {
		result, err = f.Verify(packagePool)
		if err != nil || !result {
			return
		}
	}

	return
}

// FilepathList returns list of paths to files in package repository
func (p *Package) FilepathList(packagePool aptly.PackagePool) ([]string, error) {
	var err error
	result := make([]string, len(p.Files))

	for i, f := range p.Files {
		result[i], err = packagePool.RelativePath(f.Filename, f.Checksums.MD5)
		if err != nil {
			return nil, err
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
	existing, err := collection.ByKey(p.Key())
	if err == nil {
		// check for conflict
		if existing.Equals(p) {
			// packages are the same, no need to update
			return nil
		}

		// if .Files is different, consider to be conflict
		if len(p.Files) != len(existing.Files) {
			return fmt.Errorf("unable to save: %s, conflict with existing packge", p)
		}

		for _, f := range p.Files {
			found := false
			for _, f2 := range existing.Files {
				if f2.Filename == f.Filename {
					found = true
					if f.Checksums != f2.Checksums {
						return fmt.Errorf("unable to save: %s, conflict with existing packge", p)
					} else {
						break
					}
				}

			}
			if !found {
				return fmt.Errorf("unable to save: %s, conflict with existing packge", p)
			}
		}

		// ok, .Files are the same, but some meta-data is different, proceed to saving
	} else {
		if err != database.ErrNotFound {
			return err
		}
		// ok, package doesn't exist yet
	}

	return collection.db.Put(p.Key(), p.Encode())
}

// AllPackageRefs returns list of all packages as PackageRefList
func (collection *PackageCollection) AllPackageRefs() *PackageRefList {
	return &PackageRefList{Refs: collection.db.KeysByPrefix([]byte("P"))}
}

// DeleteByKey deletes package in DB by key
func (collection *PackageCollection) DeleteByKey(key []byte) error {
	return collection.db.Delete(key)
}
