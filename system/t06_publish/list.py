from lib import BaseTest


class PublishList1Test(BaseTest):
    """
    publish list: empty list
    """
    runCmd = "aptly publish list"


class PublishList2Test(BaseTest):
    """
    publish list: several repos list
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot merge snap2 snap1",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1",
        "aptly -architectures=amd64 publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=wheezy -component=contrib snap2 ppa/smira",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -origin=origin1 snap2 ppa/tr1",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -label=label1 snap2 ppa/tr2",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -component=main,contrib snap1 snap2 ppa",
    ]
    runCmd = "aptly publish list"


class PublishList3Test(BaseTest):
    """
    publish list: several repos list, raw
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot merge snap2 snap1",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1",
        "aptly -architectures=amd64 publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -distribution=wheezy -component=contrib snap2 ppa/smira",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -origin=origin1 snap2 ppa/tr1",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -label=label1 snap2 ppa/tr2",
    ]
    runCmd = "aptly publish list -raw"
