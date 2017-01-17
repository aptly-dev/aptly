from api_lib import APITest
import xml.etree.ElementTree as ET


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

        # make sure our default layout is horizontal
        default = self.get("/api/graph.svg").content
        horizontal = self.get("/api/graph.svg?layout=horizontal").content
        self.check_equal(default, horizontal)

        # basic test of layout:
        # horizontal should be wider than vertical
        # vertical should be higher than horizontal
        vertical = self.get("/api/graph.svg?layout=vertical").content
        svgHWidth = ET.fromstring(horizontal).get('width').replace("pt","")
        svgHHeight = ET.fromstring(horizontal).get('height').replace("pt","")
        svgVWidth = ET.fromstring(vertical).get('width').replace("pt","")
        svgVHeight = ET.fromstring(vertical).get('height').replace("pt","")

        #self.check_gt(svgHWidth, svgVWidth)
        #self.check_gt(svgVHeight, svgHHeight)
