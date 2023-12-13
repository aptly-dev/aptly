from PIL import Image
import time
import re
import os
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

        os.remove("graph.png")


class GraphTest2(BaseTest):
    """
    Test that the graph is correctly generate when vertical layout is specified.
    """

    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap2 from mirror gnuplot-maverick",
        "aptly publish snapshot -skip-signing snap2",
    ]
    runCmd = "aptly graph -output=graph.png -layout=vertical"

    def check(self):
        self.check_exists_in_cwd("graph.png")

        with Image.open("graph.png") as img:
            (width, height) = img.size
            # check is horizontal
            self.check_gt(height, width)
            img.verify()

        os.remove("graph.png")


class GraphTest3(BaseTest):
    """
    Test that the graph is accessible at the temporary
    file path aptly prints.
    """

    fixtureDB = True
    fixturePool = True
    fixtureCmds = [
        "aptly snapshot create snap3 from mirror gnuplot-maverick",
        "aptly publish snapshot -skip-signing snap3",
    ]
    runCmd = "aptly graph"

    def check(self):
        assert self.output is not None

        file_regex = re.compile(r": (\S+).png")
        temp_file = file_regex.search(self.output.decode())

        assert temp_file is not None
        temp_file = temp_file.group(1) + ".png"

        self.check_exists(temp_file)
        with Image.open(temp_file) as img:
            (width, height) = img.size
            # check is horizontal
            self.check_gt(width, height)
            img.verify()

        # wait 1s to make sure it still exists
        time.sleep(1)

        assert os.path.isfile(temp_file)

        os.remove(temp_file)
