import string
import re
from lib import BaseTest


class UpdateMirror1Test(BaseTest):
    """
    update mirrors: regular update
    """
    longTest = False
    fixtureCmds = [
        "aptly -architectures=i386,amd64 mirror create --ignore-signatures varnish http://repo.varnish-cache.org/debian/ wheezy varnish-3.0",
    ]
    runCmd = "aptly mirror update --ignore-signatures varnish"

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
    outputMatchPrepare = lambda _, s: re.sub(r'Signature made .* using', '', s)

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class UpdateMirror8Test(BaseTest):
    """
    update mirrors: with sources (already in pool)
    """
    fixtureGpg = True
    fixturePool = True
    fixtureCmds = [
        "aptly mirror create --keyring=aptlytest.gpg gnuplot-maverick-src http://ppa.launchpad.net/gladky-anton/gnuplot/ubuntu/ maverick",
    ]
    runCmd = "aptly mirror update --keyring=aptlytest.gpg gnuplot-maverick-src"
    outputMatchPrepare = lambda _, s: re.sub(r'Signature made .* using', '', s)


class UpdateMirror9Test(BaseTest):
    """
    update mirrors: flat repository + sources
    """
    fixtureGpg = True
    fixtureCmds = [
        "aptly mirror create --keyring=aptlytest.gpg -with-sources flat-src http://download.opensuse.org/repositories/home:/DeepDiver1975/xUbuntu_10.04/ ./",
    ]
    runCmd = "aptly mirror update --keyring=aptlytest.gpg flat-src"
    outputMatchPrepare = lambda _, s: re.sub(r'Signature made .* using', '', s)

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))
