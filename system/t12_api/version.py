from api_lib import APITest


class VersionAPITest(APITest):
    """
    GET /version
    """

    def check(self):
        self.check_equal(self.get("/api/version").json(), {'Version': '0.9.8~dev'})
