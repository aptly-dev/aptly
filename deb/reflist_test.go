package deb

import (
	"errors"

	"github.com/smira/aptly/database"

	. "gopkg.in/check.v1"
)

type PackageRefListSuite struct {
	// Simple list with "real" packages from stanzas
	list                   *PackageList
	p1, p2, p3, p4, p5, p6 *Package
}

var _ = Suite(&PackageRefListSuite{})

func toStrSlice(reflist *PackageRefList) (result []string) {
	result = make([]string, reflist.Len())
	for i, r := range reflist.Refs {
		result[i] = string(r)
	}
	return
}

func (s *PackageRefListSuite) SetUpTest(c *C) {
	s.list = NewPackageList()

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
	db, _ := database.NewOpenDB(c.MkDir())
	coll := NewPackageCollection(db)
	coll.Update(s.p1)
	coll.Update(s.p3)

	s.list.Add(s.p1)
	s.list.Add(s.p3)
	s.list.Add(s.p5)
	s.list.Add(s.p6)

	reflist := NewPackageRefListFromPackageList(s.list)

	_, err := NewPackageListFromRefList(reflist, coll, nil)
	c.Assert(err, ErrorMatches, "unable to load package with key.*")

	coll.Update(s.p5)
	coll.Update(s.p6)

	list, err := NewPackageListFromRefList(reflist, coll, nil)
	c.Assert(err, IsNil)
	c.Check(list.Len(), Equals, 4)
	c.Check(list.Add(s.p4), ErrorMatches, "conflict in package.*")

	list, err = NewPackageListFromRefList(nil, coll, nil)
	c.Assert(err, IsNil)
	c.Check(list.Len(), Equals, 0)
}

func (s *PackageRefListSuite) TestNewPackageRefList(c *C) {
	s.list.Add(s.p1)
	s.list.Add(s.p3)
	s.list.Add(s.p5)
	s.list.Add(s.p6)

	reflist := NewPackageRefListFromPackageList(s.list)
	c.Assert(reflist.Len(), Equals, 4)
	c.Check(reflist.Refs[0], DeepEquals, []byte(s.p1.Key("")))
	c.Check(reflist.Refs[1], DeepEquals, []byte(s.p6.Key("")))
	c.Check(reflist.Refs[2], DeepEquals, []byte(s.p5.Key("")))
	c.Check(reflist.Refs[3], DeepEquals, []byte(s.p3.Key("")))

	reflist = NewPackageRefList()
	c.Check(reflist.Len(), Equals, 0)
}

func (s *PackageRefListSuite) TestPackageRefListEncodeDecode(c *C) {
	s.list.Add(s.p1)
	s.list.Add(s.p3)
	s.list.Add(s.p5)
	s.list.Add(s.p6)

	reflist := NewPackageRefListFromPackageList(s.list)

	reflist2 := &PackageRefList{}
	err := reflist2.Decode(reflist.Encode())
	c.Assert(err, IsNil)
	c.Check(reflist2.Len(), Equals, reflist.Len())
	c.Check(reflist2.Refs, DeepEquals, reflist.Refs)
}

func (s *PackageRefListSuite) TestPackageRefListForeach(c *C) {
	s.list.Add(s.p1)
	s.list.Add(s.p3)
	s.list.Add(s.p5)
	s.list.Add(s.p6)

	reflist := NewPackageRefListFromPackageList(s.list)

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
}

func (s *PackageRefListSuite) TestHas(c *C) {
	s.list.Add(s.p1)
	s.list.Add(s.p3)
	s.list.Add(s.p5)
	reflist := NewPackageRefListFromPackageList(s.list)

	c.Check(reflist.Has(s.p1), Equals, true)
	c.Check(reflist.Has(s.p3), Equals, true)
	c.Check(reflist.Has(s.p5), Equals, true)
	c.Check(reflist.Has(s.p2), Equals, true)
	c.Check(reflist.Has(s.p6), Equals, false)
}

func (s *PackageRefListSuite) TestSubstract(c *C) {
	r1 := []byte("r1")
	r2 := []byte("r2")
	r3 := []byte("r3")
	r4 := []byte("r4")
	r5 := []byte("r5")

	empty := &PackageRefList{Refs: [][]byte{}}
	l1 := &PackageRefList{Refs: [][]byte{r1, r2, r3, r4}}
	l2 := &PackageRefList{Refs: [][]byte{r1, r3}}
	l3 := &PackageRefList{Refs: [][]byte{r2, r4}}
	l4 := &PackageRefList{Refs: [][]byte{r4, r5}}
	l5 := &PackageRefList{Refs: [][]byte{r1, r2, r3}}

	c.Check(l1.Subtract(empty), DeepEquals, l1)
	c.Check(l1.Subtract(l2), DeepEquals, l3)
	c.Check(l1.Subtract(l3), DeepEquals, l2)
	c.Check(l1.Subtract(l4), DeepEquals, l5)
	c.Check(empty.Subtract(l1), DeepEquals, empty)
	c.Check(l2.Subtract(l3), DeepEquals, l2)
}

func (s *PackageRefListSuite) TestDiff(c *C) {
	db, _ := database.NewOpenDB(c.MkDir())
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

	reflistA := NewPackageRefListFromPackageList(listA)
	reflistB := NewPackageRefListFromPackageList(listB)

	diffAA, err := reflistA.Diff(reflistA, coll)
	c.Check(err, IsNil)
	c.Check(diffAA, HasLen, 0)

	diffAB, err := reflistA.Diff(reflistB, coll)
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

	diffBA, err := reflistB.Diff(reflistA, coll)
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

}

func (s *PackageRefListSuite) TestMerge(c *C) {
	db, _ := database.NewOpenDB(c.MkDir())
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

	reflistA := NewPackageRefListFromPackageList(listA)
	reflistB := NewPackageRefListFromPackageList(listB)
	reflistC := NewPackageRefListFromPackageList(listC)

	mergeAB := reflistA.Merge(reflistB, true, false)
	mergeBA := reflistB.Merge(reflistA, true, false)
	mergeAC := reflistA.Merge(reflistC, true, false)
	mergeBC := reflistB.Merge(reflistC, true, false)
	mergeCB := reflistC.Merge(reflistB, true, false)

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

	mergeABall := reflistA.Merge(reflistB, false, false)
	mergeBAall := reflistB.Merge(reflistA, false, false)
	mergeACall := reflistA.Merge(reflistC, false, false)
	mergeBCall := reflistB.Merge(reflistC, false, false)
	mergeCBall := reflistC.Merge(reflistB, false, false)

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

	mergeBCwithConflicts := reflistB.Merge(reflistC, false, true)
	c.Check(toStrSlice(mergeBCwithConflicts), DeepEquals,
		[]string{"Pall data 1.1~bp1 00000000", "Pamd64 app 1.1~bp2 00000000", "Pi386 app 1.1~bp2 00000000", "Pi386 app 1.1~bp2 00000044",
			"Pi386 dpkg 1.0 00000000", "Pi386 dpkg 1.0 00034445", "Pi386 lib 1.0 00000000"})
}

func (s *PackageRefListSuite) TestFilterLatestRefs(c *C) {
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

	result := NewPackageRefListFromPackageList(rl)
	result.FilterLatestRefs()

	c.Check(toStrSlice(result), DeepEquals,
		[]string{"Pi386 dpkg 1.6", "Pi386 lib 1.2"})
}
