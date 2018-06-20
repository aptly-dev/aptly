from api_lib import APITest


class DbAPITestCleanup(APITest):
    """
    POST /db/cleanup
    """

    def check(self):
        resp = self.post_task(
            "/api/db/cleanup"
        )

        self.check_equal(resp.status_code, 200)
