package deb

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// PackageLike is something like Package :) To be refined later
type PackageLike interface {
	GetField(string) string
	MatchesDependency(Dependency) bool
	MatchesArchitecture(string) bool
	GetName() string
	GetVersion() string
	GetArchitecture() string
}

// PackageCatalog is abstraction on top of PackageCollection and PackageList
type PackageCatalog interface {
	Scan(q PackageQuery) (result *PackageList)
	Search(dep Dependency, allMatches bool) (searchResults []*Package)
	SearchSupported() bool
	SearchByKey(arch, name, version string) (result *PackageList)
}

// PackageQuery is interface of predicate on Package
type PackageQuery interface {
	// Matches calculates match of condition against package
	Matches(pkg PackageLike) bool
	// Fast returns if search strategy is possible for this query
	Fast(list PackageCatalog) bool
	// Query performs search on package list
	Query(list PackageCatalog) *PackageList
	// String interface
	String() string
}

// OrQuery is L | R
type OrQuery struct {
	L, R PackageQuery
}

// AndQuery is L , R
type AndQuery struct {
	L, R PackageQuery
}

// NotQuery is ! Q
type NotQuery struct {
	Q PackageQuery
}

// FieldQuery is generic request against field
type FieldQuery struct {
	Field    string
	Relation int
	Value    string
	Regexp   *regexp.Regexp `codec:"-"`
}

// PkgQuery is search request against specific package
type PkgQuery struct {
	Pkg     string
	Version string
	Arch    string
}

// DependencyQuery is generic Debian-dependency like query
type DependencyQuery struct {
	Dep Dependency
}

// MatchAllQuery is query that matches all the packages
type MatchAllQuery struct{}

// Matches if any of L, R matches
func (q *OrQuery) Matches(pkg PackageLike) bool {
	return q.L.Matches(pkg) || q.R.Matches(pkg)
}

// Fast is true only if both parts are fast
func (q *OrQuery) Fast(list PackageCatalog) bool {
	return q.L.Fast(list) && q.R.Fast(list)
}

// Query strategy depends on nodes
func (q *OrQuery) Query(list PackageCatalog) (result *PackageList) {
	if q.Fast(list) {
		result = q.L.Query(list)
		result.Append(q.R.Query(list))
	} else {
		result = list.Scan(q)
	}
	return
}

// String interface
func (q *OrQuery) String() string {
	return fmt.Sprintf("(%s) | (%s)", q.L, q.R)
}

// Matches if both of L, R matches
func (q *AndQuery) Matches(pkg PackageLike) bool {
	return q.L.Matches(pkg) && q.R.Matches(pkg)
}

// Fast is true if any of the parts are fast
func (q *AndQuery) Fast(list PackageCatalog) bool {
	return q.L.Fast(list) || q.R.Fast(list)
}

// Query strategy depends on nodes
func (q *AndQuery) Query(list PackageCatalog) (result *PackageList) {
	if !q.Fast(list) {
		result = list.Scan(q)
	} else {
		if q.L.Fast(list) {
			result = q.L.Query(list)
			result = result.Scan(q.R)
		} else {
			result = q.R.Query(list)
			result = result.Scan(q.L)
		}
	}
	return
}

// String interface
func (q *AndQuery) String() string {
	return fmt.Sprintf("(%s), (%s)", q.L, q.R)
}

// Matches if not matches
func (q *NotQuery) Matches(pkg PackageLike) bool {
	return !q.Q.Matches(pkg)
}

// Fast is false
func (q *NotQuery) Fast(_ PackageCatalog) bool {
	return false
}

// Query strategy is scan always
func (q *NotQuery) Query(list PackageCatalog) (result *PackageList) {
	result = list.Scan(q)
	return
}

// String interface
func (q *NotQuery) String() string {
	return fmt.Sprintf("!(%s)", q.Q)
}

// Matches on generic field
func (q *FieldQuery) Matches(pkg PackageLike) bool {
	if q.Field == "$Version" {
		return pkg.MatchesDependency(Dependency{Pkg: pkg.GetName(), Relation: q.Relation, Version: q.Value, Regexp: q.Regexp})
	}
	if q.Field == "$Architecture" && q.Relation == VersionEqual {
		return pkg.MatchesArchitecture(q.Value)
	}

	field := pkg.GetField(q.Field)

	switch q.Relation {
	case VersionDontCare:
		return field != ""
	case VersionEqual:
		return CompareVersions(field, q.Value) == 0
	case VersionGreater:
		return CompareVersions(field, q.Value) > 0
	case VersionGreaterOrEqual:
		return CompareVersions(field, q.Value) >= 0
	case VersionLess:
		return CompareVersions(field, q.Value) < 0
	case VersionLessOrEqual:
		return CompareVersions(field, q.Value) <= 0
	case VersionPatternMatch:
		matched, err := filepath.Match(q.Value, field)
		return err == nil && matched
	case VersionRegexp:
		if q.Regexp == nil {
			q.Regexp = regexp.MustCompile(q.Value)
		}
		return q.Regexp.FindStringIndex(field) != nil

	}
	panic("unknown relation")
}

// Query runs iteration through list
func (q *FieldQuery) Query(list PackageCatalog) (result *PackageList) {
	result = list.Scan(q)
	return
}

// Fast depends on the query
func (q *FieldQuery) Fast(_ PackageCatalog) bool {
	return false
}

// String interface
func (q *FieldQuery) String() string {
	escape := func(val string) string {
		if strings.ContainsAny(val, "()|,!{} \t\n") {
			return "'" + strings.Replace(strings.Replace(val, "\\", "\\\\", -1), "'", "\\'", -1) + "'"
		}
		return val
	}

	var op string
	switch q.Relation {
	case VersionEqual:
		op = "="
	case VersionGreater:
		op = ">>"
	case VersionLess:
		op = "<<"
	case VersionRegexp:
		op = "~"
	case VersionPatternMatch:
		op = "%"
	case VersionGreaterOrEqual:
		op = ">="
	case VersionLessOrEqual:
		op = "<="
	}
	return fmt.Sprintf("%s (%s %s)", escape(q.Field), op, escape(q.Value))
}

// Matches on dependency condition
func (q *DependencyQuery) Matches(pkg PackageLike) bool {
	return pkg.MatchesDependency(q.Dep)
}

// Fast is always true for dependency query
func (q *DependencyQuery) Fast(list PackageCatalog) bool {
	return list.SearchSupported()
}

// Query runs PackageList.Search
func (q *DependencyQuery) Query(list PackageCatalog) (result *PackageList) {
	if q.Fast(list) {
		result = NewPackageList()
		for _, pkg := range list.Search(q.Dep, true) {
			result.Add(pkg)
		}
	} else {
		result = list.Scan(q)
	}

	return
}

// String interface
func (q *DependencyQuery) String() string {
	return q.Dep.String()
}

// Matches on specific properties
func (q *PkgQuery) Matches(pkg PackageLike) bool {
	return pkg.GetName() == q.Pkg && pkg.GetVersion() == q.Version && pkg.GetArchitecture() == q.Arch
}

// Fast is always true for package query
func (q *PkgQuery) Fast(_ PackageCatalog) bool {
	return true
}

// Query looks up specific package
func (q *PkgQuery) Query(list PackageCatalog) (result *PackageList) {
	return list.SearchByKey(q.Arch, q.Pkg, q.Version)
}

// String interface
func (q *PkgQuery) String() string {
	return fmt.Sprintf("%s_%s_%s", q.Pkg, q.Version, q.Arch)
}

// Matches on specific properties
func (q *MatchAllQuery) Matches(_ PackageLike) bool {
	return true
}

// Fast is always true for match all query
func (q *MatchAllQuery) Fast(_ PackageCatalog) bool {
	return true
}

// Query looks up specific package
func (q *MatchAllQuery) Query(list PackageCatalog) (result *PackageList) {
	return list.Scan(q)
}

// String interface
func (q *MatchAllQuery) String() string {
	return ""
}
