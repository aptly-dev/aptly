from api_lib import APITest
import xml.etree.ElementTree as ET


class GraphAPITest(APITest):
    """
    GET /graph.:ext
    """

    requiresDot = True

    def check(self):
        resp = self.get("/api/graph.png")
        self.check_equal(resp.headers["Content-Type"], "image/png")
        self.check_equal(resp.content[:4], '\x89PNG')

        self.check_equal(self.post("/api/repos", json={"Name": "xyz", "Comment": "fun repo"}).status_code, 201)
        resp = self.get("/api/graph.svg")
        self.check_equal(resp.headers["Content-Type"], "image/svg+xml")
        self.check_equal(resp.content[:4], '<?xm')

        resp = self.get("/api/graph.dot")
        self.check_equal(resp.headers["Content-Type"], "text/plain; charset=utf-8")
        self.check_equal(resp.content[:13], 'digraph aptly')

        # basic test of layout:
        #   horizontal should be wider than vertical
        #   vertical should be higher than horizontal
        # for this to work we need at couple of repos
        tempRepos = [self.random_name() for r in range(3)]
        for repo in tempRepos:
            self.check_equal(self.post("/api/repos", json={"Name": repo, "Comment": "graph test repo"}).status_code, 201)

        horizontal = self.get("/api/graph.svg?layout=horizontal").content
        vertical = self.get("/api/graph.svg?layout=vertical").content
        horizontalWidth = int(ET.fromstring(horizontal).get('width').replace("pt", ""))
        horizontalHeight = int(ET.fromstring(horizontal).get('height').replace("pt", ""))
        verticalWidth = int(ET.fromstring(vertical).get('width').replace("pt", ""))
        verticalHeight = int(ET.fromstring(vertical).get('height').replace("pt", ""))

        self.check_gt(horizontalWidth, verticalWidth)
        self.check_gt(verticalHeight, horizontalHeight)

        # make sure our default layout is horizontal
        self.check_equal(horizontal, self.get("/api/graph.svg").content)

        # remove the repos again
        for repo in tempRepos:
            self.check_equal(self.delete_task(
                "/api/repos/" + repo, params={"force": "1"}).json()['State'], 2
            )
