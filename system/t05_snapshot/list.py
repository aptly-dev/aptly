from lib import BaseTest


class ListSnapshot1Test(BaseTest):
    """
    list snapshots: regular list
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-contrib",
        "aptly snapshot merge snap3 snap1 snap2",
        "aptly snapshot pull snap1 snap2 snap4 mame unrar",
        "aptly repo create local-repo",
        "aptly repo add local-repo ${files}",
        "aptly snapshot create snap5 from repo local-repo",
    ]
    runCmd = "aptly snapshot list"


class ListSnapshot2Test(BaseTest):
    """
    list snapshots: empty list
    """
    runCmd = "aptly snapshot list"
