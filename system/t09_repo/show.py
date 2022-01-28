from lib import BaseTest


class ShowRepo1Test(BaseTest):
    """
    show local repo: regular
    """
    fixtureCmds = ["aptly repo create -comment=Cool -distribution=squeeze repo1"]
    runCmd = "aptly repo show repo1"


class ShowRepo2Test(BaseTest):
    """
    show local repo: -with-packages
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=wheezy -component=contrib repo2",
        "aptly repo add repo2 ${files}"
    ]
    runCmd = "aptly repo show -with-packages repo2"


class ShowRepo3Test(BaseTest):
    """
    show local repo: not found
    """
    expectedCode = 1
    runCmd = "aptly repo show repo3"


class ShowRepo4Test(BaseTest):
    """
    show local repo: json regular
    """
    fixtureCmds = ["aptly repo create -comment=Cool -distribution=squeeze repo1"]
    runCmd = "aptly repo show -json repo1"


class ShowRepo5Test(BaseTest):
    """
    show local repo: json -with-packages
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=wheezy -component=contrib repo2",
        "aptly repo add repo2 ${files}"
    ]
    runCmd = "aptly repo show -json -with-packages repo2"
