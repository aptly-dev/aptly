package deb

import (
	"fmt"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/utils"
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
func NewPackageListFromRefList(reflist *PackageRefList, collection *PackageCollection, progress aptly.Progress) (*PackageList, error) {
	// empty reflist
	if reflist == nil {
		return NewPackageList(), nil
	}

	result := &PackageList{packages: make(map[string]*Package, reflist.Len())}

	if progress != nil {
		progress.InitBar(int64(reflist.Len()), false)
	}

	err := reflist.ForEach(func(key []byte) error {
		p, err2 := collection.ByKey(key)
		if err2 != nil {
			return fmt.Errorf("unable to load package with key %s: %s", key, err2)
		}
		if progress != nil {
			progress.AddBar(1)
		}
		return result.Add(p)
	})

	if progress != nil {
		progress.ShutdownBar()
	}

	if err != nil {
		return nil, err
	}

	return result, nil
}

// Add appends package to package list, additionally checking for uniqueness
func (l *PackageList) Add(p *Package) error {
	key := string(p.ShortKey(""))
	existing, ok := l.packages[key]
	if ok {
		if !existing.Equals(p) {
			return fmt.Errorf("conflict in package %s", p)
		}
		return nil
	}
	l.packages[key] = p

	if l.indexed {
		for _, provides := range p.Provides {
			l.providesIndex[provides] = append(l.providesIndex[provides], p)
		}

		i := sort.Search(len(l.packagesIndex), func(j int) bool { return l.lessPackages(p, l.packagesIndex[j]) })

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

// ForEachIndexed calls handler for each package in list in indexed order
func (l *PackageList) ForEachIndexed(handler func(*Package) error) error {
	if !l.indexed {
		panic("list not indexed, can't iterate")
	}

	var err error
	for _, p := range l.packagesIndex {
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
				return fmt.Errorf("conflict in package %s", p)
			}
		} else {
			l.packages[k] = p
		}
	}

	return nil
}

// Remove removes package from the list, and updates index when required
func (l *PackageList) Remove(p *Package) {
	delete(l.packages, string(p.ShortKey("")))
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
func (l *PackageList) VerifyDependencies(options int, architectures []string, sources *PackageList, progress aptly.Progress) ([]Dependency, error) {
	missing := make([]Dependency, 0, 128)

	if progress != nil {
		progress.InitBar(int64(l.Len())*int64(len(architectures)), false)
	}

	for _, arch := range architectures {
		cache := make(map[string]bool, 2048)

		for _, p := range l.packages {
			if progress != nil {
				progress.AddBar(1)
			}

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

					if sources.Search(dep, false) == nil {
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

	if progress != nil {
		progress.ShutdownBar()
	}

	return missing, nil
}

// Swap swaps two packages in index
func (l *PackageList) Swap(i, j int) {
	l.packagesIndex[i], l.packagesIndex[j] = l.packagesIndex[j], l.packagesIndex[i]
}

func (l *PackageList) lessPackages(iPkg, jPkg *Package) bool {
	if iPkg.Name == jPkg.Name {
		cmp := CompareVersions(iPkg.Version, jPkg.Version)
		if cmp == 0 {
			return iPkg.Architecture < jPkg.Architecture
		}
		return cmp == 1
	}

	return iPkg.Name < jPkg.Name
}

// Less compares two packages by name (lexographical) and version (latest to oldest)
func (l *PackageList) Less(i, j int) bool {
	return l.lessPackages(l.packagesIndex[i], l.packagesIndex[j])
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

// Scan searches package index using full scan
func (l *PackageList) Scan(q PackageQuery) (result *PackageList) {
	result = NewPackageList()
	for _, pkg := range l.packages {
		if q.Matches(pkg) {
			result.Add(pkg)
		}
	}

	return
}

// Search searches package index for specified package(s) using optimized queries
func (l *PackageList) Search(dep Dependency, allMatches bool) (searchResults []*Package) {
	if !l.indexed {
		panic("list not indexed, can't search")
	}

	if dep.Relation == VersionDontCare {
		for _, p := range l.providesIndex[dep.Pkg] {
			if dep.Architecture == "" || p.MatchesArchitecture(dep.Architecture) {
				searchResults = append(searchResults, p)

				if !allMatches {
					break
				}
			}
		}
	}

	i := sort.Search(len(l.packagesIndex), func(j int) bool { return l.packagesIndex[j].Name >= dep.Pkg })

	for i < len(l.packagesIndex) && l.packagesIndex[i].Name == dep.Pkg {
		p := l.packagesIndex[i]
		if p.MatchesDependency(dep) {
			searchResults = append(searchResults, p)

			if !allMatches {
				break
			}
		}

		i++
	}

	return
}

// Filter filters package index by specified queries (ORed together), possibly pulling dependencies
func (l *PackageList) Filter(queries []PackageQuery, withDependencies bool, source *PackageList, dependencyOptions int, architecturesList []string) (*PackageList, error) {
	if !l.indexed {
		panic("list not indexed, can't filter")
	}

	result := NewPackageList()

	for _, query := range queries {
		result.Append(query.Query(l))
	}

	if withDependencies {
		added := result.Len()

		dependencySource := NewPackageList()
		if source != nil {
			dependencySource.Append(source)
		}
		dependencySource.Append(result)
		dependencySource.PrepareIndex()

		// while some new dependencies were discovered
		for added > 0 {
			added = 0

			// find missing dependencies
			missing, err := result.VerifyDependencies(dependencyOptions, architecturesList, dependencySource, nil)
			if err != nil {
				return nil, err
			}

			// try to satisfy dependencies
			for _, dep := range missing {
				searchResults := l.Search(dep, false)
				if searchResults != nil {
					for _, p := range searchResults {
						result.Add(p)
						dependencySource.Add(p)
						added++
					}
				}
			}
		}
	}

	return result, nil
}
