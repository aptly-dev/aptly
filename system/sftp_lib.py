from lib import BaseTest
import uuid
import os
import shutil
import signal
import socket
import subprocess
import tempfile

class SFTPTest(BaseTest):
    """
    BaseTest + support for SFTP
    """

    def fixture_available(self):
        return super(SFTPTest, self).fixture_available()

    def prepare(self):
        print " +++++  prepare!    ++++++"
        self.tmpdir = tempfile.mkdtemp()

        # Grab a port, then free it and hope it's still fine by the time server
        # attempts to grab it again.
        sock = socket.socket()
        sock.bind(('', 0))
        port = sock.getsockname()[1]
        sock.close()
        # FIXME: keys are somewhat hardcoded both in the test and the
        # actual code as in both cases we assume them to be ~/.ssh/id_rsa
        key = os.path.join(os.environ["HOME"], ".ssh/id_rsa")
        self.proc = subprocess.Popen(
            ["sftpserver", "-p", str(port), "-l", "INFO", "-k", key],
            preexec_fn=os.setsid,
            cwd=self.tmpdir)

        self.configOverride = {"SFTPPublishEndpoints": {
            "test1": {
                "uri": "sftp://user:asdf@localhost:" + str(port),
            }
        }}

        super(SFTPTest, self).prepare()

    def shutdown(self):
        print " ------ shtudown 0--------"
        if hasattr(self, "proc"):
            os.killpg(os.getpgid(self.proc.pid), signal.SIGTERM)  # Send the signal to all the process groups
        shutil.rmtree(self.tmpdir)
        super(SFTPTest, self).shutdown()

    def check_exists(self, path):
        if not os.path.exists(os.path.join(self.tmpdir, path)):
            raise Exception("path %s doesn't exist" % (path, ))

    def check_not_exists(self, path):
        if os.path.exists(os.path.join(self.tmpdir, path)):
            raise Exception("path %s exists" % (path, ))

    def read_file(self, path):
        with open(os.path.join(self.tmpdir, path), "r") as f:
            return f.read()
