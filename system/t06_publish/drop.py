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
