import re
from lib import BaseTest


class DiffSnapshot1Test(BaseTest):
    """
    diff two snapshots: normal diff
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-backports",
        "aptly snapshot pull snap1 snap2 snap3 'rsyslog (>= 7.4.4)'"
    ]
    runCmd = "aptly snapshot diff snap1 snap3"
    # trim trailing whitespace
    outputMatchPrepare = lambda _, s: re.sub(r'\s*$', '', s, flags=re.MULTILINE)


class DiffSnapshot2Test(BaseTest):
    """
    diff two snapshots: normal diff II
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-backports",
    ]
    runCmd = "aptly snapshot diff snap1 snap2"
    # trim trailing whitespace
    outputMatchPrepare = lambda _, s: re.sub(r'\s*$', '', s, flags=re.MULTILINE)


class DiffSnapshot3Test(BaseTest):
    """
    diff two snapshots: normal diff II + only-matching
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-backports",
    ]
    runCmd = "aptly snapshot diff -only-matching snap1 snap2"
    # trim trailing whitespace
    outputMatchPrepare = lambda _, s: re.sub(r'\s*$', '', s, flags=re.MULTILINE)


class DiffSnapshot4Test(BaseTest):
    """
    diff two snapshots: doesn't exist
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
    ]
    runCmd = "aptly snapshot diff -only-matching snap1 snap-no"
    expectedCode = 1


class DiffSnapshot5Test(BaseTest):
    """
    diff two snapshots: doesn't exist
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap2 from mirror wheezy-main",
    ]
    runCmd = "aptly snapshot diff -only-matching snap-no snap2"
    expectedCode = 1


class DiffSnapshot6Test(BaseTest):
    """
    diff two snapshots: identical snapshots
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-main",
    ]
    runCmd = "aptly snapshot diff snap1 snap2"
