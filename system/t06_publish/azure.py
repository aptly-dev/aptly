from azure_lib import AzureTest


def strip_processor(output):
    return '\n'.join(
        [
            l
            for l in output.split('\n')
            if not l.startswith(' ') and not l.startswith('Date:')
        ]
    )


class AzurePublish1Test(AzureTest):
    """
    publish to Azure: from repo
    """

    fixtureCmds = [
        'aptly repo create -distribution=maverick local-repo',
        'aptly repo add local-repo ${files}',
        'aptly repo remove local-repo libboost-program-options-dev_1.62.0.1_i386',
    ]
    runCmd = 'aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec local-repo azure:test1:'

    def check(self):
        super(AzurePublish1Test, self).check()

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

        # # verify contents except of sums
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


class AzurePublish2Test(AzureTest):
    """
    publish to Azure: publish update removed some packages
    """

    fixtureCmds = [
        'aptly repo create -distribution=maverick local-repo',
        'aptly repo add local-repo ${files}/',
        'aptly repo remove local-repo libboost-program-options-dev_1.62.0.1_i386',
        'aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec local-repo azure:test1:',
        'aptly repo remove local-repo pyspi',
    ]
    runCmd = 'aptly publish update -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick azure:test1:'

    def check(self):
        super(AzurePublish2Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/source/Sources')
        self.check_exists('public/dists/maverick/main/source/Sources.gz')
        self.check_exists('public/dists/maverick/main/source/Sources.bz2')

        self.check_not_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.dsc')
        self.check_not_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.diff.gz')
        self.check_not_exists('public/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz')
        self.check_not_exists('public/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc')
        self.check_exists(
            'public/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb'
        )

        # verify contents except of sums
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


class AzurePublish3Test(AzureTest):
    """
    publish to Azure: publish switch - removed some packages
    """

    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        'aptly snapshot create snap1 from mirror gnuplot-maverick',
        'aptly snapshot create snap2 empty',
        'aptly snapshot pull -no-deps -architectures=i386,amd64 snap2 snap1 snap3 gnuplot-x11',
        'aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick snap1 azure:test1:',
    ]
    runCmd = 'aptly publish switch -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick azure:test1: snap3'

    def check(self):
        super(AzurePublish3Test, self).check()

        self.check_exists('public/dists/maverick/InRelease')
        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/Release.gpg')

        self.check_exists('public/dists/maverick/main/binary-i386/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-i386/Packages.bz2')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.gz')
        self.check_exists('public/dists/maverick/main/binary-amd64/Packages.bz2')

        self.check_exists(
            'public/pool/main/g/gnuplot/gnuplot-x11_4.6.1-1~maverick2_i386.deb'
        )
        self.check_exists(
            'public/pool/main/g/gnuplot/gnuplot-x11_4.6.1-1~maverick2_amd64.deb'
        )
        self.check_not_exists(
            'public/pool/main/g/gnuplot/gnuplot-nox_4.6.1-1~maverick2_i386.deb'
        )
        self.check_not_exists(
            'public/pool/main/g/gnuplot/gnuplot-nox_4.6.1-1~maverick2_amd64.deb'
        )

        # verify contents except of sums
        self.check_file_contents(
            'public/dists/maverick/Release', 'release', match_prepare=strip_processor
        )
        self.check_file_contents(
            'public/dists/maverick/main/binary-i386/Packages',
            'binary',
            match_prepare=lambda s: '\n'.join(sorted(s.split('\n'))),
        )


class AzurePublish4Test(AzureTest):
    """
    publish to Azure: multiple repos, list
    """

    fixtureCmds = [
        'aptly repo create -distribution=maverick local-repo',
        'aptly repo add local-repo ${udebs}',
        'aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec local-repo azure:test1:',
        'aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=xyz local-repo azure:test1:',
        'aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec local-repo azure:test1:prefix',
    ]
    runCmd = 'aptly publish list'


class AzurePublish5Test(AzureTest):
    """
    publish to Azure: publish drop - component cleanup
    """

    fixtureCmds = [
        'aptly repo create local1',
        'aptly repo create local2',
        'aptly repo add local1 ${files}/libboost-program-options-dev_1.49.0.1_i386.deb',
        'aptly repo add local2 ${files}',
        'aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=sq1 local1 azure:test1:',
        'aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=sq2 local2 azure:test1:',
    ]
    runCmd = 'aptly publish drop sq2 azure:test1:'

    def check(self):
        super(AzurePublish5Test, self).check()

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
