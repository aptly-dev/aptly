"""
Test library.
"""

import difflib
import inspect
import json
import subprocess
import os
import posixpath
import shlex
import shutil
import string
import threading
import urllib
#import time
import SocketServer
import SimpleHTTPServer


class ThreadedTCPServer(SocketServer.ThreadingMixIn, SocketServer.TCPServer):
    pass


class FileHTTPServerRequestHandler(SimpleHTTPServer.SimpleHTTPRequestHandler):
    def translate_path(self, path):
        """Translate a /-separated PATH to the local filename syntax.

        Components that mean special things to the local file system
        (e.g. drive or directory names) are ignored.  (XXX They should
        probably be diagnosed.)

        """
        # abandon query parameters
        path = path.split('?', 1)[0]
        path = path.split('#', 1)[0]
        path = posixpath.normpath(urllib.unquote(path))
        words = path.split('/')
        words = filter(None, words)
        path = self.rootPath
        for word in words:
            drive, word = os.path.splitdrive(word)
            head, word = os.path.split(word)
            if word in (os.curdir, os.pardir):
                continue
            path = os.path.join(path, word)
        return path

    def log_message(self, format, *args):
        pass


class BaseTest(object):
    """
    Base class for all tests.
    """

    longTest = False
    fixturePool = False
    fixturePoolCopy = False
    fixtureDB = False
    fixtureGpg = False
    fixtureWebServer = False

    expectedCode = 0
    configFile = {
        "rootDir": "%s/.aptly" % os.environ["HOME"],
        "downloadConcurrency": 4,
        "architectures": [],
        "dependencyFollowSuggests": False,
        "dependencyFollowRecommends": False,
        "dependencyFollowAllVariants": False,
        "dependencyFollowSource": False,
        "gpgDisableVerify": False,
        "gpgDisableSign": False,
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
        if os.path.exists(os.path.join(os.environ["HOME"], ".gnupg", "aptlytest.gpg")):
            os.remove(os.path.join(os.environ["HOME"], ".gnupg", "aptlytest.gpg"))

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
            #start = time.time()
            os.makedirs(os.path.join(os.environ["HOME"], ".aptly"), 0755)
            os.symlink(self.fixturePoolDir, os.path.join(os.environ["HOME"], ".aptly", "pool"))
            #print "FIXTURE POOL: %.2f" % (time.time()-start)

        if self.fixturePoolCopy:
            os.makedirs(os.path.join(os.environ["HOME"], ".aptly"), 0755)
            shutil.copytree(self.fixturePoolDir, os.path.join(os.environ["HOME"], ".aptly", "pool"), ignore=shutil.ignore_patterns(".git"))

        if self.fixtureDB:
            #start = time.time()
            shutil.copytree(self.fixtureDBDir, os.path.join(os.environ["HOME"], ".aptly", "db"))
            #print "FIXTURE DB: %.2f" % (time.time()-start)

        if self.fixtureWebServer:
            self.webServerUrl = self.start_webserver(os.path.join(os.path.dirname(inspect.getsourcefile(self.__class__)),
                                                     self.fixtureWebServer))

        if self.fixtureGpg:
            self.run_cmd(["gpg", "--no-default-keyring", "--trust-model", "always", "--batch", "--keyring", "aptlytest.gpg", "--import",
                          os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "debian-archive-keyring.gpg"),
                          os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "launchpad.key"),
                          os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "flat.key")])

        if hasattr(self, "fixtureCmds"):
            for cmd in self.fixtureCmds:
                self.run_cmd(cmd)

    def run(self):
        self.output = self.output_processor(self.run_cmd(self.runCmd, self.expectedCode))

    def run_cmd(self, command, expected_code=0):
        try:
            #start = time.time()
            if not hasattr(command, "__iter__"):
                params = {
                    'files': os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files"),
                    'testfiles': os.path.join(os.path.dirname(inspect.getsourcefile(self.__class__)), self.__class__.__name__),
                }
                if self.fixtureWebServer:
                    params['url'] = self.webServerUrl

                command = string.Template(command).substitute(params)

                command = shlex.split(command)
            environ = os.environ.copy()
            environ["LC_ALL"] = "C"
            proc = subprocess.Popen(command, stderr=subprocess.STDOUT, stdout=subprocess.PIPE, env=environ)
            output, _ = proc.communicate()
            #print "CMD %s: %.2f" % (" ".join(command), time.time()-start)
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

    def read_file(self, path):
        with open(os.path.join(os.environ["HOME"], ".aptly", path), "r") as f:
            return f.read()

    def check_file_contents(self, path, gold_name, match_prepare=None):
        contents = self.read_file(path)

        self.verify_match(self.get_gold(gold_name), contents, match_prepare=match_prepare)

    def check_file(self):
        self.verify_match(self.get_gold(), open(self.checkedFile, "r").read())

    def check_exists(self, path):
        if not os.path.exists(os.path.join(os.environ["HOME"], ".aptly", path)):
            raise Exception("path %s doesn't exist" % (path, ))

    def check_not_exists(self, path):
        if os.path.exists(os.path.join(os.environ["HOME"], ".aptly", path)):
            raise Exception("path %s exists" % (path, ))

    def verify_match(self, a, b, match_prepare=None):
        if match_prepare is not None:
            a = match_prepare(a)
            b = match_prepare(b)

        if a != b:
            diff = "".join(difflib.unified_diff([l + "\n" for l in a.split("\n")], [l + "\n" for l in b.split("\n")]))

            raise Exception("content doesn't match:\n" + diff + "\n")

    check = check_output

    def prepare(self):
        self.prepare_remove_all()
        self.prepare_default_config()
        self.prepare_fixture()

    def start_webserver(self, directory):
        FileHTTPServerRequestHandler.rootPath = directory
        self.webserver = ThreadedTCPServer(("localhost", 0), FileHTTPServerRequestHandler)

        server_thread = threading.Thread(target=self.webserver.serve_forever)
        server_thread.daemon = True
        server_thread.start()

        return "http://%s:%d/" % self.webserver.server_address

    def shutdown(self):
        if hasattr(self, 'webserver'):
            self.shutdown_webserver()

    def shutdown_webserver(self):
        self.webserver.shutdown()
