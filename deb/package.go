package deb

import (
	"fmt"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/utils"
	"path/filepath"
	"strconv"
	"strings"
)

type Hash uint64

// Package is single instance of Debian package
type Package struct {
	// Basic package properties
	Name         string
	Version      string
	Architecture string
	// If this source package, this field holds "real" architecture value,
	// while Architecture would be equal to "source"
	SourceArchitecture string
	// For binary package, name of source package
	Source string
	// List of virtual packages this package provides
	Provides []string
	// Is this source package
	IsSource bool
	// Is this udeb package
	IsUdeb bool
	// Hash of files section
	FilesHash Hash
	// Is this >= 0.6 package?
	V06Plus bool
	// Offload fields
	deps  *PackageDependencies
	extra *Stanza
	files *PackageFiles
	// Mother collection
	collection *PackageCollection
}

func (h *Hash) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%08x\"", *h)), nil
}

// NewPackageFromControlFile creates Package from parsed Debian control file
func NewPackageFromControlFile(input Stanza) *Package {
	result := &Package{
		Name:         input["Package"],
		Version:      input["Version"],
		Architecture: input["Architecture"],
		Source:       input["Source"],
		V06Plus:      true,
	}

	delete(input, "Package")
	delete(input, "Version")
	delete(input, "Architecture")
	delete(input, "Source")

	filesize, _ := strconv.ParseInt(input["Size"], 10, 64)

	md5, ok := input["MD5sum"]
	if !ok {
		// there are some broken repos out there with MD5 in wrong field
		md5 = input["MD5Sum"]
	}

	result.UpdateFiles(PackageFiles{PackageFile{
		Filename:     filepath.Base(input["Filename"]),
		downloadPath: filepath.Dir(input["Filename"]),
		Checksums: utils.ChecksumInfo{
			Size:   filesize,
			MD5:    strings.TrimSpace(md5),
			SHA1:   strings.TrimSpace(input["SHA1"]),
			SHA256: strings.TrimSpace(input["SHA256"]),
		},
	}})

	delete(input, "Filename")
	delete(input, "MD5sum")
	delete(input, "MD5Sum")
	delete(input, "SHA1")
	delete(input, "SHA256")
	delete(input, "Size")

	depends := &PackageDependencies{}
	depends.Depends = parseDependencies(input, "Depends")
	depends.PreDepends = parseDependencies(input, "Pre-Depends")
	depends.Suggests = parseDependencies(input, "Suggests")
	depends.Recommends = parseDependencies(input, "Recommends")
	result.deps = depends

	result.Provides = parseDependencies(input, "Provides")

	result.extra = &input

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
		V06Plus:            true,
	}

	delete(input, "Package")
	delete(input, "Version")
	delete(input, "Architecture")

	files := make(PackageFiles, 0, 3)

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
			for i, file := range files {
				if file.Filename == filename {
					found = true
					pos = i
					break
				}
			}

			if !found {
				files = append(files, PackageFile{Filename: filename, downloadPath: input["Directory"]})
				pos = len(files) - 1
			}

			files[pos].Checksums.Size = size
			setter(&files[pos].Checksums, parts[0])
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

	result.UpdateFiles(files)

	depends := &PackageDependencies{}
	depends.BuildDepends = parseDependencies(input, "Build-Depends")
	depends.BuildDependsInDep = parseDependencies(input, "Build-Depends-Indep")
	result.deps = depends

	result.extra = &input

	return result, nil
}

// NewUdebPackageFromControlFile creates .udeb Package from parsed Debian control file
func NewUdebPackageFromControlFile(input Stanza) *Package {
	p := NewPackageFromControlFile(input)
	p.IsUdeb = true

	return p
}

// Key returns unique key identifying package
func (p *Package) Key(prefix string) []byte {
	if p.V06Plus {
		return []byte(fmt.Sprintf("%sP%s %s %s %08x", prefix, p.Architecture, p.Name, p.Version, p.FilesHash))
	}

	return p.ShortKey(prefix)
}

// ShortKey returns key for the package that should be unique in one list
func (p *Package) ShortKey(prefix string) []byte {
	return []byte(fmt.Sprintf("%sP%s %s %s", prefix, p.Architecture, p.Name, p.Version))
}

// String creates readable representation
func (p *Package) String() string {
	return fmt.Sprintf("%s_%s_%s", p.Name, p.Version, p.Architecture)
}

// GetField returns fields from package
func (p *Package) GetField(name string) string {
	switch name {
	// $Version is handled in FieldQuery
	case "$Source":
		if p.IsSource {
			return ""
		}
		source := p.Source
		if source == "" {
			return p.Name
		} else if pos := strings.Index(source, "("); pos != -1 {
			return strings.TrimSpace(source[:pos])
		}
		return source
	case "$SourceVersion":
		if p.IsSource {
			return ""
		}
		source := p.Source
		if pos := strings.Index(source, "("); pos != -1 {
			if pos2 := strings.LastIndex(source, ")"); pos2 != -1 && pos2 > pos {
				return strings.TrimSpace(source[pos+1 : pos2])
			}
		}
		return p.Version
	case "$Architecture":
		return p.Architecture
	case "$PackageType":
		if p.IsSource {
			return "source"
		}
		if p.IsUdeb {
			return "udeb"
		}
		return "deb"
	case "Name":
		return p.Name
	case "Version":
		return p.Version
	case "Architecture":
		if p.IsSource {
			return p.SourceArchitecture
		}
		return p.Architecture
	case "Source":
		return p.Source
	case "Depends":
		return strings.Join(p.Deps().Depends, ", ")
	case "Pre-Depends":
		return strings.Join(p.Deps().PreDepends, ", ")
	case "Suggests":
		return strings.Join(p.Deps().Suggests, ", ")
	case "Recommends":
		return strings.Join(p.Deps().Recommends, ", ")
	case "Provides":
		return strings.Join(p.Provides, ", ")
	case "Build-Depends":
		return strings.Join(p.Deps().BuildDepends, ", ")
	case "Build-Depends-Indep":
		return strings.Join(p.Deps().BuildDependsInDep, ", ")
	default:
		return p.Extra()[name]
	}
	return ""
}

// MatchesArchitecture checks whether packages matches specified architecture
func (p *Package) MatchesArchitecture(arch string) bool {
	if p.Architecture == "all" && arch != "source" {
		return true
	}

	return p.Architecture == arch
}

// MatchesDependency checks whether package matches specified dependency
func (p *Package) MatchesDependency(dep Dependency) bool {
	if dep.Architecture != "" && !p.MatchesArchitecture(dep.Architecture) {
		return false
	}

	if dep.Relation == VersionDontCare {
		if utils.StrSliceHasItem(p.Provides, dep.Pkg) {
			return true
		}
		return dep.Pkg == p.Name
	}

	if dep.Pkg != p.Name {
		return false
	}

	r := CompareVersions(p.Version, dep.Version)

	switch dep.Relation {
	case VersionEqual:
		return r == 0
	case VersionLess:
		return r < 0
	case VersionGreater:
		return r > 0
	case VersionLessOrEqual:
		return r <= 0
	case VersionGreaterOrEqual:
		return r >= 0
	case VersionPatternMatch:
		matched, err := filepath.Match(dep.Version, p.Version)
		return err == nil && matched
	case VersionRegexp:
		return dep.Regexp.FindStringIndex(p.Version) != nil
	}

	panic("unknown relation")
}

// GetDependencies compiles list of dependenices by flags from options
func (p *Package) GetDependencies(options int) (dependencies []string) {
	deps := p.Deps()

	dependencies = make([]string, 0, 30)
	dependencies = append(dependencies, deps.Depends...)
	dependencies = append(dependencies, deps.PreDepends...)

	if options&DepFollowRecommends == DepFollowRecommends {
		dependencies = append(dependencies, deps.Recommends...)
	}

	if options&DepFollowSuggests == DepFollowSuggests {
		dependencies = append(dependencies, deps.Suggests...)
	}

	if options&DepFollowBuild == DepFollowBuild {
		dependencies = append(dependencies, deps.BuildDepends...)
		dependencies = append(dependencies, deps.BuildDependsInDep...)
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

// Extra returns Stanza of extra fields (it may load it from collection)
func (p *Package) Extra() Stanza {
	if p.extra == nil {
		if p.collection == nil {
			panic("extra == nil && collection == nil")
		}
		p.extra = p.collection.loadExtra(p)
	}

	return *p.extra
}

// Deps returns parsed package dependencies (it may load it from collection)
func (p *Package) Deps() *PackageDependencies {
	if p.deps == nil {
		if p.collection == nil {
			panic("deps == nil && collection == nil")
		}

		p.deps = p.collection.loadDependencies(p)
	}

	return p.deps
}

// Files returns parsed files records (it may load it from collection)
func (p *Package) Files() PackageFiles {
	if p.files == nil {
		if p.collection == nil {
			panic("files == nil && collection == nil")
		}

		p.files = p.collection.loadFiles(p)
	}

	return *p.files
}

// UpdateFiles saves new state of files
func (p *Package) UpdateFiles(files PackageFiles) {
	p.files = &files
	p.FilesHash = Hash(files.Hash())
}

// Stanza creates original stanza from package
func (p *Package) Stanza() (result Stanza) {
	result = p.Extra().Copy()
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

		for _, f := range p.Files() {
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
		f := p.Files()[0]
		result["Filename"] = f.DownloadURL()
		if f.Checksums.MD5 != "" {
			result["MD5sum"] = f.Checksums.MD5
		}
		if f.Checksums.SHA1 != "" {
			result["SHA1"] = " " + f.Checksums.SHA1
		}
		if f.Checksums.SHA256 != "" {
			result["SHA256"] = " " + f.Checksums.SHA256
		}
		result["Size"] = fmt.Sprintf("%d", f.Checksums.Size)
	}

	deps := p.Deps()

	if deps.Depends != nil {
		result["Depends"] = strings.Join(deps.Depends, ", ")
	}
	if deps.PreDepends != nil {
		result["Pre-Depends"] = strings.Join(deps.PreDepends, ", ")
	}
	if deps.Suggests != nil {
		result["Suggests"] = strings.Join(deps.Suggests, ", ")
	}
	if deps.Recommends != nil {
		result["Recommends"] = strings.Join(deps.Recommends, ", ")
	}
	if p.Provides != nil {
		result["Provides"] = strings.Join(p.Provides, ", ")
	}
	if deps.BuildDepends != nil {
		result["Build-Depends"] = strings.Join(deps.BuildDepends, ", ")
	}
	if deps.BuildDependsInDep != nil {
		result["Build-Depends-Indep"] = strings.Join(deps.BuildDependsInDep, ", ")
	}

	return
}

// Equals compares two packages to be identical
func (p *Package) Equals(p2 *Package) bool {
	return p.Name == p2.Name && p.Version == p2.Version && p.SourceArchitecture == p2.SourceArchitecture &&
		p.Architecture == p2.Architecture && p.Source == p2.Source && p.IsSource == p2.IsSource &&
		p.FilesHash == p2.FilesHash
}

// LinkFromPool links package file from pool to dist's pool location
func (p *Package) LinkFromPool(publishedStorage aptly.PublishedStorage, packagePool aptly.PackagePool,
	prefix, component string, force bool) error {
	poolDir, err := p.PoolDirectory()
	if err != nil {
		return err
	}

	for i, f := range p.Files() {
		sourcePath, err := packagePool.Path(f.Filename, f.Checksums.MD5)
		if err != nil {
			return err
		}

		relPath := filepath.Join("pool", component, poolDir)
		publishedDirectory := filepath.Join(prefix, relPath)

		err = publishedStorage.LinkFromPool(publishedDirectory, packagePool, sourcePath, f.Checksums.MD5, force)
		if err != nil {
			return err
		}

		if p.IsSource {
			p.Extra()["Directory"] = relPath
		} else {
			p.Files()[i].downloadPath = relPath
		}
	}

	return nil
}

// PoolDirectory returns directory in package pool of published repository for this package files
func (p *Package) PoolDirectory() (string, error) {
	source := p.Source
	if source == "" {
		source = p.Name
	} else if pos := strings.Index(source, "("); pos != -1 {
		source = strings.TrimSpace(source[:pos])
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

	for _, f := range p.Files() {
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

	for _, f := range p.Files() {
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
	result := make([]string, len(p.Files()))

	for i, f := range p.Files() {
		result[i], err = packagePool.RelativePath(f.Filename, f.Checksums.MD5)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}
