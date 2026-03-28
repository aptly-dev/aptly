from lib import BaseTest
import os
import uuid

try:
    from google.cloud import storage

    gcs_project = os.environ.get('GCS_PROJECT')

    if gcs_project:
        gcs_client = storage.Client(project=gcs_project)
    else:
        print('GCS tests disabled: GCS_PROJECT is not set')
        gcs_client = None
except ImportError as e:
    print("GCS tests disabled: can't import google.cloud.storage", e)
    gcs_client = None
except Exception as e:
    print('GCS tests disabled: unable to initialize GCS client', e)
    gcs_client = None


class GCSTest(BaseTest):
    """
    BaseTest + support for GCS
    """

    gcsOverrides = {}

    def __init__(self) -> None:
        super(GCSTest, self).__init__()
        self.bucket_name = None
        self.bucket = None
        self.bucket_contents = None

    def fixture_available(self):
        return super(GCSTest, self).fixture_available() and gcs_client is not None

    def prepare(self):
        # GCS bucket names must be globally unique and lower-case.
        self.bucket_name = 'aptly-sys-test-' + str(uuid.uuid4()).replace('_', '-').lower()
        self.bucket = gcs_client.create_bucket(self.bucket_name)

        self.configOverride = {
            'GcsPublishEndpoints': {
                'test1': {
                    'bucket': self.bucket_name,
                    'project': gcs_project,
                },
            },
        }

        self.configOverride['GcsPublishEndpoints']['test1'].update(**self.gcsOverrides)

        super(GCSTest, self).prepare()

    def shutdown(self):
        if self.bucket is not None:
            for blob in self.bucket.list_blobs():
                blob.delete()
            self.bucket.delete(force=True)

        super(GCSTest, self).shutdown()

    def _normalize_path(self, path):
        if path.startswith('public/'):
            return path[7:]
        return path

    def check_path(self, path):
        if self.bucket_contents is None:
            self.bucket_contents = [blob.name for blob in self.bucket.list_blobs()]

        path = self._normalize_path(path)

        if path in self.bucket_contents:
            return True

        if not path.endswith('/'):
            path = path + '/'

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
        assert not mode

        path = self._normalize_path(path)
        blob = self.bucket.blob(path)
        return blob.download_as_text()
