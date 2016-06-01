from api_lib import APITest


class AuthTestGetToken(APITest):
    """
    POST /login
    """

    def check(self):
        if self.requireAuthentication:
            self.token = None
            self.requestToken()
            if len(self.api_token)<1:
                raise Exception("Login failed, no token received.")


class AuthTestRefreshToken(APITest):
    """
    POST /refresh_token
    """

    def check(self):
        if self.requireAuthentication:
            self.requestToken()
            res = self.get("/api/refresh_token")
            self.check_equal(res.status_code, 200)
            self.check_equal(res.json()["token"], self.api_token)


class VersionAPITestAuthenticationError(APITest):
    """
    GET /version (unauthenticated)
    """

    def check(self):
        if self.requireAuthentication:
            self.api_username="nobody"
            self.check_equal(self.get("/api/version").status_code, 401)
            self.api_username="admin"
            self.api_password="123"
            self.check_equal(self.get("/api/version").status_code, 401)