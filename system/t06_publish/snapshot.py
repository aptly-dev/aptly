import os
import hashlib
import inspect
from lib import BaseTest


def strip_processor(output):
    return "\n".join([l for l in output.split("\n") if not l.startswith(' ') and not l.startswith('Date:')])


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

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.bz2')

        self.check_exists('public/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')

        # verify contents except of sums
        self.check_file_contents('public/dists/maverick/Release', 'release', match_prepare=strip_processor)

        # verify signatures
        self.run_cmd(["gpg", "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/InRelease')])
        self.run_cmd(["gpg",  "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/Release.gpg'),
                      os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/Release')])

        # verify sums
        release = self.read_file('public/dists/maverick/Release').split("\n")
        release = [l for l in release if l.startswith(" ")]
        pathsSeen = set()
        for l in release:
            fileHash, fileSize, path = l.split()
            pathsSeen.add(path)

            fileSize = int(fileSize)

            st = os.stat(os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/', path))
            if fileSize != st.st_size:
                raise Exception("file size doesn't match for %s: %d != %d" % (path, fileSize, st.st_size))

            if len(fileHash) == 32:
                h = hashlib.md5()
            elif len(fileHash) == 40:
                h = hashlib.sha1()
            else:
                h = hashlib.sha256()

            h.update(self.read_file(os.path.join('public/dists/maverick', path)))

            if h.hexdigest() != fileHash:
                raise Exception("file hash doesn't match for %s: %s != %s" % (path, fileHash, h.hexdigest()))

        if pathsSeen != set(['main/binary-amd64/Packages', 'main/binary-i386/Packages', 'main/binary-i386/Packages.gz',
                             'main/binary-amd64/Packages.gz', 'main/binary-amd64/Packages.bz2', 'main/binary-i386/Packages.bz2']):
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
        self.check_exists('public/dists/squeeze/main/binary-amd64/Packages')
        self.check_exists('public/dists/squeeze/main/binary-amd64/Packages.gz')
        self.check_exists('public/dists/squeeze/main/binary-amd64/Packages.bz2')

        self.check_exists('public/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')

        # verify contents except of sums
        self.check_file_contents('public/dists/squeeze/Release', 'release', match_prepare=strip_processor)


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
        self.check_exists('public/dists/squeeze/contrib/binary-i386/Packages.gz')
        self.check_exists('public/dists/squeeze/contrib/binary-i386/Packages.bz2')
        self.check_exists('public/dists/squeeze/contrib/binary-amd64/Packages')
        self.check_exists('public/dists/squeeze/contrib/binary-amd64/Packages.gz')
        self.check_exists('public/dists/squeeze/contrib/binary-amd64/Packages.bz2')

        self.check_exists('public/pool/contrib/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')

        # verify contents except of sums
        self.check_file_contents('public/dists/squeeze/Release', 'release', match_prepare=strip_processor)


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
        self.check_not_exists('public/dists/squeeze/main/binary-amd64/Packages')
        self.check_not_exists('public/dists/squeeze/main/binary-amd64/Packages.gz')
        self.check_not_exists('public/dists/squeeze/main/binary-amd64/Packages.bz2')

        self.check_exists('public/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')

        # verify contents except of sums
        self.check_file_contents('public/dists/squeeze/Release', 'release', match_prepare=strip_processor)


class PublishSnapshot5Test(BaseTest):
    """
    publish snapshot: under prefix
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap5 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=squeeze snap5 ppa/smira"

    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSnapshot5Test, self).check()

        self.check_exists('public/ppa/smira/dists/squeeze/InRelease')
        self.check_exists('public/ppa/smira/dists/squeeze/Release')
        self.check_exists('public/ppa/smira/dists/squeeze/Release.gpg')

        self.check_exists('public/ppa/smira/dists/squeeze/main/binary-i386/Packages')
        self.check_exists('public/ppa/smira/dists/squeeze/main/binary-i386/Packages.gz')
        self.check_exists('public/ppa/smira/dists/squeeze/main/binary-i386/Packages.bz2')
        self.check_exists('public/ppa/smira/dists/squeeze/main/binary-amd64/Packages')
        self.check_exists('public/ppa/smira/dists/squeeze/main/binary-amd64/Packages.gz')
        self.check_exists('public/ppa/smira/dists/squeeze/main/binary-amd64/Packages.bz2')

        self.check_exists('public/ppa/smira/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')


class PublishSnapshot6Test(BaseTest):
    """
    publish snapshot: specify distribution
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap from mirror gnuplot-maverick",
        "aptly snapshot merge snap6 snap"
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
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.bz2')

        # verify contents except of sums
        self.check_file_contents('public/dists/maverick/Release', 'release', match_prepare=strip_processor)


class PublishSnapshot14Test(BaseTest):
    """
    publish snapshot: empty snapshot is not publishable
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
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.bz2')

        # verify contents except of sums
        self.check_file_contents('public/dists/maverick/Release', 'release', match_prepare=strip_processor)


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
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_exists('public/dists/maverick/main/source/Sources')
        self.check_exists('public/dists/maverick/main/source/Sources.gz')
        self.check_exists('public/dists/maverick/main/source/Sources.bz2')

        self.check_exists('public/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')
        self.check_exists('public/pool/main/g/gnuplot/gnuplot_4.6.1-1~maverick2.debian.tar.gz')
        self.check_exists('public/pool/main/g/gnuplot/gnuplot_4.6.1-1~maverick2.dsc')
        self.check_exists('public/pool/main/g/gnuplot/gnuplot_4.6.1.orig.tar.gz')

        # verify contents except of sums
        self.check_file_contents('public/dists/maverick/Release', 'release', match_prepare=strip_processor)
        self.check_file_contents('public/dists/maverick/main/source/Sources', 'sources', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))

        # verify signatures
        self.run_cmd(["gpg", "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/InRelease')])
        self.run_cmd(["gpg",  "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/Release.gpg'),
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
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/source/Sources')
        self.check_exists('public/dists/maverick/main/source/Sources.gz')
        self.check_exists('public/dists/maverick/main/source/Sources.bz2')

        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.dsc')
        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.diff.gz')
        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz')
        self.check_exists('public/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc')
        self.check_exists('public/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb')

        # verify contents except of sums
        self.check_file_contents('public/dists/maverick/Release', 'release', match_prepare=strip_processor)
        self.check_file_contents('public/dists/maverick/main/source/Sources', 'sources', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))
        self.check_file_contents('public/dists/maverick/main/binary-i386/Packages', 'binary', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))

        # verify signatures
        self.run_cmd(["gpg", "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/InRelease')])
        self.run_cmd(["gpg",  "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/Release.gpg'),
                      os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/Release')])
