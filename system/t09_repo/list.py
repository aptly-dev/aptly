from lib import BaseTest


class ListRepo1Test(BaseTest):
    """
    list local repos: no repos
    """
    runCmd = "aptly repo list"


class ListRepo2Test(BaseTest):
    """
    list local repo: normal
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool3 repo3",
        "aptly repo create -comment=Cool2 repo2",
        "aptly repo create repo1",
    ]
    runCmd = "aptly repo list"
