import os
import inspect
from lib import BaseTest


def changesRemove(_, s):
    return s.replace(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "changes"), "")


class EditRepo1Test(BaseTest):
    """
    edit repo: change comment
    """
    fixtureCmds = [
        "aptly repo create repo1",
    ]
    runCmd = "aptly repo edit -comment=Lala repo1"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show repo1", "repo-show")


class EditRepo2Test(BaseTest):
    """
    edit repo: change distribution & component
    """
    fixtureCmds = [
        "aptly repo create -comment=Lala -component=non-free repo2",
    ]
    runCmd = "aptly repo edit -distribution=wheezy -component=contrib repo2"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show repo2", "repo-show")


class EditRepo3Test(BaseTest):
    """
    edit repo: no such repo
    """
    runCmd = "aptly repo edit repo3"
    expectedCode = 1


class EditRepo4Test(BaseTest):
    """
    edit repo: add uploaders.json
    """
    fixtureCmds = [
        "aptly repo create repo4",
    ]
    runCmd = "aptly repo edit -uploaders-file=${changes}/uploaders2.json repo4"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show repo4", "repo_show")


class EditRepo5Test(BaseTest):
    """
    edit repo: with broken uploaders.json
    """
    fixtureCmds = [
        "aptly repo create repo5",
    ]
    runCmd = "aptly repo edit -uploaders-file=${changes}/uploaders3.json repo5"
    expectedCode = 1


class EditRepo6Test(BaseTest):
    """
    edit local repo: with missing uploaders.json
    """
    fixtureCmds = [
        "aptly repo create repo6",
    ]
    runCmd = "aptly repo edit -uploaders-file=${changes}/uploaders-not-found.json repo6"
    expectedCode = 1
    outputMatchPrepare = changesRemove


class EditRepo7Test(BaseTest):
    """
    edit local repo: remove uploaders.json
    """
    fixtureCmds = [
        "aptly repo create -uploaders-file=${changes}/uploaders2.json repo7",
    ]
    runCmd = "aptly repo edit -uploaders-file= repo7"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show repo7", "repo_show")
