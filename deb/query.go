package deb

// PackageQuery is interface of predicate on Package
type PackageQuery interface {
	// Matches calculates match of condition against package
	Matches(pkg *Package) bool
	// Searchable returns if search strategy is possible for this query
	Searchable() bool
	// Query performs search on package list
	Query(list *PackageList) *PackageList
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

// Matches if any of L, R matches
func (q *OrQuery) Matches(pkg *Package) bool {
	return q.L.Matches(pkg) || q.R.Matches(pkg)
}

// Searchable is true only if both parts are searchable
func (q *OrQuery) Searchable() bool {
	return q.L.Searchable() && q.R.Searchable()
}

// Query strategy depends on nodes
func (q *OrQuery) Query(list *PackageList) (result *PackageList) {
	if q.Searchable() {
		result = q.L.Query(list)
		result.Append(q.R.Query(list))
	} else {
		result = list.Scan(q)
	}
	return
}

// Matches if both of L, R matches
func (q *AndQuery) Matches(pkg *Package) bool {
	return q.L.Matches(pkg) && q.R.Matches(pkg)
}

// Searchable is true if any of the parts are searchable
func (q *AndQuery) Searchable() bool {
	return q.L.Searchable() || q.R.Searchable()
}

// Query strategy depends on nodes
func (q *AndQuery) Query(list *PackageList) (result *PackageList) {
	if !q.Searchable() {
		result = list.Scan(q)
	} else {
		if q.L.Searchable() {
			result = q.L.Query(list)
			result = result.Scan(q.R)
		} else {
			result = q.R.Query(list)
			result = result.Scan(q.L)
		}
	}
	return
}

// Matches if not matches
func (q *NotQuery) Matches(pkg *Package) bool {
	return !q.Q.Matches(pkg)
}

// Searchable is false
func (q *NotQuery) Searchable() bool {
	return false
}

// NotQuery strategy is scan always
func (q *NotQuery) Query(list *PackageList) (result *PackageList) {
	result = list.Scan(q)
	return
}

// Matches on generic field
func (q *FieldQuery) Matches(pkg *Package) bool {
	panic("not implemented yet")
}

// Query runs iteration through list
func (q *FieldQuery) Query(list *PackageList) (result *PackageList) {
	panic("not implemented yet")
}

// Searchable depends on the query
func (q *FieldQuery) Searchable() bool {
	return false
}

// Matches on dependency condition
func (q *DependencyQuery) Matches(pkg *Package) bool {
	return pkg.MatchesDependency(q.Dep)
}

// Searchable is always true for dependency query
func (q *DependencyQuery) Searchable() bool {
	return true
}

// Query runs PackageList.Search
func (q *DependencyQuery) Query(list *PackageList) (result *PackageList) {
	result = NewPackageList()
	for _, pkg := range list.Search(q.Dep, true) {
		result.Add(pkg)
	}

	return
}

// Matches on specific properties
func (q *PkgQuery) Matches(pkg *Package) bool {
	return pkg.Name == q.Pkg && pkg.Version == q.Version && pkg.Architecture == q.Arch
}

// Searchable is always true for package query
func (q *PkgQuery) Searchable() bool {
	return true
}

// Query looks up specific package
func (q *PkgQuery) Query(list *PackageList) (result *PackageList) {
	result = NewPackageList()

	pkg := list.packages["P"+q.Arch+" "+q.Pkg+" "+q.Version]
	if pkg != nil {
		result.Add(pkg)
	}

	return
}
