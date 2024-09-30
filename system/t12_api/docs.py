from api_lib import APITest


class TaskAPITestSwaggerDocs(APITest):
    """
    GET /docs
    """

    def check(self):
        resp = self.get("/docs/doc.json")
        self.check_equal(resp.status_code, 200)

        resp = self.get("/docs/", allow_redirects=False)
        self.check_equal(resp.status_code, 301)

        resp = self.get("/docs/index.html")
        self.check_equal(resp.status_code, 200)
