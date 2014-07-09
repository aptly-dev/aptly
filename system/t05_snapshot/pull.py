from lib import BaseTest
import re


class PullSnapshot1Test(BaseTest):
    """
    pull snapshot: simple conditions
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-non-free",
    ]
    runCmd = "aptly snapshot pull snap1 snap2 snap3 mame unrar"
    outputMatchPrepare = lambda _, output: "\n".join(sorted(output.split("\n")))

    def check(self):
        def remove_created_at(s):
            return re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly snapshot show -with-packages snap3", "snapshot_show", match_prepare=remove_created_at)


class PullSnapshot2Test(BaseTest):
    """
    pull snapshot: play with versions
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-backports",
    ]
    runCmd = "aptly snapshot pull snap1 snap2 snap3 'rsyslog (>= 7.4.4)'"
    outputMatchPrepare = lambda _, output: "\n".join(sorted(output.split("\n")))

    def check(self):
        def remove_created_at(s):
            return re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly snapshot show -with-packages snap3", "snapshot_show", match_prepare=remove_created_at)


class PullSnapshot3Test(BaseTest):
    """
    pull snapshot: play with versions + no-deps
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-backports",
    ]
    runCmd = "aptly snapshot pull -no-deps snap1 snap2 snap3 'rsyslog (>= 7.4.4)'"
    outputMatchPrepare = lambda _, output: "\n".join(sorted(output.split("\n")))

    def check(self):
        def remove_created_at(s):
            return re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly snapshot show -with-packages snap3", "snapshot_show", match_prepare=remove_created_at)


class PullSnapshot4Test(BaseTest):
    """
    pull snapshot: dry-run
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-backports",
    ]
    runCmd = "aptly snapshot pull -dry-run snap1 snap2 snap3 'rsyslog (>= 7.4.4)'"
    outputMatchPrepare = lambda _, output: "\n".join(sorted(output.split("\n")))

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly snapshot list", "snapshot_list")


class PullSnapshot5Test(BaseTest):
    """
    pull snapshot: no such snapshot
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-backports",
    ]
    runCmd = "aptly snapshot pull snap1 snap-no snap3 'rsyslog (>= 7.4.4)'"
    expectedCode = 1


class PullSnapshot6Test(BaseTest):
    """
    pull snapshot: no such snapshot
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-backports",
    ]
    runCmd = "aptly snapshot pull snap-no snap2 snap3 'rsyslog (>= 7.4.4)'"
    expectedCode = 1


class PullSnapshot7Test(BaseTest):
    """
    pull snapshot: duplicate snapshot
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-backports",
    ]
    runCmd = "aptly snapshot pull snap1 snap2 snap1 'rsyslog (>= 7.4.4)'"
    expectedCode = 1
    outputMatchPrepare = lambda _, output: "\n".join(sorted(output.split("\n")))


class PullSnapshot8Test(BaseTest):
    """
    pull snapshot: missing dependencies
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-non-free",
    ]
    runCmd = "aptly snapshot pull snap1 snap2 snap3 lunar-landing 'mars-landing (>= 1.0)'"
    outputMatchPrepare = lambda _, output: "\n".join(sorted(output.split("\n")))

    def check(self):
        def remove_created_at(s):
            return re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly snapshot show --with-packages snap3", "snapshot_show", match_prepare=remove_created_at)


class PullSnapshot9Test(BaseTest):
    """
    pull snapshot: follow sources
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-backports-src",
    ]
    runCmd = "aptly -dep-follow-source snapshot pull snap1 snap2 snap3 'rsyslog (>= 7.4.4)'"

    def check(self):
        def remove_created_at(s):
            return re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly snapshot show --with-packages snap3", "snapshot_show", match_prepare=remove_created_at)


class PullSnapshot10Test(BaseTest):
    """
    pull snapshot: follow sources + replace sources
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main-src",
        "aptly snapshot create snap2 from mirror wheezy-backports-src",
    ]
    runCmd = "aptly -dep-follow-source snapshot pull snap1 snap2 snap3 'rsyslog (>= 7.4.4)'"

    def check(self):
        def remove_created_at(s):
            return re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly snapshot show --with-packages snap3", "snapshot_show", match_prepare=remove_created_at)


class PullSnapshot11Test(BaseTest):
    """
    pull snapshot: -no-remove
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-backports",
    ]
    runCmd = "aptly snapshot pull -no-remove snap1 snap2 snap3 'rsyslog (>= 7.4.4)'"
    outputMatchPrepare = lambda _, output: "\n".join(sorted(output.split("\n")))

    def check(self):
        def remove_created_at(s):
            return re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly snapshot show -with-packages snap3", "snapshot_show", match_prepare=remove_created_at)


class PullSnapshot12Test(BaseTest):
    """
    pull snapshot: latest version is pulled by default
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create empty empty",
        "aptly snapshot create sensu from mirror sensu",
    ]
    runCmd = "aptly snapshot pull -architectures=amd64,i386 empty sensu destination sensu"


class PullSnapshot13Test(BaseTest):
    """
    pull snapshot: pull all versions
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create empty empty",
        "aptly snapshot create sensu from mirror sensu",
    ]
    runCmd = "aptly snapshot pull -architectures=amd64,i386 -all-matches empty sensu destination sensu"


class PullSnapshot14Test(BaseTest):
    """
    pull snapshot: pull with query
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create empty empty",
        "aptly snapshot create sensu from mirror sensu",
    ]
    runCmd = "aptly snapshot pull -architectures=amd64,i386 -all-matches empty sensu destination 'sensu (>0.12)' 'sensu (<0.9.6)'"
