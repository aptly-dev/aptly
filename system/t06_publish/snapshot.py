import hashlib
import inspect
import os

from lib import BaseTest, ungzip_if_required


def strip_processor(output):
    return "\n".join([l for l in output.split("\n") if not l.startswith(' ') and not l.startswith('Date:')])


def sorted_processor(output):
    return "\n".join(sorted(output.split("\n")))


class PublishSnapshot1Test(BaseTest):
    """
    publish snapshot: defaults
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot1Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Release')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists(
            'public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/Contents-i386.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Release')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages')
        self.check_exists(
            'public/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_exists(
            'public/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_exists('public/dists/maverick/main/Contents-amd64.gz')
        self.check_not_exists(
            'public/dists/maverick/main/debian-installer/binary-i386/Packages')
        self.check_not_exists(
            'public/dists/maverick/main/debian-installer/binary-amd64/Packages')

        self.check_exists(
            'public/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')

        # verify contents except of sums
        self.check_file_contents(
            'public/dists/maverick/Release', 'release', match_prepare=strip_processor)

        self.check_file_contents(
            'public/dists/maverick/main/binary-i386/Release', 'release_i386')
        self.check_file_contents(
            'public/dists/maverick/main/binary-amd64/Release', 'release_amd64')

        self.check_file_contents('public/dists/maverick/main/binary-i386/Packages',
                                 'packages_i386', match_prepare=sorted_processor)
        self.check_file_contents('public/dists/maverick/main/binary-amd64/Packages',
                                 'packages_amd64', match_prepare=sorted_processor)

        self.check_file_contents('public/dists/maverick/main/Contents-i386.gz',
                                 'contents_i386', match_prepare=ungzip_if_required, mode='b', ensure_utf8=False)
        self.check_file_contents('public/dists/maverick/main/Contents-amd64.gz',
                                 'contents_amd64', match_prepare=ungzip_if_required, mode='b', ensure_utf8=False)

        # verify signatures
        self.run_cmd([self.gpgFinder.gpg, "--no-auto-check-trustdb", "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/InRelease')])
        self.run_cmd([self.gpgFinder.gpg, "--no-auto-check-trustdb",  "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(
                          os.environ["HOME"], ".aptly", 'public/dists/maverick/Release.gpg'),
                      os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/Release')])

        # verify sums
        release = self.read_file('public/dists/maverick/Release').split("\n")
        release = [l for l in release if l.startswith(" ")]
        pathsSeen = set()
        for l in release:
            fileHash, fileSize, path = l.split()
            if "Contents" in path and not path.endswith(".gz"):
                # "Contents" are present in index, but not really written to disk
                continue

            pathsSeen.add(path)

            fileSize = int(fileSize)

            st = os.stat(os.path.join(
                os.environ["HOME"], ".aptly", 'public/dists/maverick/', path))
            if fileSize != st.st_size:
                raise Exception("file size doesn't match for %s: %d != %d" % (
                    path, fileSize, st.st_size))

            if len(fileHash) == 32:
                h = hashlib.md5()
            elif len(fileHash) == 40:
                h = hashlib.sha1()
            elif len(fileHash) == 64:
                h = hashlib.sha256()
            else:
                h = hashlib.sha512()

            h.update(self.read_file(os.path.join('public/dists/maverick', path), mode='b'))

            if h.hexdigest() != fileHash:
                raise Exception("file hash doesn't match for %s: %s != %s" % (
                    path, fileHash, h.hexdigest()))

        if pathsSeen != set(['main/binary-amd64/Packages', 'main/binary-i386/Packages', 'main/binary-i386/Packages.gz',
                             'main/binary-amd64/Packages.gz', 'main/binary-amd64/Packages.bz2', 'main/binary-i386/Packages.bz2',
                             'main/binary-amd64/Release', 'main/binary-i386/Release', 'main/Contents-amd64.gz',
                             'main/Contents-i386.gz', 'Contents-i386.gz', 'Contents-amd64.gz']):
            raise Exception("path seen wrong: %r" % (pathsSeen, ))


class PublishSnapshot2Test(BaseTest):
    """
    publish snapshot: different distribution
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap2 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=squeeze snap2"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot2Test, self).check()

        self.check_exists('public/dists/squeeze/InRelease')
        self.check_exists('public/dists/squeeze/Release')
        self.check_exists('public/dists/squeeze/Release.gpg')

        self.check_exists('public/dists/squeeze/main/binary-i386/Packages')
        self.check_exists('public/dists/squeeze/main/binary-i386/Packages.gz')
        self.check_exists('public/dists/squeeze/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/squeeze/main/Contents-i386.gz')
        self.check_exists('public/dists/squeeze/main/binary-amd64/Packages')
        self.check_exists('public/dists/squeeze/main/binary-amd64/Packages.gz')
        self.check_exists(
            'public/dists/squeeze/main/binary-amd64/Packages.bz2')
        self.check_exists('public/dists/squeeze/main/Contents-amd64.gz')

        self.check_exists(
            'public/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')

        # verify contents except of sums
        self.check_file_contents(
            'public/dists/squeeze/Release', 'release', match_prepare=strip_processor)


class PublishSnapshot3Test(BaseTest):
    """
    publish snapshot: different distribution and component
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap3 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=squeeze -component=contrib snap3"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot3Test, self).check()

        self.check_exists('public/dists/squeeze/InRelease')
        self.check_exists('public/dists/squeeze/Release')
        self.check_exists('public/dists/squeeze/Release.gpg')

        self.check_exists('public/dists/squeeze/contrib/binary-i386/Packages')
        self.check_exists(
            'public/dists/squeeze/contrib/binary-i386/Packages.gz')
        self.check_exists(
            'public/dists/squeeze/contrib/binary-i386/Packages.bz2')
        self.check_exists('public/dists/squeeze/contrib/Contents-i386.gz')
        self.check_exists('public/dists/squeeze/contrib/binary-amd64/Packages')
        self.check_exists(
            'public/dists/squeeze/contrib/binary-amd64/Packages.gz')
        self.check_exists(
            'public/dists/squeeze/contrib/binary-amd64/Packages.bz2')
        self.check_exists('public/dists/squeeze/contrib/Contents-amd64.gz')

        self.check_exists(
            'public/pool/contrib/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')

        # verify contents except of sums
        self.check_file_contents(
            'public/dists/squeeze/Release', 'release', match_prepare=strip_processor)


class PublishSnapshot4Test(BaseTest):
    """
    publish snapshot: limit architectures
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap4 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly -architectures=i386 publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=squeeze snap4"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot4Test, self).check()

        self.check_exists('public/dists/squeeze/InRelease')
        self.check_exists('public/dists/squeeze/Release')
        self.check_exists('public/dists/squeeze/Release.gpg')

        self.check_exists('public/dists/squeeze/main/binary-i386/Packages')
        self.check_exists('public/dists/squeeze/main/binary-i386/Packages.gz')
        self.check_exists('public/dists/squeeze/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/squeeze/main/Contents-i386.gz')
        self.check_not_exists(
            'public/dists/squeeze/main/binary-amd64/Packages')
        self.check_not_exists(
            'public/dists/squeeze/main/binary-amd64/Packages.gz')
        self.check_not_exists(
            'public/dists/squeeze/main/binary-amd64/Packages.bz2')
        self.check_not_exists('public/dists/squeeze/main/Contents-amd64.gz')

        self.check_exists(
            'public/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')

        # verify contents except of sums
        self.check_file_contents(
            'public/dists/squeeze/Release', 'release', match_prepare=strip_processor)


class PublishSnapshot5Test(BaseTest):
    """
    publish snapshot: under prefix
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap5 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly publish snapshot -acquire-by-hash -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=squeeze snap5 ppa/smira"

    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot5Test, self).check()

        self.check_exists('public/ppa/smira/dists/squeeze/InRelease')
        self.check_exists('public/ppa/smira/dists/squeeze/Release')
        self.check_exists('public/ppa/smira/dists/squeeze/Release.gpg')

        self.check_exists(
            'public/ppa/smira/dists/squeeze/main/binary-i386/Packages')
        self.check_exists(
            'public/ppa/smira/dists/squeeze/main/binary-i386/by-hash/MD5Sum/e98cd30fc76fbe7fa3ea25717efa1c92')
        self.check_exists(
            'public/ppa/smira/dists/squeeze/main/binary-i386/Packages.gz')
        self.check_exists(
            'public/ppa/smira/dists/squeeze/main/binary-i386/Packages.bz2')
        self.check_exists(
            'public/ppa/smira/dists/squeeze/main/binary-amd64/Packages')
        self.check_exists(
            'public/ppa/smira/dists/squeeze/main/binary-amd64/by-hash/MD5Sum/ab073d1f73bed52e7356c91161e8667e')
        self.check_exists(
            'public/ppa/smira/dists/squeeze/main/binary-amd64/Packages.gz')
        self.check_exists(
            'public/ppa/smira/dists/squeeze/main/binary-amd64/Packages.bz2')

        self.check_exists(
            'public/ppa/smira/dists/squeeze/main/Contents-i386.gz')
        self.check_exists(
            'public/ppa/smira/dists/squeeze/main/Contents-amd64.gz')

        self.check_exists(
            'public/ppa/smira/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')


class PublishSnapshot6Test(BaseTest):
    """
    publish snapshot: specify distribution
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap from mirror gnuplot-maverick",
        "aptly snapshot create snap2 from mirror wheezy-main",
        "aptly snapshot merge snap6 snap2 snap"
    ]
    runCmd = "aptly publish snapshot snap6"
    expectedCode = 1


class PublishSnapshot7Test(BaseTest):
    """
    publish snapshot: double publish under root
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap7 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap7",
    ]
    runCmd = "aptly publish snapshot snap7"
    expectedCode = 1


class PublishSnapshot8Test(BaseTest):
    """
    publish snapshot: double publish under prefix
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap8 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap8 ./ppa",
    ]
    runCmd = "aptly publish snapshot snap8 ppa"
    expectedCode = 1


class PublishSnapshot9Test(BaseTest):
    """
    publish snapshot: wrong prefix
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap9 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly publish snapshot snap9 ppa/dists/la"
    expectedCode = 1


class PublishSnapshot10Test(BaseTest):
    """
    publish snapshot: wrong prefix
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap10 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly publish snapshot snap10 ppa/pool/la"
    expectedCode = 1


class PublishSnapshot11Test(BaseTest):
    """
    publish snapshot: wrong prefix
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap11 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly publish snapshot snap11 ../la"
    expectedCode = 1


class PublishSnapshot12Test(BaseTest):
    """
    publish snapshot: no snapshot
    """
    fixtureDB = True
    runCmd = "aptly publish snapshot snap12"
    expectedCode = 1


class PublishSnapshot13Test(BaseTest):
    """
    publish snapshot: -skip-signing
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap13 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly publish snapshot -skip-signing snap13"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot13Test, self).check()

        self.check_not_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_not_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists(
            'public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/Contents-i386.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages')
        self.check_exists(
            'public/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_exists(
            'public/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_exists('public/dists/maverick/main/Contents-amd64.gz')

        # verify contents except of sums
        self.check_file_contents(
            'public/dists/maverick/Release', 'release', match_prepare=strip_processor)


class PublishSnapshot14Test(BaseTest):
    """
    publish snapshot: empty snapshot is not publishable w/o architectures list
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap14 empty",
    ]
    runCmd = "aptly publish snapshot --distribution=mars --skip-signing snap14"
    expectedCode = 1


class PublishSnapshot15Test(BaseTest):
    """
    publish snapshot: skip signing via config
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap15 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly publish snapshot snap15"
    configOverride = {"gpgDisableSign": True}
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot15Test, self).check()

        self.check_not_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_not_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists(
            'public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/Contents-i386.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages')
        self.check_exists(
            'public/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_exists(
            'public/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_exists('public/dists/maverick/main/Contents-amd64.gz')

        # verify contents except of sums
        self.check_file_contents(
            'public/dists/maverick/Release', 'release', match_prepare=strip_processor)


class PublishSnapshot16Test(BaseTest):
    """
    publish snapshot: with sources
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap16 from mirror gnuplot-maverick-src",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap16"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot16Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists(
            'public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/Contents-i386.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages')
        self.check_exists(
            'public/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_exists(
            'public/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_exists('public/dists/maverick/main/Contents-amd64.gz')
        self.check_exists('public/dists/maverick/main/source/Sources')
        self.check_exists('public/dists/maverick/main/source/Sources.gz')
        self.check_exists('public/dists/maverick/main/source/Sources.bz2')
        self.check_not_exists('public/dists/maverick/main/Contents-source.gz')

        self.check_exists(
            'public/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')
        self.check_exists(
            'public/pool/main/g/gnuplot/gnuplot_4.6.1-1~maverick2.debian.tar.gz')
        self.check_exists(
            'public/pool/main/g/gnuplot/gnuplot_4.6.1-1~maverick2.dsc')
        self.check_exists(
            'public/pool/main/g/gnuplot/gnuplot_4.6.1.orig.tar.gz')

        # verify contents except of sums
        self.check_file_contents(
            'public/dists/maverick/Release', 'release', match_prepare=strip_processor)
        self.check_file_contents('public/dists/maverick/main/source/Sources',
                                 'sources', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))

        # verify signatures
        self.run_cmd([self.gpgFinder.gpg, "--no-auto-check-trustdb", "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/InRelease')])
        self.run_cmd([self.gpgFinder.gpg, "--no-auto-check-trustdb",  "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(
                          os.environ["HOME"], ".aptly", 'public/dists/maverick/Release.gpg'),
                      os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/Release')])


class PublishSnapshot17Test(BaseTest):
    """
    publish snapshot: from local repo
    """
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
        "aptly snapshot create snap17 from repo local-repo",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick snap17"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot17Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists(
            'public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/Contents-i386.gz')
        self.check_exists('public/dists/maverick/main/source/Sources')
        self.check_exists('public/dists/maverick/main/source/Sources.gz')
        self.check_exists('public/dists/maverick/main/source/Sources.bz2')
        self.check_not_exists('public/dists/maverick/main/Contents-source.gz')

        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.dsc')
        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.diff.gz')
        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz')
        self.check_exists(
            'public/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc')
        self.check_exists(
            'public/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb')

        # verify contents except of sums
        self.check_file_contents(
            'public/dists/maverick/Release', 'release', match_prepare=strip_processor)
        self.check_file_contents('public/dists/maverick/main/source/Sources',
                                 'sources', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))
        self.check_file_contents('public/dists/maverick/main/binary-i386/Packages',
                                 'binary', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))

        # verify signatures
        self.run_cmd([self.gpgFinder.gpg, "--no-auto-check-trustdb", "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/InRelease')])
        self.run_cmd([self.gpgFinder.gpg, "--no-auto-check-trustdb",  "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(
                          os.environ["HOME"], ".aptly", 'public/dists/maverick/Release.gpg'),
                      os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/Release')])


class PublishSnapshot18Test(BaseTest):
    """
    publish snapshot: specify distribution from local repo
    """
    fixtureCmds = [
        "aptly repo create repo1",
        "aptly repo add repo1 ${files}",
        "aptly snapshot create snap1 from repo repo1",
    ]
    runCmd = "aptly publish snapshot snap1"
    expectedCode = 1


class PublishSnapshot19Test(BaseTest):
    """
    publish snapshot: guess distribution from long chain
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 from mirror gnuplot-maverick",
        "aptly snapshot create snap3 from mirror gnuplot-maverick",
        "aptly snapshot merge snap4 snap1 snap2",
        "aptly snapshot pull snap4 snap1 snap5 gnuplot",

    ]
    runCmd = "aptly publish snapshot -skip-signing snap5"
    gold_processor = BaseTest.expand_environ


class PublishSnapshot20Test(BaseTest):
    """
    publish snapshot: guess distribution from long chain including local repo
    """
    fixtureDB = True
    fixturePoolCopy = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly repo create -distribution=maverick local-repo",
        "aptly repo add local-repo ${files}",
        "aptly snapshot create snap2 from repo local-repo",
        "aptly snapshot merge snap3 snap1 snap2",

    ]
    runCmd = "aptly publish snapshot -skip-signing snap3"
    gold_processor = BaseTest.expand_environ


class PublishSnapshot21Test(BaseTest):
    """
    publish snapshot: conflict in distributions
    """
    fixtureDB = True
    fixturePoolCopy = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly repo create -distribution=squeeze local-repo",
        "aptly repo add local-repo ${files}",
        "aptly snapshot create snap2 from repo local-repo",
        "aptly snapshot merge snap3 snap1 snap2",

    ]
    runCmd = "aptly publish snapshot -skip-signing snap3"
    gold_processor = BaseTest.expand_environ
    expectedCode = 1


class PublishSnapshot22Test(BaseTest):
    """
    publish snapshot: conflict in components
    """
    fixtureDB = True
    fixturePoolCopy = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly repo create -component=contrib -distribution=maverick local-repo",
        "aptly repo add local-repo ${files}",
        "aptly snapshot create snap2 from repo local-repo",
        "aptly snapshot merge snap3 snap1 snap2",

    ]
    runCmd = "aptly publish snapshot -skip-signing snap3"
    gold_processor = BaseTest.expand_environ


class PublishSnapshot23Test(BaseTest):
    """
    publish snapshot: distribution empty plus distribution maverick
    """
    fixtureDB = True
    fixturePoolCopy = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
        "aptly snapshot create snap2 from repo local-repo",
        "aptly snapshot merge snap3 snap1 snap2",

    ]
    runCmd = "aptly publish snapshot -skip-signing snap3"
    gold_processor = BaseTest.expand_environ


class PublishSnapshot24Test(BaseTest):
    """
    publish snapshot: custom origin, notautomatic and butautomaticupgrades
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap24 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=squeeze -origin=aptly24 -notautomatic=yes -butautomaticupgrades=yes snap24"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot24Test, self).check()

        # verify contents except of sums
        self.check_file_contents(
            'public/dists/squeeze/Release', 'release', match_prepare=strip_processor)


class PublishSnapshot25Test(BaseTest):
    """
    publish snapshot: empty snapshot is publishable with architectures list
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap25 empty",
    ]
    runCmd = "aptly publish snapshot -architectures=amd64 --distribution=maverick -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap25"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot25Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_not_exists(
            'public/dists/maverick/main/binary-i386/Packages')
        self.check_not_exists(
            'public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_not_exists(
            'public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages')
        self.check_exists(
            'public/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_exists(
            'public/dists/maverick/main/binary-amd64/Packages.bz2')


class PublishSnapshot26Test(BaseTest):
    """
    publish snapshot: multiple component
    """
    fixtureDB = True
    fixturePoolCopy = True
    fixtureCmds = [
        "aptly snapshot create snap26.1 from mirror gnuplot-maverick",
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
        "aptly snapshot create snap26.2 from repo local-repo",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -component=main,contrib snap26.1 snap26.2"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot26Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists(
            'public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/Contents-i386.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages')
        self.check_exists(
            'public/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_exists(
            'public/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_exists('public/dists/maverick/main/Contents-amd64.gz')
        self.check_exists('public/dists/maverick/main/source/Sources')
        self.check_exists('public/dists/maverick/main/source/Sources.gz')
        self.check_exists('public/dists/maverick/main/source/Sources.bz2')
        self.check_not_exists('public/dists/maverick/main/Contents-source.gz')

        self.check_exists('public/dists/maverick/contrib/binary-i386/Packages')
        self.check_exists(
            'public/dists/maverick/contrib/binary-i386/Packages.gz')
        self.check_exists(
            'public/dists/maverick/contrib/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/contrib/Contents-i386.gz')
        self.check_exists(
            'public/dists/maverick/contrib/binary-amd64/Packages')
        self.check_exists(
            'public/dists/maverick/contrib/binary-amd64/Packages.gz')
        self.check_exists(
            'public/dists/maverick/contrib/binary-amd64/Packages.bz2')
        self.check_not_exists(
            'public/dists/maverick/contrib/Contents-amd64.gz')
        self.check_exists('public/dists/maverick/contrib/source/Sources')
        self.check_exists('public/dists/maverick/contrib/source/Sources.gz')
        self.check_exists('public/dists/maverick/contrib/source/Sources.bz2')

        self.check_exists(
            'public/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')
        self.check_exists('public/pool/contrib/p/pyspi/pyspi_0.6.1-1.3.dsc')
        self.check_exists(
            'public/pool/contrib/p/pyspi/pyspi_0.6.1-1.3.diff.gz')
        self.check_exists(
            'public/pool/contrib/p/pyspi/pyspi_0.6.1.orig.tar.gz')
        self.check_exists(
            'public/pool/contrib/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc')
        self.check_exists(
            'public/pool/contrib/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb')

        # verify contents except of sums
        self.check_file_contents(
            'public/dists/maverick/Release', 'release', match_prepare=strip_processor)

        # verify signatures
        self.run_cmd([self.gpgFinder.gpg, "--no-auto-check-trustdb", "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/InRelease')])
        self.run_cmd([self.gpgFinder.gpg, "--no-auto-check-trustdb",  "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(
                          os.environ["HOME"], ".aptly", 'public/dists/maverick/Release.gpg'),
                      os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/Release')])

        # verify sums
        release = self.read_file('public/dists/maverick/Release').split("\n")
        release = [l for l in release if l.startswith(" ")]
        pathsSeen = set()
        for l in release:
            fileHash, fileSize, path = l.split()
            if "Contents" in path and not path.endswith(".gz"):
                # "Contents" are present in index, but not really written to disk
                continue

            pathsSeen.add(path)

            fileSize = int(fileSize)

            st = os.stat(os.path.join(
                os.environ["HOME"], ".aptly", 'public/dists/maverick/', path))
            if fileSize != st.st_size:
                raise Exception("file size doesn't match for %s: %d != %d" % (
                    path, fileSize, st.st_size))

            if len(fileHash) == 32:
                h = hashlib.md5()
            elif len(fileHash) == 40:
                h = hashlib.sha1()
            elif len(fileHash) == 64:
                h = hashlib.sha256()
            else:
                h = hashlib.sha512()

            h.update(self.read_file(os.path.join('public/dists/maverick', path), mode='b'))

            if h.hexdigest() != fileHash:
                raise Exception("file hash doesn't match for %s: %s != %s" % (
                    path, fileHash, h.hexdigest()))

        if pathsSeen != set(['main/binary-amd64/Packages', 'main/binary-i386/Packages', 'main/binary-i386/Packages.gz',
                             'main/binary-amd64/Packages.gz', 'main/binary-amd64/Packages.bz2', 'main/binary-i386/Packages.bz2',
                             'main/source/Sources', 'main/source/Sources.gz', 'main/source/Sources.bz2',
                             'contrib/binary-amd64/Packages', 'contrib/binary-i386/Packages', 'contrib/binary-i386/Packages.gz',
                             'contrib/binary-amd64/Packages.gz', 'contrib/binary-amd64/Packages.bz2', 'contrib/binary-i386/Packages.bz2',
                             'contrib/source/Sources', 'contrib/source/Sources.gz', 'contrib/source/Sources.bz2',
                             'main/binary-amd64/Release', 'main/binary-i386/Release', 'main/source/Release',
                             'contrib/binary-amd64/Release', 'contrib/binary-i386/Release', 'contrib/source/Release',
                             'contrib/Contents-i386.gz', 'main/Contents-i386.gz', 'main/Contents-amd64.gz',
                             'Contents-i386.gz', 'Contents-amd64.gz']):
            raise Exception("path seen wrong: %r" % (pathsSeen, ))


class PublishSnapshot27Test(BaseTest):
    """
    publish snapshot: multiple component, guessing component names
    """
    fixtureDB = True
    fixturePoolCopy = True
    fixtureCmds = [
        "aptly snapshot create snap27.1 from mirror gnuplot-maverick",
        "aptly repo create -component=contrib local-repo",
        "aptly repo add local-repo ${files}",
        "aptly snapshot create snap27.2 from repo local-repo",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -component=, snap27.1 snap27.2"
    gold_processor = BaseTest.expand_environ


class PublishSnapshot28Test(BaseTest):
    """
    publish snapshot: duplicate component name (guessed)
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap28.1 from mirror gnuplot-maverick",
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
        "aptly snapshot create snap28.2 from repo local-repo",
    ]
    runCmd = "aptly publish snapshot -component=, snap28.1 snap28.2"
    expectedCode = 1


class PublishSnapshot29Test(BaseTest):
    """
    publish snapshot: duplicate component name (manual)
    """
    fixtureCmds = [
        "aptly snapshot create snap29.1 empty",
        "aptly snapshot create snap29.2 empty",
    ]
    runCmd = "aptly publish snapshot -component=b,b snap29.1 snap29.2"
    expectedCode = 1


class PublishSnapshot30Test(BaseTest):
    """
    publish snapshot: distribution conflict
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap30.1 from mirror gnuplot-maverick",
        "aptly repo create -distribution=squeeze local-repo",
        "aptly repo add local-repo ${files}",
        "aptly snapshot create snap30.2 from repo local-repo",
    ]
    runCmd = "aptly publish snapshot -component=main,contrib snap30.1 snap30.2"
    expectedCode = 1


class PublishSnapshot31Test(BaseTest):
    """
    publish snapshot: no such snapshot
    """
    fixtureCmds = [
        "aptly snapshot create snap31.1 empty",
    ]
    runCmd = "aptly publish snapshot -component=main,contrib snap31.1 snap31.2"
    expectedCode = 1


class PublishSnapshot32Test(BaseTest):
    """
    publish snapshot: mismatch in count
    """
    fixtureCmds = [
        "aptly snapshot create snap32.1 empty",
    ]
    runCmd = "aptly publish snapshot -component=main,contrib snap32.1"
    expectedCode = 2

    def outputMatchPrepare(self, s):
        return "\n".join([l for l in self.ensure_utf8(s).split("\n") if l.startswith("ERROR")])


class PublishSnapshot33Test(BaseTest):
    """
    publish snapshot: conflicting files in the snapshot
    """
    fixtureCmds = [
        "aptly repo create local-repo1",
        "aptly repo add local-repo1 ${files}",
        "aptly snapshot create snap1 from repo local-repo1",
        "aptly repo create local-repo2",
        "aptly repo add local-repo2 ${testfiles}",
        "aptly snapshot create snap2 from repo local-repo2",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick snap1",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=squeeze snap2"
    expectedCode = 1
    gold_processor = BaseTest.expand_environ


class PublishSnapshot34Test(BaseTest):
    """
    publish snapshot: -force-overwrite
    """
    fixtureCmds = [
        "aptly repo create local-repo1",
        "aptly repo add local-repo1 ${files}",
        "aptly snapshot create snap1 from repo local-repo1",
        "aptly repo create local-repo2",
        "aptly repo add local-repo2 ${testfiles}",
        "aptly snapshot create snap2 from repo local-repo2",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick snap1",
    ]
    runCmd = "aptly publish snapshot -force-overwrite -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=squeeze snap2"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot34Test, self).check()

        self.check_file_contents(
            "public/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz", "file")


class PublishSnapshot35Test(BaseTest):
    """
    publish snapshot: mirror with udebs
    """
    configOverride = {"max-tries": 1}
    fixtureGpg = True
    fixtureCmds = [
        "aptly -architectures=i386,amd64 mirror create -keyring=aptlytest.gpg -filter='$$Source (gnupg2)' -with-udebs stretch http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch main non-free",
        "aptly mirror update -keyring=aptlytest.gpg stretch",
        "aptly snapshot create stretch from mirror stretch",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec stretch"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot35Test, self).check()

        self.check_exists('public/dists/stretch/InRelease')
        self.check_exists('public/dists/stretch/Release')
        self.check_exists('public/dists/stretch/Release.gpg')

        self.check_exists('public/dists/stretch/main/binary-i386/Release')
        self.check_exists('public/dists/stretch/main/binary-i386/Packages')
        self.check_exists('public/dists/stretch/main/binary-i386/Packages.gz')
        self.check_exists('public/dists/stretch/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/stretch/main/Contents-i386.gz')
        self.check_exists(
            'public/dists/stretch/main/debian-installer/binary-i386/Release')
        self.check_exists(
            'public/dists/stretch/main/debian-installer/binary-i386/Packages')
        self.check_exists(
            'public/dists/stretch/main/debian-installer/binary-i386/Packages.gz')
        self.check_exists(
            'public/dists/stretch/main/debian-installer/binary-i386/Packages.bz2')
        self.check_exists('public/dists/stretch/main/Contents-udeb-i386.gz')
        self.check_exists('public/dists/stretch/main/binary-amd64/Release')
        self.check_exists('public/dists/stretch/main/binary-amd64/Packages')
        self.check_exists('public/dists/stretch/main/binary-amd64/Packages.gz')
        self.check_exists(
            'public/dists/stretch/main/binary-amd64/Packages.bz2')
        self.check_exists('public/dists/stretch/main/Contents-amd64.gz')
        self.check_exists(
            'public/dists/stretch/main/debian-installer/binary-amd64/Release')
        self.check_exists(
            'public/dists/stretch/main/debian-installer/binary-amd64/Packages')
        self.check_exists(
            'public/dists/stretch/main/debian-installer/binary-amd64/Packages.gz')
        self.check_exists(
            'public/dists/stretch/main/debian-installer/binary-amd64/Packages.bz2')
        self.check_exists('public/dists/stretch/main/Contents-udeb-amd64.gz')
        self.check_not_exists('public/dists/stretch/main/source/Sources')
        self.check_not_exists('public/dists/stretch/main/source/Sources.gz')
        self.check_not_exists('public/dists/stretch/main/source/Sources.bz2')

        self.check_exists(
            'public/pool/main/g/gnupg2/gpgv-udeb_2.1.18-8~deb9u4_amd64.udeb')
        self.check_exists(
            'public/pool/main/g/gnupg2/gpgv-udeb_2.1.18-8~deb9u4_i386.udeb')
        self.check_exists(
            'public/pool/main/g/gnupg2/gpgv_2.1.18-8~deb9u4_amd64.deb')
        self.check_exists(
            'public/pool/main/g/gnupg2/gpgv_2.1.18-8~deb9u4_i386.deb')

        self.check_file_contents('public/dists/stretch/main/binary-i386/Packages',
                                 'packages_i386', match_prepare=sorted_processor)
        self.check_file_contents('public/dists/stretch/main/debian-installer/binary-i386/Packages',
                                 'packages_udeb_i386', match_prepare=sorted_processor)
        self.check_file_contents('public/dists/stretch/main/binary-amd64/Packages',
                                 'packages_amd64', match_prepare=sorted_processor)
        self.check_file_contents('public/dists/stretch/main/debian-installer/binary-amd64/Packages',
                                 'packages_udeb_amd64', match_prepare=sorted_processor)

        # verify contents except of sums
        self.check_file_contents(
            'public/dists/stretch/Release', 'release', match_prepare=strip_processor)

        self.check_file_contents('public/dists/stretch/main/debian-installer/binary-i386/Release',
                                 'release_udeb_i386', match_prepare=strip_processor)

        # verify sums
        release = self.read_file('public/dists/stretch/Release').split("\n")
        release = [l for l in release if l.startswith(" ")]
        pathsSeen = set()
        for l in release:
            fileHash, fileSize, path = l.split()
            if "Contents" in path and not path.endswith(".gz"):
                # "Contents" are present in index, but not really written to disk
                continue

            pathsSeen.add(path)

            fileSize = int(fileSize)

            st = os.stat(os.path.join(
                os.environ["HOME"], ".aptly", 'public/dists/stretch/', path))
            if fileSize != st.st_size:
                raise Exception("file size doesn't match for %s: %d != %d" % (
                    path, fileSize, st.st_size))

            if len(fileHash) == 32:
                h = hashlib.md5()
            elif len(fileHash) == 40:
                h = hashlib.sha1()
            elif len(fileHash) == 64:
                h = hashlib.sha256()
            else:
                h = hashlib.sha512()

            h.update(self.read_file(os.path.join('public/dists/stretch', path), mode='b'))

            if h.hexdigest() != fileHash:
                raise Exception("file hash doesn't match for %s: %s != %s" % (
                    path, fileHash, h.hexdigest()))

        pathsExepcted = set()
        for arch in ("i386", "amd64"):
            for udeb in ("", "debian-installer/"):
                for ext in ("", ".gz", ".bz2"):
                    pathsExepcted.add(
                        "main/%sbinary-%s/Packages%s" % (udeb, arch, ext))

                pathsExepcted.add("main/Contents-%s%s.gz" %
                                  ("udeb-" if udeb != "" else "", arch))
                pathsExepcted.add("Contents-%s%s.gz" %
                                  ("udeb-" if udeb != "" else "", arch))

                pathsExepcted.add("main/%sbinary-%s/Release" % (udeb, arch))

        if pathsSeen != pathsExepcted:
            raise Exception("path seen wrong: %r != %r" %
                            (pathsSeen, pathsExepcted))


class PublishSnapshot36Test(BaseTest):
    """
    publish snapshot: -skip-contents
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap36 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -skip-contents snap36"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot36Test, self).check()

        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Release')
        self.check_not_exists('public/dists/maverick/main/Contents-i386.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Release')
        self.check_not_exists('public/dists/maverick/main/Contents-amd64.gz')


class PublishSnapshot37Test(BaseTest):
    """
    publish snapshot: mirror with double mirror update
    """
    configOverride = {"max-tries": 1}
    fixtureGpg = True
    fixtureCmds = [
        "aptly -architectures=i386,amd64 mirror create -keyring=aptlytest.gpg -filter='$$Source (gnupg2)' -with-udebs stretch http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch main non-free",
        "aptly mirror update -keyring=aptlytest.gpg stretch",
        "aptly mirror update -keyring=aptlytest.gpg stretch",
        "aptly snapshot create stretch from mirror stretch",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec stretch"
    gold_processor = BaseTest.expand_environ


class PublishSnapshot38Test(BaseTest):
    """
    publish snapshot: mirror with installer
    """
    configOverride = {"max-tries": 1}
    fixtureGpg = True
    fixtureCmds = [
        "aptly -architectures=s390x mirror create -keyring=aptlytest.gpg -filter='installer' -with-installer stretch http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch main",
        "aptly mirror update -keyring=aptlytest.gpg stretch",
        "aptly snapshot create stretch from mirror stretch",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec stretch"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot38Test, self).check()
        self.check_exists(
            'public/dists/stretch/main/installer-s390x/current/images/SHA256SUMS')
        self.check_exists(
            'public/dists/stretch/main/installer-s390x/current/images/SHA256SUMS.gpg')
        self.check_exists(
            'public/dists/stretch/main/installer-s390x/current/images/generic/debian.exec')
        self.check_exists(
            'public/dists/stretch/main/installer-s390x/current/images/MANIFEST')

        self.check_file_contents('public/dists/stretch/main/installer-s390x/current/images/SHA256SUMS',
                                 "installer_s390x", match_prepare=sorted_processor)


class PublishSnapshot39Test(BaseTest):
    """
    publish snapshot: custom suite
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -suite=stable snap1"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot39Test, self).check()

        # verify contents except of sums
        self.check_file_contents(
            'public/dists/maverick/Release', 'release', match_prepare=strip_processor)

        self.check_file_contents(
            'public/dists/maverick/main/binary-i386/Release', 'release_i386')
        self.check_file_contents(
            'public/dists/maverick/main/binary-amd64/Release', 'release_amd64')

        self.check_file_contents('public/dists/maverick/main/binary-i386/Packages',
                                 'packages_i386', match_prepare=sorted_processor)
        self.check_file_contents('public/dists/maverick/main/binary-amd64/Packages',
                                 'packages_amd64', match_prepare=sorted_processor)

        self.check_file_contents('public/dists/maverick/main/Contents-i386.gz',
                                 'contents_i386', match_prepare=ungzip_if_required, mode='b', ensure_utf8=False)
        self.check_file_contents('public/dists/maverick/main/Contents-amd64.gz',
                                 'contents_amd64', match_prepare=ungzip_if_required, mode='b', ensure_utf8=False)


class PublishSnapshot40Test(BaseTest):
    """
    publish snapshot: -skip-bz2
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap40 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -skip-bz2 snap40"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot40Test, self).check()

        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Release')
        self.check_exists('public/dists/maverick/main/binary-amd64/Release')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_not_exists('public/dists/maverick/main/binary-i386/Packages.bz2')

        self.check_exists('public/dists/maverick/main/binary-amd64/Packages')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_not_exists('public/dists/maverick/main/binary-amd64/Packages.bz2')


class PublishSnapshot41Test(BaseTest):
    """
    publish snapshot: mirror with / in distribution
    """
    configOverride = {"max-tries": 1}
    fixtureGpg = True
    fixtureCmds = [
        "aptly -architectures='i386' mirror create -keyring=aptlytest.gpg -with-sources -filter='nginx | Priority (required)'"  # continued on next line
        " -filter-with-deps=true ps41 http://repo.aptly.info/system-tests/security.debian.org/debian-security buster/updates main",
        "aptly mirror update -keyring=aptlytest.gpg ps41",
        "aptly snapshot create snap41 from mirror ps41",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap41"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot41Test, self).check()

        self.check_exists('public/dists/buster/updates/Release')
        self.check_exists('public/dists/buster/updates/Release.gpg')
        self.check_exists('public/dists/buster/updates/InRelease')
        self.check_exists('public/dists/buster/updates/main/source/Release')
        self.check_exists('public/dists/buster/updates/main/source/Sources')
        self.check_exists('public/dists/buster/updates/main/source/Sources.gz')
        self.check_exists('public/dists/buster/updates/main/source/Sources.bz2')
        self.check_exists('public/dists/buster/updates/main/binary-i386/Packages')
        self.check_exists('public/dists/buster/updates/main/binary-i386/Packages.gz')
        self.check_exists('public/dists/buster/updates/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/buster/updates/main/binary-i386/Release')
        self.check_exists('public/dists/buster/updates/main/Contents-i386.gz')
        self.check_exists('public/dists/buster/updates/Contents-i386.gz')
        self.check_exists('public/pool/main/u/util-linux/bsdutils_2.33.1-0.1+deb10u1_i386.deb')
        self.check_exists('public/pool/main/u/util-linux/fdisk_2.33.1-0.1+deb10u1_i386.deb')
        self.check_exists('public/pool/main/u/util-linux/libblkid1_2.33.1-0.1+deb10u1_i386.deb')
        self.check_exists('public/pool/main/u/util-linux/libfdisk1_2.33.1-0.1+deb10u1_i386.deb')
        self.check_exists('public/pool/main/u/util-linux/libmount1_2.33.1-0.1+deb10u1_i386.deb')
        self.check_exists('public/pool/main/u/util-linux/libsmartcols1_2.33.1-0.1+deb10u1_i386.deb')
        self.check_exists('public/pool/main/u/util-linux/libuuid1_2.33.1-0.1+deb10u1_i386.deb')
        self.check_exists('public/pool/main/u/util-linux/mount_2.33.1-0.1+deb10u1_i386.deb')
        self.check_exists('public/pool/main/u/util-linux/util-linux_2.33.1-0.1+deb10u1_i386.deb')
        self.check_exists('public/pool/main/d/dpkg/dpkg_1.19.8_i386.deb')
        self.check_exists('public/pool/main/e/e2fsprogs/e2fslibs_1.44.5-1+deb10u2_i386.deb')
        self.check_exists('public/pool/main/e/e2fsprogs/e2fsprogs_1.44.5-1+deb10u2_i386.deb')
        self.check_exists('public/pool/main/e/e2fsprogs/libcom-err2_1.44.5-1+deb10u2_i386.deb')
        self.check_exists('public/pool/main/e/e2fsprogs/libcomerr2_1.44.5-1+deb10u2_i386.deb')
        self.check_exists('public/pool/main/e/e2fsprogs/libext2fs2_1.44.5-1+deb10u2_i386.deb')
        self.check_exists('public/pool/main/e/e2fsprogs/libss2_1.44.5-1+deb10u2_i386.deb')
        self.check_exists('public/pool/main/g/gzip/gzip_1.9-3+deb10u1_i386.deb')
        self.check_exists('public/pool/main/g/glibc/libc-bin_2.28-10+deb10u2_i386.deb')
        self.check_exists('public/pool/main/g/glibc/libc6_2.28-10+deb10u2_i386.deb')
        self.check_exists('public/pool/main/g/glibc/multiarch-support_2.28-10+deb10u2_i386.deb')
        self.check_exists('public/pool/main/b/bzip2/libbz2-1.0_1.0.6-9.2~deb10u2_i386.deb')
        self.check_exists('public/pool/main/f/freetype/libfreetype6_2.9.1-3+deb10u2_i386.deb')
        self.check_exists('public/pool/main/libg/libgd2/libgd3_2.2.5-5.2+deb10u1_i386.deb')
        self.check_exists('public/pool/main/i/icu/libicu63_63.1-6+deb10u2_i386.deb')
        self.check_exists('public/pool/main/l/lz4/liblz4-1_1.8.3-1+deb10u1_i386.deb')
        self.check_exists('public/pool/main/x/xz-utils/liblzma5_5.2.4-1+deb10u1_i386.deb')
        self.check_exists('public/pool/main/n/ncurses/libncursesw6_6.1+20181013-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/ncurses/libtinfo5_6.1+20181013-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/ncurses/libtinfo6_6.1+20181013-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/ncurses/ncurses-base_6.1+20181013-2+deb10u5_all.deb')
        self.check_exists('public/pool/main/n/ncurses/ncurses-bin_6.1+20181013-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/libnginx-mod-http-auth-pam_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/libnginx-mod-http-cache-purge_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/libnginx-mod-http-dav-ext_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/libnginx-mod-http-echo_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/libnginx-mod-http-fancyindex_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/libnginx-mod-http-geoip_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/libnginx-mod-http-headers-more-filter_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/libnginx-mod-http-image-filter_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/libnginx-mod-http-lua_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/libnginx-mod-http-ndk_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/libnginx-mod-http-perl_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/libnginx-mod-http-subs-filter_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/libnginx-mod-http-uploadprogress_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/libnginx-mod-http-upstream-fair_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/libnginx-mod-http-xslt-filter_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/libnginx-mod-mail_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/libnginx-mod-nchan_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/libnginx-mod-stream_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/nginx_1.14.2-2+deb10u5_all.deb')
        self.check_exists('public/pool/main/n/nginx/nginx_1.14.2-2+deb10u5.debian.tar.xz')
        self.check_exists('public/pool/main/n/nginx/nginx_1.14.2-2+deb10u5.dsc')
        self.check_exists('public/pool/main/n/nginx/nginx_1.14.2.orig.tar.gz')
        self.check_exists('public/pool/main/n/nginx/nginx-common_1.14.2-2+deb10u5_all.deb')
        self.check_exists('public/pool/main/n/nginx/nginx-extras_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/nginx-full_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/n/nginx/nginx-light_1.14.2-2+deb10u5_i386.deb')
        self.check_exists('public/pool/main/o/openssl/libssl1.1_1.1.1n-0+deb10u6_i386.deb')
        self.check_exists('public/pool/main/s/systemd/libsystemd0_241-7~deb10u10_i386.deb')
        self.check_exists('public/pool/main/s/systemd/libudev1_241-7~deb10u10_i386.deb')
        self.check_exists('public/pool/main/t/tiff/libtiff5_4.1.0+git191117-2~deb10u9_i386.deb')
        self.check_exists('public/pool/main/t/tar/tar_1.30+dfsg-6+deb10u1_i386.deb')
        self.check_exists('public/pool/main/t/tzdata/tzdata_2021a-0+deb10u12_all.deb')
        self.check_exists('public/pool/main/libw/libwebp/libwebp6_0.6.1-2+deb10u3_i386.deb')
        self.check_exists('public/pool/main/libx/libx11/libx11-6_1.6.7-1+deb10u4_i386.deb')
        self.check_exists('public/pool/main/libx/libx11/libx11-data_1.6.7-1+deb10u4_all.deb')
        self.check_exists('public/pool/main/libx/libxml2/libxml2_2.9.4+dfsg1-7+deb10u6_i386.deb')
        self.check_exists('public/pool/main/libx/libxpm/libxpm4_3.5.12-1+deb10u2_i386.deb')
        self.check_exists('public/pool/main/libx/libxslt/libxslt1.1_1.1.32-2.2~deb10u2_i386.deb')
        self.check_exists('public/pool/main/libz/libzstd/libzstd1_1.3.8+dfsg-3+deb10u2_i386.deb')
        self.check_exists('public/pool/main/z/zlib/zlib1g_1.2.11.dfsg-1+deb10u2_i386.deb')
