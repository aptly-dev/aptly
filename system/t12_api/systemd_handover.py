import requests_unixsocket
import urllib
import os
import os.path

from lib import BaseTest


class SystemdAPIHandoverTest(BaseTest):
    aptly_server = None
    socket_path = "/tmp/_aptly_systemdapihandovertest.sock"

    def prepare(self):
        # On Debian they use /lib on other systems /usr/lib.
        systemd_activate = "/usr/lib/systemd/systemd-activate"
        if not os.path.exists(systemd_activate):
            systemd_activate = "/lib/systemd/systemd-activate"
        if not os.path.exists(systemd_activate):
            print("Could not find systemd-activate")
            return
        self.aptly_server = self._start_process("%s -l %s aptly api serve -no-lock" %
                                                (systemd_activate, self.socket_path),)
        super(SystemdAPIHandoverTest, self).prepare()

    def shutdown(self):
        if self.aptly_server is not None:
            self.aptly_server.terminate()
            self.aptly_server.wait()
            self.aptly_server = None
        if os.path.exists(self.socket_path):
            os.remove(self.socket_path)
        super(SystemdAPIHandoverTest, self).shutdown()

    def run(self):
        pass

    """
    Verify we can listen on a unix domain socket.
    """
    def check(self):
        if self.aptly_server is None:
            print("Skipping test as we failed to setup a listener.")
            return
        session = requests_unixsocket.Session()
        r = session.get('http+unix://%s/api/version' % urllib.quote(self.socket_path, safe=''))
        self.check_equal(r.json(), {'Version': os.environ['APTLY_VERSION']})
