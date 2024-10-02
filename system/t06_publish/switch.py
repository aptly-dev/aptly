import os
import hashlib
import inspect
from lib import BaseTest


def strip_processor(output):
    return "\n".join([l for l in output.split("\n") if not l.startswith(' ') and not l.startswith('Date:')])


class PublishSwitch1Test(BaseTest):
    """
    publish switch: removed some packages
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly snapshot pull -no-deps -architectures=i386,amd64 snap2 snap1 snap3 gnuplot-x11",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick snap1",
    ]
    runCmd = "aptly publish switch -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick snap3"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSwitch1Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/Contents-i386.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_exists('public/dists/maverick/main/Contents-amd64.gz')

        self.check_exists('public/pool/main/g/gnuplot/gnuplot-x11_4.6.1-1~maverick2_i386.deb')
        self.check_exists('public/pool/main/g/gnuplot/gnuplot-x11_4.6.1-1~maverick2_amd64.deb')
        self.check_not_exists('public/pool/main/g/gnuplot/gnuplot-nox_4.6.1-1~maverick2_i386.deb')
        self.check_not_exists('public/pool/main/g/gnuplot/gnuplot-nox_4.6.1-1~maverick2_amd64.deb')

        # verify contents except of sums
        self.check_file_contents('public/dists/maverick/Release', 'release', match_prepare=strip_processor)
        self.check_file_contents('public/dists/maverick/main/binary-i386/Packages', 'binary', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))

        # verify signatures
        self.run_cmd([self.gpgFinder.gpg, "--no-auto-check-trustdb", "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/InRelease')])
        self.run_cmd([self.gpgFinder.gpg, "--no-auto-check-trustdb",  "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/Release.gpg'),
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

            st = os.stat(os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/', path))
            if fileSize != st.st_size:
                raise Exception("file size doesn't match for %s: %d != %d" % (path, fileSize, st.st_size))

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
                raise Exception("file hash doesn't match for %s: %s != %s" % (path, fileHash, h.hexdigest()))

        if pathsSeen != set(['main/binary-amd64/Packages', 'main/binary-i386/Packages', 'main/binary-i386/Packages.gz',
                             'main/binary-amd64/Packages.gz', 'main/binary-amd64/Packages.bz2', 'main/binary-i386/Packages.bz2',
                             'main/binary-amd64/Release', 'main/binary-i386/Release', 'main/Contents-amd64.gz',
                             'main/Contents-i386.gz', 'Contents-i386.gz', 'Contents-amd64.gz']):
            raise Exception("path seen wrong: %r" % (pathsSeen, ))


class PublishSwitch2Test(BaseTest):
    """
    publish switch: added some packages
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly snapshot pull -no-deps -architectures=i386,amd64 snap2 snap1 snap3 gnuplot-x11",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick snap3 ppa",
    ]
    runCmd = "aptly publish switch -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick ppa snap1"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSwitch2Test, self).check()

        self.check_exists('public/ppa/dists/maverick/InRelease')
        self.check_exists('public/ppa/dists/maverick/Release')
        self.check_exists('public/ppa/dists/maverick/Release.gpg')

        self.check_exists('public/ppa/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/ppa/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/ppa/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/ppa/dists/maverick/main/Contents-i386.gz')
        self.check_exists('public/ppa/dists/maverick/main/binary-amd64/Packages')
        self.check_exists('public/ppa/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_exists('public/ppa/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_exists('public/ppa/dists/maverick/main/Contents-amd64.gz')

        self.check_exists('public/ppa/pool/main/g/gnuplot/gnuplot-x11_4.6.1-1~maverick2_i386.deb')
        self.check_exists('public/ppa/pool/main/g/gnuplot/gnuplot-x11_4.6.1-1~maverick2_amd64.deb')
        self.check_exists('public/ppa/pool/main/g/gnuplot/gnuplot-nox_4.6.1-1~maverick2_i386.deb')
        self.check_exists('public/ppa/pool/main/g/gnuplot/gnuplot-nox_4.6.1-1~maverick2_amd64.deb')

        # verify contents except of sums
        self.check_file_contents('public/ppa/dists/maverick/main/binary-i386/Packages', 'binary', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))


class PublishSwitch3Test(BaseTest):
    """
    publish switch: removed some packages, files occupied by another package
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly snapshot pull -no-deps -architectures=i386,amd64 snap2 snap1 snap3 gnuplot-x11",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick snap1",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick2 snap1",
    ]
    runCmd = "aptly publish switch -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick snap3"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSwitch3Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/Contents-i386.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_exists('public/dists/maverick/main/Contents-amd64.gz')

        self.check_exists('public/pool/main/g/gnuplot/gnuplot-x11_4.6.1-1~maverick2_i386.deb')
        self.check_exists('public/pool/main/g/gnuplot/gnuplot-x11_4.6.1-1~maverick2_amd64.deb')
        self.check_exists('public/pool/main/g/gnuplot/gnuplot-nox_4.6.1-1~maverick2_i386.deb')
        self.check_exists('public/pool/main/g/gnuplot/gnuplot-nox_4.6.1-1~maverick2_amd64.deb')


class PublishSwitch4Test(BaseTest):
    """
    publish switch: added some packages, but list of published archs doesn't change
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly snapshot pull -no-deps -architectures=i386 snap2 snap1 snap3 gnuplot-x11",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick snap3 ppa",
    ]
    runCmd = "aptly publish switch -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick ppa snap1"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSwitch4Test, self).check()

        self.check_exists('public/ppa/dists/maverick/InRelease')
        self.check_exists('public/ppa/dists/maverick/Release')
        self.check_exists('public/ppa/dists/maverick/Release.gpg')

        self.check_exists('public/ppa/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/ppa/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/ppa/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/ppa/dists/maverick/main/Contents-i386.gz')
        self.check_not_exists('public/ppa/dists/maverick/main/binary-amd64/Packages')
        self.check_not_exists('public/ppa/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_not_exists('public/ppa/dists/maverick/main/binary-amd64/Packages.bz2')

        self.check_exists('public/ppa/pool/main/g/gnuplot/gnuplot-x11_4.6.1-1~maverick2_i386.deb')
        self.check_not_exists('public/ppa/pool/main/g/gnuplot/gnuplot-x11_4.6.1-1~maverick2_amd64.deb')
        self.check_exists('public/ppa/pool/main/g/gnuplot/gnuplot-nox_4.6.1-1~maverick2_i386.deb')
        self.check_not_exists('public/ppa/pool/main/g/gnuplot/gnuplot-nox_4.6.1-1~maverick2_amd64.deb')


class PublishSwitch5Test(BaseTest):
    """
    publish switch: no such publish
    """
    fixtureCmds = [
        "aptly snapshot create snap1 empty",
    ]
    runCmd = "aptly publish switch maverick ppa snap1"
    expectedCode = 1


class PublishSwitch6Test(BaseTest):
    """
    publish switch: not a snapshot
    """
    fixtureCmds = [
        "aptly snapshot create snap1 empty",
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
        "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick local-repo",
    ]
    runCmd = "aptly publish switch maverick snap1"
    expectedCode = 1


class PublishSwitch7Test(BaseTest):
    """
    publish switch: no snapshot
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick snap1",
    ]
    runCmd = "aptly publish switch -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick snap3"
    expectedCode = 1


class PublishSwitch8Test(BaseTest):
    """
    publish switch: multi-component switching
    """
    fixtureDB = True
    fixturePoolCopy = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
        "aptly snapshot create local1 from repo local-repo",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=a,b,c snap1 snap2 local1",
        "aptly snapshot pull -no-deps -architectures=i386,amd64 snap2 snap1 snap3 gnuplot-x11",
        "aptly repo remove local-repo pyspi",
        "aptly snapshot create local2 from repo local-repo",
    ]
    runCmd = "aptly publish switch -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -component=b,c maverick snap3 local2"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSwitch8Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        for component in ("a", "b", "c"):
            self.check_exists('public/dists/maverick/' + component + '/binary-i386/Packages')
            self.check_exists('public/dists/maverick/' + component + '/binary-i386/Packages.gz')
            self.check_exists('public/dists/maverick/' + component + '/binary-i386/Packages.bz2')
            self.check_exists('public/dists/maverick/' + component + '/Contents-i386.gz')
            self.check_exists('public/dists/maverick/' + component + '/binary-amd64/Packages')
            self.check_exists('public/dists/maverick/' + component + '/binary-amd64/Packages.gz')
            self.check_exists('public/dists/maverick/' + component + '/binary-amd64/Packages.bz2')
            if component == "c":
                self.check_not_exists('public/dists/maverick/' + component + '/Contents-amd64.gz')
            else:
                self.check_exists('public/dists/maverick/' + component + '/Contents-amd64.gz')
            self.check_exists('public/dists/maverick/' + component + '/source/Sources')
            self.check_exists('public/dists/maverick/' + component + '/source/Sources.gz')
            self.check_exists('public/dists/maverick/' + component + '/source/Sources.bz2')

        self.check_exists('public/pool/a/g/gnuplot/gnuplot-x11_4.6.1-1~maverick2_i386.deb')
        self.check_exists('public/pool/a/g/gnuplot/gnuplot-x11_4.6.1-1~maverick2_amd64.deb')
        self.check_exists('public/pool/a/g/gnuplot/gnuplot-nox_4.6.1-1~maverick2_i386.deb')
        self.check_exists('public/pool/a/g/gnuplot/gnuplot-nox_4.6.1-1~maverick2_amd64.deb')

        self.check_exists('public/pool/b/g/gnuplot/gnuplot-x11_4.6.1-1~maverick2_i386.deb')
        self.check_exists('public/pool/b/g/gnuplot/gnuplot-x11_4.6.1-1~maverick2_amd64.deb')
        self.check_not_exists('public/pool/b/g/gnuplot/gnuplot-nox_4.6.1-1~maverick2_i386.deb')
        self.check_not_exists('public/pool/b/g/gnuplot/gnuplot-nox_4.6.1-1~maverick2_amd64.deb')

        self.check_exists('public/pool/c/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb')
        self.check_not_exists('public/pool/c/p/pyspi/pyspi_0.6.1-1.3.dsc')
        self.check_not_exists('public/pool/c/p/pyspi/pyspi_0.6.1-1.3.diff.gz')
        self.check_not_exists('public/pool/c/p/pyspi/pyspi_0.6.1.orig.tar.gz')
        self.check_not_exists('public/pool/c/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc')

        # verify contents except of sums
        self.check_file_contents('public/dists/maverick/Release', 'release', match_prepare=strip_processor)
        self.check_file_contents('public/dists/maverick/a/binary-i386/Packages', 'binaryA', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))
        self.check_file_contents('public/dists/maverick/b/binary-i386/Packages', 'binaryB', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))
        self.check_file_contents('public/dists/maverick/c/binary-i386/Packages', 'binaryC', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))

        # verify signatures
        self.run_cmd([self.gpgFinder.gpg, "--no-auto-check-trustdb", "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/InRelease')])
        self.run_cmd([self.gpgFinder.gpg, "--no-auto-check-trustdb",  "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/Release.gpg'),
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

            st = os.stat(os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/', path))
            if fileSize != st.st_size:
                raise Exception("file size doesn't match for %s: %d != %d" % (path, fileSize, st.st_size))

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
                raise Exception("file hash doesn't match for %s: %s != %s" % (path, fileHash, h.hexdigest()))

        if pathsSeen != set(['a/binary-amd64/Packages', 'c/source/Sources', 'c/binary-amd64/Packages.bz2', 'b/binary-amd64/Packages',
                             'a/source/Sources', 'a/binary-i386/Packages.bz2', 'b/source/Sources.bz2', 'b/binary-amd64/Packages.bz2',
                             'c/binary-i386/Packages', 'a/binary-i386/Packages', 'c/binary-amd64/Packages', 'a/source/Sources.gz',
                             'b/binary-i386/Packages.gz', 'c/binary-amd64/Packages.gz', 'a/binary-amd64/Packages.bz2', 'c/source/Sources.bz2',
                             'c/source/Sources.gz', 'a/source/Sources.bz2', 'b/binary-i386/Packages.bz2', 'a/binary-i386/Packages.gz',
                             'a/binary-amd64/Packages.gz', 'c/binary-i386/Packages.bz2', 'b/binary-amd64/Packages.gz', 'b/source/Sources',
                             'c/binary-i386/Packages.gz', 'b/source/Sources.gz', 'b/binary-i386/Packages',
                             'a/binary-amd64/Release', 'b/binary-amd64/Release', 'c/binary-amd64/Release',
                             'a/binary-i386/Release', 'b/binary-i386/Release', 'c/binary-i386/Release',
                             'a/source/Release', 'b/source/Release', 'c/source/Release',
                             'b/Contents-amd64.gz', 'c/Contents-i386.gz', 'a/Contents-i386.gz',
                             'a/Contents-amd64.gz', 'b/Contents-i386.gz', 'Contents-i386.gz', 'Contents-amd64.gz']):
            raise Exception("path seen wrong: %r" % (pathsSeen, ))


class PublishSwitch9Test(BaseTest):
    """
    publish switch: components/snapshots mismatch
    """
    fixtureCmds = [
        "aptly snapshot create snap1 empty",
        "aptly snapshot create snap2 empty",
        "aptly publish snapshot -architectures=i386 -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=a,b snap1 snap2",
    ]
    runCmd = "aptly publish switch -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -component=a,b maverick snap2"
    expectedCode = 2

    def outputMatchPrepare(self, s):
        return "\n".join([l for l in self.ensure_utf8(s).split("\n") if l.startswith("ERROR")])


class PublishSwitch10Test(BaseTest):
    """
    publish switch: conflicting files in the snapshot
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
    runCmd = "aptly publish switch -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick snap2"
    expectedCode = 1
    gold_processor = BaseTest.expand_environ


class PublishSwitch11Test(BaseTest):
    """
    publish switch: -force-overwrite
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
    runCmd = "aptly publish switch -force-overwrite -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick snap2"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSwitch11Test, self).check()

        self.check_file_contents("public/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz", "file")


class PublishSwitch12Test(BaseTest):
    """
    publish switch: wrong component names
    """
    fixtureCmds = [
        "aptly snapshot create snap1 empty",
        "aptly snapshot create snap2 empty",
        "aptly publish snapshot -architectures=i386 -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=a,b snap1 snap2",
    ]
    runCmd = "aptly publish switch -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -component=a,c maverick snap2 snap1"
    expectedCode = 1


class PublishSwitch13Test(BaseTest):
    """
    publish switch: -skip-contents
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly snapshot pull -no-deps -architectures=i386,amd64 snap2 snap1 snap3 gnuplot-x11",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -skip-contents snap1",
    ]
    runCmd = "aptly publish switch -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick snap3"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSwitch13Test, self).check()

        self.check_exists('public/dists/maverick/Release')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_not_exists('public/dists/maverick/main/Contents-i386.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages')
        self.check_not_exists('public/dists/maverick/main/Contents-amd64.gz')


class PublishSwitch14Test(BaseTest):
    """
    publish switch: removed some packages skipping cleanup
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly snapshot pull -no-deps -architectures=i386,amd64 snap2 snap1 snap3 gnuplot-x11",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick snap1",
    ]
    runCmd = "aptly publish switch -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -skip-cleanup maverick snap3"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSwitch14Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/Contents-i386.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_exists('public/dists/maverick/main/Contents-amd64.gz')

        self.check_exists('public/pool/main/g/gnuplot/gnuplot-x11_4.6.1-1~maverick2_i386.deb')
        self.check_exists('public/pool/main/g/gnuplot/gnuplot-x11_4.6.1-1~maverick2_amd64.deb')
        self.check_exists('public/pool/main/g/gnuplot/gnuplot-nox_4.6.1-1~maverick2_i386.deb')
        self.check_exists('public/pool/main/g/gnuplot/gnuplot-nox_4.6.1-1~maverick2_amd64.deb')

        # verify contents except of sums
        self.check_file_contents('public/dists/maverick/Release', 'release', match_prepare=strip_processor)
        self.check_file_contents('public/dists/maverick/main/binary-i386/Packages', 'binary', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))

        # verify signatures
        self.run_cmd([self.gpgFinder.gpg, "--no-auto-check-trustdb", "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/InRelease')])
        self.run_cmd([self.gpgFinder.gpg, "--no-auto-check-trustdb",  "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/Release.gpg'),
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

            st = os.stat(os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/', path))
            if fileSize != st.st_size:
                raise Exception("file size doesn't match for %s: %d != %d" % (path, fileSize, st.st_size))

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
                raise Exception("file hash doesn't match for %s: %s != %s" % (path, fileHash, h.hexdigest()))

        if pathsSeen != set(['main/binary-amd64/Packages', 'main/binary-i386/Packages', 'main/binary-i386/Packages.gz',
                             'main/binary-amd64/Packages.gz', 'main/binary-amd64/Packages.bz2', 'main/binary-i386/Packages.bz2',
                             'main/binary-amd64/Release', 'main/binary-i386/Release', 'main/Contents-amd64.gz',
                             'main/Contents-i386.gz', 'Contents-i386.gz', 'Contents-amd64.gz']):
            raise Exception("path seen wrong: %r" % (pathsSeen, ))


class PublishSwitch15Test(BaseTest):
    """
    publish switch: -skip-bz2
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 empty",
        "aptly snapshot pull -no-deps -architectures=i386,amd64 snap2 snap1 snap3 gnuplot-x11",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -skip-bz2 snap1",
    ]
    runCmd = "aptly publish switch -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick snap3"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSwitch15Test, self).check()

        self.check_exists('public/dists/maverick/Release')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_not_exists('public/dists/maverick/main/binary-i386/Packages.bz2')

        self.check_exists('public/dists/maverick/main/binary-amd64/Packages')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_not_exists('public/dists/maverick/main/binary-amd64/Packages.bz2')
