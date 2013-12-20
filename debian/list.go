package debian

import (
	"bytes"
	"fmt"
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

// Len returns number of packages in the list
func (l *PackageList) Len() int {
	return len(l.packages)
}

// PackageRefList is a list of keys of packages, this is basis for snapshot
// and similar stuff
//
// Refs are sorted in lexographical order
type PackageRefList struct {
	// List of package keys
	Refs [][]byte
}

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
