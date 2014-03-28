from lib import BaseTest


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
