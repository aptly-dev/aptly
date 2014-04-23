package deb

import (
	"bytes"
	"github.com/ugorji/go/codec"
	"sort"
)

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

// NewPackageRefList creates empty PackageRefList
func NewPackageRefList() *PackageRefList {
	return &PackageRefList{}
}

// NewPackageRefListFromPackageList creates PackageRefList from PackageList
func NewPackageRefListFromPackageList(list *PackageList) *PackageRefList {
	reflist := &PackageRefList{}
	reflist.Refs = make([][]byte, list.Len())

	i := 0
	for _, p := range list.packages {
		reflist.Refs[i] = p.Key("")
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

// Substract returns all packages in l that are not in r
func (l *PackageRefList) Substract(r *PackageRefList) *PackageRefList {
	result := &PackageRefList{Refs: make([][]byte, 0, 128)}

	// pointer to left and right reflists
	il, ir := 0, 0
	// length of reflists
	ll, lr := l.Len(), r.Len()

	for il < ll || ir < lr {
		if il == ll {
			// left list exhausted, we got the result
			break
		}
		if ir == lr {
			// right list exhausted, append what is left to result
			result.Refs = append(result.Refs, l.Refs[il:]...)
			break
		}

		rel := bytes.Compare(l.Refs[il], r.Refs[ir])
		if rel == 0 {
			// r contains entry from l, so we skip it
			il++
			ir++
		} else if rel < 0 {
			// item il is not in r, append
			result.Refs = append(result.Refs, l.Refs[il])
			il++
		} else {
			// skip over to next item in r
			ir++
		}
	}

	return result
}

// PackageDiff is a difference between two packages in a list.
//
// If left & right are present, difference is in package version
// If left is nil, package is present only in right
// If right is nil, package is present only in left
type PackageDiff struct {
	Left, Right *Package
}

// PackageDiffs is a list of PackageDiff records
type PackageDiffs []PackageDiff

// Diff calculates difference between two reflists
func (l *PackageRefList) Diff(r *PackageRefList, packageCollection *PackageCollection) (result PackageDiffs, err error) {
	result = make(PackageDiffs, 0, 128)

	// pointer to left and right reflists
	il, ir := 0, 0
	// length of reflists
	ll, lr := l.Len(), r.Len()
	// cached loaded packages on the left & right
	pl, pr := (*Package)(nil), (*Package)(nil)

	// until we reached end of both lists
	for il < ll || ir < lr {
		// if we've exhausted left list, pull the rest from the right
		if il == ll {
			pr, err = packageCollection.ByKey(r.Refs[ir])
			if err != nil {
				return nil, err
			}
			result = append(result, PackageDiff{Left: nil, Right: pr})
			ir++
			continue
		}
		// if we've exhausted right list, pull the rest from the left
		if ir == lr {
			pl, err = packageCollection.ByKey(l.Refs[il])
			if err != nil {
				return nil, err
			}
			result = append(result, PackageDiff{Left: pl, Right: nil})
			il++
			continue
		}

		// refs on both sides are present, load them
		rl, rr := l.Refs[il], r.Refs[ir]
		// compare refs
		rel := bytes.Compare(rl, rr)

		if rel == 0 {
			// refs are identical, so are packages, advance pointer
			il++
			ir++
			pl, pr = nil, nil
		} else {
			// load pl & pr if they haven't been loaded before
			if pl == nil {
				pl, err = packageCollection.ByKey(rl)
				if err != nil {
					return nil, err
				}
			}

			if pr == nil {
				pr, err = packageCollection.ByKey(rr)
				if err != nil {
					return nil, err
				}
			}

			// is pl & pr the same package, but different version?
			if pl.Name == pr.Name && pl.Architecture == pr.Architecture {
				result = append(result, PackageDiff{Left: pl, Right: pr})
				il++
				ir++
				pl, pr = nil, nil
			} else {
				// otherwise pl or pr is missing on one of the sides
				if rel < 0 {
					result = append(result, PackageDiff{Left: pl, Right: nil})
					il++
					pl = nil
				} else {
					result = append(result, PackageDiff{Left: nil, Right: pr})
					ir++
					pr = nil
				}
			}

		}
	}

	return
}

// Merge merges reflist r into current reflist. If overrideMatching, merge
// replaces matching packages (by architecture/name) with reference from r.
// Otherwise, all packages are saved.
func (l *PackageRefList) Merge(r *PackageRefList, overrideMatching bool) (result *PackageRefList) {
	// pointer to left and right reflists
	il, ir := 0, 0
	// length of reflists
	ll, lr := l.Len(), r.Len()

	result = &PackageRefList{}
	result.Refs = make([][]byte, 0, ll+lr)

	// until we reached end of both lists
	for il < ll || ir < lr {
		// if we've exhausted left list, pull the rest from the right
		if il == ll {
			result.Refs = append(result.Refs, r.Refs[ir:]...)
			break
		}
		// if we've exhausted right list, pull the rest from the left
		if ir == lr {
			result.Refs = append(result.Refs, l.Refs[il:]...)
			break
		}

		// refs on both sides are present, load them
		rl, rr := l.Refs[il], r.Refs[ir]
		// compare refs
		rel := bytes.Compare(rl, rr)

		if rel == 0 {
			// refs are identical, so are packages, advance pointer
			result.Refs = append(result.Refs, l.Refs[il])
			il++
			ir++
		} else {
			if overrideMatching {
				partsL := bytes.Split(rl, []byte(" "))
				archL, nameL := partsL[0][1:], partsL[1]

				partsR := bytes.Split(rr, []byte(" "))
				archR, nameR := partsR[0][1:], partsR[1]

				if bytes.Compare(archL, archR) == 0 && bytes.Compare(nameL, nameR) == 0 {
					// override with package from the right
					result.Refs = append(result.Refs, r.Refs[ir])
					il++
					ir++
					continue
				}
			}

			// otherwise append smallest of two
			if rel < 0 {
				result.Refs = append(result.Refs, l.Refs[il])
				il++
			} else {
				result.Refs = append(result.Refs, r.Refs[ir])
				ir++
			}
		}
	}

	return
}

// FilterLatestRefs takes in a reflist with potentially multiples of the same
// packages and reduces it to only the latest of each package. The operations
// are done in-place. This implements a "latest wins" approach which can be used
// while merging two or more snapshots together.
func FilterLatestRefs(r *PackageRefList) {
	// A running tab of latest seen package refs.
	latestRefs := make(map[string]int)

	for i := 0; i < len(r.Refs); i++ {
		partsL := bytes.Split(r.Refs[i], []byte(" "))
		archL, nameL, verL := partsL[0][1:], partsL[1], partsL[2]
		pkgId := string(nameL) + "." + string(archL)

		// If the package hasn't been seen before, add it and advance.
		if _, ok := latestRefs[pkgId]; !ok {
			latestRefs[pkgId] = i
			continue
		}

		// If we've already seen this package, check versions
		partsR := bytes.Split(r.Refs[latestRefs[pkgId]], []byte(" "))
		verR := partsR[2]
		vres := CompareVersions(string(verL), string(verR))

		// Remove the older or duplicate refs from the result
		if vres > 0 {
			old := latestRefs[pkgId]
			r.Refs = append(r.Refs[0:old], r.Refs[old+1:]...)
			latestRefs[pkgId] = i - 1
		} else {
			r.Refs = append(r.Refs[0:i], r.Refs[i+1:]...)
		}

		// Compensate for the reduced set
		i -= 1
	}

	return
}
