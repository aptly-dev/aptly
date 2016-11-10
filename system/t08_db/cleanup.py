from lib import BaseTest


class CleanupDB1Test(BaseTest):
    """
    cleanup db: no DB
    """
    runCmd = "aptly db cleanup"


class CleanupDB2Test(BaseTest):
    """
    cleanup db: deleting packages when mirrors are missing
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly mirror drop wheezy-main-src",
        "aptly mirror drop wheezy-main",
        "aptly mirror drop wheezy-contrib",
    ]
    runCmd = "aptly db cleanup"


class CleanupDB3Test(BaseTest):
    """
    cleanup db: deleting packages and files
    """
    fixtureDB = True
    fixturePoolCopy = True
    fixtureCmds = [
        "aptly mirror drop gnuplot-maverick-src",
        "aptly mirror drop gnuplot-maverick",
    ]
    runCmd = "aptly db cleanup"


class CleanupDB4Test(BaseTest):
    """
    cleanup db: deleting a mirror, but still referenced by snapshot
    """
    fixtureDB = True
    fixturePoolCopy = True
    fixtureCmds = [
        "aptly snapshot create gnuplot from mirror gnuplot-maverick",
        "aptly mirror drop -force gnuplot-maverick",
    ]
    runCmd = "aptly db cleanup"


class CleanupDB5Test(BaseTest):
    """
    cleanup db: create/delete snapshot, drop mirror
    """
    fixtureDB = True
    fixturePoolCopy = True
    fixtureCmds = [
        "aptly mirror drop gnuplot-maverick-src",
        "aptly snapshot create gnuplot from mirror gnuplot-maverick",
        "aptly snapshot drop gnuplot",
        "aptly mirror drop gnuplot-maverick",
    ]
    runCmd = "aptly db cleanup"


class CleanupDB6Test(BaseTest):
    """
    cleanup db: db is full
    """
    fixtureDB = True
    fixturePoolCopy = True
    runCmd = "aptly db cleanup"


class CleanupDB7Test(BaseTest):
    """
    cleanup db: local repos
    """
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
    ]
    runCmd = "aptly db cleanup"


class CleanupDB8Test(BaseTest):
    """
    cleanup db: local repos dropped
    """
    fixtureCmds = [
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
        "aptly repo drop local-repo",
    ]
    runCmd = "aptly db cleanup"


class CleanupDB9Test(BaseTest):
    """
    cleanup db: publish local repo, remove packages from repo, db cleanup
    """
    fixtureCmds = [
        "aptly repo create -distribution=abc local-repo",
        "aptly repo create -distribution=def local-repo2",
        "aptly repo add local-repo ${files}",
        "aptly publish repo -skip-signing local-repo",
        "aptly publish repo -skip-signing -architectures=i386 local-repo2",
        "aptly repo remove local-repo Name",
    ]
    runCmd = "aptly db cleanup"

    def check(self):
            self.check_output()
            self.check_cmd_output("aptly publish drop def", "publish_drop", match_prepare=self.expand_environ)


class CleanupDB10Test(BaseTest):
    """
    cleanup db: conflict in packages, should not cleanup anything
    """
    fixtureCmds = [
        "aptly repo create a",
        "aptly repo create b",
        "aptly repo add a ${files}",
        "aptly repo add b ${testfiles}"
    ]
    runCmd = "aptly db cleanup"


class CleanupDB11Test(BaseTest):
    """
    cleanup db: deleting packages and files, -verbose
    """
    fixtureDB = True
    fixturePoolCopy = True
    fixtureCmds = [
        "aptly mirror drop gnuplot-maverick-src",
        "aptly mirror drop gnuplot-maverick",
    ]
    runCmd = "aptly db cleanup -verbose"


class CleanupDB12Test(BaseTest):
    """
    cleanup db: deleting packages and files, -verbose & -dry-run
    """
    fixtureDB = True
    fixturePoolCopy = True
    fixtureCmds = [
        "aptly mirror drop gnuplot-maverick-src",
        "aptly mirror drop gnuplot-maverick",
    ]
    runCmd = "aptly db cleanup -verbose -dry-run"
