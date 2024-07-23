import requests_unixsocket
import time
import os
import urllib.error
import urllib.parse
import urllib.request

from lib import BaseTest
from testout import TestOut


class UnixSocketAPITest(BaseTest):
    aptly_server = None
    socket_path = "/tmp/_aptly_test.sock"
    base_url = ("unix://%s" % socket_path)
    aptly_out = None
    debugOutput = True

    def prepare(self):
        if self.aptly_server is None:
            UnixSocketAPITest.aptly_out = TestOut()
            self.aptly_server = self._start_process("aptly api serve -no-lock -listen=%s" % (self.base_url), stdout=UnixSocketAPITest.aptly_out, stderr=UnixSocketAPITest.aptly_out)
            time.sleep(1)
        else:
            UnixSocketAPITest.aptly_out.clear()
        super(UnixSocketAPITest, self).prepare()

    def debug_output(self):
        return UnixSocketAPITest.aptly_out.get_contents()

    def shutdown(self):
        if self.aptly_server is not None:
            self.aptly_server.terminate()
            self.aptly_server.wait()
            self.aptly_server = None
        super(UnixSocketAPITest, self).shutdown()

    def run(self):
        pass

    """
    Verify we can listen on a unix domain socket.
    """
    def check(self):
        session = requests_unixsocket.Session()
        r = session.get('http+unix://%s/api/version' % urllib.parse.quote(UnixSocketAPITest.socket_path, safe=''))
        # Just needs to come back, we actually don't care much about the code.
        # Only needs to verify that the socket is actually responding.
        self.check_equal(r.json(), {'Version': os.environ['APTLY_VERSION']})
