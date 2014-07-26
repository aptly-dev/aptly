from lib import BaseTest


class RenameRepo1Test(BaseTest):
    """
    rename repo: regular operations
    """
    fixtureCmds = [
        "aptly repo create repo1",
    ]
    runCmd = "aptly repo rename repo1 repo2"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo list", "repo_list")


class RenameRepo2Test(BaseTest):
    """
    rename repo: missing repo
    """
    runCmd = "aptly repo rename repo2 repo3"
    expectedCode = 1


class RenameRepo3Test(BaseTest):
    """
    rename repo: already exists
    """
    fixtureCmds = [
        "aptly repo create repo3",
        "aptly repo create repo4",
    ]
    runCmd = "aptly repo rename repo3 repo4"
    expectedCode = 1
