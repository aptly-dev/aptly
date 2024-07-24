import inspect
import json
import os
import random
import shutil
import string
import time

from lib import BaseTest
from testout import TestOut

try:
    import requests
except ImportError:
    requests = None

# States for returned tasks from the API
TASK_SUCCEEDED = 2
TASK_FAILED = 3


class APITest(BaseTest):
    """
    BaseTest + testing aptly API
    """
    aptly_server = None
    aptly_out = None
    debugOutput = True
    base_url = "127.0.0.1:8765"
    configOverride = {
        "FileSystemPublishEndpoints": {
            "apiandserve": {
                "rootDir": f"{os.environ['HOME']}/{BaseTest.aptlyDir}/apiandserve",
                "linkMethod": "symlink"
            }
        }
    }

    def fixture_available(self):
        return super(APITest, self).fixture_available() and requests is not None

    def prepare(self):
        if APITest.aptly_server is None:
            super(APITest, self).prepare()

            APITest.aptly_out = TestOut()

            configPath = os.path.join(os.environ["HOME"], self.aptlyConfigFile)
            APITest.aptly_server = self._start_process(f"aptly api serve -no-lock -config={configPath} -listen={self.base_url}", stdout=APITest.aptly_out, stderr=APITest.aptly_out)
            time.sleep(1)
        else:
            APITest.aptly_out.clear()

        if os.path.exists(os.path.join(os.environ["HOME"], self.aptlyDir, "upload")):
            shutil.rmtree(os.path.join(os.environ["HOME"], self.aptlyDir, "upload"))

    def debug_output(self):
        return APITest.aptly_out.get_contents()

    def run(self):
        pass

    def get(self, uri, *args, **kwargs):
        return requests.get("http://%s%s" % (self.base_url, uri), *args, **kwargs)

    def post(self, uri, *args, **kwargs):
        if "json" in kwargs:
            kwargs["data"] = json.dumps(kwargs.pop("json"))
            if "headers" not in kwargs:
                kwargs["headers"] = {}
            kwargs["headers"]["Content-Type"] = "application/json"
        return requests.post("http://%s%s" % (self.base_url, uri), *args, **kwargs)

    def _ensure_async(self, kwargs):
        # Make sure we run this as an async task
        params = kwargs.get('params', {})
        params.setdefault('_async', True)
        kwargs['params'] = params

    def post_task(self, uri, *args, **kwargs):
        self._ensure_async(kwargs)
        resp = self.post(uri, *args, **kwargs)
        if resp.status_code != 202:
            return resp

        _id = resp.json()['ID']
        resp = self.get("/api/tasks/" + str(_id) + "/wait")
        self.check_equal(resp.status_code, 200)

        return self.get("/api/tasks/" + str(_id))

    def check_task(self, task):
        self.check_equal(task.status_code, 200)
        if task.json()['State'] != TASK_SUCCEEDED:
            resp2 = self.get("/api/tasks/" + str(task.json()['ID']) + "/output")
            raise Exception(f"task failed: {resp2.text}")

    def check_task_fail(self, task, expected_output=None):
        self.check_equal(task.status_code, 200)
        if task.json()['State'] == TASK_SUCCEEDED:
            raise Exception("task expected to fail")
        if expected_output:
            resp = self.get("/api/tasks/" + str(task.json()['ID']) + "/output")
            self.check_equal(resp.json(), expected_output)

    def put(self, uri, *args, **kwargs):
        if "json" in kwargs:
            kwargs["data"] = json.dumps(kwargs.pop("json"))
            if "headers" not in kwargs:
                kwargs["headers"] = {}
            kwargs["headers"]["Content-Type"] = "application/json"
        return requests.put("http://%s%s" % (self.base_url, uri), *args, **kwargs)

    def put_task(self, uri, *args, **kwargs):
        self._ensure_async(kwargs)
        resp = self.put(uri, *args, **kwargs)
        if resp.status_code != 202:
            return resp

        _id = resp.json()['ID']
        resp = self.get("/api/tasks/" + str(_id) + "/wait")
        self.check_equal(resp.status_code, 200)

        return self.get("/api/tasks/" + str(_id))

    def delete(self, uri, *args, **kwargs):
        if "json" in kwargs:
            kwargs["data"] = json.dumps(kwargs.pop("json"))
            if "headers" not in kwargs:
                kwargs["headers"] = {}
            kwargs["headers"]["Content-Type"] = "application/json"
        return requests.delete("http://%s%s" % (self.base_url, uri), *args, **kwargs)

    def delete_task(self, uri, *args, **kwargs):
        self._ensure_async(kwargs)
        resp = self.delete(uri, *args, **kwargs)
        if resp.status_code != 202:
            return resp

        _id = resp.json()['ID']
        resp = self.get("/api/tasks/" + str(_id) + "/wait")
        self.check_equal(resp.status_code, 200)

        return self.get("/api/tasks/" + str(_id))

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
        if APITest.aptly_out is not None:
            APITest.aptly_out.close()

    def random_name(self):
        return ''.join(random.choice(string.ascii_letters + string.digits) for _ in range(15))
