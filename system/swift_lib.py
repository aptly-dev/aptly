from lib import BaseTest
import logging
import uuid
import os


try:
    import swiftclient

    if 'OS_USERNAME' in os.environ and 'OS_PASSWORD' in os.environ:
        auth_username = os.environ.get('OS_USERNAME')
        auth_password = os.environ.get('OS_PASSWORD')
        # Using auth version 2 /v2.0/
        auth_url = os.environ.get('OS_AUTH_URL')
        auth_tenant = os.environ.get('OS_TENANT_NAME')

        account_username = "%s:%s" % (auth_tenant, auth_username)
        swift_conn = swiftclient.Connection(auth_url, account_username,
                                            auth_password, auth_version=2)
    elif 'ST_USER' in os.environ and 'ST_KEY' in os.environ:
        auth_username = os.environ.get('ST_USER')
        auth_password = os.environ.get('ST_KEY')
        auth_url = os.environ.get('ST_AUTH')
        # Using auth version 1 (/auth/v1.0)
        swift_conn = swiftclient.Connection(auth_url, auth_username,
                                            auth_password, auth_version=1)
    else:
        print "Swift tests disabled: OpenStack creds not found in the environment"
        swift_conn = None
except ImportError, e:
    print "Swift tests disabled: unable to import swiftclient", e
    swift_conn = None


class SwiftTest(BaseTest):
    """
    BaseTest + support for Swift
    """

    def fixture_available(self):
        return super(SwiftTest, self).fixture_available() and swift_conn is not None

    def prepare(self):
        self.container_name = "aptly-sys-test-" + str(uuid.uuid1())
        swift_conn.put_container(self.container_name)

        self.configOverride = {"SwiftPublishEndpoints": {
            "test1": {
                "container": self.container_name,
            }
        }}

        super(SwiftTest, self).prepare()

    def _try_delete_container(self):
        if not hasattr(self, "container_name"):
            return

        try:
            for obj in swift_conn.get_container(self.container_name,
                                                full_listing=True)[1]:
                swift_conn.delete_object(self.container_name, obj.get("name"))

            swift_conn.delete_container(self.container_name)
        except swiftclient.ClientException:
            logging.exception("Error shutting down Swift container")

    def shutdown(self):
        self._try_delete_container()
        super(SwiftTest, self).shutdown()

    def check_path(self, path):
        if not hasattr(self, "container_contents"):
            self.container_contents = [obj.get('name') for obj in
                                       swift_conn.get_container(self.container_name)[1]]

        if path in self.container_contents:
            return True

        if not path.endswith("/"):
            path = path + "/"

        for item in self.container_contents:
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
        hdrs, body = swift_conn.get_object(self.container_name, path)
        return body
