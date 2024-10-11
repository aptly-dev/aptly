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


class PublishShow3Test(BaseTest):
    """
    publish show json: existing snapshot
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1",
    ]
    runCmd = "aptly publish show -json maverick"


class PublishShow4Test(BaseTest):
    """
    publish show json: under prefix
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1 ppa/smira",
    ]
    runCmd = "aptly publish show -json maverick ppa/smira"


class PublishShow5Test(BaseTest):
    """
    publish show: existing local repo
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly repo create -distribution=wheezy local-repo",
        "aptly publish repo -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -architectures=i386 local-repo"
    ]
    runCmd = "aptly publish show wheezy"
    gold_processor = BaseTest.expand_environ
