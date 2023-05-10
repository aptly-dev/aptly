from lib import BaseTest


class DropMirror1Test(BaseTest):
    """
    drop mirror: regular list
    """
    fixtureCmds = [
        "aptly mirror create --ignore-signatures mirror1 http://archive.debian.org/debian-archive/debian/ stretch",
    ]
    runCmd = "aptly mirror drop mirror1"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror1", "mirror_show", expected_code=1)


class DropMirror2Test(BaseTest):
    """
    drop mirror: in use by snapshots
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create wheez from mirror wheezy-main"
    ]
    runCmd = "aptly mirror drop wheezy-main"
    expectedCode = 1


class DropMirror3Test(BaseTest):
    """
    drop mirror: force
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create wheez from mirror wheezy-main"
    ]
    runCmd = "aptly mirror drop --force wheezy-main"


class DropMirror4Test(BaseTest):
    """
    drop mirror: no such mirror
    """
    runCmd = "aptly mirror drop mirror1"
    expectedCode = 1
