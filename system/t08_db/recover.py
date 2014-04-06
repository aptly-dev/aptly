from lib import BaseTest


class RecoverDB1Test(BaseTest):
    """
    recover db: no DB
    """
    runCmd = "aptly db recover"


class RecoverDB2Test(BaseTest):
    """
    recover db: without CURRENT files
    """
    fixtureDB = True
    runCmd = "aptly db recover"

    def prepare(self):
        super(RecoverDB2Test, self).prepare()

        self.delete_file("db/CURRENT")

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror list", "mirror_list")
