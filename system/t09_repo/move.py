from lib import BaseTest


class MoveRepo1Test(BaseTest):
    """
    move in local repo: simple move
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
        "aptly repo create -comment=Cool -distribution=squeeze repo2",
        "aptly repo add repo1 ${files}"
    ]
    runCmd = "aptly repo move repo1 repo2 'pyspi (>> 0.6.1-1.3)' libboost-program-options-dev_1.49.0.1_i386"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo1", "repo1_show")
        self.check_cmd_output("aptly repo show -with-packages repo2", "repo2_show")

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class MoveRepo2Test(BaseTest):
    """
    move in local repo: simple move w/deps
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
        "aptly repo create -comment=Cool -distribution=squeeze repo2",
        "aptly repo add repo1 ${files}"
    ]
    runCmd = "aptly -architectures=i386,amd64 repo move -with-deps repo1 repo2 'pyspi (>> 0.6.1-1.3)' libboost-program-options-dev_1.49.0.1_i386"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo1", "repo1_show")
        self.check_cmd_output("aptly repo show -with-packages repo2", "repo2_show")

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class MoveRepo3Test(BaseTest):
    """
    move in local repo: simple move w/deps but w/o archs
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
        "aptly repo create -comment=Cool -distribution=squeeze repo2",
        "aptly repo add repo1 ${files}"
    ]
    runCmd = "aptly repo move -with-deps repo1 repo2 'pyspi (>> 0.6.1-1.3)' libboost-program-options-dev_1.49.0.1_i386"
    expectedCode = 1

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class MoveRepo4Test(BaseTest):
    """
    move in local repo: dry run
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
        "aptly repo create -comment=Cool -distribution=squeeze repo2",
        "aptly repo add repo1 ${files}"
    ]
    runCmd = "aptly repo move -dry-run repo1 repo2 'pyspi (>> 0.6.1-1.3)' libboost-program-options-dev_1.49.0.1_i386"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo1", "repo1_show")
        self.check_cmd_output("aptly repo show -with-packages repo2", "repo2_show")

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class MoveRepo5Test(BaseTest):
    """
    move in local repo: wrong dep
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
        "aptly repo create -comment=Cool -distribution=squeeze repo2",
        "aptly repo add repo1 ${files}"
    ]
    runCmd = "aptly repo move repo1 repo2 'pyspi >> 0.6.1-1.3)'"
    expectedCode = 1

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class MoveRepo6Test(BaseTest):
    """
    move in local repo: same src and dest
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
    ]
    runCmd = "aptly repo move repo1 repo1 pyspi"
    expectedCode = 1


class MoveRepo7Test(BaseTest):
    """
    move in local repo: no dst
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
    ]
    runCmd = "aptly repo move repo1 repo2 pyspi"
    expectedCode = 1


class MoveRepo8Test(BaseTest):
    """
    move in local repo: no src
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo2",
    ]
    runCmd = "aptly repo move repo1 repo2 pyspi"
    expectedCode = 1
