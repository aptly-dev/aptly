from api_lib import APITest


class VersionAPITest(APITest):
    """
    GET /version
    """

    def check(self):
        self.check_equal(self.get("/api/version").json(), {'Version': '0.9.8~dev'})


class VersionAPITestAuthenticationError(APITest):
    """
    GET /version (unauthenticated)
    """

    def check(self):
        self.api_username="nobody"
        self.check_equal(self.get("/api/version").status_code, 401)