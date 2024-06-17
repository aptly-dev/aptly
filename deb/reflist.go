package deb

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/AlekSi/pointer"
	"github.com/aptly-dev/aptly/database"
	"github.com/cespare/xxhash/v2"
	"github.com/ugorji/go/codec"
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

func (l *PackageRefList) Clone() *PackageRefList {
	clone := &PackageRefList{}
	clone.Refs = make([][]byte, l.Len())
	copy(clone.Refs, l.Refs)
	return clone
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
	handle := &codec.MsgpackHandle{}
	handle.ZeroCopy = true
	decoder := codec.NewDecoderBytes(input, handle)
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

// Has checks whether package is part of reflist
func (l *PackageRefList) Has(p *Package) bool {
	key := p.Key("")

	i := sort.Search(len(l.Refs), func(j int) bool { return bytes.Compare(l.Refs[j], key) >= 0 })
	return i < len(l.Refs) && bytes.Equal(l.Refs[i], key)
}

// Strings builds list of strings with package keys
func (l *PackageRefList) Strings() []string {
	if l == nil {
		return []string{}
	}

	result := make([]string, l.Len())

	for i := 0; i < l.Len(); i++ {
		result[i] = string(l.Refs[i])
	}

	return result
}

// Subtract returns all packages in l that are not in r
func (l *PackageRefList) Subtract(r *PackageRefList) *PackageRefList {
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

// Check interface
var (
	_ json.Marshaler = PackageDiff{}
)

// MarshalJSON implements json.Marshaler interface
func (d PackageDiff) MarshalJSON() ([]byte, error) {
	serialized := struct {
		Left, Right *string
	}{}

	if d.Left != nil {
		serialized.Left = pointer.ToString(string(d.Left.Key("")))
	}
	if d.Right != nil {
		serialized.Right = pointer.ToString(string(d.Right.Key("")))
	}

	return json.Marshal(serialized)
}

// PackageDiffs is a list of PackageDiff records
type PackageDiffs []PackageDiff

// Diff calculates difference between two reflists
func (l *PackageRefList) Diff(r *PackageRefList, packageCollection *PackageCollection, result PackageDiffs) (PackageDiffs, error) {
	var err error

	if result == nil {
		result = make(PackageDiffs, 0, 128)
	}

	// pointer to left and right reflists
	il, ir := 0, 0
	// length of reflists
	ll, lr := l.Len(), r.Len()
	// cached loaded packages on the left & right
	pl, pr := (*Package)(nil), (*Package)(nil)

	// until we reached end of both lists
	for il < ll || ir < lr {
		var rl, rr []byte
		if il < ll {
			rl = l.Refs[il]
		}
		if ir < lr {
			rr = r.Refs[ir]
		}

		// compare refs
		rel := bytes.Compare(rl, rr)
		// an unset ref is less than all others, but since it represents the end
		// of a reflist, it should be *greater*, so flip the comparison result
		if rl == nil || rr == nil {
			rel = -rel
		}

		if rel == 0 {
			// refs are identical, so are packages, advance pointer
			il++
			ir++
			pl, pr = nil, nil
		} else {
			// load pl & pr if they haven't been loaded before
			if pl == nil && rl != nil {
				pl, err = packageCollection.ByKey(rl)
				if err != nil {
					return nil, err
				}
			}

			if pr == nil && rr != nil {
				pr, err = packageCollection.ByKey(rr)
				if err != nil {
					return nil, err
				}
			}

			// otherwise pl or pr is missing on one of the sides
			if rel < 0 {
				// compaction: +(,A) -(B,) --> !(A,B)
				if len(result) > 0 && result[len(result)-1].Left == nil && result[len(result)-1].Right.Name == pl.Name &&
					result[len(result)-1].Right.Architecture == pl.Architecture {
					result[len(result)-1] = PackageDiff{Left: pl, Right: result[len(result)-1].Right}
				} else {
					result = append(result, PackageDiff{Left: pl, Right: nil})
				}
				il++
				pl = nil
			} else {
				// compaction: -(A,) +(,B) --> !(A,B)
				if len(result) > 0 && result[len(result)-1].Right == nil && result[len(result)-1].Left.Name == pr.Name &&
					result[len(result)-1].Left.Architecture == pr.Architecture {
					result[len(result)-1] = PackageDiff{Left: result[len(result)-1].Left, Right: pr}
				} else {
					result = append(result, PackageDiff{Left: nil, Right: pr})
				}
				ir++
				pr = nil
			}
		}
	}

	return result, nil
}

// Merge merges reflist r into current reflist. If overrideMatching, merge
// replaces matching packages (by architecture/name) with reference from r.
// If ignoreConflicting is set, all packages are preserved, otherwise conflicting
// packages are overwritten with packages from "right" snapshot.
func (l *PackageRefList) Merge(r *PackageRefList, overrideMatching, ignoreConflicting bool) (result *PackageRefList) {
	var overriddenArch, overridenName []byte

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
			overridenName = nil
			overriddenArch = nil
		} else {
			if !ignoreConflicting || overrideMatching {
				partsL := bytes.Split(rl, []byte(" "))
				archL, nameL, versionL := partsL[0][1:], partsL[1], partsL[2]

				partsR := bytes.Split(rr, []byte(" "))
				archR, nameR, versionR := partsR[0][1:], partsR[1], partsR[2]

				if !ignoreConflicting && bytes.Equal(archL, archR) &&
					bytes.Equal(nameL, nameR) && bytes.Equal(versionL, versionR) {
					// conflicting duplicates with same arch, name, version, but different file hash
					result.Refs = append(result.Refs, r.Refs[ir])
					il++
					ir++
					overridenName = nil
					overriddenArch = nil
					continue
				}

				if overrideMatching {
					if bytes.Equal(archL, overriddenArch) && bytes.Equal(nameL, overridenName) {
						// this package has already been overridden on the right
						il++
						continue
					}

					if bytes.Equal(archL, archR) && bytes.Equal(nameL, nameR) {
						// override with package from the right
						result.Refs = append(result.Refs, r.Refs[ir])
						il++
						ir++
						overriddenArch = archL
						overridenName = nameL
						continue
					}
				}
			}

			// otherwise append smallest of two
			if rel < 0 {
				result.Refs = append(result.Refs, l.Refs[il])
				il++
			} else {
				result.Refs = append(result.Refs, r.Refs[ir])
				ir++
				overridenName = nil
				overriddenArch = nil
			}
		}
	}

	return
}

// FilterLatestRefs takes in a reflist with potentially multiples of the same
// packages and reduces it to only the latest of each package. The operations
// are done in-place. This implements a "latest wins" approach which can be used
// while merging two or more snapshots together.
func (l *PackageRefList) FilterLatestRefs() {
	var (
		lastArch, lastName, lastVer []byte
		arch, name, ver             []byte
		parts                       [][]byte
	)

	for i := 0; i < len(l.Refs); i++ {
		parts = bytes.Split(l.Refs[i][1:], []byte(" "))
		arch, name, ver = parts[0], parts[1], parts[2]

		if bytes.Equal(arch, lastArch) && bytes.Equal(name, lastName) {
			// Two packages are identical, check version and only one wins
			vres := CompareVersions(string(ver), string(lastVer))

			// Remove the older refs from the result
			if vres > 0 {
				// ver[i] > ver[i-1], remove element i-1
				l.Refs = append(l.Refs[:i-1], l.Refs[i:]...)
			} else {
				// ver[i] < ver[i-1], remove element i
				l.Refs = append(l.Refs[:i], l.Refs[i+1:]...)
				arch, name, ver = lastArch, lastName, lastVer
			}

			// Compensate for the reduced set
			i--
		}

		lastArch, lastName, lastVer = arch, name, ver
	}
}

const (
	reflistBucketCount = 1 << 6
	reflistBucketMask  = reflistBucketCount - 1
)

type reflistDigestArray [sha256.Size]byte

func bucketRefPrefix(ref []byte) []byte {
	const maxPrefixLen = 3

	// Cut out the arch, leaving behind the package name and subsequent info.
	_, ref, _ = bytes.Cut(ref, []byte{' '})

	// Strip off the lib prefix, so that "libxyz" and "xyz", which are likely
	// to be updated together, go in the same bucket.
	libPrefix := []byte("lib")
	if bytes.HasPrefix(ref, libPrefix) {
		ref = ref[len(libPrefix):]
	}

	prefixLen := len(ref)
	if maxPrefixLen < prefixLen {
		prefixLen = maxPrefixLen
	}
	prefix, _, _ := bytes.Cut(ref[:prefixLen], []byte{' '})
	return prefix
}

func bucketIdxForRef(ref []byte) int {
	return int(xxhash.Sum64(bucketRefPrefix(ref))) & reflistBucketMask
}

// SplitRefList is a list of package refs, similar to a PackageRefList. However,
// instead of storing a linear array of refs, SplitRefList splits the refs into
// PackageRefList "buckets", based on a hash of the package name inside the ref.
// Each bucket has a digest of its contents that serves as its key in the database.
//
// When serialized, a SplitRefList just becomes an array of bucket digests, and
// the buckets themselves are stored separately. Because the buckets are then
// referenced by their digests, multiple independent reflists can share buckets,
// if their buckets have matching digests.
//
// Buckets themselves may not be confirmed to a single database value; instead,
// they're split into "segments", based on the database's preferred maximum
// value size. This prevents large buckets from slowing down the database.
type SplitRefList struct {
	Buckets [][]byte

	bucketRefs []*PackageRefList
}

// NewSplitRefList creates empty SplitRefList
func NewSplitRefList() *SplitRefList {
	sl := &SplitRefList{}
	sl.reset()
	return sl
}

// NewSplitRefListFromRefList creates SplitRefList from PackageRefList
func NewSplitRefListFromRefList(reflist *PackageRefList) *SplitRefList {
	sl := NewSplitRefList()
	sl.Replace(reflist)
	return sl
}

// NewSplitRefListFromRefList creates SplitRefList from PackageList
func NewSplitRefListFromPackageList(list *PackageList) *SplitRefList {
	return NewSplitRefListFromRefList(NewPackageRefListFromPackageList(list))
}

func (sl *SplitRefList) reset() {
	sl.Buckets = make([][]byte, reflistBucketCount)
	sl.bucketRefs = make([]*PackageRefList, reflistBucketCount)
}

// Has checks whether package is part of reflist
func (sl *SplitRefList) Has(p *Package) bool {
	idx := bucketIdxForRef(p.Key(""))
	if bucket := sl.bucketRefs[idx]; bucket != nil {
		return bucket.Has(p)
	}
	return false
}

// Len returns number of refs
func (sl *SplitRefList) Len() int {
	total := 0
	for _, bucket := range sl.bucketRefs {
		if bucket != nil {
			total += bucket.Len()
		}
	}
	return total
}

func reflistDigest(l *PackageRefList) []byte {
	// Different algorithms on PackageRefLists will sometimes return a nil slice
	// of refs and other times return an empty slice. Regardless, they should
	// both be treated identically and be given an empty digest.
	if len(l.Refs) == 0 {
		return nil
	}

	h := sha256.New()
	for _, ref := range l.Refs {
		h.Write(ref)
		h.Write([]byte{0})
	}
	return h.Sum(nil)
}

// Removes all the refs inside and replaces them with those in the given reflist
func (sl *SplitRefList) Replace(reflist *PackageRefList) {
	sl.reset()

	for _, ref := range reflist.Refs {
		idx := bucketIdxForRef(ref)

		bucket := sl.bucketRefs[idx]
		if bucket == nil {
			bucket = NewPackageRefList()
			sl.bucketRefs[idx] = bucket
		}

		bucket.Refs = append(bucket.Refs, ref)
	}

	for idx, bucket := range sl.bucketRefs {
		if bucket != nil {
			sort.Sort(bucket)
			sl.Buckets[idx] = reflistDigest(bucket)
		}
	}
}

// Merge merges reflist r into current reflist (see PackageRefList.Merge)
func (sl *SplitRefList) Merge(r *SplitRefList, overrideMatching, ignoreConflicting bool) (result *SplitRefList) {
	result = NewSplitRefList()

	var empty PackageRefList
	for idx, lbucket := range sl.bucketRefs {
		rbucket := r.bucketRefs[idx]
		if lbucket == nil && rbucket == nil {
			continue
		}

		if lbucket == nil {
			lbucket = &empty
		} else if rbucket == nil {
			rbucket = &empty
		}

		result.bucketRefs[idx] = lbucket.Merge(rbucket, overrideMatching, ignoreConflicting)
		result.Buckets[idx] = reflistDigest(result.bucketRefs[idx])
	}

	return
}

// Subtract returns all packages in l that are not in r
func (sl *SplitRefList) Subtract(r *SplitRefList) (result *SplitRefList) {
	result = NewSplitRefList()

	for idx, lbucket := range sl.bucketRefs {
		rbucket := r.bucketRefs[idx]
		if lbucket != nil {
			if rbucket != nil {
				result.bucketRefs[idx] = lbucket.Subtract(rbucket)
				result.Buckets[idx] = reflistDigest(result.bucketRefs[idx])
			} else {
				result.bucketRefs[idx] = lbucket.Clone()
				result.Buckets[idx] = sl.Buckets[idx]
			}
		}
	}

	return
}

// Diff calculates difference between two reflists
func (sl *SplitRefList) Diff(r *SplitRefList, packageCollection *PackageCollection, result PackageDiffs) (PackageDiffs, error) {
	var err error

	if result == nil {
		result = make(PackageDiffs, 0, 128)
	}

	var empty PackageRefList
	for idx, lbucket := range sl.bucketRefs {
		rbucket := r.bucketRefs[idx]
		if lbucket != nil {
			if rbucket != nil {
				result, err = lbucket.Diff(rbucket, packageCollection, result)
			} else {
				result, err = lbucket.Diff(&empty, packageCollection, result)
			}
		} else if rbucket != nil {
			result, err = empty.Diff(rbucket, packageCollection, result)
		}

		if err != nil {
			return nil, err
		}
	}

	sort.Slice(result, func(i, j int) bool {
		var ri, rj []byte
		if result[i].Left != nil {
			ri = result[i].Left.Key("")
		} else {
			ri = result[i].Right.Key("")
		}
		if result[j].Left != nil {
			rj = result[j].Left.Key("")
		} else {
			rj = result[j].Right.Key("")
		}

		return bytes.Compare(ri, rj) < 0
	})

	return result, nil
}

// FilterLatestRefs reduces a reflist to the latest of each package (see PackageRefList.FilterLatestRefs)
func (sl *SplitRefList) FilterLatestRefs() {
	for idx, bucket := range sl.bucketRefs {
		if bucket != nil {
			bucket.FilterLatestRefs()
			sl.Buckets[idx] = reflistDigest(bucket)
		}
	}
}

// Flatten creates a flat PackageRefList containing all the refs in this reflist
func (sl *SplitRefList) Flatten() *PackageRefList {
	reflist := NewPackageRefList()
	sl.ForEach(func(ref []byte) error {
		reflist.Refs = append(reflist.Refs, ref)
		return nil
	})
	sort.Sort(reflist)
	return reflist
}

// ForEachBucket calls handler for each bucket in list
func (sl *SplitRefList) ForEachBucket(handler func(digest []byte, bucket *PackageRefList) error) error {
	for idx, digest := range sl.Buckets {
		if len(digest) == 0 {
			continue
		}

		bucket := sl.bucketRefs[idx]
		if bucket != nil {
			if err := handler(digest, bucket); err != nil {
				return err
			}
		}
	}

	return nil
}

// ForEach calls handler for each package ref in list
//
// IMPORTANT: unlike PackageRefList.ForEach, the order of handler invocations
// is *not* guaranteed to be sorted.
func (sl *SplitRefList) ForEach(handler func([]byte) error) error {
	for idx, digest := range sl.Buckets {
		if len(digest) == 0 {
			continue
		}

		bucket := sl.bucketRefs[idx]
		if bucket != nil {
			if err := bucket.ForEach(handler); err != nil {
				return err
			}
		}
	}

	return nil
}

// RefListDigestSet is a set of SplitRefList bucket digests
type RefListDigestSet struct {
	items map[reflistDigestArray]struct{}
}

// NewRefListDigestSet creates empty RefListDigestSet
func NewRefListDigestSet() *RefListDigestSet {
	return &RefListDigestSet{items: map[reflistDigestArray]struct{}{}}
}

// Len returns number of digests in the set
func (set *RefListDigestSet) Len() int {
	return len(set.items)
}

// ForEach calls handler for each digest in the set
func (set *RefListDigestSet) ForEach(handler func(digest []byte) error) error {
	for digest := range set.items {
		if err := handler(digest[:]); err != nil {
			return err
		}
	}

	return nil
}

// workaround for: conversion of slices to arrays requires go1.20 or later
func newRefListArray(digest []byte) reflistDigestArray {
	var array reflistDigestArray
	copy(array[:], digest)
	return array
}

// Add adds digest to set, doing nothing if the digest was already present
func (set *RefListDigestSet) Add(digest []byte) {
	set.items[newRefListArray(digest)] = struct{}{}
}

// AddAllInRefList adds all the bucket digests in a SplitRefList to the set
func (set *RefListDigestSet) AddAllInRefList(sl *SplitRefList) {
	for _, digest := range sl.Buckets {
		if len(digest) > 0 {
			set.Add(digest)
		}
	}
}

// Has checks whether a digest is part of set
func (set *RefListDigestSet) Has(digest []byte) bool {
	_, ok := set.items[newRefListArray(digest)]
	return ok
}

// Remove removes a digest from set
func (set *RefListDigestSet) Remove(digest []byte) {
	delete(set.items, newRefListArray(digest))
}

// RemoveAll removes all the digests in other from the current set
func (set *RefListDigestSet) RemoveAll(other *RefListDigestSet) {
	for digest := range other.items {
		delete(set.items, digest)
	}
}

// RefListCollection does listing, updating/adding/deleting of SplitRefLists
type RefListCollection struct {
	db database.Storage

	cache map[reflistDigestArray]*PackageRefList
}

// NewRefListCollection creates a RefListCollection
func NewRefListCollection(db database.Storage) *RefListCollection {
	return &RefListCollection{db: db, cache: make(map[reflistDigestArray]*PackageRefList)}
}

type reflistStorageFormat int

const (
	// (legacy format) all the refs are stored inline in a single value
	reflistStorageFormatInline reflistStorageFormat = iota
	// the refs are split into buckets that are stored externally from the value
	reflistStorageFormatSplit
)

// NoPadding is used because all digests are the same length, so the padding
// is useless and only serves to muddy the output.
var bucketDigestEncoding = base64.StdEncoding.WithPadding(base64.NoPadding)

func segmentPrefix(encodedDigest string) []byte {
	return []byte(fmt.Sprintf("F%s-", encodedDigest))
}

// workaround for go 1.19 instead of bytes.Clone
func cloneBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	cloned := make([]byte, len(b))
	copy(cloned, b)
	return cloned
}

func segmentIndexKey(prefix []byte, idx int) []byte {
	// Assume most buckets won't have more than 0xFFFF = ~65k segments (which
	// would be an extremely large bucket!).
	return append(cloneBytes(prefix), []byte(fmt.Sprintf("%04x", idx))...)
}

// AllBucketDigests returns a set of all the bucket digests in the database
func (collection *RefListCollection) AllBucketDigests() (*RefListDigestSet, error) {
	digests := NewRefListDigestSet()

	err := collection.db.ProcessByPrefix([]byte("F"), func(key []byte, _ []byte) error {
		if !bytes.HasSuffix(key, []byte("-0000")) {
			// Ignore additional segments for the same digest.
			return nil
		}

		encodedDigest, _, foundDash := bytes.Cut(key[1:], []byte("-"))
		if !foundDash {
			return fmt.Errorf("invalid key: %s", string(key))
		}
		digest := make([]byte, bucketDigestEncoding.DecodedLen(len(encodedDigest)))
		if _, err := bucketDigestEncoding.Decode(digest, encodedDigest); err != nil {
			return fmt.Errorf("decoding key %s: %w", string(key), err)
		}

		digests.Add(digest)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return digests, nil
}

// UnsafeDropBucket drops the bucket associated with digest from the database,
// doing so inside batch
//
// This is considered "unsafe" because no checks are performed to ensure that
// the bucket is no longer referenced by any saved reflists.
func (collection *RefListCollection) UnsafeDropBucket(digest []byte, batch database.Batch) error {
	prefix := segmentPrefix(bucketDigestEncoding.EncodeToString(digest))
	return collection.db.ProcessByPrefix(prefix, func(key []byte, _ []byte) error {
		return batch.Delete(key)
	})
}

func (collection *RefListCollection) load(sl *SplitRefList, key []byte) (reflistStorageFormat, error) {
	sl.reset()

	data, err := collection.db.Get(key)
	if err != nil {
		return 0, err
	}

	var splitOrInlineRefList struct {
		*SplitRefList
		*PackageRefList
	}
	handle := &codec.MsgpackHandle{}
	handle.ZeroCopy = true
	decoder := codec.NewDecoderBytes(data, handle)
	if err := decoder.Decode(&splitOrInlineRefList); err != nil {
		return 0, err
	}

	if splitOrInlineRefList.SplitRefList != nil {
		sl.Buckets = splitOrInlineRefList.Buckets
	} else if splitOrInlineRefList.PackageRefList != nil {
		sl.Replace(splitOrInlineRefList.PackageRefList)
		return reflistStorageFormatInline, nil
	}

	return reflistStorageFormatSplit, nil
}

func (collection *RefListCollection) loadBuckets(sl *SplitRefList) error {
	for idx := range sl.Buckets {
		if sl.bucketRefs[idx] != nil {
			continue
		}

		var bucket *PackageRefList

		if digest := sl.Buckets[idx]; len(digest) > 0 {
			cacheKey := newRefListArray(digest)
			bucket = collection.cache[cacheKey]
			if bucket == nil {
				bucket = NewPackageRefList()
				prefix := segmentPrefix(bucketDigestEncoding.EncodeToString(digest))
				err := collection.db.ProcessByPrefix(prefix, func(_ []byte, value []byte) error {
					var l PackageRefList
					if err := l.Decode(append([]byte{}, value...)); err != nil {
						return err
					}

					bucket.Refs = append(bucket.Refs, l.Refs...)
					return nil
				})

				if err != nil {
					return err
				}

				// The segments may not have been iterated in order, so make sure to re-sort
				// here.
				sort.Sort(bucket)
				collection.cache[cacheKey] = bucket
			}

			actualDigest := reflistDigest(bucket)
			if !bytes.Equal(actualDigest, digest) {
				return fmt.Errorf("corrupt reflist bucket %d: expected digest %s, got %s",
					idx,
					bucketDigestEncoding.EncodeToString(digest),
					bucketDigestEncoding.EncodeToString(actualDigest))
			}
		}

		sl.bucketRefs[idx] = bucket
	}

	return nil
}

// LoadComplete loads the reflist stored at the given key, as well as all the
// buckets referenced by a split reflist
func (collection *RefListCollection) LoadComplete(sl *SplitRefList, key []byte) error {
	if _, err := collection.load(sl, key); err != nil {
		return err
	}

	return collection.loadBuckets(sl)
}

// RefListBatch is a wrapper over a database.Batch that tracks already-written
// reflists to avoid writing them multiple times
//
// It is *not* safe to use the same underlying database.Batch that has already
// been given to UnsafeDropBucket.
type RefListBatch struct {
	batch database.Batch

	alreadyWritten *RefListDigestSet
}

// NewBatch creates a new RefListBatch wrapping the given database.Batch
func (collection *RefListCollection) NewBatch(batch database.Batch) *RefListBatch {
	return &RefListBatch{
		batch:          batch,
		alreadyWritten: NewRefListDigestSet(),
	}
}

type reflistUpdateContext struct {
	rb    *RefListBatch
	stats *RefListMigrationStats
}

func clearSegmentRefs(reflist *PackageRefList, recommendedMaxKVSize int) {
	avgRefsInSegment := recommendedMaxKVSize / 70
	reflist.Refs = make([][]byte, 0, avgRefsInSegment)
}

func flushSegmentRefs(uctx *reflistUpdateContext, prefix []byte, segment int, reflist *PackageRefList) error {
	encoded := reflist.Encode()
	err := uctx.rb.batch.Put(segmentIndexKey(prefix, segment), encoded)
	if err == nil && uctx.stats != nil {
		uctx.stats.Segments++
	}
	return err
}

func (collection *RefListCollection) updateWithContext(sl *SplitRefList, key []byte, uctx *reflistUpdateContext) error {
	if sl != nil {
		recommendedMaxKVSize := collection.db.GetRecommendedMaxKVSize()

		for idx, digest := range sl.Buckets {
			if len(digest) == 0 {
				continue
			}

			if uctx.rb.alreadyWritten.Has(digest) {
				continue
			}

			prefix := segmentPrefix(bucketDigestEncoding.EncodeToString(digest))
			if collection.db.HasPrefix(prefix) {
				continue
			}

			// All the sizing information taken from the msgpack spec:
			// https://github.com/msgpack/msgpack/blob/master/spec.md

			// Assume that a segment will have [16,2^16) elements, which would
			// fit into an array 16 and thus have 3 bytes of overhead.
			// (A database would need a massive recommendedMaxKVSize to pass
			// that limit.)
			size := len(segmentIndexKey(prefix, 0)) + 3
			segment := 0

			var reflist PackageRefList
			clearSegmentRefs(&reflist, recommendedMaxKVSize)
			for _, ref := range sl.bucketRefs[idx].Refs {
				// In order to determine the size of the ref in the database,
				// we need to know how much overhead will be added with by msgpack
				// encoding.
				requiredSize := len(ref)
				if requiredSize < 1<<5 {
					requiredSize++
				} else if requiredSize < 1<<8 {
					requiredSize += 2
				} else if requiredSize < 1<<16 {
					requiredSize += 3
				} else {
					requiredSize += 4
				}
				if size+requiredSize > recommendedMaxKVSize {
					if err := flushSegmentRefs(uctx, prefix, segment, &reflist); err != nil {
						return err
					}
					clearSegmentRefs(&reflist, recommendedMaxKVSize)
					segment++
				}

				reflist.Refs = append(reflist.Refs, ref)
				size += requiredSize
			}

			if len(reflist.Refs) > 0 {
				if err := flushSegmentRefs(uctx, prefix, segment, &reflist); err != nil {
					return err
				}
			}

			uctx.rb.alreadyWritten.Add(digest)
			if uctx.stats != nil {
				uctx.stats.Buckets++
			}
		}
	}

	var buf bytes.Buffer
	encoder := codec.NewEncoder(&buf, &codec.MsgpackHandle{})
	encoder.Encode(sl)
	err := uctx.rb.batch.Put(key, buf.Bytes())
	if err == nil && uctx.stats != nil {
		uctx.stats.Reflists++
	}
	return err
}

// UpdateInBatch will save or update the SplitRefList at key, as well as save the buckets inside,
// as part of the given batch
func (collection *RefListCollection) UpdateInBatch(sl *SplitRefList, key []byte, batch *RefListBatch) error {
	return collection.updateWithContext(sl, key, &reflistUpdateContext{rb: batch})
}

// Update will save or update the SplitRefList at key, as well as save the buckets inside
func (collection *RefListCollection) Update(sl *SplitRefList, key []byte) error {
	rb := collection.NewBatch(collection.db.CreateBatch())
	err := collection.UpdateInBatch(sl, key, rb)
	if err == nil {
		err = rb.batch.Write()
	}
	return err
}

// RefListMigrationStats counts a number of reflists, buckets, and segments
type RefListMigrationStats struct {
	Reflists, Buckets, Segments int
}

// RefListMigration wraps a RefListBatch for the purpose of migrating inline format
// reflists to split reflists
//
// Once the batch gets too large, it will automatically be flushed to the database,
// and a new batch will be created in its place.
type RefListMigration struct {
	rb *RefListBatch

	dryRun bool

	// current number of reflists/buckets/segments queued in the current, unwritten batch
	batchStats RefListMigrationStats
	flushStats RefListMigrationStats
}

// NewMigration creates an empty RefListMigration
func (collection *RefListCollection) NewMigration() *RefListMigration {
	return &RefListMigration{}
}

// NewMigrationDryRun creates an empty RefListMigration that will track the
// changes to make as usual but avoid actually writing to the db
func (collection *RefListCollection) NewMigrationDryRun() *RefListMigration {
	return &RefListMigration{dryRun: true}
}

// Stats returns statistics on the written values in the current migration
func (migration *RefListMigration) Stats() RefListMigrationStats {
	return migration.flushStats
}

// Flush will flush the current batch in the migration to the database
func (migration *RefListMigration) Flush() error {
	if migration.batchStats.Segments > 0 {
		if !migration.dryRun {
			if err := migration.rb.batch.Write(); err != nil {
				return err
			}

			// It's important that we don't clear the batch on dry runs, because
			// the batch is what contains the list of already-written buckets.
			// If we're not writing to the database, and we clear that list,
			// duplicate "writes" will occur.
			migration.rb = nil
		}

		migration.flushStats.Reflists += migration.batchStats.Reflists
		migration.flushStats.Buckets += migration.batchStats.Buckets
		migration.flushStats.Segments += migration.batchStats.Segments
		migration.batchStats = RefListMigrationStats{}
	}

	return nil
}

// LoadCompleteAndMigrate will load the reflist and its buckets as RefListCollection.LoadComplete,
// migrating any inline reflists to split ones along the way
func (collection *RefListCollection) LoadCompleteAndMigrate(sl *SplitRefList, key []byte, migration *RefListMigration) error {
	// Given enough reflists, the memory used by a batch starts to become massive, so
	// make sure to flush the written segments periodically. Note that this is only
	// checked *after* a migration of a full bucket (and all the segments inside)
	// takes place, as splitting a single bucket write into multiple batches would
	// be unsafe if an interruption occurs midway.
	const maxMigratorBatch = 50000

	format, err := collection.load(sl, key)
	if err != nil {
		return err
	}

	switch format {
	case reflistStorageFormatInline:
		if migration.rb == nil {
			migration.rb = collection.NewBatch(collection.db.CreateBatch())
		}

		collection.updateWithContext(sl, key, &reflistUpdateContext{
			rb:    migration.rb,
			stats: &migration.batchStats,
		})

		if migration.batchStats.Segments > maxMigratorBatch {
			if err := migration.Flush(); err != nil {
				return err
			}
		}

		return nil
	case reflistStorageFormatSplit:
		return collection.loadBuckets(sl)
	default:
		panic(fmt.Sprintf("unexpected format %v", format))
	}
}

// AnyRefList is implemented by both PackageRefList and SplitRefList
type AnyRefList interface {
	Has(p *Package) bool
	Len() int
	ForEach(handler func([]byte) error) error
	FilterLatestRefs()
}

// Check interface
var (
	_ AnyRefList = (*PackageRefList)(nil)
	_ AnyRefList = (*SplitRefList)(nil)
)
