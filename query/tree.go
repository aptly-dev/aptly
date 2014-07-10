package query

import (
	"github.com/smira/aptly/deb"
)

// PackageQuery is interface of predicate on Package
type PackageQuery interface {
	Matches(pkg *deb.Package) bool
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
}

// DependencyQuery is generic Debian-dependency like query
type DependencyQuery struct {
	Dep deb.Dependency
}

// Matches if any of L, R matches
func (q *OrQuery) Matches(pkg *deb.Package) bool {
	return q.L.Matches(pkg) || q.R.Matches(pkg)
}

// Matches if both of L, R matches
func (q *AndQuery) Matches(pkg *deb.Package) bool {
	return q.L.Matches(pkg) && q.R.Matches(pkg)
}

// Matches if not matches
func (q *NotQuery) Matches(pkg *deb.Package) bool {
	return !q.Q.Matches(pkg)
}

// Matches on generic field
func (q *FieldQuery) Matches(pkg *deb.Package) bool {
	panic("not implemented yet")
}

// Matches on dependency condition
func (q *DependencyQuery) Matches(pkg *deb.Package) bool {
	return pkg.MatchesDependency(q.Dep)
}
