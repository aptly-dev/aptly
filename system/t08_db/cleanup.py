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
