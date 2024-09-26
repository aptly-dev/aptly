from lib import BaseTest


class PublishRemove1Test(BaseTest):
    """
    publish remove: remove single component from published repository
    """
    fixtureCmds = [
        "aptly snapshot create snap1 empty",
        "aptly snapshot create snap2 empty",
        "aptly snapshot create snap3 empty",
        "aptly publish snapshot -architectures=i386 -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=a,b,c snap1 snap2 snap3"
    ]
    runCmd = "aptly publish remove -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick c"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishRemove1Test, self).check()

        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/a/binary-i386/Packages')
        self.check_exists('public/dists/maverick/b/binary-i386/Packages')
        self.check_not_exists('public/dists/maverick/c/binary-i386/Packages')

        release = self.read_file('public/dists/maverick/Release').split('\n')
        components = next((e.split(': ')[1] for e in release if e.startswith('Components')), None)
        components = sorted(components.split(' '))

        if ['a', 'b'] != components:
            raise Exception("value of 'Components' in release file is '%s' and does not match '%s'." % (' '.join(components), 'a b'))


class PublishRemove2Test(BaseTest):
    """
    publish remove: remove multiple components from published repository
    """
    fixtureCmds = [
        "aptly snapshot create snap1 empty",
        "aptly snapshot create snap2 empty",
        "aptly snapshot create snap3 empty",
        "aptly publish snapshot -architectures=i386 -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=a,b,c snap1 snap2 snap3"
    ]
    runCmd = "aptly publish remove -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick b c"
    gold_processor = BaseTest.expand_environ

    def check(self):
        super(PublishRemove2Test, self).check()

        self.check_exists('public/dists/maverick/Release')
        self.check_exists('public/dists/maverick/a/binary-i386/Packages')
        self.check_not_exists('public/dists/maverick/b/binary-i386/Packages')
        self.check_not_exists('public/dists/maverick/c/binary-i386/Packages')

        release = self.read_file('public/dists/maverick/Release').split('\n')
        components = next((e.split(': ')[1] for e in release if e.startswith('Components')), None)
        components = sorted(components.split(' '))

        if ['a'] != components:
            raise Exception("value of 'Components' in release file is '%s' and does not match '%s'." % (' '.join(components), 'a'))


class PublishRemove3Test(BaseTest):
    """
    publish remove: remove not existing component from published repository
    """
    fixtureCmds = [
        "aptly snapshot create snap1 empty",
        "aptly publish snapshot -architectures=i386 -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=a snap1"
    ]
    runCmd = "aptly publish remove -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick not-existent"
    expectedCode = 1
    gold_processor = BaseTest.expand_environ


class PublishRemove4Test(BaseTest):
    """
    publish remove: unspecified components
    """
    fixtureCmds = [
        "aptly snapshot create snap1 empty",
        "aptly snapshot create snap2 empty",
        "aptly snapshot create snap3 empty",
        "aptly publish snapshot -architectures=i386 -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=maverick -component=a,b,c snap1 snap2 snap3"
    ]
    runCmd = "aptly publish remove -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec maverick"
    expectedCode = 2
    gold_processor = BaseTest.expand_environ
