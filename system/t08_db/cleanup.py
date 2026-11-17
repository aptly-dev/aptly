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


class CleanupDB13Test(BaseTest):
    """
    cleanup db: appstream files survive cleanup
    """
    fixtureWebServer = "../t04_mirror/test_release2"
    fixtureGpg = True
    configOverride = {"downloadRetries": 0}
    fixtureCmds = [
        "aptly mirror create --ignore-signatures -with-appstream -architectures=amd64 appstream-test ${url} hardy main",
        "aptly mirror update -ignore-checksums --ignore-signatures appstream-test",
        "aptly snapshot create snap-appstream from mirror appstream-test",
    ]
    runCmd = "aptly db cleanup"

    def check(self):
        self.check_output()
        # verify appstream files survive cleanup by publishing the snapshot
        self.check_cmd_output(
            "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap-appstream",
            "publish", match_prepare=self.expand_environ)
        self.check_exists('public/dists/hardy/main/dep11/Components-amd64.yml.gz')
        self.check_exists('public/dists/hardy/main/dep11/icons-48x48.tar.gz')


class CleanupDB14Test(BaseTest):
    """
    cleanup db: revert split reflists
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly db cleanup",
    ]
    runCmd = "aptly db cleanup -revert-split-reflists -verbose"


class CleanupDB15Test(BaseTest):
    """
    cleanup db: revert split reflists and then convert back to split
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly db cleanup",
        "aptly db cleanup -revert-split-reflists -verbose",
    ]
    runCmd = "aptly db cleanup -verbose"
