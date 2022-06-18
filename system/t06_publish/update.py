import os
import hashlib
import inspect
from lib import BaseTest


def strip_processor(output):
    return "\n".join([l for l in output.split("\n") if not l.startswith(' ') and not l.startswith('Date:')])


class PublishUpdate1Test(BaseTest):
    """
    publish update: removed some packages
    """
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}/",
        "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick local-repo",
        "aptly repo remove local-repo pyspi"
    ]
    runCmd = "aptly publish update -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishUpdate1Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/Contents-i386.gz')
        self.check_exists('public/dists/maverick/main/Contents-i386.gz')
        self.check_exists('public/dists/maverick/main/source/Sources')
        self.check_exists('public/dists/maverick/main/source/Sources.gz')
        self.check_exists('public/dists/maverick/main/source/Sources.bz2')

        self.check_not_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.dsc')
        self.check_not_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.diff.gz')
        self.check_not_exists('public/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz')
        self.check_not_exists('public/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc')
        self.check_exists('public/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb')

        # verify contents except of sums
        self.check_file_contents('public/dists/maverick/Release', 'release', match_prepare=strip_processor)
        self.check_file_contents('public/dists/maverick/main/source/Sources', 'sources', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))
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

        if pathsSeen != set(['main/binary-i386/Packages', 'main/binary-i386/Packages.bz2', 'main/binary-i386/Packages.gz',
                             'main/source/Sources', 'main/source/Sources.gz', 'main/source/Sources.bz2',
                             'main/binary-i386/Release', 'main/source/Release', 'main/Contents-i386.gz',
                             'Contents-i386.gz']):
            raise Exception("path seen wrong: %r" % (pathsSeen, ))


class PublishUpdate2Test(BaseTest):
    """
    publish update: added some packages
    """
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}/libboost-program-options-dev_1.49.0.1_i386.deb ${files}/pyspi_0.6.1-1.3.dsc",
        "aptly publish repo -acquire-by-hash -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick local-repo",
        "aptly repo add local-repo ${files}/pyspi-0.6.1-1.3.stripped.dsc"
    ]
    runCmd = "aptly publish update -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishUpdate2Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/Contents-i386.gz')
        self.check_exists('public/dists/maverick/main/source/Sources')
        self.check_exists('public/dists/maverick/main/source/by-hash/MD5Sum/Sources')
        self.check_exists('public/dists/maverick/main/source/by-hash/MD5Sum/Sources.old')
        self.check_exists('public/dists/maverick/main/source/Sources.gz')
        self.check_exists('public/dists/maverick/main/source/by-hash/MD5Sum/Sources.gz')
        self.check_exists('public/dists/maverick/main/source/by-hash/MD5Sum/Sources.gz.old')
        self.check_exists('public/dists/maverick/main/source/Sources.bz2')
        self.check_exists('public/dists/maverick/main/source/by-hash/MD5Sum/Sources.bz2')
        self.check_exists('public/dists/maverick/main/source/by-hash/MD5Sum/Sources.bz2.old')

        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.dsc')
        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.diff.gz')
        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz')
        self.check_exists('public/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc')
        self.check_exists('public/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb')

        # verify contents except of sums
        self.check_file_contents('public/dists/maverick/main/source/Sources', 'sources', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))
        self.check_file_contents('public/dists/maverick/main/binary-i386/Packages', 'binary', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))


class PublishUpdate3Test(BaseTest):
    """
    publish update: removed some packages, files occupied by another package
    """
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}/",
        "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick local-repo",
        "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick2 local-repo",
        "aptly repo remove local-repo pyspi"
    ]
    runCmd = "aptly publish update -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishUpdate3Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/Contents-i386.gz')
        self.check_exists('public/dists/maverick/main/source/Sources')
        self.check_exists('public/dists/maverick/main/source/Sources.gz')
        self.check_exists('public/dists/maverick/main/source/Sources.bz2')

        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.dsc')
        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.diff.gz')
        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz')
        self.check_exists('public/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc')
        self.check_exists('public/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb')


class PublishUpdate4Test(BaseTest):
    """
    publish update: added some packages, but list of published archs doesn't change
    """
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}/pyspi_0.6.1-1.3.dsc",
        "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick local-repo",
        "aptly repo add local-repo ${files}/libboost-program-options-dev_1.49.0.1_i386.deb"
    ]
    runCmd = "aptly publish update -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishUpdate4Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_not_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_not_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_not_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/source/Sources')
        self.check_exists('public/dists/maverick/main/source/Sources.gz')
        self.check_exists('public/dists/maverick/main/source/Sources.bz2')

        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.dsc')
        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.diff.gz')
        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz')
        self.check_not_exists('public/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb')


class PublishUpdate5Test(BaseTest):
    """
    publish update: no such publish
    """
    runCmd = "aptly publish update maverick ppa"
    expectedCode = 1


class PublishUpdate6Test(BaseTest):
    """
    publish update: not a local repo
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1",
    ]
    runCmd = "aptly publish update maverick"
    expectedCode = 1


class PublishUpdate7Test(BaseTest):
    """
    publish update: multiple components, add some packages
    """
    fixtureCmds = [
        "aptly repo create repo1",
        "aptly repo create repo2",
        "aptly repo add repo1 ${files}/pyspi_0.6.1-1.3.dsc",
        "aptly repo add repo2 ${files}/libboost-program-options-dev_1.49.0.1_i386.deb",
        "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=main,contrib repo1 repo2",
        "aptly repo add repo1 ${files}/pyspi-0.6.1-1.3.stripped.dsc",
    ]
    runCmd = "aptly publish update -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishUpdate7Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/source/Sources')
        self.check_exists('public/dists/maverick/main/source/Sources.gz')
        self.check_exists('public/dists/maverick/main/source/Sources.bz2')

        self.check_exists('public/dists/maverick/contrib/binary-i386/Packages')
        self.check_exists('public/dists/maverick/contrib/binary-i386/Packages.gz')
        self.check_exists('public/dists/maverick/contrib/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/contrib/Contents-i386.gz')
        self.check_exists('public/dists/maverick/contrib/source/Sources')
        self.check_exists('public/dists/maverick/contrib/source/Sources.gz')
        self.check_exists('public/dists/maverick/contrib/source/Sources.bz2')

        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.dsc')
        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.diff.gz')
        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz')
        self.check_exists('public/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc')
        self.check_exists('public/pool/contrib/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb')

        # verify contents except of sums
        self.check_file_contents('public/dists/maverick/main/source/Sources', 'sources', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))
        self.check_file_contents('public/dists/maverick/main/binary-i386/Packages', 'binary', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))
        self.check_file_contents('public/dists/maverick/contrib/source/Sources', 'sources2', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))
        self.check_file_contents('public/dists/maverick/contrib/binary-i386/Packages', 'binary2', match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))


class PublishUpdate8Test(BaseTest):
    """
    publish update: update empty repos to empty repos
    """
    fixtureCmds = [
        "aptly repo create repo1",
        "aptly repo create repo2",
        "aptly publish repo -skip-signing -component=main,contrib -architectures=i386 -distribution=squeeze repo1 repo2",
    ]
    runCmd = "aptly publish update -skip-signing squeeze"
    gold_processor = BaseTest.expand_environ


class PublishUpdate9Test(BaseTest):
    """
    publish update: conflicting files in the repo
    """
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
        "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick local-repo",
        "aptly repo remove local-repo Name",
        "aptly repo add local-repo ${testfiles}",
    ]
    runCmd = "aptly publish update -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick"
    expectedCode = 1
    gold_processor = BaseTest.expand_environ


class PublishUpdate10Test(BaseTest):
    """
    publish update: -force-overwrite
    """
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
        "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick local-repo",
        "aptly repo remove local-repo Name",
        "aptly repo add local-repo ${testfiles}",
    ]
    runCmd = "aptly publish update -force-overwrite -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishUpdate10Test, self).check()

        self.check_file_contents("public/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz", "file")


class PublishUpdate11Test(BaseTest):
    """
    publish update: -skip-contents
    """
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}/",
        "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -skip-contents local-repo",
        "aptly repo remove local-repo pyspi"
    ]
    runCmd = "aptly publish update -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -skip-contents maverick"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishUpdate11Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_not_exists('public/dists/maverick/main/Contents-i386.gz')


class PublishUpdate12Test(BaseTest):
    """
    publish update: removed some packages skipping cleanup
    """
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}/",
        "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick local-repo",
        "aptly repo remove local-repo pyspi"
    ]
    runCmd = "aptly publish update -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -skip-cleanup maverick"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishUpdate12Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/Contents-i386.gz')
        self.check_exists('public/dists/maverick/main/Contents-i386.gz')
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

        if pathsSeen != set(['main/binary-i386/Packages', 'main/binary-i386/Packages.bz2', 'main/binary-i386/Packages.gz',
                             'main/source/Sources', 'main/source/Sources.gz', 'main/source/Sources.bz2',
                             'main/binary-i386/Release', 'main/source/Release', 'main/Contents-i386.gz',
                             'Contents-i386.gz']):
            raise Exception("path seen wrong: %r" % (pathsSeen, ))


class PublishUpdate13Test(BaseTest):
    """
    publish update: -skip-bz2
    """
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}/",
        "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -skip-bz2 local-repo",
        "aptly repo remove local-repo pyspi"
    ]
    runCmd = "aptly publish update -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -skip-bz2 maverick"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishUpdate13Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_not_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
