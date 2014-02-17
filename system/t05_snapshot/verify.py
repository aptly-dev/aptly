from lib import BaseTest


class VerifySnapshot1Test(BaseTest):
    """
    verify snapshot: from wheezy
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create snap1 from mirror wheezy-main"]
    runCmd = "aptly snapshot verify snap1"


class VerifySnapshot2Test(BaseTest):
    """
    verify snapshot: no snapshot
    """
    fixtureDB = True
    runCmd = "aptly snapshot verify no-such-snapshot"
    expectedCode = 1


class VerifySnapshot3Test(BaseTest):
    """
    verify snapshot: limited architectues
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create snap1 from mirror wheezy-main"]
    runCmd = "aptly -architectures=i386 snapshot verify snap1"


class VerifySnapshot4Test(BaseTest):
    """
    verify snapshot: limited architectues + suggests
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create snap1 from mirror wheezy-main"]
    runCmd = "aptly -architectures=i386 -dep-follow-suggests snapshot verify snap1"


class VerifySnapshot5Test(BaseTest):
    """
    verify snapshot: limited architectues + suggests + multiple sources
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-contrib",
        "aptly snapshot create snap3 from mirror wheezy-non-free",
    ]
    runCmd = "aptly -architectures=i386 -dep-follow-suggests snapshot verify snap1 snap2 snap3"


class VerifySnapshot6Test(BaseTest):
    """
    verify snapshot: suggests + recommends
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create snap1 from mirror wheezy-main"]
    runCmd = "aptly -dep-follow-recommends -dep-follow-suggests snapshot verify snap1"


class VerifySnapshot7Test(BaseTest):
    """
    verify snapshot: follow-all-variants
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create snap1 from mirror wheezy-main"]
    runCmd = "aptly -dep-follow-all-variants snapshot verify snap1"


class VerifySnapshot8Test(BaseTest):
    """
    verify snapshot: follow-source w/o sources
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create snap1 from mirror gnuplot-maverick"]
    runCmd = "aptly -dep-follow-source snapshot verify snap1"


class VerifySnapshot9Test(BaseTest):
    """
    verify snapshot: follow-source w sources
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create snap1 from mirror gnuplot-maverick-src"]
    runCmd = "aptly -dep-follow-source snapshot verify snap1"


class VerifySnapshot10Test(BaseTest):
    """
    verify snapshot: follow-source on whole wheezy
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main-src",
        "aptly snapshot create snap2 from mirror wheezy-contrib-src",
        "aptly snapshot create snap3 from mirror wheezy-non-free-src",
    ]
    runCmd = "aptly -dep-follow-source snapshot verify snap1 snap2 snap3"
