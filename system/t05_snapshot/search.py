from lib import BaseTest


class SearchSnapshot1Test(BaseTest):
    """
    search snapshot: regular search
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create wheezy-main from mirror wheezy-main"]
    runCmd = "aptly snapshot search wheezy-main '$$Architecture (i386), Name (% *-dev)'"


class SearchSnapshot2Test(BaseTest):
    """
    search snapshot: missing snapshot
    """
    runCmd = "aptly snapshot search snapshot-xx 'Name'"
    expectedCode = 1


class SearchSnapshot3Test(BaseTest):
    """
    search snapshot: wrong expression
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create wheezy-main from mirror wheezy-main"]
    expectedCode = 1
    runCmd = "aptly snapshot search wheezy-main '$$Architecture (i386'"


class SearchSnapshot4Test(BaseTest):
    """
    search snapshot: with-deps search
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create wheezy-main from mirror wheezy-main"]
    runCmd = "aptly snapshot search -with-deps wheezy-main 'Name (nginx)'"


class SearchSnapshot5Test(BaseTest):
    """
    search snapshot: no results
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create wheezy-main from mirror wheezy-main"]
    runCmd = "aptly snapshot search -with-deps wheezy-main 'Name (no-such-package)'"
    expectedCode = 1


class SearchSnapshot6Test(BaseTest):
    """
    search snapshot: with format
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create wheezy-main from mirror wheezy-main"]
    runCmd = "aptly snapshot search -format='{{.Package}}#{{.Version}}' wheezy-main '$$Architecture (i386), Name (% *-dev)'"


class SearchSnapshot7Test(BaseTest):
    """
    search snapshot: without query
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create wheezy-main from mirror wheezy-main"]
    runCmd = "aptly snapshot search -format='{{.Package}}#{{.Version}}' wheezy-main"
