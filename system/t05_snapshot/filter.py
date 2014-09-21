from lib import BaseTest
import re


class FilterSnapshot1Test(BaseTest):
    """
    filter snapshot: simple filter
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-non-free",
    ]
    runCmd = "aptly snapshot filter snap1 snap2 mame unrar"

    def check(self):
        def remove_created_at(s):
            return re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly snapshot show -with-packages snap2", "snapshot_show", match_prepare=remove_created_at)


class FilterSnapshot2Test(BaseTest):
    """
    filter snapshot: play with versions
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-backports",
    ]
    runCmd = "aptly snapshot filter -with-deps snap1 snap2 'rsyslog (>= 7.4.4)'"

    def check(self):
        def remove_created_at(s):
            return re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly snapshot show -with-packages snap2", "snapshot_show", match_prepare=remove_created_at)


class FilterSnapshot3Test(BaseTest):
    """
    filter snapshot: complex condition
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
    ]
    runCmd = "aptly snapshot filter snap1 snap2 'Priority (required)' nginx xyz"

    def check(self):
        def remove_created_at(s):
            return re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly snapshot show -with-packages snap2", "snapshot_show", match_prepare=remove_created_at)


class FilterSnapshot5Test(BaseTest):
    """
    filter snapshot: no such snapshot
    """
    fixtureDB = True
    fixtureCmds = [
    ]
    runCmd = "aptly snapshot filter snap1 snap2 'rsyslog (>= 7.4.4)'"
    expectedCode = 1


class FilterSnapshot6Test(BaseTest):
    """
    filter snapshot: duplicate snapshot
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot merge snap2 snap1",
    ]
    runCmd = "aptly snapshot filter snap1 snap2 'rsyslog (>= 7.4.4)'"
    expectedCode = 1


class FilterSnapshot7Test(BaseTest):
    """
    filter snapshot: follow sources
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-backports-src",
    ]
    runCmd = "aptly -dep-follow-source snapshot filter -with-deps snap1 snap2 'rsyslog (>= 7.4.4), $$Architecture (i386)'"

    def check(self):
        def remove_created_at(s):
            return re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly snapshot show --with-packages snap2", "snapshot_show", match_prepare=remove_created_at)
