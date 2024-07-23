"""
Test aptly version
"""

from lib import BaseTest


class VersionTest(BaseTest):
    """
    version should match
    """
    gold_processor = BaseTest.expand_environ

    runCmd = "aptly version"
