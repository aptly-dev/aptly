import string
import re
import os
import shutil
import inspect
from lib import BaseTest


def filterOutSignature(_, s):
    return re.sub(r'Signature made .* using', '', s)


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
        "aptly mirror create --keyring=aptlytest.gpg -architectures=amd64 flat https://cloud.r-project.org/bin/linux/debian jessie-cran3/",
    ]
    runCmd = "aptly mirror update --keyring=aptlytest.gpg flat"
    outputMatchPrepare = filterOutSignature

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
    outputMatchPrepare = filterOutSignature


class UpdateMirror9Test(BaseTest):
    """
    update mirrors: flat repository + sources
    """
    fixtureGpg = True
    fixtureCmds = [
        "aptly mirror create --keyring=aptlytest.gpg -with-sources flat-src https://cloud.r-project.org/bin/linux/debian jessie-cran3/",
    ]
    runCmd = "aptly mirror update --keyring=aptlytest.gpg flat-src"
    outputMatchPrepare = filterOutSignature

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class UpdateMirror10Test(BaseTest):
    """
    update mirrors: filtered
    """
    fixtureGpg = True
    fixtureCmds = [
        "aptly mirror create -keyring=aptlytest.gpg -with-sources -filter='!(Name (% r-*)), !($$PackageType (source))' flat-src https://cloud.r-project.org/bin/linux/debian jessie-cran3/",
    ]
    runCmd = "aptly mirror update --keyring=aptlytest.gpg flat-src"
    outputMatchPrepare = filterOutSignature

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
    outputMatchPrepare = filterOutSignature
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
    outputMatchPrepare = filterOutSignature

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


class UpdateMirror15Test(BaseTest):
    """
    update mirrors: update for mirror without MD5 checksums
    """
    longTest = False
    fixtureCmds = [
        "aptly mirror create --ignore-signatures bintray https://dl.bintray.com/smira/deb/ ./",
    ]
    runCmd = "aptly mirror update --ignore-signatures bintray"

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))

    def check(self):
        super(UpdateMirror15Test, self).check()
        # check pool
        self.check_exists('pool/c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb')


class UpdateMirror16Test(BaseTest):
    """
    update mirrors: update for mirror without MD5 checksums but with file in pool on legacy MD5 location

    as mirror lacks MD5 checksum, file would be downloaded but not re-imported
    """
    longTest = False
    fixtureCmds = [
        "aptly mirror create --ignore-signatures bintray https://dl.bintray.com/smira/deb/ ./",
    ]
    runCmd = "aptly mirror update --ignore-signatures bintray"

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))

    def prepare(self):
        super(UpdateMirror16Test, self).prepare()

        os.makedirs(os.path.join(os.environ["HOME"], ".aptly", "pool", "00", "35"))

        shutil.copy(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "libboost-program-options-dev_1.49.0.1_i386.deb"),
                    os.path.join(os.environ["HOME"], ".aptly", "pool", "00", "35"))

    def check(self):
        super(UpdateMirror16Test, self).check()
        # check pool
        self.check_not_exists('pool/c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb')


class UpdateMirror17Test(BaseTest):
    """
    update mirrors: update for mirror but with file in pool on legacy MD5 location
    """
    longTest = False
    fixtureCmds = [
        "aptly mirror create -ignore-signatures -architectures=i386 -filter=libboost-program-options-dev wheezy http://mirror.yandex.ru/debian wheezy main",
    ]
    runCmd = "aptly mirror update -ignore-signatures wheezy"

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))

    def prepare(self):
        super(UpdateMirror17Test, self).prepare()

        os.makedirs(os.path.join(os.environ["HOME"], ".aptly", "pool", "00", "35"))

        shutil.copy(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "libboost-program-options-dev_1.49.0.1_i386.deb"),
                    os.path.join(os.environ["HOME"], ".aptly", "pool", "00", "35"))

    def check(self):
        super(UpdateMirror17Test, self).check()
        # check pool
        self.check_not_exists('pool/c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb')


class UpdateMirror18Test(BaseTest):
    """
    update mirrors: update for mirror but with file in pool on legacy MD5 location and disabled legacy path support
    """
    longTest = False
    fixtureCmds = [
        "aptly mirror create -ignore-signatures -architectures=i386 -filter=libboost-program-options-dev wheezy http://mirror.yandex.ru/debian wheezy main",
    ]
    runCmd = "aptly mirror update -ignore-signatures wheezy"
    configOverride = {'skipLegacyPool': True}

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))

    def prepare(self):
        super(UpdateMirror18Test, self).prepare()

        os.makedirs(os.path.join(os.environ["HOME"], ".aptly", "pool", "00", "35"))

        shutil.copy(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "libboost-program-options-dev_1.49.0.1_i386.deb"),
                    os.path.join(os.environ["HOME"], ".aptly", "pool", "00", "35"))

    def check(self):
        super(UpdateMirror18Test, self).check()
        # check pool
        self.check_exists('pool/c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb')


class UpdateMirror19Test(BaseTest):
    """
    update mirrors: correct matching of Release checksums
    """
    longTest = False
    fixtureGpg = True
    fixtureCmds = [
        "aptly mirror create --keyring=aptlytest.gpg pagerduty http://packages.pagerduty.com/pdagent deb/"
    ]
    runCmd = "aptly mirror update --keyring=aptlytest.gpg pagerduty"
    outputMatchPrepare = filterOutSignature

    def output_processor(self, output):
        return "\n".join(line for line in output.split("\n") if ".deb" not in line)


class UpdateMirror20Test(BaseTest):
    """
    update mirrors: flat repository (internal GPG implementation)
    """
    fixtureGpg = True
    fixtureCmds = [
        "aptly mirror create --keyring=aptlytest.gpg -architectures=amd64 flat https://cloud.r-project.org/bin/linux/debian jessie-cran3/",
    ]
    configOverride = {"gpgProvider": "internal"}
    runCmd = "aptly mirror update --keyring=aptlytest.gpg flat"
    outputMatchPrepare = filterOutSignature

    def output_processor(self, output):
        return "\n".join(sorted(output.split("\n")))


class UpdateMirror21Test(BaseTest):
    """
    update mirrors: correct matching of Release checksums (internal pgp implementation)
    """
    longTest = False
    configOverride = {"gpgProvider": "internal"}
    fixtureGpg = True
    fixtureCmds = [
        "aptly mirror create --keyring=aptlytest.gpg pagerduty http://packages.pagerduty.com/pdagent deb/"
    ]
    runCmd = "aptly mirror update --keyring=aptlytest.gpg pagerduty"
    outputMatchPrepare = filterOutSignature

    def output_processor(self, output):
        return "\n".join(line for line in output.split("\n") if ".deb" not in line)
