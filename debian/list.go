package debian

import (
	"fmt"
	"github.com/smira/aptly/utils"
	"sort"
	"strings"
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
	// DepFollowBuild pulls build dependencies
	DepFollowBuild
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
	// empty reflist
	if reflist == nil {
		return NewPackageList(), nil
	}

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
	key := string(p.Key(""))
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
	delete(l.packages, string(p.Key("")))
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

// Architectures returns list of architectures present in packages and flag if source packages are present.
//
// If includeSource is true, meta-architecture "source" would be present in the list
func (l *PackageList) Architectures(includeSource bool) (result []string) {
	result = make([]string, 0, 10)
	for _, pkg := range l.packages {
		if pkg.Architecture != "all" && (pkg.Architecture != "source" || includeSource) && !utils.StrSliceHasItem(result, pkg.Architecture) {
			result = append(result, pkg.Architecture)
		}
	}
	return
}

// depSliceDeduplicate removes dups in slice of Dependencies
func depSliceDeduplicate(s []Dependency) []Dependency {
	l := len(s)
	if l < 2 {
		return s
	}
	if l == 2 {
		if s[0] == s[1] {
			return s[0:1]
		}
		return s
	}

	found := make(map[string]bool, l)
	j := 0
	for i, x := range s {
		h := x.Hash()
		if !found[h] {
			found[h] = true
			s[j] = s[i]
			j++
		}
	}

	return s[:j]
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

				variants = depSliceDeduplicate(variants)

				variantsMissing := make([]Dependency, 0, len(variants))
				missingCount := 0

				for _, dep := range variants {
					if dep.Architecture == "" {
						dep.Architecture = arch
					}

					hash := dep.Hash()
					r, ok := cache[hash]
					if ok {
						if !r {
							missingCount++
						}
						continue
					}

					if sources.Search(dep) == nil {
						variantsMissing = append(variantsMissing, dep)
						missingCount++
					} else {
						cache[hash] = true
					}
				}

				if options&DepFollowAllVariants == DepFollowAllVariants {
					missing = append(missing, variantsMissing...)
					for _, dep := range variantsMissing {
						cache[dep.Hash()] = false
					}
				} else {
					if missingCount == len(variants) {
						missing = append(missing, variantsMissing...)
						for _, dep := range variantsMissing {
							cache[dep.Hash()] = false
						}
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
		if p.MatchesDependency(dep) {
			return p
		}

		i++
	}
	return nil
}

// Filter filters package index by specified queries (ORed together), possibly pulling dependencies
func (l *PackageList) Filter(queries []string, withDependencies bool, source *PackageList, dependencyOptions int, architecturesList []string) (*PackageList, error) {
	if !l.indexed {
		panic("list not indexed, can't filter")
	}

	result := NewPackageList()

	for _, query := range queries {
		isDepQuery := strings.IndexAny(query, " (){}=<>") != -1

		if !isDepQuery {
			// try to interpret query as package string representation

			// convert Package.String() to Package.Key()
			i := strings.Index(query, "_")
			if i != -1 {
				pkg, query := query[:i], query[i+1:]
				j := strings.LastIndex(query, "_")
				if j != -1 {
					version, arch := query[:j], query[j+1:]
					p := l.packages["P"+arch+" "+pkg+" "+version]
					if p != nil {
						result.Add(p)
						continue
					}
				}
			}
		}

		// try as dependency
		dep, err := ParseDependency(query)
		if err != nil {
			if isDepQuery {
				return nil, err
			}
			// parsing failed, but probably that wasn't a dep query
			continue
		}

		i := sort.Search(len(l.packagesIndex), func(j int) bool { return l.packagesIndex[j].Name >= dep.Pkg })

		for i < len(l.packagesIndex) && l.packagesIndex[i].Name == dep.Pkg {
			p := l.packagesIndex[i]
			if p.MatchesDependency(dep) {
				result.Add(p)
			}
			i++
		}
	}

	if withDependencies {
		added := result.Len()

		dependencySource := NewPackageList()
		dependencySource.Append(source)
		dependencySource.Append(result)
		dependencySource.PrepareIndex()

		// while some new dependencies were discovered
		for added > 0 {
			added = 0

			// find missing dependencies
			missing, err := result.VerifyDependencies(dependencyOptions, architecturesList, dependencySource)
			if err != nil {
				return nil, err
			}

			// try to satisfy dependencies
			for _, dep := range missing {
				p := l.Search(dep)
				if p != nil {
					result.Add(p)
					dependencySource.Add(p)
					added++
				}
			}
		}
	}

	return result, nil
}
