import requests_unixsocket
import time
import os
import urllib

from lib import BaseTest


class UnixSocketAPITest(BaseTest):
    aptly_server = None
    socket_path = "/tmp/_aptly_test.sock"
    base_url = ("unix://%s" % socket_path)

    def prepare(self):
        if self.aptly_server is None:
            self.aptly_server = self._start_process("aptly api serve -no-lock -listen=%s" % (self.base_url),)
            time.sleep(1)
        super(UnixSocketAPITest, self).prepare()

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
        r = session.get('http+unix://%s/api/version' % urllib.quote(UnixSocketAPITest.socket_path, safe=''))
        # Just needs to come back, we actually don't care much about the code.
        # Only needs to verify that the socket is actually responding.
        self.check_equal(r.json(), {'Version': os.environ['APTLY_VERSION']})
