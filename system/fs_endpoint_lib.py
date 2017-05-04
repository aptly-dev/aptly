import os
from lib import BaseTest


class FileSystemEndpointTest(BaseTest):
    """
    BaseTest + support for filesystem endpoints
    """

    def prepare(self):
        self.configOverride = {"FileSystemPublishEndpoints": {
            "symlink": {
                "rootDir": os.path.join(os.environ["HOME"], ".aptly", "public_symlink"),
                "linkMethod": "symlink"
            },
            "hardlink": {
                "rootDir": os.path.join(os.environ["HOME"], ".aptly", "public_hardlink"),
                "linkMethod": "hardlink"
            },
            "copy": {
                "rootDir": os.path.join(os.environ["HOME"], ".aptly", "public_copy"),
                "linkMethod": "copy",
                "verifyMethod": "md5"
            },
            "copysize": {
                "rootDir": os.path.join(os.environ["HOME"], ".aptly", "public_copysize"),
                "linkMethod": "copy",
                "verifyMethod": "size"
            }
        }}
        super(FileSystemEndpointTest, self).prepare()

    def check_is_regular(self, path):
        if not os.path.isfile(os.path.join(os.environ["HOME"], ".aptly", path)):
            raise Exception("path %s is not a regular file" % (path, ))

    def check_is_symlink(self, path):
        if not os.path.islink(os.path.join(os.environ["HOME"], ".aptly", path)):
            raise Exception("path %s is not a symlink" % (path, ))

    def check_is_hardlink(self, path):
        if os.stat(os.path.join(os.environ["HOME"], ".aptly", path)) <= 1:
            raise Exception("path %s is not a hardlink" % (path, ))

    def check_is_copy(self, path):
        fullpath = os.path.join(os.environ["HOME"], ".aptly", path)
        if not (os.path.isfile(fullpath) and not self.check_is_hardlink(path)):
            raise Exception("path %s is not a copy" % (path, ))
