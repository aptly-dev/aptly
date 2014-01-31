import os
import hashlib
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
    runCmd = "aptly publish snapshot snap1"
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
        self.check_file('public/dists/maverick/Release', 'release', match_prepare=strip_processor)

        # verify signatures
        self.run_cmd(["gpg", "--verify", os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/InRelease')])
        self.run_cmd(["gpg", "--verify", os.path.join(os.environ["HOME"], ".aptly", 'public/dists/maverick/Release.gpg'),
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
    runCmd = "aptly publish snapshot -distribution=squeeze snap2"
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
        self.check_file('public/dists/squeeze/Release', 'release', match_prepare=strip_processor)


class PublishSnapshot3Test(BaseTest):
    """
    publish snapshot: different distribution and component
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap3 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly publish snapshot -distribution=squeeze -component=contrib snap3"
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
        self.check_file('public/dists/squeeze/Release', 'release', match_prepare=strip_processor)


class PublishSnapshot4Test(BaseTest):
    """
    publish snapshot: limit architectures
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap4 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly -architectures=i386 publish snapshot -distribution=squeeze snap4"
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
        self.check_file('public/dists/squeeze/Release', 'release', match_prepare=strip_processor)


class PublishSnapshot5Test(BaseTest):
    """
    publish snapshot: under prefix
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap5 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly publish snapshot -distribution=squeeze snap5 ppa/smira"

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
        "aptly publish snapshot snap7",
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
        "aptly publish snapshot snap8 ./ppa",
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
