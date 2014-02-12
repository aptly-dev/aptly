from lib import BaseTest


class DropSnapshot1Test(BaseTest):
    """
    drop snapshot: just drop
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create snap1 from mirror wheezy-non-free"]
    runCmd = "aptly snapshot drop snap1"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly snapshot show snap1", "snapshot_show", expected_code=1)


class DropSnapshot2Test(BaseTest):
    """
    drop snapshot: used as source
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-non-free",
        "aptly snapshot merge snap2 snap1",
    ]
    runCmd = "aptly snapshot drop snap1"
    expectedCode = 1


class DropSnapshot3Test(BaseTest):
    """
    drop snapshot: -force
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-non-free",
        "aptly snapshot merge snap2 snap1",
    ]
    runCmd = "aptly snapshot drop -force snap1"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly snapshot show snap1", "snapshot_show", expected_code=1)


class DropSnapshot4Test(BaseTest):
    """
    drop snapshot: already published
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1"
    ]
    runCmd = "aptly snapshot drop snap1"
    expectedCode = 1


class DropSnapshot5Test(BaseTest):
    """
    drop snapshot: already published with -force
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1"
    ]
    runCmd = "aptly snapshot drop --force snap1"
    expectedCode = 1


class DropSnapshot6Test(BaseTest):
    """
    drop snapshot: no such snapshot
    """
    fixtureDB = True
    runCmd = "aptly snapshot drop no-such-snapshot"
    expectedCode = 1


class DropSnapshot7Test(BaseTest):
    """
    drop snapshot: publish, drop publish, drop snapshot
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1",
        "aptly publish drop maverick",
    ]
    runCmd = "aptly snapshot drop snap1"
