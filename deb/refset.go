package deb

import (
	"bytes"
	"fmt"
	"slices"
	"unsafe"
)

const maxRefSetBatch = 64

// RefSetIdentity specifies which parts of a ref are used to consider them equal
//
// Note that this only takes effect *across reflists*, i.e. if a single reflist
// has duplicates according to this identity, and those duplicates are not the
// exact same bytes, then they will be preserved.
type RefSetIdentity int

const (
	// RefSetIdentityFullKey uses the full reference key (arch, name, version, hash)
	RefSetIdentityFullKey RefSetIdentity = iota
	// RefSetIdentityWithoutHash uses the reference key without the hash, so that
	// only one instance of (arch, name, version) will be present
	RefSetIdentityWithoutHash
	// RefSetIdentityWithoutVersion uses the reference key without the version
	// *or* hash, so that only one instance of (arch, name) will be present
	RefSetIdentityWithoutVersion
)

type RefSetOptions struct {
	Identity RefSetIdentity
}

type PackageRefSet struct {
	options RefSetOptions

	// Map of reference keys (as per options.Identity) to indexes into 'refs'.
	// If we stored refs directly in the map keys, it would simplify things a bit,
	// but then the final created reflist would need to be sorted from random
	// order (since Go randomizes hash ordering). But if instead, we build up the
	// final unsorted reflist in 'refs', subsets of the refs will generally be
	// ordered (since it's essentially a deduplicated concatenation of
	// previously-ordered reflists), which speeds up the final sort.
	idxmap map[string]int

	// If we're using an identity *other* than the full key, then a single reflist
	// might have multiple duplicates under the shortened key, which we want to
	// preserve. In that case, the duplicates will get placed side-by-side with
	// the first key, and a counter here is incremented to measure the number of
	// duplicated entries. Then, if this is later overridden by another reflist,
	// we set the duplicated entries to 'nil', so they can be easily removed at
	// the very end.
	idxdups map[string]int

	refs [][]byte

	// We batch together multiple lists to be merged, because merging them
	// one-by-one causes substantial performance loss (likely due to poor cache
	// efficiency?).
	pendingBatch [][][]byte
}

// NewPackageRefSet creates a new PackageRefSet with the given options
func NewPackageRefSet(options RefSetOptions) *PackageRefSet {
	rs := &PackageRefSet{
		options: options,
		idxmap:  make(map[string]int, 1024),
		refs:    make([][]byte, 0, 1024),

		pendingBatch: make([][][]byte, 0, maxRefSetBatch),
	}

	if options.Identity != RefSetIdentityFullKey {
		rs.idxdups = make(map[string]int)
	}

	return rs
}

// AddList adds the refs in the given reflist to the set
//
// If any refs are already present, according to the configured RefSetIdentity,
// they will be replaced by the new refs.
func (rs *PackageRefSet) AddList(refs *PackageRefList) {
	rs.pendingBatch = append(rs.pendingBatch, refs.Refs)
	if len(rs.pendingBatch) >= maxRefSetBatch {
		rs.flush()
	}
}

func (rs *PackageRefSet) shortenRefToKey(ref []byte) []byte {
	if rs.options.Identity == RefSetIdentityFullKey {
		return ref
	}

	// Cut out the hash.
	ref = ref[:bytes.LastIndexByte(ref, ' ')]
	if rs.options.Identity == RefSetIdentityWithoutHash {
		return ref
	}

	// Cut out the version.
	ref = ref[:bytes.LastIndexByte(ref, ' ')]
	if rs.options.Identity == RefSetIdentityWithoutVersion {
		return ref
	}

	panic(fmt.Sprintf("unexpected identity: %d", rs.options.Identity))
}

func (rs *PackageRefSet) flush() {
	for _, refs := range rs.pendingBatch {
		var prevRef []byte
		var prevKey []byte

		for _, ref := range refs {
			k := rs.shortenRefToKey(ref)
			// When indexing a map for retrieval only, we can use string(byteSlice),
			// which the compiler will optimize to avoid an excess copy:
			// https://github.com/golang/go/commit/f5f5a8b6209f84961687d993b93ea0d397f5d5bf
			// However, for *assignment*, that doesn't work, because the compiler
			// can't guarantee that the byte slice won't be modified while it's stored
			// in the map.
			//
			// But for our reflist use case, we *can* guarantee that, so constructing
			// a string from the byte slice for key use helps avoid excessive copies.
			ks := unsafe.String(unsafe.SliceData(k), len(k))

			if rs.options.Identity != RefSetIdentityFullKey && bytes.Equal(k, prevKey) {
				if !bytes.Equal(ref, prevRef) {
					// Two "duplicate" references according to a non-full-ref identity, so
					// stick them side-by-side.
					i := rs.idxmap[ks]
					dupcount := rs.idxdups[ks]

					prevOccurrenceInMiddleOfRefs := i < len(rs.refs)-dupcount-1
					if prevOccurrenceInMiddleOfRefs && rs.refs[i+dupcount+1] == nil {
						rs.refs[i+dupcount+1] = ref
					} else {
						if prevOccurrenceInMiddleOfRefs {
							// This list's previous ref overrode one from a prior reflist
							// somewhere in the middle, and there's no room after it for this
							// ref, so we need to move the whole run to the end to have room.
							//
							// (This can result in some extra intermediate nils scattered around
							// the middle, but that shouldn't be too many in practice. In the
							// future, if that does become a problem, it might be worth instead
							// trying to shift over the ref that follows this one.)
							rs.idxmap[ks] = len(rs.refs)

							run := rs.refs[i : i+dupcount+1]
							rs.refs = append(rs.refs, run...)
							rs.refs[i] = nil
							for j := range dupcount {
								rs.refs[i+1+j] = nil
							}
						}

						rs.refs = append(rs.refs, ref)
					}

					rs.idxdups[ks] += 1
				}
			} else if i, ok := rs.idxmap[ks]; ok {
				rs.refs[i] = ref

				if rs.options.Identity != RefSetIdentityFullKey {
					if dupcount := rs.idxdups[ks]; dupcount > 0 {
						for j := i + 1; j <= i+dupcount; j++ {
							rs.refs[j] = nil
						}

						delete(rs.idxdups, ks)
					}
				}
			} else {
				rs.idxmap[ks] = len(rs.refs)
				rs.refs = append(rs.refs, ref)
			}

			prevRef = ref
			prevKey = k
		}
	}

	rs.pendingBatch = rs.pendingBatch[:0]
}

// HasMatch returns true if the set contains a ref that matches the given one,
// according to the configured RefSetIdentity
func (rs *PackageRefSet) HasMatch(ref []byte) bool {
	rs.flush()
	k := rs.shortenRefToKey(ref)
	_, ok := rs.idxmap[string(k)]
	return ok
}

func sortRefSetGeneratedRefs(l *PackageRefList) {
	slices.SortFunc(l.Refs, func(a, b []byte) int {
		return bytes.Compare(a, b)
	})
	// Overridden refs end up as 'nil' which should be at the front of the
	// resulting reflist, so filter them out now.
	for len(l.Refs) > 0 && len(l.Refs[0]) == 0 {
		l.Refs = l.Refs[1:]
	}
}

// ToRefList returns a sorted PackageRefList containing all the refs in the set
//
// This will not clear the state of the set and thus may be safely called multiple times.
func (rs *PackageRefSet) ToRefList() *PackageRefList {
	rs.flush()
	result := &PackageRefList{Refs: slices.Clone(rs.refs)}
	sortRefSetGeneratedRefs(result)
	return result
}

// SplitRefSet is a PackageRefSet split into buckets, akin to SplitRefList
type SplitRefSet struct {
	options RefSetOptions
	buckets []*PackageRefSet

	knownDigests *RefListDigestSet
	lastDigests  [][]byte
}

// NewSplitRefSet creates a new SplitRefSet with the given options
func NewSplitRefSet(options RefSetOptions) *SplitRefSet {
	buckets := make([]*PackageRefSet, reflistBucketCount)
	for i := range buckets {
		buckets[i] = NewPackageRefSet(options)
	}
	srs := &SplitRefSet{options: options, buckets: buckets}

	if options.Identity == RefSetIdentityFullKey {
		srs.knownDigests = NewRefListDigestSet()
	} else {
		srs.lastDigests = make([][]byte, reflistBucketCount)
	}

	return srs
}

// AddList adds the refs from the given SplitRefList to the set
func (srs *SplitRefSet) AddList(sl *SplitRefList) {
	for i, bucket := range sl.bucketRefs {
		digest := sl.Buckets[i]
		if len(digest) == 0 || bucket == nil {
			continue
		}

		if srs.options.Identity == RefSetIdentityFullKey {
			// If we're using the full ref as the identity, then order of invoking
			// AddList does not matter, so if we've ever seen this bucket before, we
			// know that the refs inside are present, and re-adding them is a no-op.
			if srs.knownDigests.Has(digest) {
				continue
			}
			srs.knownDigests.Add(digest)
		} else {
			// When we're *not* using the full ref as the identity, order *does*
			// matter, e.g. adding the SplitRefLists A, B, A in sequence would result
			// in first any matches in A being overridden by B, but then those same
			// matches being overriden by A again.
			//
			// However, we can still check the most *recent* digest, since adding the
			// same bucket twice in a row remains a no-op.
			if bytes.Equal(srs.lastDigests[i], digest) {
				continue
			}
			srs.lastDigests[i] = digest
		}

		srs.buckets[i].AddList(bucket)
	}
}

// HasMatch returns true if the set contains a ref that matches the given one,
// according to the configured RefSetIdentity
func (srs *SplitRefSet) HasMatch(ref []byte) bool {
	i := bucketIdxForRef(ref)
	return srs.buckets[i].HasMatch(ref)
}

// ToRefList returns a SplitRefList containing all the refs in the set
//
// This will not clear the state of the set and thus may be safely called multiple times.
func (srs *SplitRefSet) ToRefList() *SplitRefList {
	result := NewSplitRefList()
	for i, rs := range srs.buckets {
		rs.flush()
		if len(rs.refs) == 0 {
			continue
		}

		result.bucketRefs[i] = rs.ToRefList()
		result.Buckets[i] = reflistDigest(result.bucketRefs[i])
	}
	return result
}

// ToFlattenedRefList returns a flattened PackageRefList containing all the refs in the set
//
// This is equivalent to ToRefList().Flatten().
func (srs *SplitRefSet) ToFlattenedRefList() *PackageRefList {
	result := &PackageRefList{Refs: make([][]byte, 0, 128)}
	for _, rs := range srs.buckets {
		rs.flush()
		result.Refs = append(result.Refs, rs.refs...)
	}
	sortRefSetGeneratedRefs(result)
	return result
}
