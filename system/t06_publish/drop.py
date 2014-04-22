from lib import BaseTest


class PublishDrop1Test(BaseTest):
    """
    publish drop: existing snapshot
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1",
    ]
    runCmd = "aptly publish drop maverick"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishDrop1Test, self).check()

        self.check_not_exists('public/dists/')
        self.check_not_exists('public/pool/')


class PublishDrop2Test(BaseTest):
    """
    publish drop: under prefix
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 ppa/smira",
    ]
    runCmd = "aptly publish drop maverick ppa/smira"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishDrop2Test, self).check()

        self.check_not_exists('public/ppa/smira/dists/')
        self.check_not_exists('public/ppa/smira/pool/')
        self.check_exists('public/ppa/smira/')


class PublishDrop3Test(BaseTest):
    """
    publish drop: drop one distribution
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=sq1 snap1",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=sq2 snap1",
    ]
    runCmd = "aptly publish drop sq1"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishDrop3Test, self).check()

        self.check_not_exists('public/dists/sq1')
        self.check_exists('public/dists/sq2')
        self.check_exists('public/pool/main/')


class PublishDrop4Test(BaseTest):
    """
    publish drop: drop one of components
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=sq1 -component=contrib snap1",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=sq2 snap1",
    ]
    runCmd = "aptly publish drop sq1"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishDrop4Test, self).check()

        self.check_not_exists('public/dists/sq1')
        self.check_exists('public/dists/sq2')
        self.check_not_exists('public/pool/contrib/')
        self.check_exists('public/pool/main/')


class PublishDrop5Test(BaseTest):
    """
    publish drop: component cleanup
    """
    fixtureCmds = [
        "aptly repo create local1",
        "aptly repo create local2",
        "aptly repo add local1 ${files}/libboost-program-options-dev_1.49.0.1_i386.deb",
        "aptly repo add local2 ${files}",
        "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=sq1 local1",
        "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=sq2 local2",
    ]
    runCmd = "aptly publish drop sq2"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishDrop5Test, self).check()

        self.check_exists('public/dists/sq1')
        self.check_not_exists('public/dists/sq2')
        self.check_exists('public/pool/main/')

        self.check_not_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.dsc')
        self.check_not_exists('public/pool/main/p/pyspi/pyspi_0.6.1-1.3.diff.gz')
        self.check_not_exists('public/pool/main/p/pyspi/pyspi_0.6.1.orig.tar.gz')
        self.check_not_exists('public/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc')
        self.check_exists('public/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb')


class PublishDrop6Test(BaseTest):
    """
    publish drop: no publish
    """
    runCmd = "aptly publish drop sq1"
    expectedCode = 1
