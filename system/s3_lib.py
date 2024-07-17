from lib import BaseTest
import uuid
import os

try:
    import boto

    if 'AWS_SECRET_ACCESS_KEY' in os.environ and 'AWS_ACCESS_KEY_ID' in os.environ:
        s3_conn = boto.connect_s3()
    else:
        print("S3 tests disabled: AWS creds not found in the environment (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)")
        s3_conn = None
except ImportError as e:
    print("S3 tests disabled: can't import boto", e)
    s3_conn = None


class S3Test(BaseTest):
    """
    BaseTest + support for S3
    """

    s3Overrides = {}

    def fixture_available(self):
        return super(S3Test, self).fixture_available() and s3_conn is not None

    def prepare(self):
        self.bucket_name = "aptly-sys-test-" + str(uuid.uuid1())
        self.bucket = s3_conn.create_bucket(self.bucket_name)
        self.configOverride = {"S3PublishEndpoints": {
            "test1": {
                "region": "us-east-1",
                "bucket": self.bucket_name,
                "awsAccessKeyID": os.environ["AWS_ACCESS_KEY_ID"],
                "awsSecretAccessKey": os.environ["AWS_SECRET_ACCESS_KEY"]
            }
        }}

        self.configOverride["S3PublishEndpoints"]["test1"].update(**self.s3Overrides)

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

    def read_file(self, path, mode=''):
        # We don't support reading as binary here.
        assert not mode

        if path.startswith("public/"):
            path = path[7:]

        key = self.bucket.get_key(path)
        return key.get_contents_as_string()
