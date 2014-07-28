import re
from lib import BaseTest


class EditMirror1Test(BaseTest):
    """
    edit mirror: enable filter
    """
    fixtureDB = True
    runCmd = "aptly mirror edit -filter=nginx -filter-with-deps wheezy-main"

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
        "aptly mirror create -ignore-signatures -filter='nginx | Priority (required)' mirror5 http://security.debian.org/ wheezy/updates main",
    ]
    runCmd = "aptly mirror edit -filter= mirror5"

    def check(self):
        def removeDates(s):
            return re.sub(r"(Date|Valid-Until): [,0-9:+A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly mirror show mirror5", "mirror_show", match_prepare=removeDates)
