"""
Test help screens
"""

from lib import BaseTest


class MainTest(BaseTest):
    """
    main
    """
    runCmd = "aptly"


class MirrorTest(BaseTest):
    """
    main
    """
    runCmd = "aptly mirror"


class MirrorCreateTest(BaseTest):
    """
    main
    """
    runCmd = "aptly mirror create"


class MainHelpTest(BaseTest):
    """
    main
    """
    runCmd = "aptly help"


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
