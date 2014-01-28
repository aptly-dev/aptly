from lib import BaseTest
import re


class ShowSnapshot1Test(BaseTest):
    """
    show snapshot: from mirror
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create snap1 from mirror wheezy-non-free"]
    runCmd = "aptly snapshot show --with-packages snap1"
    outputMatchPrepare = lambda _, s: re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)


class ShowSnapshot2Test(BaseTest):
    """
    show snapshot: no snapshot
    """
    fixtureDB = True
    runCmd = "aptly snapshot show no-such-snapshot"
    expectedCode = 1


class ShowSnapshot3Test(BaseTest):
    """
    show snapshot: from mirror w/o packages
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create snap1 from mirror wheezy-non-free"]
    runCmd = "aptly snapshot show snap1"
    outputMatchPrepare = lambda _, s: re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)
