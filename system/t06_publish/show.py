from lib import BaseTest


class PublishShow1Test(BaseTest):
    """
    publish show: existing snapshot
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1",
    ]
    runCmd = "aptly publish show maverick"


class PublishShow2Test(BaseTest):
    """
    publish show: under prefix
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 ppa/smira",
    ]
    runCmd = "aptly publish show maverick ppa/smira"
