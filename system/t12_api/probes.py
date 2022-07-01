from api_lib import APITest


class ReadyAPITest(APITest):
    """
    GET /ready
    """

    def check(self):
        resp = self.get("/api/ready")
        self.check_equal(resp.status_code, 200)

        readyStatus = "{\"Status\":\"Aptly is ready\"}"
        self.check_equal(readyStatus, resp.text)


class HealthyAPITest(APITest):
    """
    GET /healthy
    """

    def check(self):
        resp = self.get("/api/healthy")
        self.check_equal(resp.status_code, 200)

        healthyStatus = "{\"Status\":\"Aptly is healthy\"}"
        self.check_equal(healthyStatus, resp.text)
