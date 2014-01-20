"""
Test config file generation.
"""

import os
from lib import BaseTest


class Test(BaseTest):
    runCmd = "aptly"
    checkedFile = os.path.join(os.environ["HOME"], ".aptly.conf")

    check = BaseTest.check_file
    gold_processor = BaseTest.expand_environ
    prepare = BaseTest.prepare_remove_all
