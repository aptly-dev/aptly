from lib import BaseTest


class SearchRepo1Test(BaseTest):
    """
    search repo: regular search
    """
    fixtureDB = True
    outputMatchPrepare = lambda _, s: "\n".join(sorted(s.split("\n")))
    fixtureCmds = ["aptly repo create wheezy-main", "aptly repo import wheezy-main wheezy-main Name"]
    runCmd = "aptly repo search wheezy-main '$$Architecture (i386), Name (% *-dev)'"


class SearchRepo2Test(BaseTest):
    """
    search repo: missing repo
    """
    runCmd = "aptly repo search repo-xx 'Name'"
    expectedCode = 1


class SearchRepo3Test(BaseTest):
    """
    search repo: wrong expression
    """
    fixtureDB = True
    fixtureCmds = ["aptly repo create wheezy-main", "aptly repo import wheezy-main wheezy-main Name"]
    expectedCode = 1
    runCmd = "aptly repo search wheezy-main '$$Architecture (i386'"


class SearchRepo4Test(BaseTest):
    """
    search repo: with-deps search
    """
    fixtureDB = True
    fixtureCmds = ["aptly repo create wheezy-main", "aptly repo import wheezy-main wheezy-main Name"]
    outputMatchPrepare = lambda _, s: "\n".join(sorted(s.split("\n")))
    runCmd = "aptly repo search -with-deps wheezy-main 'Name (nginx)'"


class SearchRepo5Test(BaseTest):
    """
    search repo: with -format
    """
    fixtureDB = True
    outputMatchPrepare = lambda _, s: "\n".join(sorted(s.split("\n")))
    fixtureCmds = ["aptly repo create wheezy-main", "aptly repo import wheezy-main wheezy-main Name"]
    runCmd = "aptly repo search -format='{{.Package}}#{{.Version}}' wheezy-main '$$Architecture (i386), Name (% *-dev)'"

class SearchRepo6Test(BaseTest):
    """
    search repo: without query
    """
    fixtureDB = True
    outputMatchPrepare = lambda _, s: "\n".join(sorted(s.split("\n")))
    fixtureCmds = ["aptly repo create wheezy-main", "aptly repo import wheezy-main wheezy-main Name"]
    runCmd = "aptly repo search wheezy-main"
