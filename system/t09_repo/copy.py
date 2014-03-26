from lib import BaseTest


class CopyRepo1Test(BaseTest):
    """
    copy in local repo: simple copy
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
        "aptly repo create -comment=Cool -distribution=squeeze repo2",
        "aptly repo add repo1 ${files}"
    ]
    runCmd = "aptly repo copy repo1 repo2 'pyspi (>> 0.6.1-1.3)' libboost-program-options-dev_1.49.0.1_i386"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo1", "repo1_show")
        self.check_cmd_output("aptly repo show -with-packages repo2", "repo2_show")

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class CopyRepo2Test(BaseTest):
    """
    copy in local repo: simple copy w/deps
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
        "aptly repo create -comment=Cool -distribution=squeeze repo2",
        "aptly repo add repo1 ${files}"
    ]
    runCmd = "aptly -architectures=i386,amd64 repo copy -with-deps repo1 repo2 'pyspi (>> 0.6.1-1.3)' libboost-program-options-dev_1.49.0.1_i386"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo1", "repo1_show")
        self.check_cmd_output("aptly repo show -with-packages repo2", "repo2_show")

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class CopyRepo3Test(BaseTest):
    """
    copy in local repo: simple copy w/deps but w/o archs
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
        "aptly repo create -comment=Cool -distribution=squeeze repo2",
        "aptly repo add repo1 ${files}"
    ]
    runCmd = "aptly repo copy -with-deps repo1 repo2 'pyspi (>> 0.6.1-1.3)' libboost-program-options-dev_1.49.0.1_i386"
    expectedCode = 1

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class CopyRepo4Test(BaseTest):
    """
    copy in local repo: dry run
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
        "aptly repo create -comment=Cool -distribution=squeeze repo2",
        "aptly repo add repo1 ${files}"
    ]
    runCmd = "aptly repo copy -dry-run repo1 repo2 'pyspi (>> 0.6.1-1.3)' libboost-program-options-dev_1.49.0.1_i386"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo1", "repo1_show")
        self.check_cmd_output("aptly repo show -with-packages repo2", "repo2_show")

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class CopyRepo5Test(BaseTest):
    """
    copy in local repo: wrong dep
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
        "aptly repo create -comment=Cool -distribution=squeeze repo2",
        "aptly repo add repo1 ${files}"
    ]
    runCmd = "aptly repo copy repo1 repo2 'pyspi >> 0.6.1-1.3)'"
    expectedCode = 1

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class CopyRepo6Test(BaseTest):
    """
    copy in local repo: same src and dest
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
    ]
    runCmd = "aptly repo copy repo1 repo1 pyspi"
    expectedCode = 1


class CopyRepo7Test(BaseTest):
    """
    copy in local repo: no dst
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
    ]
    runCmd = "aptly repo copy repo1 repo2 pyspi"
    expectedCode = 1


class CopyRepo8Test(BaseTest):
    """
    copy in local repo: no src
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo2",
    ]
    runCmd = "aptly repo copy repo1 repo2 pyspi"
    expectedCode = 1
