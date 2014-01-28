from lib import BaseTest
import re


class MergeSnapshot1Test(BaseTest):
    """
    merge snapshots: two snapshots
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-non-free",
    ]
    runCmd = "aptly snapshot merge snap3 snap1 snap2"

    def check(self):
        def remove_created_at(s):
            return re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly snapshot show -with-packages snap3", "snapshot_show", match_prepare=remove_created_at)


class MergeSnapshot2Test(BaseTest):
    """
    merge snapshots: one snapshot
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
    ]
    runCmd = "aptly snapshot merge snap2 snap1"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly snapshot diff snap1 snap2", "snapshot_diff")


class MergeSnapshot3Test(BaseTest):
    """
    merge snapshots: three snapshots
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-non-free",
        "aptly snapshot create snap3 from mirror wheezy-backports",
    ]
    runCmd = "aptly snapshot merge snap4 snap1 snap2 snap3"

    def check(self):
        def remove_created_at(s):
            return re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly snapshot show -with-packages snap4", "snapshot_show", match_prepare=remove_created_at)


class MergeSnapshot4Test(BaseTest):
    """
    merge snapshots: no such snapshot
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
    ]
    runCmd = "aptly snapshot merge snap3 snap1 snap2"
    expectedCode = 1


class MergeSnapshot5Test(BaseTest):
    """
    merge snapshots: duplicate snapshot
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
    ]
    runCmd = "aptly snapshot merge snap1 snap1"
    expectedCode = 1
