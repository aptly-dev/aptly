import re

from lib import BaseTest


class ShowMirror1Test(BaseTest):
    """
    show mirror: regular mirror
    """
    fixtureCmds = ["aptly mirror create --ignore-signatures mirror1 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch"]
    runCmd = "aptly mirror show mirror1"


class ShowMirror2Test(BaseTest):
    """
    show mirror: missing mirror
    """
    runCmd = "aptly mirror show mirror-xx"
    expectedCode = 1


class ShowMirror3Test(BaseTest):
    """
    show mirror: regular mirror with packages
    """
    fixtureDB = True
    runCmd = "aptly mirror show --with-packages wheezy-contrib"

    def outputMatchPrepare(self, s):
        return re.sub(r"Last update: [0-9:+A-Za-z -]+\n", "", s)


class ShowMirror4Test(BaseTest):
    """
    show mirror: mirror with filter
    """
    fixtureCmds = [
        "aptly mirror create -ignore-signatures -filter='nginx | Priority (required)' -filter-with-deps=true mirror4 http://repo.aptly.info/system-tests/archive.debian.org/debian-security/ stretch/updates main"
    ]
    runCmd = "aptly mirror show mirror4"

    def outputMatchPrepare(self, s):
        return re.sub(r"(Date): [,0-9:+A-Za-z -]+\n", "", s)


class ShowMirror5Test(BaseTest):
    """
    show mirror: regular mirror
    """
    fixtureCmds = ["aptly mirror create --ignore-signatures mirror1 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch"]
    runCmd = "aptly mirror show -json mirror1"

    def outputMatchPrepare(_, s):
        return re.sub(r'[ ]*"UUID": "[\w-]+",?\n', '', s)


class ShowMirror6Test(BaseTest):
    """
    show mirror: missing mirror
    """
    runCmd = "aptly mirror show -json mirror-xx"
    expectedCode = 1


class ShowMirror7Test(BaseTest):
    """
    show mirror: regular mirror with packages
    """
    fixtureDB = True
    runCmd = "aptly mirror show -json --with-packages wheezy-contrib"


class ShowMirror8Test(BaseTest):
    """
    show mirror: mirror with filter
    """
    fixtureCmds = [
        "aptly mirror create -ignore-signatures -filter='nginx | Priority (required)' -filter-with-deps=true mirror4 http://repo.aptly.info/system-tests/archive.debian.org/debian-security/ stretch/updates main"
    ]
    runCmd = "aptly mirror show -json mirror4"

    def outputMatchPrepare(self, s):
        s = re.sub(r'[ ]*"UUID": "[\w-]+",?\n', '', s)
        s = re.sub('"Date": .*', '"Date": "anytime",', s)
        s = re.sub('"Valid-Until": .*', '"Valid-Until": "anytime",', s)
        return s
