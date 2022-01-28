package deb

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/utils"
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
	// DepVerboseResolve emits additional logs while dependencies are being resolved
	DepVerboseResolve
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
	// Indexed list of packages, sorted by name internally
	packagesIndex []*Package
	// Map of packages for each virtual package (provides)
	providesIndex map[string][]*Package
	// Package key generation function
	keyFunc func(p *Package) string
	// Allow duplicates?
	duplicatesAllowed bool
	// Has index been prepared?
	indexed bool
}

// PackageConflictError means that package can't be added to the list due to error
type PackageConflictError struct {
	error
}

// Verify interface
var (
	_ sort.Interface = &PackageList{}
	_ PackageCatalog = &PackageList{}
)

func packageShortKey(p *Package) string {
	return string(p.ShortKey(""))
}

func packageFullKey(p *Package) string {
	return string(p.Key(""))
}

// NewPackageList creates empty package list without duplicate package
func NewPackageList() *PackageList {
	return NewPackageListWithDuplicates(false, 1000)
}

// NewPackageListWithDuplicates creates empty package list which might allow or block duplicate packages
func NewPackageListWithDuplicates(duplicates bool, capacity int) *PackageList {
	if capacity == 0 {
		capacity = 1000
	}

	result := &PackageList{
		packages:          make(map[string]*Package, capacity),
		duplicatesAllowed: duplicates,
		keyFunc:           packageShortKey,
	}

	if duplicates {
		result.keyFunc = packageFullKey
	}

	return result
}

// NewPackageListFromRefList loads packages list from PackageRefList
func NewPackageListFromRefList(reflist *PackageRefList, collection *PackageCollection, progress aptly.Progress) (*PackageList, error) {
	// empty reflist
	if reflist == nil {
		return NewPackageList(), nil
	}

	result := NewPackageListWithDuplicates(false, reflist.Len())

	if progress != nil {
		progress.InitBar(int64(reflist.Len()), false, aptly.BarGeneralBuildPackageList)
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

// Has checks whether package is already in the list
func (l *PackageList) Has(p *Package) bool {
	key := l.keyFunc(p)
	_, ok := l.packages[key]

	return ok
}

// Add appends package to package list, additionally checking for uniqueness
func (l *PackageList) Add(p *Package) error {
	key := l.keyFunc(p)
	existing, ok := l.packages[key]
	if ok {
		if !existing.Equals(p) {
			return &PackageConflictError{fmt.Errorf("conflict in package %s", p)}
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
	delete(l.packages, l.keyFunc(p))
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
		if pkg.Architecture != ArchitectureAll && (pkg.Architecture != ArchitectureSource || includeSource) && !utils.StrSliceHasItem(result, pkg.Architecture) {
			result = append(result, pkg.Architecture)
		}
	}
	return
}

// Strings builds list of strings with package keys
func (l *PackageList) Strings() []string {
	result := make([]string, l.Len())
	i := 0

	for _, p := range l.packages {
		result[i] = string(p.Key(""))
		i++
	}

	return result
}

// FullNames builds a list of package {name}_{version}_{arch}
func (l *PackageList) FullNames() []string {
	result := make([]string, l.Len())
	i := 0

	for _, p := range l.packages {
		result[i] = p.GetFullName()
		i++
	}

	return result
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
// Analysis would be performed for each architecture, in specified sources
func (l *PackageList) VerifyDependencies(options int, architectures []string, sources *PackageList, progress aptly.Progress) ([]Dependency, error) {
	l.PrepareIndex()
	missing := make([]Dependency, 0, 128)

	if progress != nil {
		progress.InitBar(int64(l.Len())*int64(len(architectures)), false, aptly.BarGeneralVerifyDependencies)
	}

	for _, arch := range architectures {
		cache := make(map[string]bool, 2048)

		for _, p := range l.packagesIndex {
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

				for _, dep := range variants {
					if dep.Architecture == "" {
						dep.Architecture = arch
					}

					hash := dep.Hash()
					satisfied, ok := cache[hash]
					if !ok {
						satisfied = sources.Search(dep, false) != nil
						cache[hash] = satisfied
					}

					if !satisfied && !ok {
						variantsMissing = append(variantsMissing, dep)
					}

					if satisfied && options&DepFollowAllVariants == 0 {
						variantsMissing = nil
						break
					}
				}

				missing = append(missing, variantsMissing...)
			}
		}
	}

	if progress != nil {
		progress.ShutdownBar()
	}

	if options&DepVerboseResolve == DepVerboseResolve && progress != nil {
		missingStr := make([]string, len(missing))
		for i := range missing {
			missingStr[i] = missing[i].String()
		}
		progress.ColoredPrintf("@{y}Missing dependencies:@| %s", strings.Join(missingStr, ", "))
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
	if l.indexed {
		return
	}

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
	result = NewPackageListWithDuplicates(l.duplicatesAllowed, 0)
	for _, pkg := range l.packages {
		if q.Matches(pkg) {
			result.Add(pkg)
		}
	}

	return
}

// SearchSupported returns true for PackageList
func (l *PackageList) SearchSupported() bool {
	return true
}

// SearchByKey looks up package by exact key reference
func (l *PackageList) SearchByKey(arch, name, version string) (result *PackageList) {
	result = NewPackageListWithDuplicates(l.duplicatesAllowed, 0)

	pkg := l.packages["P"+arch+" "+name+" "+version]
	if pkg != nil {
		result.Add(pkg)
	}

	return
}

// Search searches package index for specified package(s) using optimized queries
func (l *PackageList) Search(dep Dependency, allMatches bool) (searchResults []*Package) {
	if !l.indexed {
		panic("list not indexed, can't search")
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

	return
}

// Filter filters package index by specified queries (ORed together), possibly pulling dependencies
func (l *PackageList) Filter(queries []PackageQuery, withDependencies bool, source *PackageList, dependencyOptions int, architecturesList []string) (*PackageList, error) {
	return l.FilterWithProgress(queries, withDependencies, source, dependencyOptions, architecturesList, nil)
}

// FilterWithProgress filters package index by specified queries (ORed together), possibly pulling dependencies and displays progress
func (l *PackageList) FilterWithProgress(queries []PackageQuery, withDependencies bool, source *PackageList, dependencyOptions int, architecturesList []string, progress aptly.Progress) (*PackageList, error) {
	if !l.indexed {
		panic("list not indexed, can't filter")
	}

	result := NewPackageList()

	for _, query := range queries {
		result.Append(query.Query(l))
	}

	if withDependencies {
		added := result.Len()
		result.PrepareIndex()

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
			missing, err := result.VerifyDependencies(dependencyOptions, architecturesList, dependencySource, progress)
			if err != nil {
				return nil, err
			}

			// try to satisfy dependencies
			for _, dep := range missing {
				if dependencyOptions&DepFollowAllVariants == 0 {
					// dependency might have already been satisfied
					// with packages already been added
					//
					// when follow-all-variants is enabled, we need to try to expand anyway,
					// as even if dependency is satisfied now, there might be other ways to satisfy dependency
					if result.Search(dep, false) != nil {
						if dependencyOptions&DepVerboseResolve == DepVerboseResolve && progress != nil {
							progress.ColoredPrintf("@{y}Already satisfied dependency@|: %s with %s", &dep, result.Search(dep, true))
						}
						continue
					}
				}

				searchResults := l.Search(dep, true)
				if len(searchResults) > 0 {
					for _, p := range searchResults {
						if result.Has(p) {
							continue
						}

						if dependencyOptions&DepVerboseResolve == DepVerboseResolve && progress != nil {
							progress.ColoredPrintf("@{g}Injecting package@|: %s", p)
						}
						result.Add(p)
						dependencySource.Add(p)
						added++
						if dependencyOptions&DepFollowAllVariants == 0 {
							break
						}
					}
				} else {
					if dependencyOptions&DepVerboseResolve == DepVerboseResolve && progress != nil {
						progress.ColoredPrintf("@{r}Unsatisfied dependency@|: %s", dep.String())
					}

				}
			}
		}
	}

	return result, nil
}
