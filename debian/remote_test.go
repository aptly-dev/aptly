package debian

import (
	"errors"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/console"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/files"
	"github.com/smira/aptly/http"
	"github.com/smira/aptly/utils"
	"io"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"os"
)

type NullVerifier struct {
}

func (n *NullVerifier) InitKeyring() error {
	return nil
}

func (n *NullVerifier) AddKeyring(keyring string) {
}

func (n *NullVerifier) VerifyDetachedSignature(signature, cleartext io.Reader) error {
	return nil
}

func (n *NullVerifier) VerifyClearsigned(clearsigned io.Reader) error {
	return nil
}

func (n *NullVerifier) ExtractClearsigned(clearsigned io.Reader) (text *os.File, err error) {
	text, _ = ioutil.TempFile("", "aptly-test")
	io.Copy(text, clearsigned)
	text.Seek(0, 0)
	os.Remove(text.Name())

	return
}

type PackageListMixinSuite struct {
	p1, p2, p3 *Package
	list       *PackageList
	reflist    *PackageRefList
}

func (s *PackageListMixinSuite) SetUpPackages() {
	s.list = NewPackageList()

	s.p1 = NewPackageFromControlFile(packageStanza.Copy())
	stanza := packageStanza.Copy()
	stanza["Package"] = "mars-invaders"
	s.p2 = NewPackageFromControlFile(stanza)
	stanza = packageStanza.Copy()
	stanza["Package"] = "lonely-strangers"
	s.p3 = NewPackageFromControlFile(stanza)

	s.list.Add(s.p1)
	s.list.Add(s.p2)
	s.list.Add(s.p3)

	s.reflist = NewPackageRefListFromPackageList(s.list)
}

type RemoteRepoSuite struct {
	PackageListMixinSuite
	repo              *RemoteRepo
	flat              *RemoteRepo
	downloader        *http.FakeDownloader
	progress          aptly.Progress
	db                database.Storage
	packageCollection *PackageCollection
	packagePool       aptly.PackagePool
}

var _ = Suite(&RemoteRepoSuite{})

func (s *RemoteRepoSuite) SetUpTest(c *C) {
	s.repo, _ = NewRemoteRepo("yandex", "http://mirror.yandex.ru/debian", "squeeze", []string{"main"}, []string{}, false)
	s.flat, _ = NewRemoteRepo("exp42", "http://repos.express42.com/virool/precise/", "./", []string{}, []string{}, false)
	s.downloader = http.NewFakeDownloader().ExpectResponse("http://mirror.yandex.ru/debian/dists/squeeze/Release", exampleReleaseFile)
	s.progress = console.NewProgress()
	s.db, _ = database.OpenDB(c.MkDir())
	s.packageCollection = NewPackageCollection(s.db)
	s.packagePool = files.NewPackagePool(c.MkDir())
	s.SetUpPackages()
	s.progress.Start()
}

func (s *RemoteRepoSuite) TearDownTest(c *C) {
	s.progress.Shutdown()
	s.db.Close()
}

func (s *RemoteRepoSuite) TestInvalidURL(c *C) {
	_, err := NewRemoteRepo("s", "http://lolo%2", "squeeze", []string{"main"}, []string{}, false)
	c.Assert(err, ErrorMatches, ".*hexadecimal escape in host.*")
}

func (s *RemoteRepoSuite) TestFlatCreation(c *C) {
	c.Check(s.flat.Distribution, Equals, "")
	c.Check(s.flat.Architectures, IsNil)
	c.Check(s.flat.Components, IsNil)

	_, err := NewRemoteRepo("fl", "http://some.repo/", "./", []string{"main"}, []string{}, false)
	c.Check(err, ErrorMatches, "components aren't supported for flat repos")
}

func (s *RemoteRepoSuite) TestString(c *C) {
	c.Check(s.repo.String(), Equals, "[yandex]: http://mirror.yandex.ru/debian/ squeeze")
	c.Check(s.flat.String(), Equals, "[exp42]: http://repos.express42.com/virool/precise/ ./")

	s.repo.DownloadSources = true
	s.flat.DownloadSources = true
	c.Check(s.repo.String(), Equals, "[yandex]: http://mirror.yandex.ru/debian/ squeeze [src]")
	c.Check(s.flat.String(), Equals, "[exp42]: http://repos.express42.com/virool/precise/ ./ [src]")
}

func (s *RemoteRepoSuite) TestNumPackages(c *C) {
	c.Check(s.repo.NumPackages(), Equals, 0)
	s.repo.packageRefs = s.reflist
	c.Check(s.repo.NumPackages(), Equals, 3)
}

func (s *RemoteRepoSuite) TestIsFlat(c *C) {
	c.Check(s.repo.IsFlat(), Equals, false)
	c.Check(s.flat.IsFlat(), Equals, true)
}

func (s *RemoteRepoSuite) TestRefList(c *C) {
	s.repo.packageRefs = s.reflist
	c.Check(s.repo.RefList(), Equals, s.reflist)
}

func (s *RemoteRepoSuite) TestReleaseURL(c *C) {
	c.Assert(s.repo.ReleaseURL("Release").String(), Equals, "http://mirror.yandex.ru/debian/dists/squeeze/Release")
	c.Assert(s.repo.ReleaseURL("InRelease").String(), Equals, "http://mirror.yandex.ru/debian/dists/squeeze/InRelease")

	c.Assert(s.flat.ReleaseURL("Release").String(), Equals, "http://repos.express42.com/virool/precise/Release")
}

func (s *RemoteRepoSuite) TestBinaryURL(c *C) {
	c.Assert(s.repo.BinaryURL("main", "amd64").String(), Equals, "http://mirror.yandex.ru/debian/dists/squeeze/main/binary-amd64/Packages")
}

func (s *RemoteRepoSuite) TestSourcesURL(c *C) {
	c.Assert(s.repo.SourcesURL("main").String(), Equals, "http://mirror.yandex.ru/debian/dists/squeeze/main/source/Sources")
}

func (s *RemoteRepoSuite) TestFlatBinaryURL(c *C) {
	c.Assert(s.flat.FlatBinaryURL().String(), Equals, "http://repos.express42.com/virool/precise/Packages")
}

func (s *RemoteRepoSuite) TestFlatSourcesURL(c *C) {
	c.Assert(s.flat.FlatSourcesURL().String(), Equals, "http://repos.express42.com/virool/precise/Sources")
}

func (s *RemoteRepoSuite) TestPackageURL(c *C) {
	c.Assert(s.repo.PackageURL("pool/main/0/0ad/0ad_0~r11863-2_i386.deb").String(), Equals,
		"http://mirror.yandex.ru/debian/pool/main/0/0ad/0ad_0~r11863-2_i386.deb")
}

func (s *RemoteRepoSuite) TestFetch(c *C) {
	err := s.repo.Fetch(s.downloader, nil)
	c.Assert(err, IsNil)
	c.Assert(s.repo.Architectures, DeepEquals, []string{"amd64", "armel", "armhf", "i386", "powerpc"})
	c.Assert(s.repo.Components, DeepEquals, []string{"main"})
	c.Assert(s.downloader.Empty(), Equals, true)

	c.Check(s.repo.ReleaseFiles, HasLen, 39)
	c.Check(s.repo.ReleaseFiles["main/binary-i386/Packages.bz2"], DeepEquals,
		utils.ChecksumInfo{
			Size:   734,
			MD5:    "7954ed80936429687122b554620c1b5b",
			SHA1:   "95a463a0739bf9ff622c8d68f6e4598d400f5248",
			SHA256: "377890a26f99db55e117dfc691972dcbbb7d8be1630c8fc8297530c205377f2b"})
}

func (s *RemoteRepoSuite) TestFetchNullVerifier1(c *C) {
	downloader := http.NewFakeDownloader()
	downloader.ExpectError("http://mirror.yandex.ru/debian/dists/squeeze/InRelease", errors.New("404"))
	downloader.ExpectResponse("http://mirror.yandex.ru/debian/dists/squeeze/Release", exampleReleaseFile)
	downloader.ExpectResponse("http://mirror.yandex.ru/debian/dists/squeeze/Release.gpg", "GPG")

	err := s.repo.Fetch(downloader, &NullVerifier{})
	c.Assert(err, IsNil)
	c.Assert(s.repo.Architectures, DeepEquals, []string{"amd64", "armel", "armhf", "i386", "powerpc"})
	c.Assert(s.repo.Components, DeepEquals, []string{"main"})
	c.Assert(downloader.Empty(), Equals, true)
}

func (s *RemoteRepoSuite) TestFetchNullVerifier2(c *C) {
	downloader := http.NewFakeDownloader()
	downloader.ExpectResponse("http://mirror.yandex.ru/debian/dists/squeeze/InRelease", exampleReleaseFile)

	err := s.repo.Fetch(downloader, &NullVerifier{})
	c.Assert(err, IsNil)
	c.Assert(s.repo.Architectures, DeepEquals, []string{"amd64", "armel", "armhf", "i386", "powerpc"})
	c.Assert(s.repo.Components, DeepEquals, []string{"main"})
	c.Assert(downloader.Empty(), Equals, true)
}

func (s *RemoteRepoSuite) TestFetchWrongArchitecture(c *C) {
	s.repo, _ = NewRemoteRepo("s", "http://mirror.yandex.ru/debian/", "squeeze", []string{"main"}, []string{"xyz"}, false)
	err := s.repo.Fetch(s.downloader, nil)
	c.Assert(err, ErrorMatches, "architecture xyz not available in repo.*")
}

func (s *RemoteRepoSuite) TestFetchWrongComponent(c *C) {
	s.repo, _ = NewRemoteRepo("s", "http://mirror.yandex.ru/debian/", "squeeze", []string{"xyz"}, []string{"i386"}, false)
	err := s.repo.Fetch(s.downloader, nil)
	c.Assert(err, ErrorMatches, "component xyz not available in repo.*")
}

func (s *RemoteRepoSuite) TestEncodeDecode(c *C) {
	repo := &RemoteRepo{}
	err := repo.Decode(s.repo.Encode())
	c.Assert(err, IsNil)

	c.Check(repo.Name, Equals, "yandex")
	c.Check(repo.ArchiveRoot, Equals, "http://mirror.yandex.ru/debian/")
}

func (s *RemoteRepoSuite) TestKey(c *C) {
	c.Assert(len(s.repo.Key()), Equals, 37)
	c.Assert(s.repo.Key()[0], Equals, byte('R'))
}

func (s *RemoteRepoSuite) TestRefKey(c *C) {
	c.Assert(len(s.repo.RefKey()), Equals, 37)
	c.Assert(s.repo.RefKey()[0], Equals, byte('E'))
	c.Assert(s.repo.RefKey()[1:], DeepEquals, s.repo.Key()[1:])
}

func (s *RemoteRepoSuite) TestDownload(c *C) {
	s.repo.Architectures = []string{"i386"}

	err := s.repo.Fetch(s.downloader, nil)
	c.Assert(err, IsNil)

	s.downloader.ExpectError("http://mirror.yandex.ru/debian/dists/squeeze/main/binary-i386/Packages.bz2", errors.New("HTTP 404"))
	s.downloader.ExpectError("http://mirror.yandex.ru/debian/dists/squeeze/main/binary-i386/Packages.gz", errors.New("HTTP 404"))
	s.downloader.ExpectResponse("http://mirror.yandex.ru/debian/dists/squeeze/main/binary-i386/Packages", examplePackagesFile)
	s.downloader.ExpectResponse("http://mirror.yandex.ru/debian/pool/main/a/amanda/amanda-client_3.3.1-3~bpo60+1_amd64.deb", "xyz")

	err = s.repo.Download(s.progress, s.downloader, s.packageCollection, s.packagePool, false)
	c.Assert(err, IsNil)
	c.Assert(s.downloader.Empty(), Equals, true)
	c.Assert(s.repo.packageRefs, NotNil)

	pkg, err := s.packageCollection.ByKey(s.repo.packageRefs.Refs[0])
	c.Assert(err, IsNil)

	result, err := pkg.VerifyFiles(s.packagePool)
	c.Check(result, Equals, true)
	c.Check(err, IsNil)

	c.Check(pkg.Name, Equals, "amanda-client")
}

func (s *RemoteRepoSuite) TestDownloadWithSources(c *C) {
	s.repo.Architectures = []string{"i386"}
	s.repo.DownloadSources = true

	err := s.repo.Fetch(s.downloader, nil)
	c.Assert(err, IsNil)

	s.downloader.ExpectError("http://mirror.yandex.ru/debian/dists/squeeze/main/binary-i386/Packages.bz2", errors.New("HTTP 404"))
	s.downloader.ExpectError("http://mirror.yandex.ru/debian/dists/squeeze/main/binary-i386/Packages.gz", errors.New("HTTP 404"))
	s.downloader.ExpectResponse("http://mirror.yandex.ru/debian/dists/squeeze/main/binary-i386/Packages", examplePackagesFile)
	s.downloader.ExpectError("http://mirror.yandex.ru/debian/dists/squeeze/main/source/Sources.bz2", errors.New("HTTP 404"))
	s.downloader.ExpectError("http://mirror.yandex.ru/debian/dists/squeeze/main/source/Sources.gz", errors.New("HTTP 404"))
	s.downloader.ExpectResponse("http://mirror.yandex.ru/debian/dists/squeeze/main/source/Sources", exampleSourcesFile)
	s.downloader.AnyExpectResponse("http://mirror.yandex.ru/debian/pool/main/a/amanda/amanda-client_3.3.1-3~bpo60+1_amd64.deb", "xyz")
	s.downloader.AnyExpectResponse("http://mirror.yandex.ru/debian/pool/main/a/access-modifier-checker/access-modifier-checker_1.0-4.dsc", "abc")
	s.downloader.AnyExpectResponse("http://mirror.yandex.ru/debian/pool/main/a/access-modifier-checker/access-modifier-checker_1.0.orig.tar.gz", "abcd")
	s.downloader.AnyExpectResponse("http://mirror.yandex.ru/debian/pool/main/a/access-modifier-checker/access-modifier-checker_1.0-4.debian.tar.gz", "abcde")

	err = s.repo.Download(s.progress, s.downloader, s.packageCollection, s.packagePool, false)
	c.Assert(err, IsNil)
	c.Assert(s.downloader.Empty(), Equals, true)
	c.Assert(s.repo.packageRefs, NotNil)

	pkg, err := s.packageCollection.ByKey(s.repo.packageRefs.Refs[0])
	c.Assert(err, IsNil)

	result, err := pkg.VerifyFiles(s.packagePool)
	c.Check(result, Equals, true)
	c.Check(err, IsNil)

	c.Check(pkg.Name, Equals, "amanda-client")

	pkg, err = s.packageCollection.ByKey(s.repo.packageRefs.Refs[1])
	c.Assert(err, IsNil)

	result, err = pkg.VerifyFiles(s.packagePool)
	c.Check(result, Equals, true)
	c.Check(err, IsNil)

	c.Check(pkg.Name, Equals, "access-modifier-checker")
}

func (s *RemoteRepoSuite) TestDownloadFlat(c *C) {
	downloader := http.NewFakeDownloader()
	downloader.ExpectResponse("http://repos.express42.com/virool/precise/Release", exampleReleaseFile)
	downloader.ExpectError("http://repos.express42.com/virool/precise/Packages.bz2", errors.New("HTTP 404"))
	downloader.ExpectError("http://repos.express42.com/virool/precise/Packages.gz", errors.New("HTTP 404"))
	downloader.ExpectResponse("http://repos.express42.com/virool/precise/Packages", examplePackagesFile)
	downloader.ExpectResponse("http://repos.express42.com/virool/precise/pool/main/a/amanda/amanda-client_3.3.1-3~bpo60+1_amd64.deb", "xyz")

	err := s.flat.Fetch(downloader, nil)
	c.Assert(err, IsNil)

	err = s.flat.Download(s.progress, downloader, s.packageCollection, s.packagePool, false)
	c.Assert(err, IsNil)
	c.Assert(downloader.Empty(), Equals, true)
	c.Assert(s.flat.packageRefs, NotNil)

	pkg, err := s.packageCollection.ByKey(s.flat.packageRefs.Refs[0])
	c.Assert(err, IsNil)

	result, err := pkg.VerifyFiles(s.packagePool)
	c.Check(result, Equals, true)
	c.Check(err, IsNil)

	c.Check(pkg.Name, Equals, "amanda-client")
}

func (s *RemoteRepoSuite) TestDownloadWithSourcesFlat(c *C) {
	s.flat.DownloadSources = true

	downloader := http.NewFakeDownloader()
	downloader.ExpectResponse("http://repos.express42.com/virool/precise/Release", exampleReleaseFile)
	downloader.ExpectError("http://repos.express42.com/virool/precise/Packages.bz2", errors.New("HTTP 404"))
	downloader.ExpectError("http://repos.express42.com/virool/precise/Packages.gz", errors.New("HTTP 404"))
	downloader.ExpectResponse("http://repos.express42.com/virool/precise/Packages", examplePackagesFile)
	downloader.ExpectError("http://repos.express42.com/virool/precise/Sources.bz2", errors.New("HTTP 404"))
	downloader.ExpectError("http://repos.express42.com/virool/precise/Sources.gz", errors.New("HTTP 404"))
	downloader.ExpectResponse("http://repos.express42.com/virool/precise/Sources", exampleSourcesFile)
	downloader.AnyExpectResponse("http://repos.express42.com/virool/precise/pool/main/a/amanda/amanda-client_3.3.1-3~bpo60+1_amd64.deb", "xyz")
	downloader.AnyExpectResponse("http://repos.express42.com/virool/precise/pool/main/a/access-modifier-checker/access-modifier-checker_1.0-4.dsc", "abc")
	downloader.AnyExpectResponse("http://repos.express42.com/virool/precise/pool/main/a/access-modifier-checker/access-modifier-checker_1.0.orig.tar.gz", "abcd")
	downloader.AnyExpectResponse("http://repos.express42.com/virool/precise/pool/main/a/access-modifier-checker/access-modifier-checker_1.0-4.debian.tar.gz", "abcde")

	err := s.flat.Fetch(downloader, nil)
	c.Assert(err, IsNil)

	err = s.flat.Download(s.progress, downloader, s.packageCollection, s.packagePool, false)
	c.Assert(err, IsNil)
	c.Assert(downloader.Empty(), Equals, true)
	c.Assert(s.flat.packageRefs, NotNil)

	pkg, err := s.packageCollection.ByKey(s.flat.packageRefs.Refs[0])
	c.Assert(err, IsNil)

	result, err := pkg.VerifyFiles(s.packagePool)
	c.Check(result, Equals, true)
	c.Check(err, IsNil)

	c.Check(pkg.Name, Equals, "amanda-client")

	pkg, err = s.packageCollection.ByKey(s.flat.packageRefs.Refs[1])
	c.Assert(err, IsNil)

	result, err = pkg.VerifyFiles(s.packagePool)
	c.Check(result, Equals, true)
	c.Check(err, IsNil)

	c.Check(pkg.Name, Equals, "access-modifier-checker")
}

type RemoteRepoCollectionSuite struct {
	PackageListMixinSuite
	db         database.Storage
	collection *RemoteRepoCollection
}

var _ = Suite(&RemoteRepoCollectionSuite{})

func (s *RemoteRepoCollectionSuite) SetUpTest(c *C) {
	s.db, _ = database.OpenDB(c.MkDir())
	s.collection = NewRemoteRepoCollection(s.db)
	s.SetUpPackages()
}

func (s *RemoteRepoCollectionSuite) TearDownTest(c *C) {
	s.db.Close()
}

func (s *RemoteRepoCollectionSuite) TestAddByName(c *C) {
	r, err := s.collection.ByName("yandex")
	c.Assert(err, ErrorMatches, "*.not found")

	repo, _ := NewRemoteRepo("yandex", "http://mirror.yandex.ru/debian/", "squeeze", []string{"main"}, []string{}, false)
	c.Assert(s.collection.Add(repo), IsNil)
	c.Assert(s.collection.Add(repo), ErrorMatches, ".*already exists")

	r, err = s.collection.ByName("yandex")
	c.Assert(err, IsNil)
	c.Assert(r.String(), Equals, repo.String())

	collection := NewRemoteRepoCollection(s.db)
	r, err = collection.ByName("yandex")
	c.Assert(err, IsNil)
	c.Assert(r.String(), Equals, repo.String())
}

func (s *RemoteRepoCollectionSuite) TestByUUID(c *C) {
	r, err := s.collection.ByUUID("some-uuid")
	c.Assert(err, ErrorMatches, "*.not found")

	repo, _ := NewRemoteRepo("yandex", "http://mirror.yandex.ru/debian/", "squeeze", []string{"main"}, []string{}, false)
	c.Assert(s.collection.Add(repo), IsNil)

	r, err = s.collection.ByUUID(repo.UUID)
	c.Assert(err, IsNil)
	c.Assert(r.String(), Equals, repo.String())
}

func (s *RemoteRepoCollectionSuite) TestUpdateLoadComplete(c *C) {
	repo, _ := NewRemoteRepo("yandex", "http://mirror.yandex.ru/debian/", "squeeze", []string{"main"}, []string{}, false)
	c.Assert(s.collection.Update(repo), IsNil)

	collection := NewRemoteRepoCollection(s.db)
	r, err := collection.ByName("yandex")
	c.Assert(err, IsNil)
	c.Assert(r.packageRefs, IsNil)

	repo.packageRefs = s.reflist
	c.Assert(s.collection.Update(repo), IsNil)

	collection = NewRemoteRepoCollection(s.db)
	r, err = collection.ByName("yandex")
	c.Assert(err, IsNil)
	c.Assert(r.packageRefs, IsNil)
	c.Assert(r.NumPackages(), Equals, 0)
	c.Assert(s.collection.LoadComplete(r), IsNil)
	c.Assert(r.NumPackages(), Equals, 3)
}

func (s *RemoteRepoCollectionSuite) TestForEachAndLen(c *C) {
	repo, _ := NewRemoteRepo("yandex", "http://mirror.yandex.ru/debian/", "squeeze", []string{"main"}, []string{}, false)
	s.collection.Add(repo)

	count := 0
	err := s.collection.ForEach(func(*RemoteRepo) error {
		count++
		return nil
	})
	c.Assert(count, Equals, 1)
	c.Assert(err, IsNil)

	c.Check(s.collection.Len(), Equals, 1)

	e := errors.New("c")

	err = s.collection.ForEach(func(*RemoteRepo) error {
		return e
	})
	c.Assert(err, Equals, e)
}

func (s *RemoteRepoCollectionSuite) TestDrop(c *C) {
	repo1, _ := NewRemoteRepo("yandex", "http://mirror.yandex.ru/debian/", "squeeze", []string{"main"}, []string{}, false)
	s.collection.Add(repo1)

	repo2, _ := NewRemoteRepo("tyndex", "http://mirror.yandex.ru/debian/", "wheezy", []string{"main"}, []string{}, false)
	s.collection.Add(repo2)

	r1, _ := s.collection.ByUUID(repo1.UUID)
	c.Check(r1, Equals, repo1)

	err := s.collection.Drop(repo1)
	c.Check(err, IsNil)

	_, err = s.collection.ByUUID(repo1.UUID)
	c.Check(err, ErrorMatches, "mirror .* not found")

	collection := NewRemoteRepoCollection(s.db)
	_, err = collection.ByName("yandex")
	c.Check(err, ErrorMatches, "mirror .* not found")

	r2, _ := collection.ByName("tyndex")
	c.Check(r2.String(), Equals, repo2.String())

	c.Check(func() { s.collection.Drop(repo1) }, Panics, "repo not found!")
}

const exampleReleaseFile = `Origin: LP-PPA-agenda-developers-daily
Label: Agenda Daily Builds
Suite: precise
Version: 12.04
Codename: precise
Date: Thu, 05 Dec 2013  8:14:32 UTC
Architectures: amd64 armel armhf i386 powerpc
Components: main
Description: Ubuntu Precise 12.04
MD5Sum:
 6a5fc91b7277021999268e04a8d74d4c              134 main/binary-amd64/Release
 01ff4a18aab39546fde304a35350fc2d              643 main/binary-amd64/Packages.gz
 52ded91eeb8490b02016335aa3343492             1350 main/binary-amd64/Packages
 5216f9ffe55d151cd7ce7b98b7a43bd7              735 main/binary-amd64/Packages.bz2
 d41d8cd98f00b204e9800998ecf8427e                0 main/binary-armel/Packages
 4059d198768f9f8dc9372dc1c54bc3c3               14 main/binary-armel/Packages.bz2
 7a9de1fb7bf60d416a77d9c9a9716675              134 main/binary-armel/Release
 9d10bb61e59bd799891ae4fbcf447ec9               29 main/binary-armel/Packages.gz
 9d10bb61e59bd799891ae4fbcf447ec9               29 main/binary-armhf/Packages.gz
 c63d31e8e3a5650c29a7124e541d6c23              134 main/binary-armhf/Release
 4059d198768f9f8dc9372dc1c54bc3c3               14 main/binary-armhf/Packages.bz2
 d41d8cd98f00b204e9800998ecf8427e                0 main/binary-armhf/Packages
 c8d336856df67d509032bb54145c2f89              826 main/binary-i386/Packages
 92262f0668b265401291f0467bc93763              133 main/binary-i386/Release
 7954ed80936429687122b554620c1b5b              734 main/binary-i386/Packages.bz2
 e2eef4fe7d285b12c511adfa3a39069e              641 main/binary-i386/Packages.gz
 4059d198768f9f8dc9372dc1c54bc3c3               14 main/binary-powerpc/Packages.bz2
 9d10bb61e59bd799891ae4fbcf447ec9               29 main/binary-powerpc/Packages.gz
 d41d8cd98f00b204e9800998ecf8427e                0 main/binary-powerpc/Packages
 b079563fd3367c11f7be049bc686dd10              136 main/binary-powerpc/Release
 9d10bb61e59bd799891ae4fbcf447ec9               29 main/debian-installer/binary-amd64/Packages.gz
 d41d8cd98f00b204e9800998ecf8427e                0 main/debian-installer/binary-amd64/Packages
 4059d198768f9f8dc9372dc1c54bc3c3               14 main/debian-installer/binary-amd64/Packages.bz2
 9d10bb61e59bd799891ae4fbcf447ec9               29 main/debian-installer/binary-armel/Packages.gz
 d41d8cd98f00b204e9800998ecf8427e                0 main/debian-installer/binary-armel/Packages
 4059d198768f9f8dc9372dc1c54bc3c3               14 main/debian-installer/binary-armel/Packages.bz2
 9d10bb61e59bd799891ae4fbcf447ec9               29 main/debian-installer/binary-armhf/Packages.gz
 d41d8cd98f00b204e9800998ecf8427e                0 main/debian-installer/binary-armhf/Packages
 4059d198768f9f8dc9372dc1c54bc3c3               14 main/debian-installer/binary-armhf/Packages.bz2
 d41d8cd98f00b204e9800998ecf8427e                0 main/debian-installer/binary-i386/Packages
 9d10bb61e59bd799891ae4fbcf447ec9               29 main/debian-installer/binary-i386/Packages.gz
 4059d198768f9f8dc9372dc1c54bc3c3               14 main/debian-installer/binary-i386/Packages.bz2
 d41d8cd98f00b204e9800998ecf8427e                0 main/debian-installer/binary-powerpc/Packages
 4059d198768f9f8dc9372dc1c54bc3c3               14 main/debian-installer/binary-powerpc/Packages.bz2
 9d10bb61e59bd799891ae4fbcf447ec9               29 main/debian-installer/binary-powerpc/Packages.gz
 3481d65651306df1596dca9078c2506a              135 main/source/Release
 0459b7e4512db5479cb982bac6e2f9a1             2003 main/source/Sources
 3d83a489f1bd3c04226aa6520b8a6d07              656 main/source/Sources.bz2
 b062b5b77094aeeb05ca8dbb1ecf68a9              592 main/source/Sources.gz
SHA1:
 fb0b7c8935623ed7d8c45044ba62225fd8cbd4ad              134 main/binary-amd64/Release
 b5d62bcec4ec18b88d664255e9051645bab7bd01              643 main/binary-amd64/Packages.gz
 ed47aae8926d22d529c27b40b61604aed2cb5f2f             1350 main/binary-amd64/Packages
 5b9b171ffcea36e869eba31bcc0e1bfb2a6ad84f              735 main/binary-amd64/Packages.bz2
 da39a3ee5e6b4b0d3255bfef95601890afd80709                0 main/binary-armel/Packages
 64a543afbb5f4bf728636bdcbbe7a2ed0804adc2               14 main/binary-armel/Packages.bz2
 b89234a7efb74d02f15b88e264b5cd2ae1e5dc2d              134 main/binary-armel/Release
 3df6ca52b6e8ecfb4a8fac6b8e02c777e3c7960d               29 main/binary-armel/Packages.gz
 3df6ca52b6e8ecfb4a8fac6b8e02c777e3c7960d               29 main/binary-armhf/Packages.gz
 585a452e27c2e7e047c49d4b0a7459d8c627aa08              134 main/binary-armhf/Release
 64a543afbb5f4bf728636bdcbbe7a2ed0804adc2               14 main/binary-armhf/Packages.bz2
 da39a3ee5e6b4b0d3255bfef95601890afd80709                0 main/binary-armhf/Packages
 1d2f0cd7a3c9e687b853eb277e241cd712b6e3b1              826 main/binary-i386/Packages
 16020809662f9bda36eb516d0995658dd94d1ad5              133 main/binary-i386/Release
 95a463a0739bf9ff622c8d68f6e4598d400f5248              734 main/binary-i386/Packages.bz2
 bf8c0dec9665ba78311c97cae1755d4b2e60af76              641 main/binary-i386/Packages.gz
 64a543afbb5f4bf728636bdcbbe7a2ed0804adc2               14 main/binary-powerpc/Packages.bz2
 3df6ca52b6e8ecfb4a8fac6b8e02c777e3c7960d               29 main/binary-powerpc/Packages.gz
 da39a3ee5e6b4b0d3255bfef95601890afd80709                0 main/binary-powerpc/Packages
 cf2ae2d98f535d90209f2c4e5790f95b393d8c2b              136 main/binary-powerpc/Release
 3df6ca52b6e8ecfb4a8fac6b8e02c777e3c7960d               29 main/debian-installer/binary-amd64/Packages.gz
 da39a3ee5e6b4b0d3255bfef95601890afd80709                0 main/debian-installer/binary-amd64/Packages
 64a543afbb5f4bf728636bdcbbe7a2ed0804adc2               14 main/debian-installer/binary-amd64/Packages.bz2
 3df6ca52b6e8ecfb4a8fac6b8e02c777e3c7960d               29 main/debian-installer/binary-armel/Packages.gz
 da39a3ee5e6b4b0d3255bfef95601890afd80709                0 main/debian-installer/binary-armel/Packages
 64a543afbb5f4bf728636bdcbbe7a2ed0804adc2               14 main/debian-installer/binary-armel/Packages.bz2
 3df6ca52b6e8ecfb4a8fac6b8e02c777e3c7960d               29 main/debian-installer/binary-armhf/Packages.gz
 da39a3ee5e6b4b0d3255bfef95601890afd80709                0 main/debian-installer/binary-armhf/Packages
 64a543afbb5f4bf728636bdcbbe7a2ed0804adc2               14 main/debian-installer/binary-armhf/Packages.bz2
 da39a3ee5e6b4b0d3255bfef95601890afd80709                0 main/debian-installer/binary-i386/Packages
 3df6ca52b6e8ecfb4a8fac6b8e02c777e3c7960d               29 main/debian-installer/binary-i386/Packages.gz
 64a543afbb5f4bf728636bdcbbe7a2ed0804adc2               14 main/debian-installer/binary-i386/Packages.bz2
 da39a3ee5e6b4b0d3255bfef95601890afd80709                0 main/debian-installer/binary-powerpc/Packages
 64a543afbb5f4bf728636bdcbbe7a2ed0804adc2               14 main/debian-installer/binary-powerpc/Packages.bz2
 3df6ca52b6e8ecfb4a8fac6b8e02c777e3c7960d               29 main/debian-installer/binary-powerpc/Packages.gz
 49cfec0c9b1df3a25e983a3ddf29d15b0e376e02              135 main/source/Release
 6b92e0fc84307226172696fde59ca5f33f380b57             2003 main/source/Sources
 ecb8afea11030a5df46941cb8ec297ca24c85736              656 main/source/Sources.bz2
 923e71383969c91146f12fa8cd121397f2467a2e              592 main/source/Sources.gz
SHA256:
 8c0314cfb1b48a8daf47f77420330fd0d78a31897eeb46e05a51964c9f2c02df              134 main/binary-amd64/Release
 81b072773d2fdd8471473e060d3bf73255e4c00d322cf387654736ea196e83b4              643 main/binary-amd64/Packages.gz
 c7bb299483277bbf7bf4165042edaf547f5fa18f5782c7d2cd8407a38a327cc8             1350 main/binary-amd64/Packages
 d263f735c3830caa33ae6441529bd4f8e382205af597ab2cdfcea73afdaa21ab              735 main/binary-amd64/Packages.bz2
 e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855                0 main/binary-armel/Packages
 d3dda84eb03b9738d118eb2be78e246106900493c0ae07819ad60815134a8058               14 main/binary-armel/Packages.bz2
 75ede815b020626c6aa16201d24099ed7e06f03643d0cf38ef194f1029ea648b              134 main/binary-armel/Release
 825d493158fe0f50ca1acd70367aefa391170563af2e4ee9cedbcbe6796c8384               29 main/binary-armel/Packages.gz
 825d493158fe0f50ca1acd70367aefa391170563af2e4ee9cedbcbe6796c8384               29 main/binary-armhf/Packages.gz
 d25382b633c4a1621f8df6ce86e5c63da2e506a377e05ae9453238bb18191540              134 main/binary-armhf/Release
 d3dda84eb03b9738d118eb2be78e246106900493c0ae07819ad60815134a8058               14 main/binary-armhf/Packages.bz2
 e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855                0 main/binary-armhf/Packages
 b1bb341bb613363ca29440c2eb9c08a9289de5458209990ec502ed27711a83a2              826 main/binary-i386/Packages
 e5aaceaac5ecb59143a4b4ed2bf700fe85d6cf08addd10cf2058bde697b7b219              133 main/binary-i386/Release
 377890a26f99db55e117dfc691972dcbbb7d8be1630c8fc8297530c205377f2b              734 main/binary-i386/Packages.bz2
 6361e8efc67d2e7c1a8db45388aec0311007c0a1bd96698623ddeb5ed0bdc914              641 main/binary-i386/Packages.gz
 d3dda84eb03b9738d118eb2be78e246106900493c0ae07819ad60815134a8058               14 main/binary-powerpc/Packages.bz2
 825d493158fe0f50ca1acd70367aefa391170563af2e4ee9cedbcbe6796c8384               29 main/binary-powerpc/Packages.gz
 e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855                0 main/binary-powerpc/Packages
 03b5c97a99aa799964eb5a77f8a62ad38a241b93a87eacac6cf75a270a6d417c              136 main/binary-powerpc/Release
 825d493158fe0f50ca1acd70367aefa391170563af2e4ee9cedbcbe6796c8384               29 main/debian-installer/binary-amd64/Packages.gz
 e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855                0 main/debian-installer/binary-amd64/Packages
 d3dda84eb03b9738d118eb2be78e246106900493c0ae07819ad60815134a8058               14 main/debian-installer/binary-amd64/Packages.bz2
 825d493158fe0f50ca1acd70367aefa391170563af2e4ee9cedbcbe6796c8384               29 main/debian-installer/binary-armel/Packages.gz
 e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855                0 main/debian-installer/binary-armel/Packages
 d3dda84eb03b9738d118eb2be78e246106900493c0ae07819ad60815134a8058               14 main/debian-installer/binary-armel/Packages.bz2
 825d493158fe0f50ca1acd70367aefa391170563af2e4ee9cedbcbe6796c8384               29 main/debian-installer/binary-armhf/Packages.gz
 e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855                0 main/debian-installer/binary-armhf/Packages
 d3dda84eb03b9738d118eb2be78e246106900493c0ae07819ad60815134a8058               14 main/debian-installer/binary-armhf/Packages.bz2
 e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855                0 main/debian-installer/binary-i386/Packages
 825d493158fe0f50ca1acd70367aefa391170563af2e4ee9cedbcbe6796c8384               29 main/debian-installer/binary-i386/Packages.gz
 d3dda84eb03b9738d118eb2be78e246106900493c0ae07819ad60815134a8058               14 main/debian-installer/binary-i386/Packages.bz2
 e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855                0 main/debian-installer/binary-powerpc/Packages
 d3dda84eb03b9738d118eb2be78e246106900493c0ae07819ad60815134a8058               14 main/debian-installer/binary-powerpc/Packages.bz2
 825d493158fe0f50ca1acd70367aefa391170563af2e4ee9cedbcbe6796c8384               29 main/debian-installer/binary-powerpc/Packages.gz
 d683102993b6f11067ce86d73111f067e36a199e9dc1f4295c8b19c274dc9ef8              135 main/source/Release
 45f868fd5d9efe611d67572ffcf96a00a5b9ec38ea5102753290c38c36b8c282             2003 main/source/Sources
 d178f1e310218d9f0f16c37d0780637f1cf3640a94a7fb0e24dc940c51b1e115              656 main/source/Sources.bz2
 080228b550da407fb8ac73fb30b37323468fd2b2de98dd56a324ee7d701f6103              592 main/source/Sources.gz`

const examplePackagesFile = `Package: amanda-client
Source: amanda
Version: 1:3.3.1-3~bpo60+1
Installed-Size: 880
Maintainer: Bdale Garbee <bdale@gag.com>
Architecture: i386
Replaces: amanda-common (<< 1:2.5.2p1-3)
Depends: libc6 (>= 2.3), libcurl3 (>= 7.16.2-1), libglib2.0-0 (>= 2.12.0), libreadline6 (>= 6.0), libssl0.9.8 (>= 0.9.8m-1), amanda-common (= 1:3.3.1-3~bpo60+1)
Suggests: gnuplot, dump, smbclient
Conflicts: amanda, amanda-common (<< 1:2.5.2p1-3)
Description: Advanced Maryland Automatic Network Disk Archiver (Client)
Description-md5: 21af3684379a64cacc51c39152ab1062
Section: utils
Priority: optional
Filename: pool/main/a/amanda/amanda-client_3.3.1-3~bpo60+1_amd64.deb
Size: 3
MD5sum: d16fb36f0911f878998c136191af705e
SHA1: 66b27417d37e024c46526c2f6d358a754fc552f3
SHA256: 3608bca1e44ea6c4d268eb6db02260269892c0b42b86bbf1e77a6fa16c3c9282
`

const exampleSourcesFile = sourcePackageMeta
