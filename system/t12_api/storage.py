from api_lib import APITest


class TaskAPITestSwaggerDocs(APITest):
    """
    GET /docs
    """

    def check(self):
        resp = self.get("/api/storage")
        self.check_equal(resp.status_code, 200)
