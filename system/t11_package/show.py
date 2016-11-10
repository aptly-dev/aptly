from lib import BaseTest


class ShowPackage1Test(BaseTest):
    """
    show package: regular show
    """
    fixtureDB = True
    outputMatchPrepare = lambda _, s: "\n".join(sorted(s.split("\n")))
    runCmd = "aptly package show 'Name (% nginx-extras*)'"


class ShowPackage2Test(BaseTest):
    """
    show package: missing package
    """
    runCmd = "aptly package show 'Name (package-xx)'"


class ShowPackage3Test(BaseTest):
    """
    show package: by key
    """
    fixtureDB = True
    outputMatchPrepare = lambda _, s: "\n".join(sorted(s.split("\n")))
    runCmd = "aptly package show nginx-full_1.2.1-2.2+wheezy2_amd64"


class ShowPackage4Test(BaseTest):
    """
    show package: with files
    """
    fixtureDB = True
    outputMatchPrepare = lambda _, s: "\n".join(sorted(s.split("\n")))
    gold_processor = BaseTest.expand_environ
    runCmd = "aptly package show -with-files nginx-full_1.2.1-2.2+wheezy2_amd64"


class ShowPackage5Test(BaseTest):
    """
    show package: with inclusion
    """
    fixtureDB = True
    outputMatchPrepare = lambda _, s: "\n".join(sorted(s.split("\n")))
    runCmd = "aptly package show -with-references nginx-full_1.2.1-2.2+wheezy2_amd64"


class ShowPackage6Test(BaseTest):
    """
    show package: with inclusion + more inclusions
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-contrib",
        "aptly snapshot create snap3 from mirror wheezy-main-src",
        "aptly snapshot merge snap4 snap1 snap2 snap3",
        "aptly repo create repo1",
        "aptly repo import wheezy-main repo1 nginx",
    ]
    outputMatchPrepare = lambda _, s: "\n".join(sorted(s.split("\n")))
    runCmd = "aptly package show -with-references nginx-full_1.2.1-2.2+wheezy2_amd64"


class ShowPackage7Test(BaseTest):
    """
    show package: with duplicates
    """
    fixtureCmds = [
        "aptly repo create a",
        "aptly repo create b",
        "aptly repo add a ${files}",
        "aptly repo add b ${testfiles}"
    ]
    outputMatchPrepare = lambda _, s: "\n".join(sorted(s.split("\n")))
    runCmd = "aptly package show -with-references \"pyspi (0.6.1-1.3)\""
