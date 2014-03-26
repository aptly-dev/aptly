from lib import BaseTest


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
