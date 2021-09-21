from lib import BaseTest
import re


class ShowSnapshot1Test(BaseTest):
    """
    show snapshot: from mirror
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create snap1 from mirror wheezy-non-free"]
    runCmd = "aptly snapshot show --with-packages snap1"

    def outputMatchPrepare(_, s):
        return re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)


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

    def outputMatchPrepare(_, s):
        return re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)


class ShowSnapshot4Test(BaseTest):
    """
    show snapshot json: from mirror w/o packages
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create snap1 from mirror wheezy-non-free"]
    runCmd = "aptly snapshot show -json snap1"

    def outputMatchPrepare(_, s):
        return re.sub(r'[ ]*"CreatedAt": "[^"]+",?\n', '', s)


class ShowSnapshot5Test(BaseTest):
    """
    show snapshot json: from mirror with packages
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create snap1 from mirror gnuplot-maverick"]
    runCmd = "aptly snapshot show -json -with-packages snap1"

    def outputMatchPrepare(_, s):
        return re.sub(r'[ ]*"CreatedAt": "[^"]+",?\n', '', s)


class ShowSnapshot6Test(BaseTest):
    """
    show snapshot json: from local repo w/o packages
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=wheezy -component=contrib repo1",
        "aptly repo add repo1 ${files}",
        "aptly snapshot create snap1 from repo repo1"
    ]
    runCmd = "aptly snapshot show -json snap1"

    def outputMatchPrepare(_, s):
        return re.sub(r'[ ]*"CreatedAt": "[^"]+",?\n', '', s)


class ShowSnapshot7Test(BaseTest):
    """
    show snapshot json: from local repo with packages
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=wheezy -component=contrib repo1",
        "aptly repo add repo1 ${files}",
        "aptly snapshot create snap1 from repo repo1"
    ]
    runCmd = "aptly snapshot show -json -with-packages snap1"

    def outputMatchPrepare(_, s):
        return re.sub(r'[ ]*"CreatedAt": "[^"]+",?\n', '', s)


class ShowSnapshot8Test(BaseTest):
    """
    show snapshot json: from local repo w/o packages
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 from mirror sensu",
        "aptly snapshot pull snap1 snap2 snap3 sensu"
    ]
    runCmd = "aptly snapshot show -json snap3"

    def outputMatchPrepare(_, s):
        return re.sub(r'[ ]*"CreatedAt": "[^"]+",?\n', '', s)


class ShowSnapshot9Test(BaseTest):
    """
    show snapshot json: from local repo with packages
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 from mirror sensu",
        "aptly snapshot pull snap1 snap2 snap3 sensu"
    ]
    runCmd = "aptly snapshot show -json -with-packages snap3"

    def outputMatchPrepare(_, s):
        return re.sub(r'[ ]*"CreatedAt": "[^"]+",?\n', '', s)
