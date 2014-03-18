from lib import BaseTest
import re


class CreateSnapshot1Test(BaseTest):
    """
    create snapshot: from mirror
    """
    fixtureDB = True
    runCmd = "aptly snapshot create snap1 from mirror wheezy-main"

    def check(self):
        def remove_created_at(s):
            return re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly snapshot show -with-packages snap1", "snapshot_show", match_prepare=remove_created_at)


class CreateSnapshot2Test(BaseTest):
    """
    create snapshot: no mirror
    """
    fixtureDB = True
    runCmd = "aptly snapshot create snap1 from mirror no-such-mirror"
    expectedCode = 1


class CreateSnapshot3Test(BaseTest):
    """
    create snapshot: duplicate name
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create snap1 from mirror wheezy-main"]
    runCmd = "aptly snapshot create snap1 from mirror wheezy-contrib"
    expectedCode = 1


class CreateSnapshot4Test(BaseTest):
    """
    create snapshot: empty
    """
    runCmd = "aptly snapshot create snap4 empty"

    def check(self):
        def remove_created_at(s):
            return re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly snapshot show -with-packages snap4", "snapshot_show", match_prepare=remove_created_at)


class CreateSnapshot5Test(BaseTest):
    """
    create snapshot: empty duplicate name
    """
    fixtureCmds = ["aptly snapshot create snap5 empty"]
    runCmd = "aptly snapshot create snap5 empty"
    expectedCode = 1


class CreateSnapshot6Test(BaseTest):
    """
    create snapshot: from repo
    """
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}"
    ]
    runCmd = "aptly snapshot create snap6 from repo local-repo"

    def check(self):
        def remove_created_at(s):
            return re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly snapshot show -with-packages snap6", "snapshot_show", match_prepare=remove_created_at)


class CreateSnapshot7Test(BaseTest):
    """
    create snapshot: no repo
    """
    runCmd = "aptly snapshot create snap1 from repo no-such-repo"
    expectedCode = 1


class CreateSnapshot8Test(BaseTest):
    """
    create snapshot: duplicate name from repo
    """
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
        "aptly snapshot create snap8 from repo local-repo"
    ]
    runCmd = "aptly snapshot create snap8 from repo local-repo"
    expectedCode = 1


class CreateSnapshot9Test(BaseTest):
    """
    create snapshot: from empty repo
    """
    fixtureCmds = [
        "aptly repo create local-repo",
    ]
    runCmd = "aptly snapshot create snap9 from repo local-repo"
    expectedCode = 1
