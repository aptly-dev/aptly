from lib import BaseTest


class PublishList1Test(BaseTest):
    """
    publish list: empty list
    """
    runCmd = "aptly publish list"


class PublishList2Test(BaseTest):
    """
    publish list: several snapshots list
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot merge snap2 snap1",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1",
        "aptly -architectures=amd64 publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=wheezy -component=contrib snap2 ppa/smira",
    ]
    runCmd = "aptly publish list"
