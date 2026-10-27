package deb

import (
	"bytes"
	"fmt"
	"sort"
	"testing"
)

func oldMergeInline(l, r *PackageRefList, overrideMatching, ignoreConflicting bool) (result *PackageRefList) {
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

func oldMergeSplit(sl, r *SplitRefList, overrideMatching, ignoreConflicting bool) (result *SplitRefList) {
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

		result.bucketRefs[idx] = oldMergeInline(lbucket, rbucket, overrideMatching, ignoreConflicting)
		result.Buckets[idx] = reflistDigest(result.bucketRefs[idx])
	}

	return
}

const nMergeListLen = 4096

var nMergeIdentities = []RefSetIdentity{RefSetIdentityFullKey, RefSetIdentityWithoutHash, RefSetIdentityWithoutVersion}
var nMergeDupEveryN = []int{2, 8, 128, 1024}
var nMergeListCounts = []int{2, 8, 32, 128, 512}

func benchmarkMergeCombinations(b *testing.B, cb func(*testing.B, []*PackageRefList, RefSetIdentity)) {
	for _, identity := range nMergeIdentities {
		for _, dupEveryN := range nMergeDupEveryN {
			for _, count := range nMergeListCounts {
				var identityS string
				switch identity {
				case RefSetIdentityFullKey:
					identityS = "FullKey"
				case RefSetIdentityWithoutHash:
					identityS = "WithoutHash"
				case RefSetIdentityWithoutVersion:
					identityS = "WithoutVersion"
				}

				b.Run(fmt.Sprintf("%s-DupEvery%d-Count%d", identityS, dupEveryN, count), func(b *testing.B) {
					lists := make([]*PackageRefList, count)
					for i := 0; i < count; i++ {
						lists[i] = NewPackageRefList()
					}

					for i := 0; i < nMergeListLen; i++ {
						for j := 0; j < count; j++ {
							if i%(j+1) == 0 || i%dupEveryN == 0 {
								lists[j].Refs = append(lists[j].Refs,
									[]byte(fmt.Sprintf("Pamd64 %dpackage %d 000000", i, i)))
							}
						}
					}

					for i := 0; i < count; i++ {
						sort.Sort(lists[i])
					}

					cb(b, lists, identity)
				})
			}
		}
	}
}

func BenchmarkReflistNMergeOld(b *testing.B) {
	benchmarkMergeCombinations(b, func(b *testing.B, lists []*PackageRefList, identity RefSetIdentity) {
		ignoreDuplicates := identity == RefSetIdentityFullKey
		overrideMatching := identity == RefSetIdentityWithoutVersion

		for b.Loop() {
			ret := lists[0]
			for i := 1; i < len(lists); i++ {
				ret = oldMergeInline(ret, lists[i], overrideMatching, ignoreDuplicates)
			}
		}
	})
}

func BenchmarkReflistNMergeByRefSet(b *testing.B) {
	benchmarkMergeCombinations(b, func(b *testing.B, lists []*PackageRefList, identity RefSetIdentity) {
		for b.Loop() {
			set := NewPackageRefSet(RefSetOptions{Identity: identity})
			for _, l := range lists {
				set.AddList(l)
			}
			set.ToRefList()
		}
	})
}

func BenchmarkSplitReflistNMergeOld(b *testing.B) {
	benchmarkMergeCombinations(b, func(b *testing.B, lists []*PackageRefList, identity RefSetIdentity) {
		sls := make([]*SplitRefList, 0, len(lists))
		for _, l := range lists {
			sls = append(sls, NewSplitRefListFromRefList(l))
		}

		ignoreDuplicates := identity == RefSetIdentityFullKey
		overrideMatching := identity == RefSetIdentityWithoutVersion

		for b.Loop() {
			ret := sls[0]
			for i := 1; i < len(sls); i++ {
				ret = oldMergeSplit(ret, sls[i], overrideMatching, ignoreDuplicates)
			}
		}
	})
}

func BenchmarkSplitReflistNMergeByRefSet(b *testing.B) {
	benchmarkMergeCombinations(b, func(b *testing.B, lists []*PackageRefList, identity RefSetIdentity) {
		sls := make([]*SplitRefList, 0, len(lists))
		for _, l := range lists {
			sls = append(sls, NewSplitRefListFromRefList(l))
		}

		for b.Loop() {
			set := NewSplitRefSet(RefSetOptions{Identity: identity})
			for _, sl := range sls {
				set.AddList(sl)
			}
			set.ToRefList()
		}
	})
}
