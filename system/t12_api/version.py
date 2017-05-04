from api_lib import APITest
import os


class VersionAPITest(APITest):
    """
    GET /version
    """

    def check(self):
        self.check_equal(self.get("/api/version").json(), {'Version': os.environ['APTLY_VERSION']})
