from lib import BaseTest


class ImportRepo1Test(BaseTest):
    """
    import to local repo: simple import
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
        "aptly repo add repo1 ${files}"
    ]
    runCmd = "aptly repo import wheezy-main repo1 'nginx (>= 1.2)' unpaper_0.4.2-1_amd64"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo1", "repo_show")

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class ImportRepo2Test(BaseTest):
    """
    import to local repo: import w/deps
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
        "aptly repo add repo1 ${files}"
    ]
    runCmd = "aptly -architectures=i386,amd64 repo import -with-deps wheezy-main repo1 dpkg_1.16.12_i386 userinfo_2.2-3_amd64"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo1", "repo_show")

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class ImportRepo3Test(BaseTest):
    """
    import to local repo: simple move w/deps but w/o archs
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
    ]
    runCmd = "aptly repo import -with-deps wheezy-contrib repo1 redeclipse"
    expectedCode = 1

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class ImportRepo4Test(BaseTest):
    """
    import to local repo: dry run
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
    ]
    runCmd = "aptly -architectures=i386,amd64 repo import -dry-run -with-deps wheezy-contrib repo1 redeclipse-dbg"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo1", "repo_show")

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class ImportRepo5Test(BaseTest):
    """
    import to local repo: wrong dep
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
    ]
    runCmd = "aptly repo import wheezy-contrib repo1 'pyspi >> 0.6.1-1.3)'"
    expectedCode = 1

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class ImportRepo6Test(BaseTest):
    """
    import to local repo: non-updated mirror
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
        "aptly mirror create --ignore-signatures mirror1 http://mirror.yandex.ru/debian/ wheezy",
    ]
    runCmd = "aptly repo import mirror1 repo1 nginx"
    expectedCode = 1


class ImportRepo7Test(BaseTest):
    """
    import to local repo: no dst
    """
    fixtureDB = True
    fixtureCmds = [
    ]
    runCmd = "aptly repo import wheezy-contrib repo1 nginx"
    expectedCode = 1


class ImportRepo8Test(BaseTest):
    """
    import to local repo: no src
    """
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
    ]
    runCmd = "aptly repo import wheezy-main repo1 pyspi"
    expectedCode = 1


class ImportRepo9Test(BaseTest):
    """
    import to local repo: import with complex query
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly repo create -comment=Cool -distribution=squeeze repo1",
    ]
    runCmd = "aptly repo import wheezy-main repo1 '(httpd, $$Source (=nginx)) | exim4'"

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))
