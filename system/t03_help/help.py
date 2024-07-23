"""
Test help screens
"""

import re
from lib import BaseTest


class MainTest(BaseTest):
    """
    main
    """
    expectedCode = 2
    runCmd = "aptly"

    def outputMatchPrepare(_, s):
        return re.sub(r'  -(cpuprofile|memprofile|memstats|meminterval)=.*\n', '', s, flags=re.MULTILINE)


class MirrorTest(BaseTest):
    """
    main
    """
    expectedCode = 2
    runCmd = "aptly mirror"


class MirrorCreateTest(BaseTest):
    """
    main
    """
    expectedCode = 2
    runCmd = "aptly mirror create"


class MainHelpTest(BaseTest):
    """
    main
    """
    runCmd = "aptly help"

    def outputMatchPrepare(_, s):
        return re.sub(r'  -(cpuprofile|memprofile|memstats|meminterval)=.*\n', '', s, flags=re.MULTILINE)


class MirrorHelpTest(BaseTest):
    """
    main
    """
    runCmd = "aptly help mirror"


class MirrorCreateHelpTest(BaseTest):
    """
    main
    """
    runCmd = "aptly help mirror create"


class WrongFlagTest(BaseTest):
    """
    main
    """
    expectedCode = 2
    runCmd = "aptly mirror create -fxz=sss"
