package debian

import (
	"github.com/smira/aptly/database"
	. "launchpad.net/gocheck"
	"os"
	"path/filepath"
)

type NullSigner struct{}

func (n *NullSigner) SetKey(keyRef string) {

}

func (n *NullSigner) DetachedSign(source string, destination string) error {
	return nil
}

func (n *NullSigner) ClearSign(source string, destination string) error {
	return nil
}

type PublishedRepoSuite struct {
	PackageListMixinSuite
	repo              *PublishedRepo
	packageRepo       *Repository
	db                database.Storage
	packageCollection *PackageCollection
}

var _ = Suite(&PublishedRepoSuite{})

func (s *PublishedRepoSuite) SetUpTest(c *C) {
	s.SetUpPackages()

	s.db, _ = database.OpenDB(c.MkDir())

	s.packageRepo = NewRepository(c.MkDir())

	repo, _ := NewRemoteRepo("yandex", "http://mirror.yandex.ru/debian/", "squeeze", []string{"main"}, []string{})
	repo.packageRefs = s.reflist

	snapshot, _ := NewSnapshotFromRepository("snap", repo)

	s.repo = NewPublishedRepo("ppa", "squeeze", "main", nil, snapshot)

	s.packageCollection = NewPackageCollection(s.db)
	s.packageCollection.Update(s.p1)
	s.packageCollection.Update(s.p2)
	s.packageCollection.Update(s.p3)

	poolPath, _ := s.packageRepo.PoolPath(s.p1.Filename, s.p1.HashMD5)
	err := os.MkdirAll(filepath.Dir(poolPath), 0755)
	f, err := os.Create(poolPath)
	c.Assert(err, IsNil)
	f.Close()
}

func (s *PublishedRepoSuite) TestPublish(c *C) {
	err := s.repo.Publish(s.packageRepo, s.packageCollection, &NullSigner{})
	c.Assert(err, IsNil)

	c.Check(s.repo.Architectures, DeepEquals, []string{"i386"})

	rf, err := os.Open(filepath.Join(s.packageRepo.RootPath, "public/ppa/dists/squeeze/Release"))
	c.Assert(err, IsNil)

	cfr := NewControlFileReader(rf)
	st, err := cfr.ReadStanza()
	c.Assert(err, IsNil)

	c.Check(st["Origin"], Equals, "ppa squeeze")
	c.Check(st["Components"], Equals, "main")
	c.Check(st["Architectures"], Equals, "i386")

	pf, err := os.Open(filepath.Join(s.packageRepo.RootPath, "public/ppa/dists/squeeze/main/binary-i386/Packages"))
	c.Assert(err, IsNil)

	cfr = NewControlFileReader(pf)

	for i := 0; i < 3; i++ {
		st, err = cfr.ReadStanza()
		c.Assert(err, IsNil)

		c.Check(st["Filename"], Equals, "pool/main/a/alien-arena/alien-arena-common_7.40-2_i386.deb")
	}

	st, err = cfr.ReadStanza()
	c.Assert(err, IsNil)
	c.Assert(st, IsNil)

	_, err = os.Stat(filepath.Join(s.packageRepo.RootPath, "public/ppa/pool/main/a/alien-arena/alien-arena-common_7.40-2_i386.deb"))
	c.Assert(err, IsNil)
}
