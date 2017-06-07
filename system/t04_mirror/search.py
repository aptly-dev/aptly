from lib import BaseTest


class SearchMirror1Test(BaseTest):
    """
    search mirror: regular search
    """
    fixtureDB = True
    runCmd = "aptly mirror search wheezy-main '$$Architecture (i386), Name (% *-dev)'"


class SearchMirror2Test(BaseTest):
    """
    search mirror: missing mirror
    """
    runCmd = "aptly mirror search mirror-xx 'Name'"
    expectedCode = 1


class SearchMirror3Test(BaseTest):
    """
    search mirror: wrong expression
    """
    fixtureDB = True
    expectedCode = 1
    runCmd = "aptly mirror search wheezy-main '$$Architecture (i386'"


class SearchMirror4Test(BaseTest):
    """
    search mirror: with-deps search
    """
    fixtureDB = True
    runCmd = "aptly mirror search -with-deps wheezy-main 'Name (nginx)'"


class SearchMirror5Test(BaseTest):
    """
    search mirror: regular search
    """
    fixtureDB = True
    runCmd = "aptly mirror search -format='{{.Package}}#{{.Version}}' wheezy-main '$$Architecture (i386), Name (% *-dev)'"


class SearchMirror6Test(BaseTest):
    """
    search mirror: no query
    """
    fixtureDB = True
    runCmd = "aptly mirror search -format='{{.Package}}#{{.Version}}' wheezy-main"
