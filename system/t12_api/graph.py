from api_lib import APITest


class GraphAPITest(APITest):
    """
    GET /graph.:ext
    """

    def check(self):
        resp = self.get("/api/graph.png")
        self.check_equal(resp.headers["Content-Type"], "image/png")
        self.check_equal(resp.content[:4], '\x89PNG')

        self.check_equal(self.post("/api/repos", json={"Name": "xyz", "Comment": "fun repo"}).status_code, 201)
        resp = self.get("/api/graph.svg")
        self.check_equal(resp.headers["Content-Type"], "image/svg+xml")
        self.check_equal(resp.content[:4], '<?xm')
