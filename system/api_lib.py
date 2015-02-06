from lib import BaseTest
import time
import json
import random
import string
import os
import inspect
import shutil

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

        if os.path.exists(os.path.join(os.environ["HOME"], ".aptly", "upload")):
            shutil.rmtree(os.path.join(os.environ["HOME"], ".aptly", "upload"))

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

    def put(self, uri, *args, **kwargs):
        if "json" in kwargs:
            kwargs["data"] = json.dumps(kwargs.pop("json"))
            if not "headers" in kwargs:
                kwargs["headers"] = {}
            kwargs["headers"]["Content-Type"] = "application/json"
        return requests.put("http://%s%s" % (self.base_url, uri), *args, **kwargs)

    def delete(self, uri, *args, **kwargs):
        if "json" in kwargs:
            kwargs["data"] = json.dumps(kwargs.pop("json"))
            if not "headers" in kwargs:
                kwargs["headers"] = {}
            kwargs["headers"]["Content-Type"] = "application/json"
        return requests.delete("http://%s%s" % (self.base_url, uri), *args, **kwargs)

    def upload(self, uri, *filenames, **kwargs):
        upload_name = kwargs.pop("upload_name", None)
        directory = kwargs.pop("directory", "files")
        assert kwargs == {}

        files = {}

        for filename in filenames:
            fp = open(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), directory, filename), "rb")
            if upload_name is not None:
                upload_filename = upload_name
            else:
                upload_filename = filename
            files[upload_filename] = (upload_filename, fp)

        return self.post(uri, files=files)

    @classmethod
    def shutdown_class(cls):
        if cls.aptly_server is not None:
            cls.aptly_server.terminate()
            cls.aptly_server.wait()
            cls.aptly_server = None

    def random_name(self):
        return ''.join(random.choice(string.ascii_letters + string.digits) for _ in range(15))
