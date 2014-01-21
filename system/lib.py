"""
Test library.
"""

import difflib
import inspect
import json
import subprocess
import os
import shutil
import string


class BaseTest(object):
    """
    Base class for all tests.
    """

    longTest = False
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

    def prepare_fixture(self):
        if hasattr(self, "fixtureCmds"):
            for cmd in self.fixtureCmds:
                self.run_cmd(cmd)

    def run(self):
        self.output = self.output_processor(self.run_cmd(self.runCmd, self.expectedCode))

    def run_cmd(self, command, expected_code=0):
        try:
            proc = subprocess.Popen(command.split(" "), stderr=subprocess.STDOUT, stdout=subprocess.PIPE)
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
        self.verify_match(self.get_gold(), self.output)

    def check_cmd_output(self, command, gold_name):
        self.verify_match(self.get_gold(gold_name), self.run_cmd(command))

    def verify_match(self, a, b):
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

