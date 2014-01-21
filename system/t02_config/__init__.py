"""
Test config file
"""

import os
from lib import BaseTest


class CreateConfigTest(BaseTest):
    """
    new file is generated if missing
    """
    runCmd = "aptly"
    checkedFile = os.path.join(os.environ["HOME"], ".aptly.conf")

    check = BaseTest.check_file
    gold_processor = BaseTest.expand_environ
    prepare = BaseTest.prepare_remove_all


class BadConfigTest(BaseTest):
    """
    broken config file
    """
    runCmd = "aptly"
    expectedCode = 1

    def prepare(self):
        self.prepare_remove_all()

        f = open(os.path.join(os.environ["HOME"], ".aptly.conf"), "w")
        f.write("{some crap")
        f.close()
