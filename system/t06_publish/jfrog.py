from jfrog_lib import JFrogTest


def strip_processor(output):
    return '\n'.join(
        [
            l
            for l in output.split('\n')
            if not l.startswith(' ') and not l.startswith('Date:')
        ]
    )


class JFrogPublish1Test(JFrogTest):
    """
    publish to JFrog: from repo
    """

    fixtureCmds = [
        'aptly repo create -distribution=maverick local-repo',
        'aptly repo add local-repo ${files}',
        'aptly repo remove local-repo libboost-program-options-dev_1.62.0.1_i386',
    ]
    runCmd = 'aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec local-repo jfrog:test1:'

    def check(self):
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
        self.check_exists(
            'public/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb'
        )

        # verify contents except sums/date chunks
        self.check_file_contents(
            'public/dists/maverick/Release', 'release', match_prepare=strip_processor
        )
        self.check_file_contents(
            'public/dists/maverick/main/source/Sources',
            'sources',
            match_prepare=lambda s: '\n'.join(sorted(s.split('\n'))),
        )
        self.check_file_contents(
            'public/dists/maverick/main/binary-i386/Packages',
            'binary',
            match_prepare=lambda s: '\n'.join(sorted(s.split('\n'))),
        )


class JFrogPublish2Test(JFrogTest):
    """
    publish to JFrog: update after removing package from repo
    """

    fixtureCmds = [
        'aptly repo create -distribution=maverick local-repo',
        'aptly repo add local-repo ${files}/',
        'aptly repo remove local-repo libboost-program-options-dev_1.62.0.1_i386',
        'aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec local-repo jfrog:test1:',
        'aptly repo remove local-repo pyspi',
    ]
    runCmd = 'aptly publish update -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick jfrog:test1:'

    def check(self):
        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_not_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.dsc')
        self.check_not_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.diff.gz')
        self.check_not_exists('public/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz')
        self.check_not_exists('public/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc')
        self.check_exists(
            'public/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb'
        )


class JFrogPublish3Test(JFrogTest):
    """
    publish to JFrog: publish drop performs cleanup
    """

    fixtureCmds = [
        'aptly repo create local1',
        'aptly repo create local2',
        'aptly repo add local1 ${files}/libboost-program-options-dev_1.49.0.1_i386.deb',
        'aptly repo add local2 ${files}',
        'aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=sq1 local1 jfrog:test1:',
        'aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=sq2 local2 jfrog:test1:',
    ]
    runCmd = 'aptly publish drop sq2 jfrog:test1:'

    def check(self):
        self.check_exists('public/dists/sq1')
        self.check_not_exists('public/dists/sq2')
        self.check_exists('public/pool/main/')

        self.check_not_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.dsc')
        self.check_not_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.diff.gz')
        self.check_not_exists('public/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz')
        self.check_not_exists('public/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc')
        self.check_exists(
            'public/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb'
        )
