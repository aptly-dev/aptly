"""
Test aptly version
"""

from lib import BaseTest


class VersionTest(BaseTest):
    """
    version should match
    """

    runCmd = "aptly version"
