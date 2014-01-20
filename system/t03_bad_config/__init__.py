"""
Start with bad config.
"""

import os
from lib import BaseTest


class Test(BaseTest):
    runCmd = "aptly"
    expectedCode = 1

    def prepare(self):
        self.prepare_remove_all()

        f = open(os.path.join(os.environ["HOME"], ".aptly.conf"), "w")
        f.write("{some crap")
        f.close()
