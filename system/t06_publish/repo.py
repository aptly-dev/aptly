import os
import hashlib
import inspect
from lib import BaseTest


def strip_processor(output):
    return "\n".join([l for l in output.split("\n") if not l.startswith(' ') and not l.startswith('Date:')])


class PublishRepo1Test(BaseTest):
    """
    publish repo: default
    """
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
    ]
    runCmd = "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick local-repo"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishRepo1Test, self).check()

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

        if pathsSeen != set(['main/binary-i386/Packages', 'main/binary-i386/Packages.bz2', 'main/binary-i386/Packages.gz',
                             'main/source/Sources', 'main/source/Sources.gz', 'main/source/Sources.bz2',
                             'main/binary-i386/Release', 'main/source/Release']):
            raise Exception("path seen wrong: %r" % (pathsSeen, ))


class PublishRepo2Test(BaseTest):
    """
    publish repo: different component
    """
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
    ]
    runCmd = "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=contrib local-repo"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishRepo2Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/contrib/binary-i386/Packages')
        self.check_exists('public/dists/maverick/contrib/binary-i386/Packages.gz')
        self.check_exists('public/dists/maverick/contrib/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/contrib/source/Sources')
        self.check_exists('public/dists/maverick/contrib/source/Sources.gz')
        self.check_exists('public/dists/maverick/contrib/source/Sources.bz2')

        self.check_exists('public/pool/contrib/p/pyspi/pyspi_0.6.1-1.3.dsc')
        self.check_exists('public/pool/contrib/p/pyspi/pyspi_0.6.1-1.3.diff.gz')
        self.check_exists('public/pool/contrib/p/pyspi/pyspi_0.6.1.orig.tar.gz')
        self.check_exists('public/pool/contrib/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc')
        self.check_exists('public/pool/contrib/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb')


class PublishRepo3Test(BaseTest):
    """
    publish repo: different architectures
    """
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
    ]
    runCmd = "aptly -architectures=i386 publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=contrib local-repo"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishRepo3Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/contrib/binary-i386/Packages')
        self.check_exists('public/dists/maverick/contrib/binary-i386/Packages.gz')
        self.check_exists('public/dists/maverick/contrib/binary-i386/Packages.bz2')
        self.check_not_exists('public/dists/maverick/contrib/source/Sources')
        self.check_not_exists('public/dists/maverick/contrib/source/Sources.gz')
        self.check_not_exists('public/dists/maverick/contrib/source/Sources.bz2')

        self.check_not_exists('public/pool/contrib/p/pyspi/pyspi_0.6.1-1.3.dsc')
        self.check_not_exists('public/pool/contrib/p/pyspi/pyspi_0.6.1-1.3.diff.gz')
        self.check_not_exists('public/pool/contrib/p/pyspi/pyspi_0.6.1.orig.tar.gz')
        self.check_not_exists('public/pool/contrib/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc')
        self.check_exists('public/pool/contrib/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb')


class PublishRepo4Test(BaseTest):
    """
    publish repo: under prefix
    """
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
    ]
    runCmd = "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick local-repo ppa"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishRepo4Test, self).check()

        self.check_exists('public/ppa/dists/maverick/InRelease')
        self.check_exists('public/ppa/dists/maverick/Release')
        self.check_exists('public/ppa/dists/maverick/Release.gpg')

        self.check_exists('public/ppa/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/ppa/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/ppa/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/ppa/dists/maverick/main/source/Sources')
        self.check_exists('public/ppa/dists/maverick/main/source/Sources.gz')
        self.check_exists('public/ppa/dists/maverick/main/source/Sources.bz2')

        self.check_exists('public/ppa/pool/main/p/pyspi/pyspi_0.6.1-1.3.dsc')
        self.check_exists('public/ppa/pool/main/p/pyspi/pyspi_0.6.1-1.3.diff.gz')
        self.check_exists('public/ppa/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz')
        self.check_exists('public/ppa/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc')
        self.check_exists('public/ppa/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb')


class PublishRepo5Test(BaseTest):
    """
    publish repo: specify distribution
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
    ]
    runCmd = "aptly publish repo local-repo"
    expectedCode = 1


class PublishRepo6Test(BaseTest):
    """
    publish repo: double publish under root
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
        "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick local-repo",
    ]
    runCmd = "aptly publish repo -distribution=maverick local-repo"
    expectedCode = 1


class PublishRepo7Test(BaseTest):
    """
    publish repo: double publish under prefix
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
        "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick local-repo ./ppa",
    ]
    runCmd = "aptly publish repo -distribution=maverick local-repo ppa"
    expectedCode = 1


class PublishRepo8Test(BaseTest):
    """
    publish repo: wrong prefix
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
    ]
    runCmd = "aptly publish repo -distribution=maverick local-repo ppa/dists/la"
    expectedCode = 1


class PublishRepo9Test(BaseTest):
    """
    publish repo: wrong prefix
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
    ]
    runCmd = "aptly publish repo -distribution=maverick local-repo ppa/pool/la"
    expectedCode = 1


class PublishRepo10Test(BaseTest):
    """
    publish repo: wrong prefix
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
    ]
    runCmd = "aptly publish repo -distribution=maverick local-repo ../la"
    expectedCode = 1


class PublishRepo11Test(BaseTest):
    """
    publish repo: no snapshot
    """
    runCmd = "aptly publish repo local-repo"
    expectedCode = 1


class PublishRepo12Test(BaseTest):
    """
    publish repo: -skip-signing
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
    ]
    runCmd = "aptly publish repo -skip-signing -distribution=maverick local-repo"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishRepo12Test, self).check()

        self.check_not_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_not_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/source/Sources')
        self.check_exists('public/dists/maverick/main/source/Sources.gz')
        self.check_exists('public/dists/maverick/main/source/Sources.bz2')

        # verify contents except of sums
        self.check_file_contents('public/dists/maverick/Release', 'release', match_prepare=strip_processor)


class PublishRepo13Test(BaseTest):
    """
    publish repo: empty repo is not publishable w/o architectures list
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly repo create local-repo",
    ]
    runCmd = "aptly publish repo --distribution=mars --skip-signing local-repo"
    expectedCode = 1


class PublishRepo14Test(BaseTest):
    """
    publish repo: publishing defaults from local repo
    """
    fixtureCmds = [
        "aptly repo create -distribution=maverick -component=contrib local-repo",
        "aptly repo add local-repo ${files}",
    ]
    runCmd = "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec local-repo"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishRepo14Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/contrib/binary-i386/Packages')
        self.check_exists('public/dists/maverick/contrib/binary-i386/Packages.gz')
        self.check_exists('public/dists/maverick/contrib/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/contrib/source/Sources')
        self.check_exists('public/dists/maverick/contrib/source/Sources.gz')
        self.check_exists('public/dists/maverick/contrib/source/Sources.bz2')

        self.check_exists('public/pool/contrib/p/pyspi/pyspi_0.6.1-1.3.dsc')
        self.check_exists('public/pool/contrib/p/pyspi/pyspi_0.6.1-1.3.diff.gz')
        self.check_exists('public/pool/contrib/p/pyspi/pyspi_0.6.1.orig.tar.gz')
        self.check_exists('public/pool/contrib/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc')
        self.check_exists('public/pool/contrib/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb')


class PublishRepo15Test(BaseTest):
    """
    publish repo: custom label
    """
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
    ]
    runCmd = "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=contrib -label=label15 local-repo"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishRepo15Test, self).check()

        # verify contents except of sums
        self.check_file_contents('public/dists/maverick/Release', 'release', match_prepare=strip_processor)


class PublishRepo16Test(BaseTest):
    """
    publish repo: empty repo is publishable with architectures list
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly repo create local-repo",
    ]
    runCmd = "aptly publish repo  -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -architectures=source,i386 --distribution=maverick local-repo"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishRepo16Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/source/Sources')
        self.check_exists('public/dists/maverick/main/source/Sources.gz')
        self.check_exists('public/dists/maverick/main/source/Sources.bz2')


class PublishRepo17Test(BaseTest):
    """
    publish repo: multiple component
    """
    fixtureCmds = [
        "aptly repo create repo1",
        "aptly repo create repo2",
        "aptly repo add repo1 ${files}/libboost-program-options-dev_1.49.0.1_i386.deb ${files}/pyspi_0.6.1-1.3.dsc",
        "aptly repo add repo2 ${files}/pyspi-0.6.1-1.3.stripped.dsc",
    ]
    runCmd = "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -component=main,contrib -distribution=maverick repo1 repo2"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishRepo17Test, self).check()

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
        self.check_exists('public/dists/maverick/contrib/source/Sources')
        self.check_exists('public/dists/maverick/contrib/source/Sources.gz')
        self.check_exists('public/dists/maverick/contrib/source/Sources.bz2')

        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.dsc')
        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.diff.gz')
        self.check_exists('public/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz')
        self.check_exists('public/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb')

        self.check_exists('public/pool/contrib/p/pyspi/pyspi_0.6.1-1.3.diff.gz')
        self.check_exists('public/pool/contrib/p/pyspi/pyspi_0.6.1.orig.tar.gz')
        self.check_exists('public/pool/contrib/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc')

        # verify contents except of sums
        self.check_file_contents('public/dists/maverick/Release', 'release', match_prepare=strip_processor)

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

        if pathsSeen != set(['main/binary-i386/Packages', 'main/binary-i386/Packages.gz',
                             'main/binary-i386/Packages.bz2',
                             'main/source/Sources', 'main/source/Sources.gz', 'main/source/Sources.bz2',
                             'contrib/binary-i386/Packages', 'contrib/binary-i386/Packages.gz',
                             'contrib/binary-i386/Packages.bz2',
                             'contrib/source/Sources', 'contrib/source/Sources.gz', 'contrib/source/Sources.bz2',
                             'main/source/Release', 'contrib/source/Release',
                             'main/binary-i386/Release', 'contrib/binary-i386/Release']):
            raise Exception("path seen wrong: %r" % (pathsSeen, ))


class PublishRepo18Test(BaseTest):
    """
    publish repo: multiple component, guessing component names
    """
    fixtureCmds = [
        "aptly repo create -distribution=squeeze -component=main repo1",
        "aptly repo create -distribution=squeeze -component=contrib repo2",
        "aptly repo add repo1 ${files}/libboost-program-options-dev_1.49.0.1_i386.deb ${files}/pyspi_0.6.1-1.3.dsc",
        "aptly repo add repo2 ${files}/pyspi-0.6.1-1.3.stripped.dsc",
    ]
    runCmd = "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -component=, repo1 repo2"
    gold_processor = BaseTest.expand_environ


class PublishRepo19Test(BaseTest):
    """
    publish repo: duplicate component name (guessed)
    """
    fixtureCmds = [
        "aptly repo create -distribution=squeeze -component=contrib repo1",
        "aptly repo create -distribution=squeeze -component=contrib repo2",
        "aptly repo add repo1 ${files}/libboost-program-options-dev_1.49.0.1_i386.deb ${files}/pyspi_0.6.1-1.3.dsc",
        "aptly repo add repo2 ${files}/pyspi-0.6.1-1.3.stripped.dsc",
    ]
    runCmd = "aptly publish repo -component=, repo1 repo2"
    expectedCode = 1


class PublishRepo20Test(BaseTest):
    """
    publish repo: duplicate component name (manual)
    """
    fixtureCmds = [
        "aptly repo create -distribution=squeeze -component=main repo1",
        "aptly repo create -distribution=squeeze -component=contrib repo2",
    ]
    runCmd = "aptly publish repo -component=b,b repo1 repo2"
    expectedCode = 1


class PublishRepo21Test(BaseTest):
    """
    publish repo: distribution conflict
    """
    fixtureCmds = [
        "aptly repo create -distribution=squeeze -component=main repo1",
        "aptly repo create -distribution=wheezy -component=contrib repo2",
    ]
    runCmd = "aptly publish repo -component=, repo1 repo2"
    expectedCode = 1


class PublishRepo22Test(BaseTest):
    """
    publish reop: no such repo
    """
    fixtureCmds = [
        "aptly repo create -distribution=squeeze -component=main repo1",
    ]
    runCmd = "aptly publish repo -component=, repo1 repo2"
    expectedCode = 1


class PublishRepo23Test(BaseTest):
    """
    publish repo: mismatch in count
    """
    fixtureCmds = [
        "aptly repo create -distribution=squeeze -component=main repo1",
    ]
    runCmd = "aptly publish repo -component=main,contrib repo1"
    expectedCode = 2
    outputMatchPrepare = lambda _, s: "\n".join([l for l in s.split("\n") if l.startswith("ERROR")])


class PublishRepo24Test(BaseTest):
    """
    publish repo: conflicting files in the repo
    """
    fixtureCmds = [
        "aptly repo create local-repo1",
        "aptly repo add local-repo1 ${files}",
        "aptly repo create local-repo2",
        "aptly repo add local-repo2 ${testfiles}",
        "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick local-repo1",
    ]
    runCmd = "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=squeeze local-repo2"
    expectedCode = 1
    gold_processor = BaseTest.expand_environ


class PublishRepo25Test(BaseTest):
    """
    publish repo: -force-overwrite
    """
    fixtureCmds = [
        "aptly repo create local-repo1",
        "aptly repo add local-repo1 ${files}",
        "aptly repo create local-repo2",
        "aptly repo add local-repo2 ${testfiles}",
        "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick local-repo1",
    ]
    runCmd = "aptly publish repo -force-overwrite -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=squeeze local-repo2"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishRepo25Test, self).check()

        self.check_file_contents("public/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz", "file")
