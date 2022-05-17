from lib import BaseTest
import uuid
import os

try:
    from azure.storage.blob import BlobServiceClient

    azure_storage_account = os.environ.get('AZURE_STORAGE_ACCOUNT')
    azure_storage_access_key = os.environ.get('AZURE_STORAGE_ACCESS_KEY')
    azure_storage_endpoint = os.environ.get(
        'AZURE_STORAGE_ENDPOINT',
        f'https://{azure_storage_account}.blob.core.windows.net',
    )
    if azure_storage_account is not None and azure_storage_access_key is not None:
        blob_client = BlobServiceClient(
            account_url=azure_storage_endpoint,
            credential=azure_storage_access_key,
        )
    else:
        print('Azure tests disabled: Azure creds not found in the environment')
        blob_client = None
except ImportError as e:
    print("Azure tests disabled: can't import azure.storage.blob", e)
    blob_client = None


class AzureTest(BaseTest):
    """
    BaseTest + support for Azure Blob Storage
    """

    use_azure_pool = False

    def __init__(self) -> None:
        super(AzureTest, self).__init__()
        self.container_name = None
        self.container = None
        self.container_contents = None

    def fixture_available(self):
        return super(AzureTest, self).fixture_available() and blob_client is not None

    def prepare(self):
        self.container_name = 'aptly-sys-test-' + str(uuid.uuid1())
        self.container = blob_client.create_container(
            self.container_name, public_access='blob'
        )

        self.azure_endpoint = {
            'accountName': azure_storage_account,
            'accountKey': azure_storage_access_key,
            'container': self.container_name,
            'endpoint': azure_storage_endpoint,
        }

        self.configOverride = {
            'AzurePublishEndpoints': {
                'test1': self.azure_endpoint,
            },
        }
        if self.use_azure_pool:
            self.configOverride['packagePoolStorage'] = {
                'type': 'azure',
                **self.azure_endpoint,
            }

        super(AzureTest, self).prepare()

    def shutdown(self):
        if self.container_name is not None:
            blob_client.delete_container(self.container_name)

        super(AzureTest, self).shutdown()

    def check_path(self, path):
        if self.container_contents is None:
            self.container_contents = [
                p.name for p in self.container.list_blobs() if p.name is not None
            ]

        if path.startswith('public/'):
            path = path.removeprefix('public/')

        if path in self.container_contents:
            return True

        if not path.endswith('/'):
            path = path + '/'

        for item in self.container_contents:
            if item.startswith(path):
                return True

        return False

    def check_exists(self, path):
        if not self.check_path(path):
            raise Exception("path %s doesn't exist" % (path,))

    def check_exists_azure_only(self, path):
        self.check_exists(path)
        BaseTest.check_not_exists(self, path)

    def check_not_exists(self, path):
        if self.check_path(path):
            raise Exception('path %s exists' % (path,))

    def read_file(self, path, mode=''):
        assert not mode

        if path.startswith('public/'):
            path = path.removeprefix('public/')

        blob = self.container.download_blob(path)
        return blob.readall().decode('utf-8')
