from lib import BaseTest


class RenameSnapshot1Test(BaseTest):
    """
    rename snapshot: regular operations
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snapshot1 from mirror wheezy-contrib",
    ]
    runCmd = "aptly snapshot rename snapshot1 snapshot2"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly snapshot list", "snapshot_list")


class RenameSnapshot2Test(BaseTest):
    """
    rename snapshot: missing snapshot
    """
    runCmd = "aptly snapshot rename snapshot2 snapshot3"
    expectedCode = 1


class RenameSnapshot3Test(BaseTest):
    """
    rename snapshot: already exists
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snapshot3 from mirror wheezy-contrib",
        "aptly snapshot create snapshot4 from mirror wheezy-contrib",
    ]
    runCmd = "aptly snapshot rename snapshot3 snapshot4"
    expectedCode = 1
