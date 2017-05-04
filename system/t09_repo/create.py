import os
import inspect
from lib import BaseTest


def changesRemove(_, s):
    return s.replace(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "changes"), "")


class CreateRepo1Test(BaseTest):
    """
    create local repo: regular repo
    """
    runCmd = "aptly repo create repo1"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show repo1", "repo_show")


class CreateRepo2Test(BaseTest):
    """
    create local repo: regular repo with comment & publishing defaults
    """
    runCmd = "aptly repo create -comment=Repository2 -distribution=maverick -component=non-free repo2"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show repo2", "repo_show")


class CreateRepo3Test(BaseTest):
    """
    create local repo: duplicate name
    """
    fixtureCmds = ["aptly repo create repo3"]
    runCmd = "aptly repo create -comment=Repository3 repo3"
    expectedCode = 1


class CreateRepo4Test(BaseTest):
    """
    create local repo: with uploaders.json
    """
    runCmd = "aptly repo create -uploaders-file=${changes}/uploaders2.json repo4"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show repo4", "repo_show")


class CreateRepo5Test(BaseTest):
    """
    create local repo: with broken uploaders.json
    """
    runCmd = "aptly repo create -uploaders-file=${changes}/uploaders3.json repo5"
    expectedCode = 1


class CreateRepo6Test(BaseTest):
    """
    create local repo: with missing uploaders.json
    """
    runCmd = "aptly repo create -uploaders-file=${changes}/uploaders-not-found.json repo6"
    expectedCode = 1
    outputMatchPrepare = changesRemove


class CreateRepo7Test(BaseTest):
    """
    create local repo: from snapshot
    """
    fixtureCmds = [
        "aptly repo create repo1",
        "aptly repo add repo1 ${files}",
        "aptly snapshot create snap1 from repo repo1",
    ]
    runCmd = "aptly repo create repo2 from snapshot snap1"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo2", "repo_show")


class CreateRepo8Test(BaseTest):
    """
    create local repo: from missing snapshot
    """
    runCmd = "aptly repo create repo2 from snapshot snap-missing"
    expectedCode = 1
