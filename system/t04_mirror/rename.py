from lib import BaseTest


class RenameMirror1Test(BaseTest):
    """
    rename mirror: regular operations
    """
    fixtureDB = True
    runCmd = "aptly mirror rename wheezy-main wheezy-main-cool"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror list", "mirror_list")


class RenameMirror2Test(BaseTest):
    """
    rename mirror: missing mirror
    """
    runCmd = "aptly mirror rename wheezy-main wheezy-main-cool"
    expectedCode = 1


class RenameMirror3Test(BaseTest):
    """
    rename mirror: already exists
    """
    fixtureDB = True
    runCmd = "aptly mirror rename wheezy-main wheezy-contrib"
    expectedCode = 1
