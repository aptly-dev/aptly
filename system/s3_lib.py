from lib import BaseTest
import uuid
import os

try:
    import boto

    if 'AWS_SECRET_ACCESS_KEY' in os.environ and 'AWS_ACCESS_KEY_ID' in os.environ:
        s3_conn = boto.connect_s3()
    else:
        s3_conn = None
except ImportError:
    s3_conn = None


class S3Test(BaseTest):
    """
    BaseTest + support for S3
    """

    def fixture_available(self):
        return super(S3Test, self).fixture_available() and s3_conn is not None

    def prepare(self):
        self.bucket_name = "aptly-sys-test-" + str(uuid.uuid1())
        self.bucket = s3_conn.create_bucket(self.bucket_name)
        self.configOverride = {"S3PublishEndpoints": {
            "test1": {
                "region": "us-east-1",
                "bucket": self.bucket_name,
            }
        }}

        super(S3Test, self).prepare()

    def shutdown(self):
        if hasattr(self, "bucket_name"):
            if hasattr(self, "bucket"):
                keys = self.bucket.list()
                if keys:
                    self.bucket.delete_keys(keys)
            s3_conn.delete_bucket(self.bucket_name)

        super(S3Test, self).shutdown()

    def check_path(self, path):
        if not hasattr(self, "bucket_contents"):
            self.bucket_contents = [key.name for key in self.bucket.list()]

        if path.startswith("public/"):
            path = path[7:]

        if path in self.bucket_contents:
            return True

        if not path.endswith("/"):
            path = path + "/"

        for item in self.bucket_contents:
            if item.startswith(path):
                return True

        return False

    def check_exists(self, path):
        if not self.check_path(path):
            raise Exception("path %s doesn't exist" % (path, ))

    def check_not_exists(self, path):
        if self.check_path(path):
            raise Exception("path %s exists" % (path, ))

    def read_file(self, path):
        if path.startswith("public/"):
            path = path[7:]

        key = self.bucket.get_key(path)
        return key.get_contents_as_string()
