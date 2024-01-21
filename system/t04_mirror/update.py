import inspect
import os
import re
import shutil
import string

from lib import BaseTest


def filterOutSignature(_, s):
    return re.sub(r'Signature made .* using', '', s)


def filterOutRedirects(_, s):
    return re.sub(r'Following redirect to .+?\n', '', s)


class UpdateMirror1Test(BaseTest):
    """
    update mirrors: regular update
    """
    sortOutput = True
    longTest = False
    fixtureCmds = [
        "aptly -architectures=i386,amd64 mirror create --ignore-signatures varnish http://repo.aptly.info/system-tests/packagecloud.io/varnishcache/varnish30/debian/ wheezy main",
    ]
    runCmd = "aptly mirror update --ignore-signatures varnish"
    outputMatchPrepare = filterOutRedirects


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
    configOverride = {
        "downloadRetries": 0,
    }
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
    configOverride = {
        "downloadRetries": 0,
    }
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
    configOverride = {
        "downloadRetries": 0,
    }
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
    configOverride = {
        "downloadRetries": 0,
    }

    runCmd = "aptly mirror update -ignore-checksums --ignore-signatures failure"

    def gold_processor(self, gold):
        return string.Template(gold).substitute({'url': self.webServerUrl})


class UpdateMirror7Test(BaseTest):
    """
    update mirrors: flat repository
    """
    sortOutput = True
    fixtureGpg = True
    fixtureCmds = [
            "aptly mirror create --keyring=aptlytest.gpg -architectures=amd64 flat http://repo.aptly.info/system-tests/cloud.r-project.org/bin/linux/debian bullseye-cran40/",
    ]
    runCmd = "aptly mirror update --keyring=aptlytest.gpg flat"
    outputMatchPrepare = filterOutSignature


class UpdateMirror8Test(BaseTest):
    """
    update mirrors: with sources (already in pool)
    """
    configOverride = {"max-tries": 1}

    fixtureGpg = True
    fixturePool = True
    fixtureCmds = [
        "aptly mirror create --keyring=aptlytest.gpg gnuplot-maverick-src http://repo.aptly.info/system-tests/ppa.launchpad.net/gladky-anton/gnuplot/ubuntu/ maverick",
    ]
    runCmd = "aptly mirror update --keyring=aptlytest.gpg gnuplot-maverick-src"
    outputMatchPrepare = filterOutSignature


class UpdateMirror9Test(BaseTest):
    """
    update mirrors: flat repository + sources
    """
    sortOutput = True
    fixtureGpg = True
    fixtureCmds = [
        "aptly mirror create --keyring=aptlytest.gpg -with-sources flat-src http://repo.aptly.info/system-tests/cloud.r-project.org/bin/linux/debian bullseye-cran40/",
    ]
    runCmd = "aptly mirror update --keyring=aptlytest.gpg flat-src"
    outputMatchPrepare = filterOutSignature


class UpdateMirror10Test(BaseTest):
    """
    update mirrors: filtered
    """
    sortOutput = True
    fixtureGpg = True
    fixtureCmds = [
        "aptly mirror create -keyring=aptlytest.gpg -with-sources -filter='!(Name (% r-*)), !($$PackageType (source))' flat-src http://repo.aptly.info/system-tests/cloud.r-project.org/bin/linux/debian bullseye-cran40/",
    ]
    runCmd = "aptly mirror update --keyring=aptlytest.gpg flat-src"
    outputMatchPrepare = filterOutSignature


class UpdateMirror11Test(BaseTest):
    """
    update mirrors: update over FTP
    """
    configOverride = {"max-tries": 1}
    sortOutput = True
    longTest = False
    fixtureGpg = True
    requiresFTP = True
    fixtureCmds = [
        "aptly mirror create -keyring=aptlytest.gpg -filter='Priority (required), Name (% s*)' "
        "-architectures=i386 stretch-main https://snapshot.debian.org/archive/debian/20220201T025006Z/ stretch main",
    ]
    outputMatchPrepare = filterOutSignature
    runCmd = "aptly mirror update -keyring=aptlytest.gpg stretch-main"


class UpdateMirror12Test(BaseTest):
    """
    update mirrors: update with udebs
    """
    configOverride = {"max-tries": 1}
    sortOutput = True
    longTest = False
    fixtureGpg = True
    fixtureCmds = [
        "aptly -architectures=i386,amd64 mirror create -keyring=aptlytest.gpg -filter='$$Source (gnupg2)' -with-udebs stretch http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch main non-free",
    ]
    runCmd = "aptly mirror update -keyring=aptlytest.gpg stretch"
    outputMatchPrepare = filterOutSignature


class UpdateMirror13Test(BaseTest):
    """
    update mirrors: regular update with --skip-existing-packages option
    """
    sortOutput = True
    longTest = False
    fixtureCmds = [
        "aptly -architectures=i386,amd64 mirror create --ignore-signatures varnish http://repo.aptly.info/system-tests/packagecloud.io/varnishcache/varnish30/debian/ wheezy main",
    ]
    runCmd = "aptly mirror update --ignore-signatures --skip-existing-packages varnish"
    outputMatchPrepare = filterOutRedirects


class UpdateMirror14Test(BaseTest):
    """
    update mirrors: regular update with --skip-existing-packages option
    """
    sortOutput = True
    longTest = False
    fixtureCmds = [
        "aptly -architectures=i386,amd64 mirror create --ignore-signatures varnish http://repo.aptly.info/system-tests/packagecloud.io/varnishcache/varnish30/debian/ wheezy main",
        "aptly mirror update --ignore-signatures --skip-existing-packages varnish"
    ]
    runCmd = "aptly mirror update --ignore-signatures --skip-existing-packages varnish"
    outputMatchPrepare = filterOutRedirects


class UpdateMirror15Test(BaseTest):
    """
    update mirrors: update for mirror without MD5 checksums
    """
    # TODO spin up a Python server to serve that data from fixtures directory, instead of using bintray
    # e.g. python3 -m http.server --directory src/aptly/system/t04_mirror/test_release/
    # but that fixture seems to have the wrong hashes...
    skipTest = "Using deprecated service - bintray"
    sortOutput = True
    longTest = False
    fixtureCmds = [
        "aptly mirror create --ignore-signatures bintray https://dl.bintray.com/smira/deb/ ./",
        # TODO note the ./ is "flat" whereas putting "hardy" looks into dists/hardy
        # "aptly mirror create --ignore-signatures bintray http://localhost:8000/ hardy",
    ]
    runCmd = "aptly mirror update --ignore-signatures bintray"

    def check(self):
        super(UpdateMirror15Test, self).check()
        # check pool
        self.check_exists(
            'pool/c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb')


class UpdateMirror16Test(BaseTest):
    """
    update mirrors: update for mirror without MD5 checksums but with file in pool on legacy MD5 location

    as mirror lacks MD5 checksum, file would be downloaded but not re-imported
    """
    skipTest = "Using deprecated service - bintray"
    sortOutput = True
    longTest = False
    fixtureCmds = [
        "aptly mirror create --ignore-signatures bintray https://dl.bintray.com/smira/deb/ ./",
    ]
    runCmd = "aptly mirror update --ignore-signatures bintray"

    def prepare(self):
        super(UpdateMirror16Test, self).prepare()

        os.makedirs(os.path.join(
            os.environ["HOME"], ".aptly", "pool", "00", "35"))

        shutil.copy(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "libboost-program-options-dev_1.49.0.1_i386.deb"),
                    os.path.join(os.environ["HOME"], ".aptly", "pool", "00", "35"))

    def check(self):
        super(UpdateMirror16Test, self).check()
        # check pool
        self.check_not_exists(
            'pool/c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb')


class UpdateMirror17Test(BaseTest):
    """
    update mirrors: update for mirror but with file in pool on legacy MD5 location
    """
    sortOutput = True
    longTest = False
    fixtureCmds = [
        "aptly mirror create -ignore-signatures -architectures=i386 -filter=libboost-program-options-dev stretch http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian stretch main",
    ]
    runCmd = "aptly mirror update -ignore-signatures stretch"

    def prepare(self):
        super(UpdateMirror17Test, self).prepare()

        os.makedirs(os.path.join(
            os.environ["HOME"], ".aptly", "pool", "e0", "bb"))

        shutil.copy(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "libboost-program-options-dev_1.62.0.1_i386.deb"),
                    os.path.join(os.environ["HOME"], ".aptly", "pool", "e0", "bb"))

    def check(self):
        super(UpdateMirror17Test, self).check()
        # check pool
        self.check_not_exists(
            'pool/db/a2/f225645a2a8bd8378e2f64bd1faa_libboost-program-options-dev_1.62.0.1_i386.deb')


class UpdateMirror18Test(BaseTest):
    """
    update mirrors: update for mirror but with file in pool on legacy MD5 location and disabled legacy path support
    """
    sortOutput = True
    longTest = False
    fixtureCmds = [
        "aptly mirror create -ignore-signatures -architectures=i386 -filter=libboost-program-options-dev stretch http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian stretch main",
    ]
    runCmd = "aptly mirror update -ignore-signatures stretch"
    configOverride = {'skipLegacyPool': True}

    def prepare(self):
        super(UpdateMirror18Test, self).prepare()

        os.makedirs(os.path.join(
            os.environ["HOME"], ".aptly", "pool", "e0", "bb"))

        shutil.copy(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "libboost-program-options-dev_1.62.0.1_i386.deb"),
                    os.path.join(os.environ["HOME"], ".aptly", "pool", "e0", "bb"))

    def check(self):
        super(UpdateMirror18Test, self).check()
        # check pool
        self.check_exists(
            'pool/db/a2/f225645a2a8bd8378e2f64bd1faa_libboost-program-options-dev_1.62.0.1_i386.deb')


class UpdateMirror19Test(BaseTest):
    """
    update mirrors: correct matching of Release checksums
    """
    configOverride = {"max-tries": 1}
    longTest = False
    fixtureGpg = True
    fixtureCmds = [
        "aptly mirror create --keyring=aptlytest.gpg pagerduty http://repo.aptly.info/system-tests/packages.pagerduty.com/pdagent deb/"
    ]
    runCmd = "aptly mirror update --keyring=aptlytest.gpg pagerduty"
    outputMatchPrepare = filterOutSignature

    def output_processor(self, output):
        return "\n".join(line for line in self.ensure_utf8(output).split("\n") if ".deb" not in line)


class UpdateMirror20Test(BaseTest):
    """
    update mirrors: flat repository (internal GPG implementation)
    """
    sortOutput = True
    fixtureGpg = True
    configOverride = {"gpgProvider": "internal"}
    fixtureCmds = [
        "gpg --no-default-keyring --keyring aptlytest.gpg --export-options export-minimal --export -o " + os.path.join(
            os.environ["HOME"], ".gnupg/aptlytest-gpg1.gpg"),
        "aptly mirror create --keyring=aptlytest-gpg1.gpg -architectures=amd64 --filter='r-cran-class' flat http://repo.aptly.info/system-tests/cloud.r-project.org/bin/linux/debian bullseye-cran40/",
    ]
    runCmd = "aptly mirror update --keyring=aptlytest-gpg1.gpg flat"
    outputMatchPrepare = filterOutSignature

    def teardown(self):
        self.run_cmd(["rm", "-f", os.path.join(os.environ["HOME"], ".gnupg/aptlytest-gpg1.gpg")])


class UpdateMirror21Test(BaseTest):
    """
    update mirrors: correct matching of Release checksums (internal pgp implementation)
    """
    longTest = False
    configOverride = {"gpgProvider": "internal", "max-tries": 1}
    fixtureGpg = True
    fixtureCmds = [
        "gpg --no-default-keyring --keyring aptlytest.gpg --export-options export-minimal --export -o " + os.path.join(
            os.environ["HOME"], ".gnupg/aptlytest-gpg1.gpg"),
        "aptly mirror create --keyring=aptlytest-gpg1.gpg pagerduty http://repo.aptly.info/system-tests/packages.pagerduty.com/pdagent deb/"
    ]
    runCmd = "aptly mirror update --keyring=aptlytest-gpg1.gpg pagerduty"
    outputMatchPrepare = filterOutSignature

    def output_processor(self, output):
        return "\n".join(line for line in self.ensure_utf8(output).split("\n") if ".deb" not in line)

    def teardown(self):
        self.run_cmd(["rm", "-f", os.path.join(os.environ["HOME"], ".gnupg/aptlytest-gpg1.gpg")])


class UpdateMirror22Test(BaseTest):
    """
    update mirrors: SHA512 checksums only
    """
    configOverride = {"gpgProvider": "internal"}
    fixtureGpg = True
    fixtureCmds = [
        "gpg --no-default-keyring --keyring aptlytest.gpg --export-options export-minimal --export -o " + os.path.join(
            os.environ["HOME"], ".gnupg/aptlytest-gpg1.gpg"),
        "aptly mirror create --keyring=aptlytest-gpg1.gpg --filter=nomatch libnvidia-container http://repo.aptly.info/system-tests/nvidia.github.io/libnvidia-container/stable/ubuntu16.04/amd64 ./"
    ]
    runCmd = "aptly mirror update --keyring=aptlytest-gpg1.gpg libnvidia-container"

    def outputMatchPrepare(self, s):
        return re.sub(r'Signature made .* using|Packages filtered: .* -> 0.', '', s)

    def teardown(self):
        self.run_cmd(["rm", "-f", os.path.join(os.environ["HOME"], ".gnupg/aptlytest-gpg1.gpg")])


class UpdateMirror23Test(BaseTest):
    """
    update mirrors: update with installer
    """
    configOverride = {"max-tries": 1}
    sortOutput = True
    longTest = False
    fixtureGpg = True
    fixtureCmds = [
        "aptly -architectures=s390x mirror create -keyring=aptlytest.gpg -filter='installer' -with-installer stretch http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch main non-free",
    ]
    runCmd = "aptly mirror update -keyring=aptlytest.gpg stretch"
    outputMatchPrepare = filterOutSignature


class UpdateMirror24Test(BaseTest):
    """
    update mirrors: update with installer with separate gpg file
    """
    configOverride = {"max-tries": 1}
    sortOutput = True
    longTest = False
    fixtureGpg = True
    fixtureCmds = [
        "aptly -architectures=amd64 mirror create -keyring=aptlytest.gpg -filter='installer' -with-installer trusty http://repo.aptly.info/system-tests/us.archive.ubuntu.com/ubuntu/ trusty main restricted",
    ]
    runCmd = "aptly mirror update -keyring=aptlytest.gpg trusty"
    outputMatchPrepare = filterOutSignature
