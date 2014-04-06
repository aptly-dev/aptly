"""
Test config file
"""

import os
import re
import inspect
from lib import BaseTest


class CreateConfigTest(BaseTest):
    """
    new file is generated if missing
    """
    runCmd = "aptly mirror list"
    checkedFile = os.path.join(os.environ["HOME"], ".aptly.conf")

    check = BaseTest.check_file
    gold_processor = BaseTest.expand_environ
    prepare = BaseTest.prepare_remove_all


class BadConfigTest(BaseTest):
    """
    broken config file
    """
    runCmd = "aptly mirror list"
    expectedCode = 1

    gold_processor = BaseTest.expand_environ

    def prepare(self):
        self.prepare_remove_all()

        f = open(os.path.join(os.environ["HOME"], ".aptly.conf"), "w")
        f.write("{some crap")
        f.close()


class ConfigInFileTest(BaseTest):
    """
    config in other file test
    """
    runCmd = ["aptly", "mirror", "list",
              "-config=%s" % (os.path.join(os.path.dirname(inspect.getsourcefile(BadConfigTest)), "aptly.conf"), )]
    prepare = BaseTest.prepare_remove_all

    outputMatchPrepare = lambda _, s: re.sub(r'  -(cpuprofile|memprofile|memstats|meminterval)=.*\n', '', s, flags=re.MULTILINE)


class ConfigInMissingFileTest(BaseTest):
    """
    config in other file test
    """
    runCmd = ["aptly", "mirror", "list", "-config=nosuchfile.conf"]
    expectedCode = 1
    prepare = BaseTest.prepare_remove_all
