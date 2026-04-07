from lib import BaseTest
import uuid
import os

try:
    import requests

    if 'JFROG_URL' in os.environ and 'JFROG_USERNAME' in os.environ and \
       os.environ['JFROG_URL'] != "" and os.environ['JFROG_USERNAME'] != "":
        jfrog_ready = True
    else:
        print("JFrog tests disabled: JFrog creds not found in the environment (JFROG_URL, JFROG_USERNAME, JFROG_PASSWORD)")
        jfrog_ready = False
except ImportError as e:
    print("JFrog tests disabled: can't import requests", e)
    jfrog_ready = False


class JFrogTest(BaseTest):
    """
    BaseTest + support for JFrog
    """

    jfrogOverrides = {}

    def fixture_available(self):
        return super(JFrogTest, self).fixture_available() and jfrog_ready

    def prepare(self):
        self.repository = "aptly-sys-test-" + str(uuid.uuid1())
        self.jfrog_url = os.environ["JFROG_URL"]
        self.jfrog_username = os.environ["JFROG_USERNAME"]
        self.jfrog_password = os.environ["JFROG_PASSWORD"]

        # Create repository via REST API
        auth = (self.jfrog_username, self.jfrog_password)
        create_url = f"{self.jfrog_url}/api/repositories/{self.repository}"
        payload = {
            "key": self.repository,
            "rclass": "local",
            "packageType": "generic"
        }
        res = requests.put(create_url, json=payload, auth=auth)
        if res.status_code >= 400:
            raise Exception(f"Failed to create JFrog repository: {res.text}")

        self.configOverride = {"JFrogPublishEndpoints": {
            "test1": {
                "url": self.jfrog_url,
                "repository": self.repository,
                "username": self.jfrog_username,
                "password": self.jfrog_password
            }
        }}

        self.configOverride["JFrogPublishEndpoints"]["test1"].update(**self.jfrogOverrides)

        super(JFrogTest, self).prepare()

    def shutdown(self):
        if hasattr(self, "repository"):
            auth = (self.jfrog_username, self.jfrog_password)
            delete_url = f"{self.jfrog_url}/api/repositories/{self.repository}"
            requests.delete(delete_url, auth=auth)

        super(JFrogTest, self).shutdown()

    def check_path(self, path):
        if path.startswith("public/"):
            path = path[7:]

        # Check against JFrog Artifactory API
        auth = (self.jfrog_username, self.jfrog_password)
        check_url = f"{self.jfrog_url}/api/storage/{self.repository}/{path}"
        res = requests.head(check_url, auth=auth)
        if res.status_code == 200:
            return True
        return False

    def check_exists(self, path):
        if not self.check_path(path):
            raise Exception("path %s doesn't exist" % (path, ))

    def check_not_exists(self, path):
        if self.check_path(path):
            raise Exception("path %s exists" % (path, ))

    def read_file(self, path, mode=''):
        assert not mode
        if path.startswith("public/"):
            path = path[7:]

        auth = (self.jfrog_username, self.jfrog_password)
        get_url = f"{self.jfrog_url}/{self.repository}/{path}"
        res = requests.get(get_url, auth=auth)
        res.raise_for_status()
        return res.text
