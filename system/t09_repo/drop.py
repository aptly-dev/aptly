from lib import BaseTest


class DropRepo1Test(BaseTest):
    """
    drop repo: regular drop
    """
    fixtureCmds = [
        "aptly repo create repo1",
    ]
    runCmd = "aptly repo drop repo1"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show repo1", "repo-show", expected_code=1)


class DropRepo2Test(BaseTest):
    """
    drop repo: in use by snapshots
    """
    fixtureCmds = [
        "aptly repo create repo2",
        "aptly repo add repo2 ${files}",
        "aptly snapshot create local from repo repo2",
    ]
    runCmd = "aptly repo drop repo2"
    expectedCode = 1


class DropRepo3Test(BaseTest):
    """
    drop repo: force
    """
    fixtureCmds = [
        "aptly repo create repo3",
        "aptly repo add repo3 ${files}",
        "aptly snapshot create local from repo repo3",
    ]
    runCmd = "aptly repo drop --force repo3"


class DropRepo4Test(BaseTest):
    """
    drop repo: no such repo
    """
    runCmd = "aptly repo drop repo4"
    expectedCode = 1


class DropRepo5Test(BaseTest):
    """
    drop repo: published
    """
    fixtureCmds = [
        "aptly repo create repo5",
        "aptly repo add repo5 ${files}",
        "aptly publish repo -skip-signing -distribution=squeeze repo5",
    ]
    runCmd = "aptly repo drop repo5"
    expectedCode = 1
