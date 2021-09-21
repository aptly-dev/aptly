"""
Testing serving public repo
"""

import httplib
import os
import signal
import subprocess
import shlex
import time

from lib import BaseTest


class RootDirInaccessible(BaseTest):
    """
    serve command aborts if rootDir is inaccessible
    """
    skipTest = 'User is root' if os.environ['USER'] == 'root' else False
    fixtureDB = False
    fixturePool = False

    configOverride = {
        "rootDir": "/root"  # any directory that exists but is not writable
    }

    runCmd = "aptly serve -listen=127.0.0.1:8765"
    expectedCode = 1


class Serve1Test(BaseTest):
    """
    serve public: two publishes, verify HTTP
    """
    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly snapshot create snap2 from mirror gnuplot-maverick-src",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap1",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec snap2 debian",
        "aptly publish snapshot -keyring=${files}/aptly.pub -secret-keyring=${files}/aptly.sec -component=main,contrib snap1 snap2 multi",
    ]
    runCmd = "aptly serve -listen=127.0.0.1:8765"

    def run(self):
        try:
            proc = subprocess.Popen(shlex.split(self.runCmd), stderr=subprocess.STDOUT, stdout=subprocess.PIPE, bufsize=0)

            try:
                time.sleep(1)

                conn = httplib.HTTPConnection("127.0.0.1", 8765)
                conn.request("GET", "/")
                r = conn.getresponse()
                if r.status != 200:
                    raise Exception("Expected status 200 != %d" % r.status)
                self.http_response = r.read()

                output = os.read(proc.stdout.fileno(), 8192)

            finally:
                proc.send_signal(signal.SIGINT)
                proc.wait()

            if proc.returncode != -2 and proc.returncode != 2:
                raise Exception("exit code %d != %d (output: %s)" % (proc.returncode, -2, output))
            self.output = output
        except Exception, e:
            raise Exception("Running command %s failed: %s" % (self.runCmd, str(e)))

    def check(self):
        self.check_output()
        self.verify_match(self.get_gold('http'), self.http_response, match_prepare=lambda s: "\n".join(sorted(s.split("\n"))))


class Serve2Test(BaseTest):
    """
    serve public: no publishes
    """
    runCmd = "aptly serve -listen=127.0.0.1:8765"
