"""
Test library.
"""

import difflib
import inspect
import subprocess
import os
import shutil
import string

class BaseTest(object):
    """
    Base class for all tests.
    """

    expectedCode = 0

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
        f = open(os.path.join(os.environ["HOME"], ".aptly.conf"), "w")
        f.write(config_file)
        f.close()

    def run(self):
        try:
            proc = subprocess.Popen(self.runCmd.split(" "), stderr=subprocess.STDOUT, stdout=subprocess.PIPE)
            self.output, _ = proc.communicate()
            if proc.returncode != self.expectedCode:
                raise Exception("exit code %d != %d" % (proc.returncode, self.expectedCode))
        except Exception, e:
            raise Exception("Running command %s failed: %s" % (self.runCmd, str(e)))

    def gold_processor(self, gold):
        return gold

    def expand_environ(self, gold):
        return string.Template(gold).substitute(os.environ)

    def get_gold(self):
        gold = os.path.join(os.path.dirname(inspect.getsourcefile(self.__class__)), "gold")
        return self.gold_processor(open(gold, "r").read())

    def check_output(self):
        self.verify_match(self.get_gold(), self.output)

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

config_file = """
{
  "rootDir": "%s/.aptly",
  "downloadConcurrency": 4,
  "architectures": [],
  "dependencyFollowSuggests": false,
  "dependencyFollowRecommends": false,
  "dependencyFollowAllVariants": false
}
""" % (os.environ["HOME"])
