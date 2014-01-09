package debian

import (
	"bytes"
	"fmt"
	"github.com/smira/aptly/utils"
	"github.com/ugorji/go/codec"
	"sort"
)

// PackageList is list of unique (by key) packages
//
// It could be seen as repo snapshot, repo contents, result of filtering,
// merge, etc.
type PackageList struct {
	packages map[string]*Package
}

// NewPackageList creates empty package list
func NewPackageList() *PackageList {
	return &PackageList{packages: make(map[string]*Package, 1000)}
}

// NewPackageListFromRefList loads packages list from PackageRefList
func NewPackageListFromRefList(reflist *PackageRefList, collection *PackageCollection) (*PackageList, error) {
	result := &PackageList{packages: make(map[string]*Package, reflist.Len())}

	err := reflist.ForEach(func(key []byte) error {
		p, err := collection.ByKey(key)
		if err != nil {
			return fmt.Errorf("unable to load package with key %s: %s", key, err)
		}
		return result.Add(p)
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// Add appends package to package list, additionally checking for uniqueness
func (l *PackageList) Add(p *Package) error {
	key := string(p.Key())
	existing, ok := l.packages[key]
	if ok {
		if !existing.Equals(p) {
			return fmt.Errorf("conflict in package %s: %#v != %#v", p, existing, p)
		}
		return nil
	}
	l.packages[key] = p
	return nil
}

// ForEach calls handler for each package in list
func (l *PackageList) ForEach(handler func(*Package) error) error {
	var err error
	for _, p := range l.packages {
		err = handler(p)
		if err != nil {
			return err
		}
	}
	return err
}

// Len returns number of packages in the list
func (l *PackageList) Len() int {
	return len(l.packages)
}

// Architectures returns list of architectures present in packages
func (l *PackageList) Architectures() (result []string) {
	result = make([]string, 0, 10)
	for _, pkg := range l.packages {
		if pkg.Architecture != "all" && !utils.StrSliceHasItem(result, pkg.Architecture) {
			result = append(result, pkg.Architecture)
		}
	}
	return
}

// Dependency options
const (
	// DepFollowSource pulls source packages when required
	DepFollowSource = 1 << iota
	// DepFollowSuggests pulls from suggests
	DepFollowSuggests
	// DepFollowRecommends pulls from recommends
	DepFollowRecommends
	// DepFollowAllVariants follows all variants if depends on "a | b"
	DepFollowAllVariants
)

// VerifyDependencies looks for missing dependencies in package list.
//
// Analysis would be peformed for each architecture, in specified sources
func (l *PackageList) VerifyDependencies(options int, architectures []string, sources *PackageIndexedList) ([]Dependency, error) {
	missing := make([]Dependency, 0, 128)

	for _, arch := range architectures {
		cache := make(map[string]bool, 2048)

		for _, p := range l.packages {
			if !p.MatchesArchitecture(arch) {
				continue
			}

			for _, dep := range p.GetDependencies(options) {
				dep, err := parseDependency(dep)
				if err != nil {
					return nil, fmt.Errorf("unable to process package %s: %s", p, err)
				}

				dep.Architecture = arch

				hash := dep.Hash()
				_, ok := cache[hash]
				if ok {
					continue
				}

				if sources.Search(dep) == nil {
					missing = append(missing, dep)
					cache[hash] = false
				} else {
					cache[hash] = true
				}
			}
		}
	}

	return missing, nil
}

// PackageIndexedList is a list of packages optimized for satisfying searches
type PackageIndexedList struct {
	// List of packages, sorted by name internally
	packages []*Package
	// Map of packages for each virtual package
	providesList map[string][]*Package
}

// Verify interface
var (
	_ sort.Interface = &PackageIndexedList{}
)

// NewPackageIndexedList creates empty PackageIndexedList
func NewPackageIndexedList() *PackageIndexedList {
	return &PackageIndexedList{
		packages: make([]*Package, 0, 1024),
	}
}

// Len returns number of refs
func (l *PackageIndexedList) Len() int {
	return len(l.packages)
}

// Swap swaps two refs
func (l *PackageIndexedList) Swap(i, j int) {
	l.packages[i], l.packages[j] = l.packages[j], l.packages[i]
}

// Compare compares two refs in lexographical order
func (l *PackageIndexedList) Less(i, j int) bool {
	return l.packages[i].Name < l.packages[j].Name
}

// PrepareIndex prepares list for indexing
func (l *PackageIndexedList) PrepareIndex() {
	sort.Sort(l)

	l.providesList = make(map[string][]*Package, 128)
	for _, p := range l.packages {
		if p.Provides != "" {
			l.providesList[p.Provides] = append(l.providesList[p.Provides], p)
		}
	}
}

// Append adds more packages to the index
func (l *PackageIndexedList) Append(pl *PackageList) {
	pp := make([]*Package, pl.Len())
	i := 0
	for _, p := range pl.packages {
		pp[i] = p
		i++
	}

	l.packages = append(l.packages, pp...)
}

// Search searches package index for specified package
func (l *PackageIndexedList) Search(dep Dependency) *Package {
	if dep.Relation == VersionDontCare {
		for _, p := range l.providesList[dep.Pkg] {
			if p.MatchesArchitecture(dep.Architecture) {
				return p
			}
		}
	}

	i := sort.Search(len(l.packages), func(j int) bool { return l.packages[j].Name >= dep.Pkg })

	for i < len(l.packages) && l.packages[i].Name == dep.Pkg {
		p := l.packages[i]
		if p.MatchesArchitecture(dep.Architecture) {
			if dep.Relation == VersionDontCare {
				return p
			}

			r := CompareVersions(p.Version, dep.Version)
			switch dep.Relation {
			case VersionEqual:
				if r == 0 {
					return p
				}
			case VersionLess:
				if r < 0 {
					return p
				}
			case VersionGreater:
				if r > 0 {
					return p
				}
			case VersionLessOrEqual:
				if r <= 0 {
					return p
				}
			case VersionGreaterOrEqual:
				if r >= 0 {
					return p
				}
			}
		}
		i++
	}
	return nil
}

// PackageRefList is a list of keys of packages, this is basis for snapshot
// and similar stuff
//
// Refs are sorted in lexicographical order
type PackageRefList struct {
	// List of package keys
	Refs [][]byte
}

// Verify interface
var (
	_ sort.Interface = &PackageRefList{}
)

// NewPackageRefListFromPackageList creates PackageRefList from PackageList
func NewPackageRefListFromPackageList(list *PackageList) *PackageRefList {
	reflist := &PackageRefList{}
	reflist.Refs = make([][]byte, list.Len())

	i := 0
	for _, p := range list.packages {
		reflist.Refs[i] = p.Key()
		i++
	}

	sort.Sort(reflist)

	return reflist
}

// Len returns number of refs
func (l *PackageRefList) Len() int {
	return len(l.Refs)
}

// Swap swaps two refs
func (l *PackageRefList) Swap(i, j int) {
	l.Refs[i], l.Refs[j] = l.Refs[j], l.Refs[i]
}

// Compare compares two refs in lexographical order
func (l *PackageRefList) Less(i, j int) bool {
	return bytes.Compare(l.Refs[i], l.Refs[j]) < 0
}

// Encode does msgpack encoding of PackageRefList
func (l *PackageRefList) Encode() []byte {
	var buf bytes.Buffer

	encoder := codec.NewEncoder(&buf, &codec.MsgpackHandle{})
	encoder.Encode(l)

	return buf.Bytes()
}

// Decode decodes msgpack representation into PackageRefLit
func (l *PackageRefList) Decode(input []byte) error {
	decoder := codec.NewDecoderBytes(input, &codec.MsgpackHandle{})
	return decoder.Decode(l)
}

// ForEach calls handler for each package ref in list
func (l *PackageRefList) ForEach(handler func([]byte) error) error {
	var err error
	for _, p := range l.Refs {
		err = handler(p)
		if err != nil {
			return err
		}
	}
	return err
}
