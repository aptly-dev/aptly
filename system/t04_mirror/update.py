from lib import BaseTest


class UpdateMirror1Test(BaseTest):
    """
    update mirrors: regular update
    """
    longTest = True
    fixtureCmds = [
        "aptly -architectures=i386,amd64 mirror create alsa-ppa http://ppa.launchpad.net/alsa-backports/ubuntu/ hardy main",
    ]
    runCmd = "aptly mirror update alsa-ppa"

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class UpdateMirror2Test(BaseTest):
    """
    update mirrors: no such repo
    """
    runCmd = "aptly mirror update mirror-xyz"
    expectedCode = 1
