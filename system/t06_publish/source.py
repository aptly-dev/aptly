from lib import BaseTest


class PublishSourceAdd1Test(BaseTest):
    """
    publish source add: add single source
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-contrib",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=wheezy -component=main snap1",
        "aptly publish source add -component=contrib wheezy snap2"
    ]
    runCmd = "aptly publish update -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec"

    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSourceAdd1Test, self).check()
        self.check_exists('public/dists/wheezy/contrib/binary-i386/Packages')
        self.check_exists('public/dists/wheezy/contrib/binary-i386/Packages.gz')
        self.check_exists('public/dists/wheezy/contrib/binary-i386/Packages.bz2')
        self.check_exists('public/dists/wheezy/contrib/Contents-i386.gz')
        self.check_exists('public/dists/wheezy/contrib/binary-amd64/Packages')
        self.check_exists('public/dists/wheezy/contrib/binary-amd64/Packages.gz')
        self.check_exists('public/dists/wheezy/contrib/binary-amd64/Packages.bz2')
        self.check_exists('public/dists/wheezy/contrib/Contents-amd64.gz')

        release = self.read_file('public/dists/wheezy/Release').split('\n')
        components = next((e.split(': ')[1] for e in release if e.startswith('Components')), None)
        components = sorted(components.split(' '))
        if ['contrib', 'main'] != components:
            raise Exception("value of 'Components' in release file is '%s' and does not match '%s'." % (' '.join(components), 'contrib main'))


class PublishSourceAdd2Test(BaseTest):
    """
    publish source add: add multiple sources
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-contrib",
        "aptly snapshot create snap3 from mirror wheezy-non-free",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=wheezy -component=main snap1",
        "aptly publish source add -component=contrib,non-free wheezy snap2 snap3"
    ]
    runCmd = "aptly publish update -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec"

    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishSourceAdd2Test, self).check()
        self.check_exists('public/dists/wheezy/contrib/binary-i386/Packages')
        self.check_exists('public/dists/wheezy/contrib/binary-i386/Packages.gz')
        self.check_exists('public/dists/wheezy/contrib/binary-i386/Packages.bz2')
        self.check_exists('public/dists/wheezy/contrib/Contents-i386.gz')
        self.check_exists('public/dists/wheezy/contrib/binary-amd64/Packages')
        self.check_exists('public/dists/wheezy/contrib/binary-amd64/Packages.gz')
        self.check_exists('public/dists/wheezy/contrib/binary-amd64/Packages.bz2')
        self.check_exists('public/dists/wheezy/contrib/Contents-amd64.gz')

        self.check_exists('public/dists/wheezy/non-free/binary-i386/Packages')
        self.check_exists('public/dists/wheezy/non-free/binary-i386/Packages.gz')
        self.check_exists('public/dists/wheezy/non-free/binary-i386/Packages.bz2')
        self.check_exists('public/dists/wheezy/non-free/Contents-i386.gz')
        self.check_exists('public/dists/wheezy/non-free/binary-amd64/Packages')
        self.check_exists('public/dists/wheezy/non-free/binary-amd64/Packages.gz')
        self.check_exists('public/dists/wheezy/non-free/binary-amd64/Packages.bz2')
        self.check_exists('public/dists/wheezy/non-free/Contents-amd64.gz')

        release = self.read_file('public/dists/wheezy/Release').split('\n')
        components = next((e.split(': ')[1] for e in release if e.startswith('Components')), None)
        components = sorted(components.split(' '))
        if ['contrib', 'main', 'non-free'] != components:
            raise Exception("value of 'Components' in release file is '%s' and does not match '%s'." % (' '.join(components), 'contrib main non-free'))


class PublishSourceAdd3Test(BaseTest):
    """
    publish source add: (re-)add already added source
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=wheezy -component=main snap1",
    ]
    runCmd = "aptly publish add -component=main wheezy snap2"
    expectedCode = 1
    gold_processor = BaseTest.expand_environ


class PublishSourceList1Test(BaseTest):
    """
    publish source list: show source changes
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-contrib",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=wheezy -component=main snap1",
    ]
    runCmd = "aptly publish source list"

    gold_processor = BaseTest.expand_environ


class PublishSourceDrop1Test(BaseTest):
    """
    publish source drop: drop source changes
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-contrib",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=wheezy -component=main snap1",
    ]
    runCmd = "aptly publish source drop"

    gold_processor = BaseTest.expand_environ


class PublishSourceUpdate1Test(BaseTest):
    """
    publish source update: Update single source
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=wheezy -component=main snap1",
    ]
    runCmd = "aptly publish source update -component=main wheezy snap2"

    gold_processor = BaseTest.expand_environ


class PublishSourceUpdate2Test(BaseTest):
    """
    publish source update: Update multiple sources
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-main",
        "aptly snapshot create snap3 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=wheezy -component=main,test snap1 snap2",
    ]
    runCmd = "aptly publish source update -component=main,test wheezy snap2 snap3"

    gold_processor = BaseTest.expand_environ


class PublishSourceUpdate3Test(BaseTest):
    """
    publish source update: Update not existing source
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=wheezy -component=main snap1",
    ]
    runCmd = "aptly publish source update -component=not-existent wheezy snap1"

    gold_processor = BaseTest.expand_environ


class PublishSourceRemove1Test(BaseTest):
    """
    publish source remove: Remove single source
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-contrib",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=wheezy -component=main,contrib snap1 snap2",
    ]
    runCmd = "aptly publish source remove -component=contrib wheezy"

    gold_processor = BaseTest.expand_environ


class PublishSourceRemove2Test(BaseTest):
    """
    publish source remove: Remove multiple sources
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-contrib",
        "aptly snapshot create snap3 from mirror wheezy-non-free",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=wheezy -component=main,contrib,non-free snap1 snap2 snap3",
    ]
    runCmd = "aptly publish source remove -component=contrib,non-free wheezy"

    gold_processor = BaseTest.expand_environ


class PublishSourceRemove3Test(BaseTest):
    """
    publish source remove: Remove not-existing source
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=wheezy -component=main snap1",
    ]
    runCmd = "aptly publish source remove -component=not-existent wheezy"
    expectedCode = 1
    gold_processor = BaseTest.expand_environ
