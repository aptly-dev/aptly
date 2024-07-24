import inspect
import os
import subprocess
import tempfile

from api_lib import APITest


def check_gpgkey_exists(gpg_key, keyring):
    p = subprocess.Popen([
        "gpg", "--no-default-keyring",
        "--keyring", keyring,
        "--fingerprint", gpg_key],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    p.communicate()
    if p.returncode != 0:
        raise Exception("gpg key does not exists")


class GPGAPITestAddKey(APITest):
    """
    POST /gpg/key
    """
    requiresGPG2 = True

    def check(self):
        with tempfile.NamedTemporaryFile(suffix=".pub") as keyring:
            gpgkeyid = "9E3E53F19C7DE460"
            resp = self.post("/api/gpg/key", json={
                "Keyserver": "keyserver.ubuntu.com",
                "Keyring": keyring.name,
                "GpgKeyID": gpgkeyid
            })

            self.check_equal(resp.status_code, 200)
            check_gpgkey_exists(gpgkeyid, keyring.name)


class GPGAPITestAddKeyArmor(APITest):
    """
    POST /gpg/key
    """
    def check(self):
        keyfile = os.path.join(os.path.dirname(inspect.getsourcefile(APITest)),
                               "files") + "/launchpad.key"
        gpgkeyid = "3B1F56C0"

        with open(keyfile, 'r') as keyf:
            gpgkeyarmor = keyf.read()

        with tempfile.NamedTemporaryFile(suffix=".pub") as keyring:
            resp = self.post("/api/gpg/key", json={
                "Keyring": keyring.name,
                "GpgKeyArmor": gpgkeyarmor
            })

            self.check_equal(resp.status_code, 200)
            check_gpgkey_exists(gpgkeyid, keyring.name)
