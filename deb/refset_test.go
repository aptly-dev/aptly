package deb

import (
	"fmt"

	"github.com/aptly-dev/aptly/database/goleveldb"
	. "gopkg.in/check.v1"
)

type PackageRefSetSuite struct{}

var _ = Suite(&PackageRefSetSuite{})

func matchedByRefSetFrom(ref []byte, identity RefSetIdentity, ls ...AnyRefList) bool {
	switch ls[0].(type) {
	case *PackageRefList:
		rs := NewPackageRefSet(RefSetOptions{Identity: identity})
		for _, l := range ls {
			rs.AddList(l.(*PackageRefList))
		}
		return rs.HasMatch(ref)
	case *SplitRefList:
		rs := NewSplitRefSet(RefSetOptions{Identity: identity})
		for _, l := range ls {
			rs.AddList(l.(*SplitRefList))
		}
		return rs.HasMatch(ref)
	default:
		panic(fmt.Sprintf("unexpected reflist type %T", ls[0]))
	}
}

func mergeThroughRefSet(identity RefSetIdentity, ls ...AnyRefList) AnyRefList {
	switch ls[0].(type) {
	case *PackageRefList:
		rs := NewPackageRefSet(RefSetOptions{Identity: identity})
		for _, l := range ls {
			rs.AddList(l.(*PackageRefList))
		}
		return rs.ToRefList()
	case *SplitRefList:
		rs := NewSplitRefSet(RefSetOptions{Identity: identity})
		for _, l := range ls {
			rs.AddList(l.(*SplitRefList))
		}
		return rs.ToRefList()
	default:
		panic(fmt.Sprintf("unexpected reflist type %T", ls[0]))
	}
}

func mergeFlatThroughRefSet(identity RefSetIdentity, ls ...AnyRefList) *PackageRefList {
	switch ls[0].(type) {
	case *PackageRefList:
		rs := NewPackageRefSet(RefSetOptions{Identity: identity})
		for _, l := range ls {
			rs.AddList(l.(*PackageRefList))
		}
		return rs.ToRefList()
	case *SplitRefList:
		rs := NewSplitRefSet(RefSetOptions{Identity: identity})
		for _, l := range ls {
			rs.AddList(l.(*SplitRefList))
		}
		return rs.ToFlattenedRefList()
	default:
		panic(fmt.Sprintf("unexpected reflist type %T", ls[0]))
	}
}

func (s *PackageRefSetSuite) TestMerging(c *C) {
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
			_ = coll.Update(p)
		}

		listA := NewPackageList()
		_ = listA.Add(packages[0])
		_ = listA.Add(packages[1])
		_ = listA.Add(packages[2])
		_ = listA.Add(packages[3])
		_ = listA.Add(packages[7])

		listB := NewPackageList()
		_ = listB.Add(packages[0])
		_ = listB.Add(packages[2])
		_ = listB.Add(packages[4])
		_ = listB.Add(packages[5])
		_ = listB.Add(packages[6])

		listC := NewPackageList()
		_ = listC.Add(packages[0])
		_ = listC.Add(packages[8])
		_ = listC.Add(packages[9])

		reflistA := f.newFromPackageList(listA)
		reflistB := f.newFromPackageList(listB)
		reflistC := f.newFromPackageList(listC)

		c.Check(matchedByRefSetFrom(packages[1].Key(""), RefSetIdentityFullKey, reflistA, reflistB), Equals, true)
		c.Check(matchedByRefSetFrom(packages[1].Key(""), RefSetIdentityWithoutHash, reflistA, reflistB), Equals, true)
		c.Check(matchedByRefSetFrom(packages[1].Key(""), RefSetIdentityWithoutVersion, reflistA, reflistB), Equals, true)

		c.Check(matchedByRefSetFrom(packages[8].Key(""), RefSetIdentityFullKey, reflistA, reflistB), Equals, false)
		c.Check(matchedByRefSetFrom(packages[8].Key(""), RefSetIdentityWithoutHash, reflistA, reflistB), Equals, true)
		c.Check(matchedByRefSetFrom(packages[8].Key(""), RefSetIdentityWithoutVersion, reflistA, reflistB), Equals, true)

		c.Check(matchedByRefSetFrom(packages[1].Key(""), RefSetIdentityFullKey, reflistB, reflistC), Equals, false)
		c.Check(matchedByRefSetFrom(packages[1].Key(""), RefSetIdentityWithoutHash, reflistB, reflistC), Equals, false)
		c.Check(matchedByRefSetFrom(packages[1].Key(""), RefSetIdentityWithoutVersion, reflistB, reflistC), Equals, true)

		mergeAB := mergeThroughRefSet(RefSetIdentityWithoutVersion, reflistA, reflistB)
		mergeBA := mergeThroughRefSet(RefSetIdentityWithoutVersion, reflistB, reflistA)
		mergeAC := mergeThroughRefSet(RefSetIdentityWithoutVersion, reflistA, reflistC)
		mergeBC := mergeThroughRefSet(RefSetIdentityWithoutVersion, reflistB, reflistC)
		mergeCB := mergeThroughRefSet(RefSetIdentityWithoutVersion, reflistC, reflistB)

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

		c.Check(toStrSlice(mergeAB), DeepEquals, toStrSlice(mergeFlatThroughRefSet(RefSetIdentityWithoutVersion, reflistA, reflistB)))
		c.Check(toStrSlice(mergeBA), DeepEquals, toStrSlice(mergeFlatThroughRefSet(RefSetIdentityWithoutVersion, reflistB, reflistA)))
		c.Check(toStrSlice(mergeAC), DeepEquals, toStrSlice(mergeFlatThroughRefSet(RefSetIdentityWithoutVersion, reflistA, reflistC)))
		c.Check(toStrSlice(mergeBC), DeepEquals, toStrSlice(mergeFlatThroughRefSet(RefSetIdentityWithoutVersion, reflistB, reflistC)))
		c.Check(toStrSlice(mergeCB), DeepEquals, toStrSlice(mergeFlatThroughRefSet(RefSetIdentityWithoutVersion, reflistC, reflistB)))

		mergeABall := mergeThroughRefSet(RefSetIdentityWithoutHash, reflistA, reflistB)
		mergeBAall := mergeThroughRefSet(RefSetIdentityWithoutHash, reflistB, reflistA)
		mergeACall := mergeThroughRefSet(RefSetIdentityWithoutHash, reflistA, reflistC)
		mergeBCall := mergeThroughRefSet(RefSetIdentityWithoutHash, reflistB, reflistC)
		mergeCBall := mergeThroughRefSet(RefSetIdentityWithoutHash, reflistC, reflistB)

		verifyRefListIntegrity(c, mergeABall)
		verifyRefListIntegrity(c, mergeBAall)
		verifyRefListIntegrity(c, mergeACall)
		verifyRefListIntegrity(c, mergeBCall)
		verifyRefListIntegrity(c, mergeCBall)

		c.Check(toStrSlice(mergeABall), DeepEquals, toStrSlice(mergeFlatThroughRefSet(RefSetIdentityWithoutHash, reflistA, reflistB)))
		c.Check(toStrSlice(mergeBAall), DeepEquals, toStrSlice(mergeFlatThroughRefSet(RefSetIdentityWithoutHash, reflistB, reflistA)))
		c.Check(toStrSlice(mergeACall), DeepEquals, toStrSlice(mergeFlatThroughRefSet(RefSetIdentityWithoutHash, reflistA, reflistC)))
		c.Check(toStrSlice(mergeBCall), DeepEquals, toStrSlice(mergeFlatThroughRefSet(RefSetIdentityWithoutHash, reflistB, reflistC)))
		c.Check(toStrSlice(mergeCBall), DeepEquals, toStrSlice(mergeFlatThroughRefSet(RefSetIdentityWithoutHash, reflistC, reflistB)))

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

		mergeBCwithConflicts := mergeThroughRefSet(RefSetIdentityFullKey, reflistB, reflistC)
		c.Check(toStrSlice(mergeBCwithConflicts), DeepEquals,
			[]string{"Pall data 1.1~bp1 00000000", "Pamd64 app 1.1~bp2 00000000", "Pi386 app 1.1~bp2 00000000", "Pi386 app 1.1~bp2 00000044",
				"Pi386 dpkg 1.0 00000000", "Pi386 dpkg 1.0 00034445", "Pi386 lib 1.0 00000000"})
	})
}

func (s *PackageRefSetSuite) TestMergingEdgeCases(c *C) {
	forEachRefList(func(f reflistFactory) {
		a := f.newFromRefs(
			[]byte("Pamd64 abc 1 00000000"),
			[]byte("Pamd64 abc 2 00000000"),
			[]byte("Pamd64 abc 5 00000000"),
			[]byte("Pamd64 abc 7 00000000"),
			[]byte("Pamd64 xyz 2 00000000"),
		)

		b := f.newFromRefs(
			[]byte("Pamd64 abc 3 00000000"),
			[]byte("Pamd64 abc 5 00000000"),
			[]byte("Pamd64 wxyz 1 00000000"),
		)

		m := mergeThroughRefSet(RefSetIdentityWithoutVersion, a, b)
		c.Check(toStrSlice(m), DeepEquals,
			[]string{"Pamd64 abc 3 00000000", "Pamd64 abc 5 00000000", "Pamd64 wxyz 1 00000000", "Pamd64 xyz 2 00000000"})

		// Interior run of duplicates *expands* across merges.
		m2 := mergeThroughRefSet(RefSetIdentityWithoutVersion, m, a)
		c.Check(toStrSlice(m2), DeepEquals,
			[]string{
				"Pamd64 abc 1 00000000", "Pamd64 abc 2 00000000",
				"Pamd64 abc 5 00000000", "Pamd64 abc 7 00000000",
				"Pamd64 wxyz 1 00000000", "Pamd64 xyz 2 00000000"})

		// Last occurrence of a reflist overrides prior ones.
		m3 := mergeThroughRefSet(RefSetIdentityWithoutVersion, b, a, b)
		c.Check(toStrSlice(m3), DeepEquals,
			[]string{"Pamd64 abc 3 00000000", "Pamd64 abc 5 00000000", "Pamd64 wxyz 1 00000000", "Pamd64 xyz 2 00000000"})
	})
}
