from lib import BaseTest
import time
import json
import random
import string

try:
    import requests
except ImportError:
    requests = None


class APITest(BaseTest):
    """
    BaseTest + testing aptly API
    """
    aptly_server = None
    base_url = "127.0.0.1:8765"

    def fixture_available(self):
        return super(APITest, self).fixture_available() and requests is not None

    def prepare(self):
        if APITest.aptly_server is None:
            super(APITest, self).prepare()

            APITest.aptly_server = self._start_process("aptly api serve -listen=%s" % (self.base_url),)
            time.sleep(1)

    def run(self):
        pass

    def get(self, uri, *args, **kwargs):
        return requests.get("http://%s%s" % (self.base_url, uri), *args, **kwargs)

    def post(self, uri, *args, **kwargs):
        if "json" in kwargs:
            kwargs["data"] = json.dumps(kwargs.pop("json"))
            if not "headers" in kwargs:
                kwargs["headers"] = {}
            kwargs["headers"]["Content-Type"] = "application/json"
        return requests.post("http://%s%s" % (self.base_url, uri), *args, **kwargs)

    @classmethod
    def shutdown_class(cls):
        if cls.aptly_server is not None:
            cls.aptly_server.terminate()
            cls.aptly_server.wait()
            cls.aptly_server = None

    def random_name(self):
        return ''.join(random.choice(string.ascii_letters + string.digits) for _ in range(15))
