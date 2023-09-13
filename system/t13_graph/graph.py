from PIL import Image
from lib import BaseTest


class GraphTest1(BaseTest):
    """
    Test that graph is generated correctly and accessible at the specified output path.
    """

    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror gnuplot-maverick",
        "aptly publish snapshot -skip-signing snap1",
    ]
    runCmd = "aptly graph -output=graph.png -layout=horizontal"

    def check(self):
        self.check_exists_in_cwd("graph.png")

        with Image.open("graph.png") as img:
            (width, height) = img.size
            # check is horizontal
            self.check_gt(width, height)
            img.verify()
