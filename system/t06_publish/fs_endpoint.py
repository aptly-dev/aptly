from fs_endpoint_lib import FileSystemEndpointTest


class FSEndpointPublishSnapshot1Test(FileSystemEndpointTest):
    """
    publish snapshot: using symlinks
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:symlink:"
    gold_processor = FileSystemEndpointTest.expand_environ

    def check(self):
        super(FSEndpointPublishSnapshot1Test, self).check()

        self.check_is_regular('public_symlink/dists/maverick/InRelease')
        self.check_is_regular('public_symlink/dists/maverick/Release')
        self.check_is_regular('public_symlink/dists/maverick/Release.gpg')

        self.check_is_regular('public_symlink/dists/maverick/main/binary-i386/Release')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-i386/Packages')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-i386/Packages.gz')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_is_regular('public_symlink/dists/maverick/main/Contents-i386.gz')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-amd64/Release')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-amd64/Packages')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_is_regular('public_symlink/dists/maverick/main/Contents-amd64.gz')
        self.check_not_exists('public_symlink/dists/maverick/main/debian-installer/binary-i386/Packages')
        self.check_not_exists('public_symlink/dists/maverick/main/debian-installer/binary-amd64/Packages')

        self.check_is_symlink('public_symlink/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')


class FSEndpointPublishSnapshot2Test(FileSystemEndpointTest):
    """
    publish snapshot: using hardlinks
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:hardlink:"
    gold_processor = FileSystemEndpointTest.expand_environ

    def check(self):
        super(FSEndpointPublishSnapshot2Test, self).check()

        self.check_is_regular('public_hardlink/dists/maverick/InRelease')
        self.check_is_regular('public_hardlink/dists/maverick/Release')
        self.check_is_regular('public_hardlink/dists/maverick/Release.gpg')

        self.check_is_regular('public_hardlink/dists/maverick/main/binary-i386/Release')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-i386/Packages')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-i386/Packages.gz')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_is_regular('public_hardlink/dists/maverick/main/Contents-i386.gz')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-amd64/Release')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-amd64/Packages')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_is_regular('public_hardlink/dists/maverick/main/Contents-amd64.gz')
        self.check_not_exists('public_hardlink/dists/maverick/main/debian-installer/binary-i386/Packages')
        self.check_not_exists('public_hardlink/dists/maverick/main/debian-installer/binary-amd64/Packages')

        self.check_is_hardlink('public_hardlink/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')


class FSEndpointPublishSnapshot3Test(FileSystemEndpointTest):
    """
    publish snapshot: using copy
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:copy:"
    gold_processor = FileSystemEndpointTest.expand_environ

    def check(self):
        super(FSEndpointPublishSnapshot3Test, self).check()

        self.check_is_regular('public_copy/dists/maverick/InRelease')
        self.check_is_regular('public_copy/dists/maverick/Release')
        self.check_is_regular('public_copy/dists/maverick/Release.gpg')

        self.check_is_regular('public_copy/dists/maverick/main/binary-i386/Release')
        self.check_is_regular('public_copy/dists/maverick/main/binary-i386/Packages')
        self.check_is_regular('public_copy/dists/maverick/main/binary-i386/Packages.gz')
        self.check_is_regular('public_copy/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_is_regular('public_copy/dists/maverick/main/Contents-i386.gz')
        self.check_is_regular('public_copy/dists/maverick/main/binary-amd64/Release')
        self.check_is_regular('public_copy/dists/maverick/main/binary-amd64/Packages')
        self.check_is_regular('public_copy/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_is_regular('public_copy/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_is_regular('public_copy/dists/maverick/main/Contents-amd64.gz')
        self.check_not_exists('public_copy/dists/maverick/main/debian-installer/binary-i386/Packages')
        self.check_not_exists('public_copy/dists/maverick/main/debian-installer/binary-amd64/Packages')

        self.check_is_copy('public_copy/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')


class FSEndpointPublishSnapshot4Test(FileSystemEndpointTest):
    """
    publish snapshot: using copy, symlink and hardlink variants
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:copy:",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:symlink:",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:hardlink:"
    gold_processor = FileSystemEndpointTest.expand_environ

    def check(self):
        super(FSEndpointPublishSnapshot4Test, self).check()

        self.check_is_regular('public_copy/dists/maverick/InRelease')
        self.check_is_regular('public_copy/dists/maverick/Release')
        self.check_is_regular('public_copy/dists/maverick/Release.gpg')

        self.check_is_regular('public_copy/dists/maverick/main/binary-i386/Release')
        self.check_is_regular('public_copy/dists/maverick/main/binary-i386/Packages')
        self.check_is_regular('public_copy/dists/maverick/main/binary-i386/Packages.gz')
        self.check_is_regular('public_copy/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_is_regular('public_copy/dists/maverick/main/Contents-i386.gz')
        self.check_is_regular('public_copy/dists/maverick/main/binary-amd64/Release')
        self.check_is_regular('public_copy/dists/maverick/main/binary-amd64/Packages')
        self.check_is_regular('public_copy/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_is_regular('public_copy/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_is_regular('public_copy/dists/maverick/main/Contents-amd64.gz')
        self.check_not_exists('public_copy/dists/maverick/main/debian-installer/binary-i386/Packages')
        self.check_not_exists('public_copy/dists/maverick/main/debian-installer/binary-amd64/Packages')

        self.check_is_copy('public_copy/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')

        self.check_is_regular('public_symlink/dists/maverick/InRelease')
        self.check_is_regular('public_symlink/dists/maverick/Release')
        self.check_is_regular('public_symlink/dists/maverick/Release.gpg')

        self.check_is_regular('public_symlink/dists/maverick/main/binary-i386/Release')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-i386/Packages')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-i386/Packages.gz')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_is_regular('public_symlink/dists/maverick/main/Contents-i386.gz')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-amd64/Release')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-amd64/Packages')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_is_regular('public_symlink/dists/maverick/main/Contents-amd64.gz')
        self.check_not_exists('public_symlink/dists/maverick/main/debian-installer/binary-i386/Packages')
        self.check_not_exists('public_symlink/dists/maverick/main/debian-installer/binary-amd64/Packages')

        self.check_is_symlink('public_symlink/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')

        self.check_is_regular('public_hardlink/dists/maverick/InRelease')
        self.check_is_regular('public_hardlink/dists/maverick/Release')
        self.check_is_regular('public_hardlink/dists/maverick/Release.gpg')

        self.check_is_regular('public_hardlink/dists/maverick/main/binary-i386/Release')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-i386/Packages')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-i386/Packages.gz')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_is_regular('public_hardlink/dists/maverick/main/Contents-i386.gz')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-amd64/Release')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-amd64/Packages')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_is_regular('public_hardlink/dists/maverick/main/Contents-amd64.gz')
        self.check_not_exists('public_hardlink/dists/maverick/main/debian-installer/binary-i386/Packages')
        self.check_not_exists('public_hardlink/dists/maverick/main/debian-installer/binary-amd64/Packages')

        self.check_is_hardlink('public_hardlink/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')


class FSEndpointPublishSnapshot5Test(FileSystemEndpointTest):
    """
    publish snapshot: using copy, symlink and hardlink variants under prefixes
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:copy:snap_copy/daily",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:symlink:snap_symlink/daily",
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:hardlink:snap_hardlink/daily"
    gold_processor = FileSystemEndpointTest.expand_environ

    def check(self):
        super(FSEndpointPublishSnapshot5Test, self).check()

        self.check_is_regular('public_copy/snap_copy/daily/dists/maverick/InRelease')
        self.check_is_regular('public_copy/snap_copy/daily/dists/maverick/Release')
        self.check_is_regular('public_copy/snap_copy/daily/dists/maverick/Release.gpg')

        self.check_is_regular('public_copy/snap_copy/daily/dists/maverick/main/binary-i386/Release')
        self.check_is_regular('public_copy/snap_copy/daily/dists/maverick/main/binary-i386/Packages')
        self.check_is_regular('public_copy/snap_copy/daily/dists/maverick/main/binary-i386/Packages.gz')
        self.check_is_regular('public_copy/snap_copy/daily/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_is_regular('public_copy/snap_copy/daily/dists/maverick/main/Contents-i386.gz')
        self.check_is_regular('public_copy/snap_copy/daily/dists/maverick/main/binary-amd64/Release')
        self.check_is_regular('public_copy/snap_copy/daily/dists/maverick/main/binary-amd64/Packages')
        self.check_is_regular('public_copy/snap_copy/daily/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_is_regular('public_copy/snap_copy/daily/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_is_regular('public_copy/snap_copy/daily/dists/maverick/main/Contents-amd64.gz')
        self.check_not_exists('public_copy/snap_copy/daily/dists/maverick/main/debian-installer/binary-i386/Packages')
        self.check_not_exists('public_copy/snap_copy/daily/dists/maverick/main/debian-installer/binary-amd64/Packages')

        self.check_is_copy('public_copy/snap_copy/daily/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')

        self.check_is_regular('public_symlink/snap_symlink/daily/dists/maverick/InRelease')
        self.check_is_regular('public_symlink/snap_symlink/daily/dists/maverick/Release')
        self.check_is_regular('public_symlink/snap_symlink/daily/dists/maverick/Release.gpg')

        self.check_is_regular('public_symlink/snap_symlink/daily/dists/maverick/main/binary-i386/Release')
        self.check_is_regular('public_symlink/snap_symlink/daily/dists/maverick/main/binary-i386/Packages')
        self.check_is_regular('public_symlink/snap_symlink/daily/dists/maverick/main/binary-i386/Packages.gz')
        self.check_is_regular('public_symlink/snap_symlink/daily/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_is_regular('public_symlink/snap_symlink/daily/dists/maverick/main/Contents-i386.gz')
        self.check_is_regular('public_symlink/snap_symlink/daily/dists/maverick/main/binary-amd64/Release')
        self.check_is_regular('public_symlink/snap_symlink/daily/dists/maverick/main/binary-amd64/Packages')
        self.check_is_regular('public_symlink/snap_symlink/daily/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_is_regular('public_symlink/snap_symlink/daily/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_is_regular('public_symlink/snap_symlink/daily/dists/maverick/main/Contents-amd64.gz')
        self.check_not_exists('public_symlink/snap_symlink/daily/dists/maverick/main/debian-installer/binary-i386/Packages')
        self.check_not_exists('public_symlink/snap_symlink/daily/dists/maverick/main/debian-installer/binary-amd64/Packages')

        self.check_is_symlink('public_symlink/snap_symlink/daily/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')

        self.check_is_regular('public_hardlink/snap_hardlink/daily/dists/maverick/InRelease')
        self.check_is_regular('public_hardlink/snap_hardlink/daily/dists/maverick/Release')
        self.check_is_regular('public_hardlink/snap_hardlink/daily/dists/maverick/Release.gpg')

        self.check_is_regular('public_hardlink/snap_hardlink/daily/dists/maverick/main/binary-i386/Release')
        self.check_is_regular('public_hardlink/snap_hardlink/daily/dists/maverick/main/binary-i386/Packages')
        self.check_is_regular('public_hardlink/snap_hardlink/daily/dists/maverick/main/binary-i386/Packages.gz')
        self.check_is_regular('public_hardlink/snap_hardlink/daily/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_is_regular('public_hardlink/snap_hardlink/daily/dists/maverick/main/Contents-i386.gz')
        self.check_is_regular('public_hardlink/snap_hardlink/daily/dists/maverick/main/binary-amd64/Release')
        self.check_is_regular('public_hardlink/snap_hardlink/daily/dists/maverick/main/binary-amd64/Packages')
        self.check_is_regular('public_hardlink/snap_hardlink/daily/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_is_regular('public_hardlink/snap_hardlink/daily/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_is_regular('public_hardlink/snap_hardlink/daily/dists/maverick/main/Contents-amd64.gz')
        self.check_not_exists('public_hardlink/snap_hardlink/daily/dists/maverick/main/debian-installer/binary-i386/Packages')
        self.check_not_exists('public_hardlink/snap_hardlink/daily/dists/maverick/main/debian-installer/binary-amd64/Packages')

        self.check_is_hardlink('public_hardlink/snap_hardlink/daily/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')


class FSEndpointPublishSnapshot6Test(FileSystemEndpointTest):
    """
    publish snapshot: drop one
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:copy:",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:symlink:",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:hardlink:"
    ]
    runCmd = "aptly publish drop maverick filesystem:copy:"
    gold_processor = FileSystemEndpointTest.expand_environ

    def check(self):
        super(FSEndpointPublishSnapshot6Test, self).check()

        self.check_not_exists('public_copy/dists/')
        self.check_not_exists('public_copy/pool/')

        self.check_is_regular('public_symlink/dists/maverick/InRelease')
        self.check_is_regular('public_symlink/dists/maverick/Release')
        self.check_is_regular('public_symlink/dists/maverick/Release.gpg')

        self.check_is_regular('public_symlink/dists/maverick/main/binary-i386/Release')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-i386/Packages')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-i386/Packages.gz')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_is_regular('public_symlink/dists/maverick/main/Contents-i386.gz')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-amd64/Release')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-amd64/Packages')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_is_regular('public_symlink/dists/maverick/main/Contents-amd64.gz')
        self.check_not_exists('public_symlink/dists/maverick/main/debian-installer/binary-i386/Packages')
        self.check_not_exists('public_symlink/dists/maverick/main/debian-installer/binary-amd64/Packages')

        self.check_is_symlink('public_symlink/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')

        self.check_is_regular('public_hardlink/dists/maverick/InRelease')
        self.check_is_regular('public_hardlink/dists/maverick/Release')
        self.check_is_regular('public_hardlink/dists/maverick/Release.gpg')

        self.check_is_regular('public_hardlink/dists/maverick/main/binary-i386/Release')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-i386/Packages')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-i386/Packages.gz')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_is_regular('public_hardlink/dists/maverick/main/Contents-i386.gz')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-amd64/Release')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-amd64/Packages')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_is_regular('public_hardlink/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_is_regular('public_hardlink/dists/maverick/main/Contents-amd64.gz')
        self.check_not_exists('public_hardlink/dists/maverick/main/debian-installer/binary-i386/Packages')
        self.check_not_exists('public_hardlink/dists/maverick/main/debian-installer/binary-amd64/Packages')

        self.check_is_hardlink('public_hardlink/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')


class FSEndpointPublishSnapshot7Test(FileSystemEndpointTest):
    """
    publish snapshot: drop two
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:copy:",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:symlink:",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:hardlink:",
        "aptly publish drop maverick filesystem:copy:"
    ]
    runCmd = "aptly publish drop maverick filesystem:hardlink:"
    gold_processor = FileSystemEndpointTest.expand_environ

    def check(self):
        super(FSEndpointPublishSnapshot7Test, self).check()

        self.check_not_exists('public_copy/dists/')
        self.check_not_exists('public_copy/pool/')

        self.check_is_regular('public_symlink/dists/maverick/InRelease')
        self.check_is_regular('public_symlink/dists/maverick/Release')
        self.check_is_regular('public_symlink/dists/maverick/Release.gpg')

        self.check_is_regular('public_symlink/dists/maverick/main/binary-i386/Release')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-i386/Packages')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-i386/Packages.gz')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_is_regular('public_symlink/dists/maverick/main/Contents-i386.gz')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-amd64/Release')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-amd64/Packages')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_is_regular('public_symlink/dists/maverick/main/binary-amd64/Packages.bz2')
        self.check_is_regular('public_symlink/dists/maverick/main/Contents-amd64.gz')
        self.check_not_exists('public_symlink/dists/maverick/main/debian-installer/binary-i386/Packages')
        self.check_not_exists('public_symlink/dists/maverick/main/debian-installer/binary-amd64/Packages')

        self.check_is_symlink('public_symlink/pool/main/g/gnuplot/gnuplot-doc_4.6.1-1~maverick2_all.deb')

        self.check_not_exists('public_hardlink/dists/')
        self.check_not_exists('public_hardlink/pool/')


class FSEndpointPublishSnapshot8Test(FileSystemEndpointTest):
    """
    publish snapshot: remove snapshot
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:copy:",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:symlink:",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:hardlink:",
        "aptly publish drop maverick filesystem:copy:",
        "aptly publish drop maverick filesystem:symlink:",
        "aptly publish drop maverick filesystem:hardlink:",
    ]
    runCmd = "aptly snapshot drop snap1"
    gold_processor = FileSystemEndpointTest.expand_environ


class FSEndpointPublishSnapshot9Test(FileSystemEndpointTest):
    """
    publish snapshot: remove snapshot error
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:copy:",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:symlink:",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:hardlink:",
        "aptly publish drop maverick filesystem:copy:",
    ]
    runCmd = "aptly snapshot drop snap1"
    expectedCode = 1


class FSEndpointPublishSnapshot10Test(FileSystemEndpointTest):
    """
    publish list: several repos list
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:copy:",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:symlink:snap_symlink/daily",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 filesystem:hardlink:"
    ]
    runCmd = "aptly publish list -raw"


class FSEndpointPublishSnapshot11Test(FileSystemEndpointTest):
    """
    publish snapshot: conflicting files in the snapshot using symlink method
    """
    fixtureCmds = [
        "aptly repo create local-repo1",
        "aptly repo add local-repo1 ${files}",
        "aptly snapshot create snap1 from repo local-repo1",
        "aptly repo create local-repo2",
        "aptly repo add local-repo2 ${testfiles}",
        "aptly snapshot create snap2 from repo local-repo2",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick snap1 filesystem:symlink:"
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=squeeze snap2 filesystem:symlink:"
    expectedCode = 1
    gold_processor = FileSystemEndpointTest.expand_environ


class FSEndpointPublishSnapshot12Test(FileSystemEndpointTest):
    """
    publish snapshot: conflicting files in the snapshot using symlink method. -force-overwrite
    """
    fixtureCmds = [
        "aptly repo create local-repo1",
        "aptly repo add local-repo1 ${files}",
        "aptly snapshot create snap1 from repo local-repo1",
        "aptly repo create local-repo2",
        "aptly repo add local-repo2 ${testfiles}",
        "aptly snapshot create snap2 from repo local-repo2",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick snap1 filesystem:symlink:"
    ]
    runCmd = "aptly publish snapshot -force-overwrite -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=squeeze snap2 filesystem:symlink:"
    gold_processor = FileSystemEndpointTest.expand_environ

    def check(self):
        super(FSEndpointPublishSnapshot12Test, self).check()

        self.check_file_contents("public_symlink/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz", "file")


class FSEndpointPublishSnapshot13Test(FileSystemEndpointTest):
    """
    publish snapshot: conflicting files in the snapshot using copy method with md5 verification
    """
    fixtureCmds = [
        "aptly repo create local-repo1",
        "aptly repo add local-repo1 ${files}",
        "aptly snapshot create snap1 from repo local-repo1",
        "aptly repo create local-repo2",
        "aptly repo add local-repo2 ${testfiles}",
        "aptly snapshot create snap2 from repo local-repo2",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick snap1 filesystem:copy:"
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=squeeze snap2 filesystem:copy:"
    expectedCode = 1
    gold_processor = FileSystemEndpointTest.expand_environ


class FSEndpointPublishSnapshot14Test(FileSystemEndpointTest):
    """
    publish snapshot: conflicting files in the snapshot using copy method with md5 verification. -force-overwrite
    """
    fixtureCmds = [
        "aptly repo create local-repo1",
        "aptly repo add local-repo1 ${files}",
        "aptly snapshot create snap1 from repo local-repo1",
        "aptly repo create local-repo2",
        "aptly repo add local-repo2 ${testfiles}",
        "aptly snapshot create snap2 from repo local-repo2",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick snap1 filesystem:copy:"
    ]
    runCmd = "aptly publish snapshot -force-overwrite -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=squeeze snap2 filesystem:copy:"
    gold_processor = FileSystemEndpointTest.expand_environ

    def check(self):
        super(FSEndpointPublishSnapshot14Test, self).check()

        self.check_file_contents("public_copy/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz", "file")


class FSEndpointPublishSnapshot15Test(FileSystemEndpointTest):
    """
    publish snapshot: conflicting files in the snapshot using copy method with size verification
    """
    fixtureCmds = [
        "aptly repo create local-repo1",
        "aptly repo add local-repo1 ${files}",
        "aptly snapshot create snap1 from repo local-repo1",
        "aptly repo create local-repo2",
        "aptly repo add local-repo2 ${testfiles}",
        "aptly snapshot create snap2 from repo local-repo2",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick snap1 filesystem:copysize:"
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=squeeze snap2 filesystem:copysize:"
    expectedCode = 1
    gold_processor = FileSystemEndpointTest.expand_environ


class FSEndpointPublishSnapshot16Test(FileSystemEndpointTest):
    """
    publish snapshot: conflicting files in the snapshot using copy method with size verification. -force-overwrite
    """
    fixtureCmds = [
        "aptly repo create local-repo1",
        "aptly repo add local-repo1 ${files}",
        "aptly snapshot create snap1 from repo local-repo1",
        "aptly repo create local-repo2",
        "aptly repo add local-repo2 ${testfiles}",
        "aptly snapshot create snap2 from repo local-repo2",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick snap1 filesystem:copysize:"
    ]
    runCmd = "aptly publish snapshot -force-overwrite -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=squeeze snap2 filesystem:copysize:"
    gold_processor = FileSystemEndpointTest.expand_environ

    def check(self):
        super(FSEndpointPublishSnapshot16Test, self).check()

        self.check_file_contents("public_copysize/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz", "file")


class FSEndpointPublishSnapshot17Test(FileSystemEndpointTest):
    """
    publish snapshot: conflicting files in the snapshot using copy method with md5 verification
    """
    fixtureCmds = [
        "aptly repo create local-repo1",
        "aptly repo add local-repo1 ${testfiles}/1",
        "aptly snapshot create snap1 from repo local-repo1",
        "aptly repo create local-repo2",
        "aptly repo add local-repo2 ${testfiles}/2",
        "aptly snapshot create snap2 from repo local-repo2",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick snap1 filesystem:copy:"
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=squeeze snap2 filesystem:copy:"
    expectedCode = 1
    gold_processor = FileSystemEndpointTest.expand_environ


class FSEndpointPublishSnapshot18Test(FileSystemEndpointTest):
    """
    publish snapshot: conflicting files in the snapshot using copy method with size verification (not detected!)
    """
    fixtureCmds = [
        "aptly repo create local-repo1",
        "aptly repo add local-repo1 ${testfiles}/1",
        "aptly snapshot create snap1 from repo local-repo1",
        "aptly repo create local-repo2",
        "aptly repo add local-repo2 ${testfiles}/2",
        "aptly snapshot create snap2 from repo local-repo2",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick snap1 filesystem:copysize:"
    ]
    runCmd = "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=squeeze snap2 filesystem:copysize:"
    gold_processor = FileSystemEndpointTest.expand_environ
