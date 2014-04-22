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

        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.bz2')

        self.check_exists('public/pool/main/g/gnuplot/gnuplot-x11_4.6.1-1~maverick2_i386.deb')
        self.check_exists('public/pool/main/g/gnuplot/gnuplot-x11_4.6.1-1~maverick2_amd64.deb')
        self.check_not_exists('public/pool/main/g/gnuplot/gnuplot-nox_4.6.1-1~maverick2_i386.deb')
        self.check_not_exists('public/pool/main/g/gnuplot/gnuplot-nox_4.6.1-1~maverick2_amd64.deb')

        # verify contents except of sums
        self.check_file_contents('public/dists/maverick/Release', 'release', match_prepare=strip_processor)
        self.check_file_contents('public/dists/maverick/main/binary-i386/Packages', 'binary', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))

        # verify signatures
        self.run_cmd(["gpg", "--no-auto-check-trustdb", "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
                      "--verify", os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/InRelease')])
        self.run_cmd(["gpg", "--no-auto-check-trustdb",  "--keyring", os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "aptly.pub"),
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

        self.check_exists('public/ppa/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/ppa/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/ppa/dists/maverick/main/binary-amd64/Packages')
        self.check_exists('public/ppa/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_exists('public/ppa/dists/maverick/main/binary-amd64/Packages.bz2')

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

        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.bz2')

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

        self.check_exists('public/ppa/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/ppa/dists/maverick/main/binary-i386/Packages.bz2')
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
