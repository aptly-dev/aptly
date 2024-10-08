package deb

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/aptly-dev/aptly/database/goleveldb"

	. "gopkg.in/check.v1"
)

type PackageRefListSuite struct {
	p1, p2, p3, p4, p5, p6 *Package
}

var _ = Suite(&PackageRefListSuite{})

func verifyRefListIntegrity(c *C, rl AnyRefList) AnyRefList {
	if rl, ok := rl.(*SplitRefList); ok {
		for idx, bucket := range rl.bucketRefs {
			if bucket == nil {
				bucket = NewPackageRefList()
			}
			c.Check(rl.Buckets[idx], DeepEquals, reflistDigest(bucket))
		}
	}

	return rl
}

func getRefs(rl AnyRefList) (refs [][]byte) {
	switch rl := rl.(type) {
	case *PackageRefList:
		refs = rl.Refs
	case *SplitRefList:
		refs = rl.Flatten().Refs
	default:
		panic(fmt.Sprintf("unexpected reflist type %t", rl))
	}

	// Hack so that passing getRefs-returned slices to DeepEquals won't fail given a nil
	// slice and an empty slice.
	if len(refs) == 0 {
		refs = nil
	}
	return
}

func toStrSlice(reflist AnyRefList) (result []string) {
	result = make([]string, reflist.Len())
	for i, r := range getRefs(reflist) {
		result[i] = string(r)
	}
	return
}

type reflistFactory struct {
	new                func() AnyRefList
	newFromRefs        func(refs ...[]byte) AnyRefList
	newFromPackageList func(list *PackageList) AnyRefList
}

func forEachRefList(test func(f reflistFactory)) {
	test(reflistFactory{
		new: func() AnyRefList {
			return NewPackageRefList()
		},
		newFromRefs: func(refs ...[]byte) AnyRefList {
			return &PackageRefList{Refs: refs}
		},
		newFromPackageList: func(list *PackageList) AnyRefList {
			return NewPackageRefListFromPackageList(list)
		},
	})

	test(reflistFactory{
		new: func() AnyRefList {
			return NewSplitRefList()
		},
		newFromRefs: func(refs ...[]byte) AnyRefList {
			return NewSplitRefListFromRefList(&PackageRefList{Refs: refs})
		},
		newFromPackageList: func(list *PackageList) AnyRefList {
			return NewSplitRefListFromPackageList(list)
		},
	})
}

func (s *PackageRefListSuite) SetUpTest(c *C) {
	s.p1 = NewPackageFromControlFile(packageStanza.Copy())
	s.p2 = NewPackageFromControlFile(packageStanza.Copy())
	stanza := packageStanza.Copy()
	stanza["Package"] = "mars-invaders"
	s.p3 = NewPackageFromControlFile(stanza)
	stanza = packageStanza.Copy()
	stanza["Source"] = "unknown-planet"
	s.p4 = NewPackageFromControlFile(stanza)
	stanza = packageStanza.Copy()
	stanza["Package"] = "lonely-strangers"
	s.p5 = NewPackageFromControlFile(stanza)
	stanza = packageStanza.Copy()
	stanza["Version"] = "99.1"
	s.p6 = NewPackageFromControlFile(stanza)
}

func (s *PackageRefListSuite) TestNewPackageListFromRefList(c *C) {
	forEachRefList(func(f reflistFactory) {
		list := NewPackageList()

		db, _ := goleveldb.NewOpenDB(c.MkDir())
		coll := NewPackageCollection(db)
		coll.Update(s.p1)
		coll.Update(s.p3)

		list.Add(s.p1)
		list.Add(s.p3)
		list.Add(s.p5)
		list.Add(s.p6)

		reflist := f.newFromPackageList(list)

		_, err := NewPackageListFromRefList(reflist, coll, nil)
		c.Assert(err, ErrorMatches, "unable to load package with key.*")

		coll.Update(s.p5)
		coll.Update(s.p6)

		list, err = NewPackageListFromRefList(reflist, coll, nil)
		c.Assert(err, IsNil)
		c.Check(list.Len(), Equals, 4)
		c.Check(list.Add(s.p4), ErrorMatches, "conflict in package.*")

		list, err = NewPackageListFromRefList(nil, coll, nil)
		c.Assert(err, IsNil)
		c.Check(list.Len(), Equals, 0)
	})
}

func (s *PackageRefListSuite) TestNewPackageRefList(c *C) {
	forEachRefList(func(f reflistFactory) {
		list := NewPackageList()
		list.Add(s.p1)
		list.Add(s.p3)
		list.Add(s.p5)
		list.Add(s.p6)

		reflist := f.newFromPackageList(list)
		verifyRefListIntegrity(c, reflist)
		c.Assert(reflist.Len(), Equals, 4)
		refs := getRefs(reflist)
		c.Check(refs[0], DeepEquals, []byte(s.p1.Key("")))
		c.Check(refs[1], DeepEquals, []byte(s.p6.Key("")))
		c.Check(refs[2], DeepEquals, []byte(s.p5.Key("")))
		c.Check(refs[3], DeepEquals, []byte(s.p3.Key("")))

		reflist = f.new()
		c.Check(reflist.Len(), Equals, 0)
	})
}

func (s *PackageRefListSuite) TestPackageRefListForeach(c *C) {
	forEachRefList(func(f reflistFactory) {
		list := NewPackageList()
		list.Add(s.p1)
		list.Add(s.p3)
		list.Add(s.p5)
		list.Add(s.p6)

		reflist := f.newFromPackageList(list)

		Len := 0
		err := reflist.ForEach(func([]byte) error {
			Len++
			return nil
		})

		c.Check(Len, Equals, 4)
		c.Check(err, IsNil)

		e := errors.New("b")

		err = reflist.ForEach(func([]byte) error {
			return e
		})

		c.Check(err, Equals, e)
	})
}

func (s *PackageRefListSuite) TestHas(c *C) {
	forEachRefList(func(f reflistFactory) {
		list := NewPackageList()
		list.Add(s.p1)
		list.Add(s.p3)
		list.Add(s.p5)
		reflist := f.newFromPackageList(list)

		c.Check(reflist.Has(s.p1), Equals, true)
		c.Check(reflist.Has(s.p3), Equals, true)
		c.Check(reflist.Has(s.p5), Equals, true)
		c.Check(reflist.Has(s.p2), Equals, true)
		c.Check(reflist.Has(s.p6), Equals, false)
	})
}

func subtractRefLists(l, r AnyRefList) AnyRefList {
	switch l := l.(type) {
	case *PackageRefList:
		return l.Subtract(r.(*PackageRefList))
	case *SplitRefList:
		return l.Subtract(r.(*SplitRefList))
	default:
		panic(fmt.Sprintf("unexpected reflist type %t", l))
	}
}

func (s *PackageRefListSuite) TestSubtract(c *C) {
	forEachRefList(func(f reflistFactory) {
		r1 := []byte("Pall r1")
		r2 := []byte("Pall r2")
		r3 := []byte("Pall r3")
		r4 := []byte("Pall r4")
		r5 := []byte("Pall r5")

		empty := f.newFromRefs()
		l1 := f.newFromRefs(r1, r2, r3, r4)
		l2 := f.newFromRefs(r1, r3)
		l3 := f.newFromRefs(r2, r4)
		l4 := f.newFromRefs(r4, r5)
		l5 := f.newFromRefs(r1, r2, r3)

		c.Check(getRefs(subtractRefLists(l1, empty)), DeepEquals, getRefs(l1))
		c.Check(getRefs(subtractRefLists(l1, l2)), DeepEquals, getRefs(l3))
		c.Check(getRefs(subtractRefLists(l1, l3)), DeepEquals, getRefs(l2))
		c.Check(getRefs(subtractRefLists(l1, l4)), DeepEquals, getRefs(l5))
		c.Check(getRefs(subtractRefLists(empty, l1)), DeepEquals, getRefs(empty))
		c.Check(getRefs(subtractRefLists(l2, l3)), DeepEquals, getRefs(l2))
	})
}

func diffRefLists(l, r AnyRefList, packageCollection *PackageCollection) (PackageDiffs, error) {
	switch l := l.(type) {
	case *PackageRefList:
		return l.Diff(r.(*PackageRefList), packageCollection, nil)
	case *SplitRefList:
		return l.Diff(r.(*SplitRefList), packageCollection, nil)
	default:
		panic(fmt.Sprintf("unexpected reflist type %t", l))
	}
}

func (s *PackageRefListSuite) TestDiff(c *C) {
	forEachRefList(func(f reflistFactory) {
		db, _ := goleveldb.NewOpenDB(c.MkDir())
		coll := NewPackageCollection(db)

		packages := []*Package{
			{Name: "lib", Version: "1.0", Architecture: "i386"},      //0
			{Name: "dpkg", Version: "1.7", Architecture: "i386"},     //1
			{Name: "data", Version: "1.1~bp1", Architecture: "all"},  //2
			{Name: "app", Version: "1.1~bp1", Architecture: "i386"},  //3
			{Name: "app", Version: "1.1~bp2", Architecture: "i386"},  //4
			{Name: "app", Version: "1.1~bp2", Architecture: "amd64"}, //5
			{Name: "xyz", Version: "3.0", Architecture: "sparc"},     //6
		}

		for _, p := range packages {
			coll.Update(p)
		}

		listA := NewPackageList()
		listA.Add(packages[0])
		listA.Add(packages[1])
		listA.Add(packages[2])
		listA.Add(packages[3])
		listA.Add(packages[6])

		listB := NewPackageList()
		listB.Add(packages[0])
		listB.Add(packages[2])
		listB.Add(packages[4])
		listB.Add(packages[5])

		reflistA := f.newFromPackageList(listA)
		reflistB := f.newFromPackageList(listB)

		diffAA, err := diffRefLists(reflistA, reflistA, coll)
		c.Check(err, IsNil)
		c.Check(diffAA, HasLen, 0)

		diffAB, err := diffRefLists(reflistA, reflistB, coll)
		c.Check(err, IsNil)
		c.Check(diffAB, HasLen, 4)

		c.Check(diffAB[0].Left, IsNil)
		c.Check(diffAB[0].Right.String(), Equals, "app_1.1~bp2_amd64")

		c.Check(diffAB[1].Left.String(), Equals, "app_1.1~bp1_i386")
		c.Check(diffAB[1].Right.String(), Equals, "app_1.1~bp2_i386")

		c.Check(diffAB[2].Left.String(), Equals, "dpkg_1.7_i386")
		c.Check(diffAB[2].Right, IsNil)

		c.Check(diffAB[3].Left.String(), Equals, "xyz_3.0_sparc")
		c.Check(diffAB[3].Right, IsNil)

		diffBA, err := diffRefLists(reflistB, reflistA, coll)
		c.Check(err, IsNil)
		c.Check(diffBA, HasLen, 4)

		c.Check(diffBA[0].Right, IsNil)
		c.Check(diffBA[0].Left.String(), Equals, "app_1.1~bp2_amd64")

		c.Check(diffBA[1].Right.String(), Equals, "app_1.1~bp1_i386")
		c.Check(diffBA[1].Left.String(), Equals, "app_1.1~bp2_i386")

		c.Check(diffBA[2].Right.String(), Equals, "dpkg_1.7_i386")
		c.Check(diffBA[2].Left, IsNil)

		c.Check(diffBA[3].Right.String(), Equals, "xyz_3.0_sparc")
		c.Check(diffBA[3].Left, IsNil)
	})
}

func (s *PackageRefListSuite) TestDiffCompactsAtEnd(c *C) {
	forEachRefList(func(f reflistFactory) {
		db, _ := goleveldb.NewOpenDB(c.MkDir())
		coll := NewPackageCollection(db)

		packages := []*Package{
			{Name: "app", Version: "1.1~bp1", Architecture: "i386"},  //0
			{Name: "app", Version: "1.1~bp2", Architecture: "i386"},  //1
			{Name: "app", Version: "1.1~bp2", Architecture: "amd64"}, //2
		}

		for _, p := range packages {
			coll.Update(p)
		}

		listA := NewPackageList()
		listA.Add(packages[0])

		listB := NewPackageList()
		listB.Add(packages[1])
		listB.Add(packages[2])

		reflistA := f.newFromPackageList(listA)
		reflistB := f.newFromPackageList(listB)

		diffAB, err := diffRefLists(reflistA, reflistB, coll)
		c.Check(err, IsNil)
		c.Check(diffAB, HasLen, 2)

		c.Check(diffAB[0].Left, IsNil)
		c.Check(diffAB[0].Right.String(), Equals, "app_1.1~bp2_amd64")

		c.Check(diffAB[1].Left.String(), Equals, "app_1.1~bp1_i386")
		c.Check(diffAB[1].Right.String(), Equals, "app_1.1~bp2_i386")
	})
}

func mergeRefLists(l, r AnyRefList, overrideMatching, ignoreConflicting bool) AnyRefList {
	switch l := l.(type) {
	case *PackageRefList:
		return l.Merge(r.(*PackageRefList), overrideMatching, ignoreConflicting)
	case *SplitRefList:
		return l.Merge(r.(*SplitRefList), overrideMatching, ignoreConflicting)
	default:
		panic(fmt.Sprintf("unexpected reflist type %t", l))
	}
}

func (s *PackageRefListSuite) TestMerge(c *C) {
	forEachRefList(func(f reflistFactory) {
		db, _ := goleveldb.NewOpenDB(c.MkDir())
		coll := NewPackageCollection(db)

		packages := []*Package{
			{Name: "lib", Version: "1.0", Architecture: "i386"},                      //0
			{Name: "dpkg", Version: "1.7", Architecture: "i386"},                     //1
			{Name: "data", Version: "1.1~bp1", Architecture: "all"},                  //2
			{Name: "app", Version: "1.1~bp1", Architecture: "i386"},                  //3
			{Name: "app", Version: "1.1~bp2", Architecture: "i386"},                  //4
			{Name: "app", Version: "1.1~bp2", Architecture: "amd64"},                 //5
			{Name: "dpkg", Version: "1.0", Architecture: "i386"},                     //6
			{Name: "xyz", Version: "1.0", Architecture: "sparc"},                     //7
			{Name: "dpkg", Version: "1.0", Architecture: "i386", FilesHash: 0x34445}, //8
			{Name: "app", Version: "1.1~bp2", Architecture: "i386", FilesHash: 0x44}, //9
		}

		for _, p := range packages {
			p.V06Plus = true
			coll.Update(p)
		}

		listA := NewPackageList()
		listA.Add(packages[0])
		listA.Add(packages[1])
		listA.Add(packages[2])
		listA.Add(packages[3])
		listA.Add(packages[7])

		listB := NewPackageList()
		listB.Add(packages[0])
		listB.Add(packages[2])
		listB.Add(packages[4])
		listB.Add(packages[5])
		listB.Add(packages[6])

		listC := NewPackageList()
		listC.Add(packages[0])
		listC.Add(packages[8])
		listC.Add(packages[9])

		reflistA := f.newFromPackageList(listA)
		reflistB := f.newFromPackageList(listB)
		reflistC := f.newFromPackageList(listC)

		mergeAB := mergeRefLists(reflistA, reflistB, true, false)
		mergeBA := mergeRefLists(reflistB, reflistA, true, false)
		mergeAC := mergeRefLists(reflistA, reflistC, true, false)
		mergeBC := mergeRefLists(reflistB, reflistC, true, false)
		mergeCB := mergeRefLists(reflistC, reflistB, true, false)

		verifyRefListIntegrity(c, mergeAB)
		verifyRefListIntegrity(c, mergeBA)
		verifyRefListIntegrity(c, mergeAC)
		verifyRefListIntegrity(c, mergeBC)
		verifyRefListIntegrity(c, mergeCB)

		c.Check(toStrSlice(mergeAB), DeepEquals,
			[]string{"Pall data 1.1~bp1 00000000", "Pamd64 app 1.1~bp2 00000000", "Pi386 app 1.1~bp2 00000000", "Pi386 dpkg 1.0 00000000", "Pi386 lib 1.0 00000000", "Psparc xyz 1.0 00000000"})
		c.Check(toStrSlice(mergeBA), DeepEquals,
			[]string{"Pall data 1.1~bp1 00000000", "Pamd64 app 1.1~bp2 00000000", "Pi386 app 1.1~bp1 00000000", "Pi386 dpkg 1.7 00000000", "Pi386 lib 1.0 00000000", "Psparc xyz 1.0 00000000"})
		c.Check(toStrSlice(mergeAC), DeepEquals,
			[]string{"Pall data 1.1~bp1 00000000", "Pi386 app 1.1~bp2 00000044", "Pi386 dpkg 1.0 00034445", "Pi386 lib 1.0 00000000", "Psparc xyz 1.0 00000000"})
		c.Check(toStrSlice(mergeBC), DeepEquals,
			[]string{"Pall data 1.1~bp1 00000000", "Pamd64 app 1.1~bp2 00000000", "Pi386 app 1.1~bp2 00000044", "Pi386 dpkg 1.0 00034445", "Pi386 lib 1.0 00000000"})
		c.Check(toStrSlice(mergeCB), DeepEquals,
			[]string{"Pall data 1.1~bp1 00000000", "Pamd64 app 1.1~bp2 00000000", "Pi386 app 1.1~bp2 00000000", "Pi386 dpkg 1.0 00000000", "Pi386 lib 1.0 00000000"})

		mergeABall := mergeRefLists(reflistA, reflistB, false, false)
		mergeBAall := mergeRefLists(reflistB, reflistA, false, false)
		mergeACall := mergeRefLists(reflistA, reflistC, false, false)
		mergeBCall := mergeRefLists(reflistB, reflistC, false, false)
		mergeCBall := mergeRefLists(reflistC, reflistB, false, false)

		verifyRefListIntegrity(c, mergeABall)
		verifyRefListIntegrity(c, mergeBAall)
		verifyRefListIntegrity(c, mergeACall)
		verifyRefListIntegrity(c, mergeBCall)
		verifyRefListIntegrity(c, mergeCBall)

		c.Check(mergeABall, DeepEquals, mergeBAall)
		c.Check(toStrSlice(mergeBAall), DeepEquals,
			[]string{"Pall data 1.1~bp1 00000000", "Pamd64 app 1.1~bp2 00000000", "Pi386 app 1.1~bp1 00000000", "Pi386 app 1.1~bp2 00000000",
				"Pi386 dpkg 1.0 00000000", "Pi386 dpkg 1.7 00000000", "Pi386 lib 1.0 00000000", "Psparc xyz 1.0 00000000"})

		c.Check(mergeBCall, Not(DeepEquals), mergeCBall)
		c.Check(toStrSlice(mergeACall), DeepEquals,
			[]string{"Pall data 1.1~bp1 00000000", "Pi386 app 1.1~bp1 00000000", "Pi386 app 1.1~bp2 00000044", "Pi386 dpkg 1.0 00034445",
				"Pi386 dpkg 1.7 00000000", "Pi386 lib 1.0 00000000", "Psparc xyz 1.0 00000000"})
		c.Check(toStrSlice(mergeBCall), DeepEquals,
			[]string{"Pall data 1.1~bp1 00000000", "Pamd64 app 1.1~bp2 00000000", "Pi386 app 1.1~bp2 00000044", "Pi386 dpkg 1.0 00034445",
				"Pi386 lib 1.0 00000000"})

		mergeBCwithConflicts := mergeRefLists(reflistB, reflistC, false, true)
		c.Check(toStrSlice(mergeBCwithConflicts), DeepEquals,
			[]string{"Pall data 1.1~bp1 00000000", "Pamd64 app 1.1~bp2 00000000", "Pi386 app 1.1~bp2 00000000", "Pi386 app 1.1~bp2 00000044",
				"Pi386 dpkg 1.0 00000000", "Pi386 dpkg 1.0 00034445", "Pi386 lib 1.0 00000000"})
	})
}

func (s *PackageRefListSuite) TestFilterLatestRefs(c *C) {
	forEachRefList(func(f reflistFactory) {
		packages := []*Package{
			{Name: "lib", Version: "1.0", Architecture: "i386"},
			{Name: "lib", Version: "1.2~bp1", Architecture: "i386"},
			{Name: "lib", Version: "1.2", Architecture: "i386"},
			{Name: "dpkg", Version: "1.2", Architecture: "i386"},
			{Name: "dpkg", Version: "1.3", Architecture: "i386"},
			{Name: "dpkg", Version: "1.3~bp2", Architecture: "i386"},
			{Name: "dpkg", Version: "1.5", Architecture: "i386"},
			{Name: "dpkg", Version: "1.6", Architecture: "i386"},
		}

		rl := NewPackageList()
		rl.Add(packages[0])
		rl.Add(packages[1])
		rl.Add(packages[2])
		rl.Add(packages[3])
		rl.Add(packages[4])
		rl.Add(packages[5])
		rl.Add(packages[6])
		rl.Add(packages[7])

		result := f.newFromPackageList(rl)
		result.FilterLatestRefs()

		verifyRefListIntegrity(c, result)
		c.Check(toStrSlice(result), DeepEquals,
			[]string{"Pi386 dpkg 1.6", "Pi386 lib 1.2"})
	})
}

func (s *PackageRefListSuite) TestPackageRefListEncodeDecode(c *C) {
	list := NewPackageList()
	list.Add(s.p1)
	list.Add(s.p3)
	list.Add(s.p5)
	list.Add(s.p6)

	reflist := NewPackageRefListFromPackageList(list)

	reflist2 := &PackageRefList{}
	err := reflist2.Decode(reflist.Encode())
	c.Assert(err, IsNil)
	c.Check(reflist2.Len(), Equals, reflist.Len())
	c.Check(reflist2.Refs, DeepEquals, reflist.Refs)
}

func (s *PackageRefListSuite) TestRefListBucketPrefix(c *C) {
	c.Check(bucketRefPrefix([]byte("Pall abcd 1.0")), DeepEquals, []byte("abc"))
	c.Check(bucketRefPrefix([]byte("Pall libabcd 1.0")), DeepEquals, []byte("abc"))
	c.Check(bucketRefPrefix([]byte("Pamd64 xy 1.0")), DeepEquals, []byte("xy"))
}

func (s *PackageRefListSuite) TestRefListBucketIdx(c *C) {
	c.Check(bucketIdxForRef(s.p1.Key("")), Equals, 46)
	c.Check(bucketIdxForRef(s.p2.Key("")), Equals, 46)
	c.Check(bucketIdxForRef(s.p3.Key("")), Equals, 26)
	c.Check(bucketIdxForRef(s.p4.Key("")), Equals, 46)
	c.Check(bucketIdxForRef(s.p5.Key("")), Equals, 4)
	c.Check(bucketIdxForRef(s.p6.Key("")), Equals, 46)
}

func (s *PackageRefListSuite) TestSplitRefListBuckets(c *C) {
	list := NewPackageList()
	list.Add(s.p1)
	list.Add(s.p3)
	list.Add(s.p4)
	list.Add(s.p5)
	list.Add(s.p6)

	sl := NewSplitRefListFromPackageList(list)
	verifyRefListIntegrity(c, sl)

	c.Check(hex.EncodeToString(sl.Buckets[4]), Equals, "7287aed32daad5d1aab4e89533bde135381d932e60548cfc00b882ca8858ae07")
	c.Check(toStrSlice(sl.bucketRefs[4]), DeepEquals, []string{string(s.p5.Key(""))})
	c.Check(hex.EncodeToString(sl.Buckets[26]), Equals, "f31fc28e82368b63c8be47eefc64b8e217e2e5349c7e3827b98f80536b956f6e")
	c.Check(toStrSlice(sl.bucketRefs[26]), DeepEquals, []string{string(s.p3.Key(""))})
	c.Check(hex.EncodeToString(sl.Buckets[46]), Equals, "55e70286393afc5da5046d68c632d35f98bec24781ae433bd1a1069b52853367")
	c.Check(toStrSlice(sl.bucketRefs[46]), DeepEquals, []string{string(s.p1.Key("")), string(s.p6.Key(""))})
}

func (s *PackageRefListSuite) TestRefListDigestSet(c *C) {
	list := NewPackageList()
	list.Add(s.p1)
	list.Add(s.p3)
	list.Add(s.p4)
	list.Add(s.p5)
	list.Add(s.p6)

	sl := NewSplitRefListFromPackageList(list)

	set := NewRefListDigestSet()
	c.Check(set.Len(), Equals, 0)

	err := sl.ForEachBucket(func(digest []byte, bucket *PackageRefList) error {
		c.Check(set.Has(digest), Equals, false)
		return nil
	})
	c.Assert(err, IsNil)

	set.AddAllInRefList(sl)
	c.Check(set.Len(), Equals, 3)

	err = sl.ForEachBucket(func(digest []byte, bucket *PackageRefList) error {
		c.Check(set.Has(digest), Equals, true)
		return nil
	})
	c.Assert(err, IsNil)

	firstDigest := sl.Buckets[bucketIdxForRef(s.p1.Key(""))]
	set.Remove(firstDigest)
	c.Check(set.Len(), Equals, 2)

	err = sl.ForEachBucket(func(digest []byte, bucket *PackageRefList) error {
		c.Check(set.Has(digest), Equals, !bytes.Equal(digest, firstDigest))
		return nil
	})
	c.Assert(err, IsNil)

	set2 := NewRefListDigestSet()
	set2.AddAllInRefList(sl)
	set2.RemoveAll(set)

	err = sl.ForEachBucket(func(digest []byte, bucket *PackageRefList) error {
		c.Check(set2.Has(digest), Equals, bytes.Equal(digest, firstDigest))
		return nil
	})
	c.Assert(err, IsNil)
}

func (s *PackageRefListSuite) TestRefListCollectionLoadSave(c *C) {
	db, _ := goleveldb.NewOpenDB(c.MkDir())
	reflistCollection := NewRefListCollection(db)
	packageCollection := NewPackageCollection(db)

	packageCollection.Update(s.p1)
	packageCollection.Update(s.p2)
	packageCollection.Update(s.p3)
	packageCollection.Update(s.p4)
	packageCollection.Update(s.p5)
	packageCollection.Update(s.p6)

	list := NewPackageList()
	list.Add(s.p1)
	list.Add(s.p2)
	list.Add(s.p3)
	list.Add(s.p4)
	list.Add(s.p5)

	key := []byte("test")

	reflist := NewPackageRefListFromPackageList(list)
	db.Put(key, reflist.Encode())

	sl := NewSplitRefList()
	err := reflistCollection.LoadComplete(sl, key)
	c.Assert(err, IsNil)
	verifyRefListIntegrity(c, sl)
	c.Check(toStrSlice(sl), DeepEquals, toStrSlice(reflist))

	list.Add(s.p6)
	sl = NewSplitRefListFromPackageList(list)
	err = reflistCollection.Update(sl, key)
	c.Assert(err, IsNil)

	sl = NewSplitRefList()
	err = reflistCollection.LoadComplete(sl, key)
	c.Assert(err, IsNil)
	verifyRefListIntegrity(c, sl)
	c.Check(toStrSlice(sl), DeepEquals, toStrSlice(NewPackageRefListFromPackageList(list)))
}

func (s *PackageRefListSuite) TestRefListCollectionMigrate(c *C) {
	db, _ := goleveldb.NewOpenDB(c.MkDir())
	reflistCollection := NewRefListCollection(db)
	packageCollection := NewPackageCollection(db)

	packageCollection.Update(s.p1)
	packageCollection.Update(s.p2)
	packageCollection.Update(s.p3)
	packageCollection.Update(s.p4)
	packageCollection.Update(s.p5)
	packageCollection.Update(s.p6)

	list := NewPackageList()
	list.Add(s.p1)
	list.Add(s.p2)
	list.Add(s.p3)
	list.Add(s.p4)
	list.Add(s.p5)

	key := []byte("test")

	reflist := NewPackageRefListFromPackageList(list)
	db.Put(key, reflist.Encode())

	sl := NewSplitRefList()
	format, err := reflistCollection.load(sl, key)
	c.Assert(err, IsNil)
	c.Check(format, Equals, reflistStorageFormatInline)

	migrator := reflistCollection.NewMigration()
	err = reflistCollection.LoadCompleteAndMigrate(sl, key, migrator)
	c.Assert(err, IsNil)
	verifyRefListIntegrity(c, sl)
	c.Check(toStrSlice(sl), DeepEquals, toStrSlice(NewPackageRefListFromPackageList(list)))

	stats := migrator.Stats()
	c.Check(stats.Reflists, Equals, 0)
	c.Check(stats.Buckets, Equals, 0)
	c.Check(stats.Segments, Equals, 0)

	err = migrator.Flush()
	c.Assert(err, IsNil)
	stats = migrator.Stats()
	c.Check(stats.Reflists, Equals, 1)
	c.Check(stats.Buckets, Not(Equals), 0)
	c.Check(stats.Segments, Equals, stats.Segments)

	sl = NewSplitRefList()
	err = reflistCollection.LoadComplete(sl, key)
	c.Assert(err, IsNil)
	verifyRefListIntegrity(c, sl)
	c.Check(toStrSlice(sl), DeepEquals, toStrSlice(NewPackageRefListFromPackageList(list)))

	format, err = reflistCollection.load(sl, key)
	c.Assert(err, IsNil)
	c.Check(format, Equals, reflistStorageFormatSplit)
}
