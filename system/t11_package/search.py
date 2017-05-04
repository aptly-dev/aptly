from lib import BaseTest


class SearchPackage1Test(BaseTest):
    """
    search package: regular search
    """
    fixtureDB = True
    runCmd = "aptly package search '$$Architecture (i386), Name (% *-dev)'"


class SearchPackage2Test(BaseTest):
    """
    search package: missing package
    """
    runCmd = "aptly package search 'Name (package-xx)'"
    expectedCode = 1


class SearchPackage3Test(BaseTest):
    """
    search package: by key
    """
    fixtureDB = True
    runCmd = "aptly package search nginx-full_1.2.1-2.2+wheezy2_amd64"


class SearchPackage4Test(BaseTest):
    """
    search package: by dependency
    """
    fixtureDB = True
    runCmd = "aptly package search coreutils"


class SearchPackage5Test(BaseTest):
    """
    search package: with format
    """
    fixtureDB = True
    runCmd = "aptly package search -format='{{.Package}}#{{.Version}}' '$$Architecture (i386), Name (% *-dev)'"


class SearchPackage6Test(BaseTest):
    """
    search package: no query
    """
    fixtureDB = True
    runCmd = "aptly package search"
