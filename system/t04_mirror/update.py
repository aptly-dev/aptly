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
        "aptly mirror create --keyring=aptlytest.gpg flat http://download.opensuse.org/repositories/home:/monkeyiq/Debian_7.0/ ./",
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
        "aptly mirror create --keyring=aptlytest.gpg -with-sources flat-src http://download.opensuse.org/repositories/home:/monkeyiq/Debian_7.0/ ./",
    ]
    runCmd = "aptly mirror update --keyring=aptlytest.gpg flat-src"
    outputMatchPrepare = lambda _, s: re.sub(r'Signature made .* using', '', s)

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class UpdateMirror10Test(BaseTest):
    """
    update mirrors: filtered
    """
    fixtureGpg = True
    fixtureCmds = [
        "aptly mirror create -keyring=aptlytest.gpg -with-sources -filter='!(Name (% libferris*)), !($$PackageType (source))' flat-src http://download.opensuse.org/repositories/home:/monkeyiq/Debian_7.0/ ./",
    ]
    runCmd = "aptly mirror update --keyring=aptlytest.gpg flat-src"
    outputMatchPrepare = lambda _, s: re.sub(r'Signature made .* using', '', s)

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class UpdateMirror11Test(BaseTest):
    """
    update mirrors: update over FTP
    """
    longTest = False
    fixtureGpg = True
    fixtureCmds = [
        "aptly mirror create -keyring=aptlytest.gpg -filter='Priority (required), Name (% s*)' -architectures=i386 wheezy-main ftp://ftp.ru.debian.org/debian/ wheezy main",
    ]
    outputMatchPrepare = lambda _, s: re.sub(r'Signature made .* using', '', s)
    runCmd = "aptly mirror update -keyring=aptlytest.gpg wheezy-main"

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class UpdateMirror12Test(BaseTest):
    """
    update mirrors: update with udebs
    """
    longTest = False
    fixtureGpg = True
    fixtureCmds = [
        "aptly -architectures=i386,amd64 mirror create -keyring=aptlytest.gpg -filter='$$Source (gnupg)' -with-udebs wheezy http://mirror.yandex.ru/debian/ wheezy main non-free",
    ]
    runCmd = "aptly mirror update -keyring=aptlytest.gpg wheezy"
    outputMatchPrepare = lambda _, s: re.sub(r'Signature made .* using', '', s)

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class UpdateMirror13Test(BaseTest):
    """
    update mirrors: regular update with --skip-existing-packages option
    """
    longTest = False
    fixtureCmds = [
        "aptly -architectures=i386,amd64 mirror create --ignore-signatures varnish http://repo.varnish-cache.org/debian/ wheezy varnish-3.0",
    ]
    runCmd = "aptly mirror update --ignore-signatures --skip-existing-packages varnish"

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class UpdateMirror14Test(BaseTest):
    """
    update mirrors: regular update with --skip-existing-packages option
    """
    longTest = False
    fixtureCmds = [
        "aptly -architectures=i386,amd64 mirror create --ignore-signatures varnish http://repo.varnish-cache.org/debian/ wheezy varnish-3.0",
        "aptly mirror update --ignore-signatures --skip-existing-packages varnish"
    ]
    runCmd = "aptly mirror update --ignore-signatures --skip-existing-packages varnish"

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


