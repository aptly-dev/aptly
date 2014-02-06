from lib import BaseTest
import re


class ShowMirror1Test(BaseTest):
    """
    show mirror: regular mirror
    """
    fixtureCmds = ["aptly mirror create --ignore-signatures mirror1 http://mirror.yandex.ru/debian/ wheezy"]
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
    outputMatchPrepare = lambda _, s: re.sub(r"Last update: [0-9:+A-Za-z -]+\n", "", s)
