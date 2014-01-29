"""
Test library.
"""

import difflib
import inspect
import json
import subprocess
import os
import shlex
import shutil
import string


class BaseTest(object):
    """
    Base class for all tests.
    """

    longTest = False
    fixturePool = False
    fixtureDB = False

    expectedCode = 0
    configFile = {
        "rootDir": "%s/.aptly" % os.environ["HOME"],
        "downloadConcurrency": 4,
        "architectures": [],
        "dependencyFollowSuggests": False,
        "dependencyFollowRecommends": False,
        "dependencyFollowAllVariants": False
    }
    configOverride = {}

    fixtureDBDir = os.path.join(os.environ["HOME"], "aptly-fixture-db")
    fixturePoolDir = os.path.join(os.environ["HOME"], "aptly-fixture-pool")

    outputMatchPrepare = None

    def test(self):
        self.prepare()
        self.run()
        self.check()

    def prepare_remove_all(self):
        if os.path.exists(os.path.join(os.environ["HOME"], ".aptly")):
            shutil.rmtree(os.path.join(os.environ["HOME"], ".aptly"))
        if os.path.exists(os.path.join(os.environ["HOME"], ".aptly.conf")):
            os.remove(os.path.join(os.environ["HOME"], ".aptly.conf"))

    def prepare_default_config(self):
        cfg = self.configFile.copy()
        cfg.update(**self.configOverride)
        f = open(os.path.join(os.environ["HOME"], ".aptly.conf"), "w")
        f.write(json.dumps(cfg))
        f.close()

    def fixture_available(self):
        if self.fixturePool and not os.path.exists(self.fixturePoolDir):
            return False
        if self.fixtureDB and not os.path.exists(self.fixtureDBDir):
            return False

        return True

    def prepare_fixture(self):
        if self.fixturePool:
            os.makedirs(os.path.join(os.environ["HOME"], ".aptly"), 0755)
            os.symlink(self.fixturePoolDir, os.path.join(os.environ["HOME"], ".aptly", "pool"))

        if self.fixtureDB:
            shutil.copytree(self.fixtureDBDir, os.path.join(os.environ["HOME"], ".aptly", "db"))

        if hasattr(self, "fixtureCmds"):
            for cmd in self.fixtureCmds:
                self.run_cmd(cmd)

    def run(self):
        self.output = self.output_processor(self.run_cmd(self.runCmd, self.expectedCode))

    def run_cmd(self, command, expected_code=0):
        try:
            proc = subprocess.Popen(shlex.split(command), stderr=subprocess.STDOUT, stdout=subprocess.PIPE)
            output, _ = proc.communicate()
            if proc.returncode != expected_code:
                raise Exception("exit code %d != %d (output: %s)" % (proc.returncode, expected_code, output))
            return output
        except Exception, e:
            raise Exception("Running command %s failed: %s" % (command, str(e)))

    def gold_processor(self, gold):
        return gold

    def output_processor(self, output):
        return output

    def expand_environ(self, gold):
        return string.Template(gold).substitute(os.environ)

    def get_gold(self, gold_name="gold"):
        gold = os.path.join(os.path.dirname(inspect.getsourcefile(self.__class__)), self.__class__.__name__ + "_" + gold_name)
        return self.gold_processor(open(gold, "r").read())

    def check_output(self):
        self.verify_match(self.get_gold(), self.output, match_prepare=self.outputMatchPrepare)

    def check_cmd_output(self, command, gold_name, match_prepare=None, expected_code=0):
        self.verify_match(self.get_gold(gold_name), self.run_cmd(command, expected_code=expected_code), match_prepare)

    def verify_match(self, a, b, match_prepare=None):
        if match_prepare is not None:
            a = match_prepare(a)
            b = match_prepare(b)

        if a != b:
            diff = "".join(difflib.unified_diff([l + "\n" for l in a.split("\n")], [l + "\n" for l in b.split("\n")]))

            raise Exception("content doesn't match:\n" + diff + "\n")

    def check_file(self):
        self.verify_match(self.get_gold(), open(self.checkedFile, "r").read())

    check = check_output

    def prepare(self):
        self.prepare_remove_all()
        self.prepare_default_config()
        self.prepare_fixture()
