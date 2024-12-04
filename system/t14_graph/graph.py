"""
Test aptly graph
"""

import os
import re

from lib import BaseTest


class CreateGraphTest(BaseTest):
    """
    open graph in viewer
    """
    fixtureCmds = ["mkdir -p ../build", "ln -fs /bin/true ../build/xdg-open"]
    environmentOverride = {"PATH": os.environ["PATH"] + ":../build"}
    runCmd = "aptly graph"

    def outputMatchPrepare(self, s):
        return re.sub(r"[0-9]", "", s)

    def teardown(self):
        self.run_cmd(["rm", "-f", "../build/xdg-open"])


class CreateGraphOutputTest(BaseTest):
    """
    open graph in viewer
    """
    runCmd = "aptly graph -output /tmp/aptly-graph.png"

    def teardown(self):
        self.run_cmd(["rm", "-f", "/tmp/aptly-graph.png"])
