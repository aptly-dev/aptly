import os

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


class RecoverDB3Test(BaseTest):
    """
    recover db: dangling reference
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly repo create db3test",
        "aptly repo add db3test changes/hardlink_0.2.1_amd64.deb",
    ]

    runCmd = "aptly db recover"

    def prepare(self):
        super(RecoverDB3Test, self).prepare()

        self.run_cmd(["go", "run", "files/corruptdb.go",
                      "-db", os.path.join(os.environ["HOME"], self.aptlyDir, "db"),
                      "-prefix", "Pamd64 hardlink 0.2.1"])

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly db cleanup", "cleanup")
