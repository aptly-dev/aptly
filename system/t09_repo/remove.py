from lib import BaseTest


class RemoveRepo1Test(BaseTest):
    """
    remove from local repo: as dep
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool local-repo",
        "aptly repo add local-repo ${files}"
    ]
    runCmd = "aptly repo remove local-repo pyspi some"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages local-repo", "repo_show")

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class RemoveRepo2Test(BaseTest):
    """
    remove from local repo: as dep with version, key
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool local-repo",
        "aptly repo add local-repo ${files}"
    ]
    runCmd = "aptly repo remove local-repo 'pyspi (>> 0.6.1-1.3)' libboost-program-options-dev_1.49.0.1_i386"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages local-repo", "repo_show")

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class RemoveRepo3Test(BaseTest):
    """
    remove from local repo: no such repo
    """
    runCmd = "aptly repo remove local-repo 'pyspi (>> 0.6.1-1.3)' libboost-program-options-dev_1.49.0.1_i386"
    expectedCode = 1


class RemoveRepo4Test(BaseTest):
    """
    remove from local repo: dry run
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool local-repo",
        "aptly repo add local-repo ${files}"
    ]
    runCmd = "aptly repo remove -dry-run local-repo 'pyspi (>> 0.6.1-1.3)' libboost-program-options-dev_1.49.0.1_i386"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages local-repo", "repo_show")

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))

