package debian

import (
	"fmt"
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
//
// TODO: Error handling
func (l *PackageList) ForEach(handler func(*Package)) {
	for _, p := range l.packages {
		handler(p)
	}
}

// Length returns number of packages in the list
func (l *PackageList) Length() int {
	return len(l.packages)
}
