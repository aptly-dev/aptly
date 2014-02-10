import string
from lib import BaseTest


class UpdateMirror1Test(BaseTest):
    """
    update mirrors: regular update
    """
    longTest = True
    fixtureCmds = [
        "aptly -architectures=i386,amd64 mirror create --ignore-signatures alsa-ppa http://ppa.launchpad.net/alsa-backports/ubuntu/ hardy main",
    ]
    runCmd = "aptly mirror update --ignore-signatures alsa-ppa"

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class UpdateMirror2Test(BaseTest):
    """
    update mirrors: no such repo
    """
    runCmd = "aptly mirror update mirror-xyz"
    expectedCode = 1


class UpdateMirror3Test(BaseTest):
    """
    update mirrors: wrong checksum in release file
    """
    fixtureCmds = [
        "aptly mirror create --ignore-signatures failure ${url} hardy main",
    ]
    fixtureWebServer = "test_release"
    runCmd = "aptly mirror update --ignore-signatures failure"
    expectedCode = 1

    def gold_processor(self, gold):
        return string.Template(gold).substitute({'url': self.webServerUrl})


class UpdateMirror4Test(BaseTest):
    """
    update mirrors: wrong checksum in release file, but ignore
    """
    fixtureCmds = [
        "aptly mirror create --ignore-signatures failure ${url} hardy main",
    ]
    fixtureWebServer = "test_release"
    runCmd = "aptly mirror update -ignore-checksums --ignore-signatures failure"
    expectedCode = 1

    def gold_processor(self, gold):
        return string.Template(gold).substitute({'url': self.webServerUrl})


class UpdateMirror5Test(BaseTest):
    """
    update mirrors: wrong checksum in package
    """
    fixtureCmds = [
        "aptly mirror create --ignore-signatures failure ${url} hardy main",
    ]
    fixtureWebServer = "test_release2"
    runCmd = "aptly mirror update --ignore-signatures failure"
    expectedCode = 1

    def gold_processor(self, gold):
        return string.Template(gold).substitute({'url': self.webServerUrl})


class UpdateMirror6Test(BaseTest):
    """
    update mirrors: wrong checksum in package, but ignore
    """
    fixtureCmds = [
        "aptly mirror create --ignore-signatures failure ${url} hardy main",
    ]
    fixtureWebServer = "test_release2"
    runCmd = "aptly mirror update -ignore-checksums --ignore-signatures failure"

    def gold_processor(self, gold):
        return string.Template(gold).substitute({'url': self.webServerUrl})


class UpdateMirror7Test(BaseTest):
    """
    update mirrors: flat repository
    """
    fixtureGpg = True
    fixtureCmds = [
        "aptly mirror create --keyring=aptlytest.gpg flat http://download.opensuse.org/repositories/home:/DeepDiver1975/xUbuntu_10.04/ ./",
    ]
    runCmd = "aptly mirror update --keyring=aptlytest.gpg flat"

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))
