package debian

import (
	"bytes"
	"fmt"
	"github.com/smira/aptly/utils"
	"github.com/ugorji/go/codec"
	"sort"
)

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

// PackageList is list of unique (by key) packages
//
// It could be seen as repo snapshot, repo contents, result of filtering,
// merge, etc.
//
// If indexed, PackageList starts supporting searching
type PackageList struct {
	// Straight list of packages as map
	packages map[string]*Package
	// Has index been prepared?
	indexed bool
	// Indexed list of packages, sorted by name internally
	packagesIndex []*Package
	// Map of packages for each virtual package (provides)
	providesIndex map[string][]*Package
}

// Verify interface
var (
	_ sort.Interface = &PackageList{}
)

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

	if l.indexed {
		for _, provides := range p.Provides {
			l.providesIndex[provides] = append(l.providesIndex[provides], p)
		}

		i := sort.Search(len(l.packagesIndex), func(j int) bool { return l.packagesIndex[j].Name >= p.Name })

		// insert p into l.packagesIndex in position i
		l.packagesIndex = append(l.packagesIndex, nil)
		copy(l.packagesIndex[i+1:], l.packagesIndex[i:])
		l.packagesIndex[i] = p
	}
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

// Append adds content from one package list to another
func (l *PackageList) Append(pl *PackageList) error {
	if l.indexed {
		panic("Append not supported when indexed")
	}
	for k, p := range pl.packages {
		existing, ok := l.packages[k]
		if ok {
			if !existing.Equals(p) {
				return fmt.Errorf("conflict in package %s: %#v != %#v", p, existing, p)
			}
		} else {
			l.packages[k] = p
		}
	}

	return nil
}

// Remove removes package from the list, and updates index when required
func (l *PackageList) Remove(p *Package) {
	delete(l.packages, string(p.Key()))
	if l.indexed {
		for _, provides := range p.Provides {
			for i, pkg := range l.providesIndex[provides] {
				if pkg.Equals(p) {
					// remove l.ProvidesIndex[provides][i] w/o preserving order
					l.providesIndex[provides][len(l.providesIndex[provides])-1], l.providesIndex[provides][i], l.providesIndex[provides] =
						nil, l.providesIndex[provides][len(l.providesIndex[provides])-1], l.providesIndex[provides][:len(l.providesIndex[provides])-1]
					break
				}
			}
		}

		i := sort.Search(len(l.packagesIndex), func(j int) bool { return l.packagesIndex[j].Name >= p.Name })
		for i < len(l.packagesIndex) && l.packagesIndex[i].Name == p.Name {
			if l.packagesIndex[i].Equals(p) {
				// remove l.packagesIndex[i] preserving order
				copy(l.packagesIndex[i:], l.packagesIndex[i+1:])
				l.packagesIndex[len(l.packagesIndex)-1] = nil
				l.packagesIndex = l.packagesIndex[:len(l.packagesIndex)-1]
				break
			}
			i++
		}
	}
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

// VerifyDependencies looks for missing dependencies in package list.
//
// Analysis would be peformed for each architecture, in specified sources
func (l *PackageList) VerifyDependencies(options int, architectures []string, sources *PackageList) ([]Dependency, error) {
	missing := make([]Dependency, 0, 128)

	for _, arch := range architectures {
		cache := make(map[string]bool, 2048)

		for _, p := range l.packages {
			if !p.MatchesArchitecture(arch) {
				continue
			}

			for _, dep := range p.GetDependencies(options) {
				variants, err := ParseDependencyVariants(dep)
				if err != nil {
					return nil, fmt.Errorf("unable to process package %s: %s", p, err)
				}

				variantsMissing := make([]Dependency, 0, len(variants))

				for _, dep := range variants {
					dep.Architecture = arch

					hash := dep.Hash()
					_, ok := cache[hash]
					if ok {
						continue
					}

					if sources.Search(dep) == nil {
						variantsMissing = append(variantsMissing, dep)
						cache[hash] = false
					} else {
						cache[hash] = true
					}
				}

				if options&DepFollowAllVariants == DepFollowAllVariants {
					missing = append(missing, variantsMissing...)
				} else {
					if len(variantsMissing) == len(variants) {
						missing = append(missing, variantsMissing...)
					}
				}
			}
		}
	}

	return missing, nil
}

// Swap swaps two packages in index
func (l *PackageList) Swap(i, j int) {
	l.packagesIndex[i], l.packagesIndex[j] = l.packagesIndex[j], l.packagesIndex[i]
}

// Compare compares two names in lexographical order
func (l *PackageList) Less(i, j int) bool {
	return l.packagesIndex[i].Name < l.packagesIndex[j].Name
}

// PrepareIndex prepares list for indexing
func (l *PackageList) PrepareIndex() {
	l.packagesIndex = make([]*Package, l.Len())
	l.providesIndex = make(map[string][]*Package, 128)

	i := 0
	for _, p := range l.packages {
		l.packagesIndex[i] = p
		i++

		for _, provides := range p.Provides {
			l.providesIndex[provides] = append(l.providesIndex[provides], p)
		}
	}

	sort.Sort(l)

	l.indexed = true
}

// Search searches package index for specified package
func (l *PackageList) Search(dep Dependency) *Package {
	if !l.indexed {
		panic("list not indexed, can't search")
	}

	if dep.Relation == VersionDontCare {
		for _, p := range l.providesIndex[dep.Pkg] {
			if p.MatchesArchitecture(dep.Architecture) {
				return p
			}
		}
	}

	i := sort.Search(len(l.packagesIndex), func(j int) bool { return l.packagesIndex[j].Name >= dep.Pkg })

	for i < len(l.packagesIndex) && l.packagesIndex[i].Name == dep.Pkg {
		p := l.packagesIndex[i]
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
