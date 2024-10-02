from lib import BaseTest


class PublishAdd1Test(BaseTest):
    """
    publish add: add new component to snapshot publish
    """
    fixtureCmds = [
        "aptly snapshot create snap1 empty",
        "aptly snapshot create snap2 empty",
        "aptly snapshot create snap3 empty",
        "aptly publish snapshot -architectures=i386 -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=a,b snap1 snap2",
    ]
    runCmd = "aptly publish add -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -component=c maverick snap3"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishAdd1Test, self).check()

        self.check_exists('public/dists/maverick/a/binary-i386/Packages')
        self.check_exists('public/dists/maverick/b/binary-i386/Packages')
        self.check_exists('public/dists/maverick/c/binary-i386/Packages')


class PublishAdd2Test(BaseTest):
    """
    publish add: add new component to local repo publish
    """
    fixtureCmds = [
        "aptly repo create local-repo1",
        "aptly repo create local-repo2",
        "aptly repo create local-repo3",
        "aptly publish repo -architectures=i386 -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=a,b local-repo1 local-repo2",
    ]
    runCmd = "aptly publish add -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -component=c maverick local-repo3"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishAdd2Test, self).check()

        self.check_exists('public/dists/maverick/a/binary-i386/Packages')
        self.check_exists('public/dists/maverick/b/binary-i386/Packages')
        self.check_exists('public/dists/maverick/c/binary-i386/Packages')


class PublishAdd3Test(BaseTest):
    """
    publish add: add already existing component
    """
    fixtureCmds = [
        "aptly snapshot create snap1 empty",
        "aptly snapshot create snap2 empty",
        "aptly snapshot create snap3 empty",
        "aptly publish snapshot -architectures=i386 -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=a,b snap1 snap2",
    ]
    runCmd = "aptly publish add -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -component=b maverick snap3"
    expectedCode = 1
