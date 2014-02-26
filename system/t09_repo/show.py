from lib import BaseTest


class ShowRepo1Test(BaseTest):
    """
    show local repo: regular
    """
    fixtureCmds = ["aptly repo create -comment=Cool repo1"]
    runCmd = "aptly repo show repo1"


class ShowRepo2Test(BaseTest):
    """
    show local repo: -with-packages
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool repo2",
        "aptly repo add repo2 ${files}"
    ]
    runCmd = "aptly repo show -with-packages repo2"


class ShowRepo3Test(BaseTest):
    """
    show local repo: not found
    """
    expectedCode = 1
    runCmd = "aptly repo show repo3"
