import re

from lib import BaseTest


class EditMirror1Test(BaseTest):
    """
    edit mirror: enable filter & download sources
    """
    fixtureDB = True
    runCmd = "aptly mirror edit -filter=nginx -filter-with-deps -with-sources wheezy-main"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show wheezy-main", "mirror_show", match_prepare=lambda s: re.sub(r"Last update: [0-9:+A-Za-z -]+\n", "", s))


class EditMirror2Test(BaseTest):
    """
    edit mirror: missing mirror
    """
    runCmd = "aptly mirror edit wheezy-main"
    expectedCode = 1


class EditMirror3Test(BaseTest):
    """
    edit mirror: no changes
    """
    fixtureDB = True
    runCmd = "aptly mirror edit wheezy-main"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show wheezy-main", "mirror_show", match_prepare=lambda s: re.sub(r"Last update: [0-9:+A-Za-z -]+\n", "", s))


class EditMirror4Test(BaseTest):
    """
    edit mirror: wrong query
    """
    fixtureDB = True
    runCmd = "aptly mirror edit -filter=| wheezy-main"
    expectedCode = 1


class EditMirror5Test(BaseTest):
    """
    edit mirror: remove filter
    """
    fixtureCmds = [
        "aptly mirror create -ignore-signatures -filter='nginx | Priority (required)' mirror5 http://repo.aptly.info/system-tests/archive.debian.org/debian-security/ stretch/updates main",
    ]
    runCmd = "aptly mirror edit -filter= mirror5"

    def check(self):
        def removeDates(s):
            return re.sub(r"(Date|Valid-Until): [,0-9:+A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly mirror show mirror5", "mirror_show", match_prepare=removeDates)


class EditMirror6Test(BaseTest):
    """
    edit mirror: change architectures
    """
    fixtureCmds = [
        "aptly mirror create -ignore-signatures -architectures=amd64 mirror6 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian stretch main"
    ]
    runCmd = "aptly mirror edit -ignore-signatures -architectures=amd64,i386 mirror6"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror6", "mirror_show", match_prepare=lambda s: re.sub(r"Last update: [0-9:+A-Za-z -]+\n", "", s))


class EditMirror7Test(BaseTest):
    """
    edit mirror: change architectures to missing archs
    """
    fixtureCmds = [
        "aptly mirror create -ignore-signatures -architectures=amd64 stretch http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian stretch main"
    ]
    runCmd = "aptly mirror edit -ignore-signatures -architectures=amd64,x56 stretch"
    expectedCode = 1


class EditMirror8Test(BaseTest):
    """
    edit mirror: enable udebs
    """
    fixtureDB = True
    runCmd = "aptly mirror edit -with-udebs wheezy-main"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show wheezy-main", "mirror_show", match_prepare=lambda s: re.sub(r"Last update: [0-9:+A-Za-z -]+\n", "", s))


class EditMirror9Test(BaseTest):
    """
    edit mirror: flat mirror with udebs
    """
    fixtureCmds = ["aptly mirror create -keyring=aptlytest.gpg mirror9 http://repo.aptly.info/system-tests/pkg.jenkins.io/debian-stable binary/"]
    fixtureGpg = True
    runCmd = "aptly mirror edit -with-udebs mirror9"
    expectedCode = 1


class EditMirror10Test(BaseTest):
    """
    edit mirror: change archive url
    """
    requiresFTP = True
    fixtureCmds = ["aptly mirror create -ignore-signatures mirror10 http://repo.aptly.info/system-tests/ftp.ru.debian.org/debian bookworm main"]
    runCmd = "aptly mirror edit -ignore-signatures -archive-url http://repo.aptly.info/system-tests/ftp.ch.debian.org/debian mirror10"
