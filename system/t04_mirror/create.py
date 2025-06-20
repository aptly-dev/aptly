from pathlib import Path
import re
import os

from lib import BaseTest


class CreateMirror1Test(BaseTest):
    """
    create mirror: all architectures + all components
    """
    runCmd = "aptly mirror create --ignore-signatures mirror1 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror1", "mirror_show")


class CreateMirror2Test(BaseTest):
    """
    create mirror: all architectures and 1 component
    """
    runCmd = "aptly mirror create --ignore-signatures mirror2  http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch main"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror2", "mirror_show")


class CreateMirror3Test(BaseTest):
    """
    create mirror: some architectures and 2 components
    """
    runCmd = "aptly -architectures=i386,amd64 mirror create --ignore-signatures mirror3 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch main contrib"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror3", "mirror_show")


class CreateMirror4Test(BaseTest):
    """
    create mirror: missing component
    """
    expectedCode = 1

    runCmd = "aptly -architectures=i386,amd64 mirror create --ignore-signatures mirror4 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch life"


class CreateMirror5Test(BaseTest):
    """
    create mirror: missing architecture
    """
    expectedCode = 1

    runCmd = "aptly -architectures=i386,nano68 mirror create --ignore-signatures mirror5 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch"


class CreateMirror6Test(BaseTest):
    """
    create mirror: missing release
    """
    expectedCode = 1
    requiresGPG2 = True

    configOverride = {"max-tries": 1}
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror6 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ suslik"


class CreateMirror7Test(BaseTest):
    """
    create mirror: architectures fixed via config file
    """
    runCmd = "aptly mirror create --ignore-signatures mirror7 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch main contrib"
    configOverride = {"architectures": ["i386", "amd64"]}

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror7", "mirror_show")


class CreateMirror8Test(BaseTest):
    """
    create mirror: already exists
    """
    fixtureCmds = [
        "aptly mirror create --ignore-signatures mirror8 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch main contrib"
    ]
    runCmd = "aptly mirror create --ignore-signatures mirror8 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch main contrib"
    expectedCode = 1


class CreateMirror9Test(BaseTest):
    """
    create mirror: repo with InRelease verification
    """
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror9 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch-backports"
    fixtureGpg = True
    requiresGPG2 = True

    def outputMatchPrepare(self, s):
        return re.sub(r'Signature made .* using|Warning: using insecure memory!\n', '', s)

    def check(self):
        def removeDates(s):
            return re.sub(r"(Date|Valid-Until): [,0-9:+A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly mirror show mirror9",
                              "mirror_show", match_prepare=removeDates)


class CreateMirror10Test(BaseTest):
    """
    create mirror: repo with InRelease verification, failure
    """
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror10 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch-backports"
    fixtureGpg = False
    gold_processor = BaseTest.expand_environ
    expectedCode = 1

    def outputMatchPrepare(self, s):
        return re.sub(r'Signature made .* using|gpgv: keyblock resource .*$|gpgv: Can\'t check signature: .*$', '', s, flags=re.MULTILINE)


class CreateMirror11Test(BaseTest):
    """
    create mirror: repo with Release + Release.gpg verification
    """
    configOverride = {"max-tries": 1}
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror11 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch"
    fixtureGpg = True

    def outputMatchPrepare(self, s):
        return re.sub(r'Signature made .* using', '', s)

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror11", "mirror_show")


class CreateMirror12Test(BaseTest):
    """
    create mirror: repo with Release+Release.gpg verification, failure
    """
    configOverride = {"max-tries": 1}
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror12 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch"
    fixtureGpg = False
    gold_processor = BaseTest.expand_environ
    expectedCode = 1

    def outputMatchPrepare(self, s):
        return re.sub(r'Signature made .* using|gpgv: keyblock resource .*$|gpgv: Can\'t check signature: .*$', '', s, flags=re.MULTILINE)


class CreateMirror13Test(BaseTest):
    """
    create mirror: skip verification using config file
    """
    runCmd = "aptly mirror create mirror13 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch"
    configOverride = {"gpgDisableVerify": True}

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror13", "mirror_show")


class CreateMirror14Test(BaseTest):
    """
    create mirror: flat repository
    """
    runCmd = "aptly mirror create -keyring=aptlytest.gpg mirror14  http://repo.aptly.info/system-tests/cloud.r-project.org/bin/linux/debian bullseye-cran40/"
    fixtureGpg = True

    def outputMatchPrepare(self, s):
        return re.sub(r'Signature made .* using', '', s)

    def check(self):
        def removeDates(s):
            return re.sub(r"(Date|Valid-Until): [,0-9:+A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly mirror show mirror14",
                              "mirror_show", match_prepare=removeDates)


class CreateMirror15Test(BaseTest):
    """
    create mirror: flat repository + components
    """
    runCmd = "aptly mirror create -keyring=aptlytest.gpg mirror14  http://repo.aptly.info/system-tests/cloud.r-project.org/bin/linux/debian bullseye-cran40/ main"
    expectedCode = 1


class CreateMirror16Test(BaseTest):
    """
    create mirror: there's no "source" architecture
    """
    expectedCode = 1

    runCmd = "aptly -architectures=source mirror create -ignore-signatures mirror16 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch"


class CreateMirror17Test(BaseTest):
    """
    create mirror: mirror with sources enabled
    """
    runCmd = "aptly -architectures=i386 mirror create -ignore-signatures -with-sources mirror17 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror17", "mirror_show")


class CreateMirror18Test(BaseTest):
    """
    create mirror: mirror with ppa URL
    """
    fixtureGpg = True
    configOverride = {
        "max-tries": 1,
        "ppaDistributorID": "ubuntu",
        "ppaCodename": "maverick",
    }

    fixtureCmds = [
        "gpg --no-default-keyring --keyring=ppa.gpg --keyserver=hkp://keyserver.ubuntu.com:80 --recv-keys 5BFCD481D86D5824470E469F9000B1C3A01F726C 02219381E9161C78A46CB2BFA5279A973B1F56C0",
        f"chmod 400 {os.path.join(os.environ['HOME'], '.gnupg/ppa.gpg')}"
    ]
    runCmd = "aptly mirror create -keyring=ppa.gpg mirror18 ppa:gladky-anton/gnuplot"

    def outputMatchPrepare(self, s):
        return re.sub(r'Signature made .* using', '', s)

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror18", "mirror_show")


class CreateMirror19Test(BaseTest):
    """
    create mirror: mirror with / in distribution
    """
    fixtureGpg = True

    runCmd = "aptly -architectures='i386' mirror create -keyring=aptlytest.gpg -with-sources mirror19 http://repo.aptly.info/system-tests/archive.debian.org/debian-security/ stretch/updates main"

    def outputMatchPrepare(self, s):
        return re.sub(r'Signature made .* using', '', s)

    def check(self):
        def removeDates(s):
            return re.sub(r"(Date|Valid-Until): [,0-9:+A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly mirror show mirror19",
                              "mirror_show", match_prepare=removeDates)


class CreateMirror20Test(BaseTest):
    """
    create mirror: using failing HTTP_PROXY
    """
    fixtureGpg = True
    configOverride = {"max-tries": 1}

    runCmd = "aptly -architectures='i386' mirror create -keyring=aptlytest.gpg -with-sources mirror20 http://repo.aptly.info/system-tests/archive.debian.org/debian-security/ stretch/updates main"
    environmentOverride = {"HTTP_PROXY": "127.0.0.1:3137"}
    expectedCode = 1

    def outputMatchPrepare(self, s):
        return s.replace(
            'getsockopt: ', ''
        ).replace(
            'connect: ', ''
        ).replace(
            'proxyconnect tcp', 'http: error connecting to proxy http://127.0.0.1:3137'
        ).replace(
            'Get http://repo.aptly.info/system-tests/archive.debian.org/debian-security/dists/stretch/updates/Release:',
            'Get "http://repo.aptly.info/system-tests/archive.debian.org/debian-security/dists/stretch/updates/Release":'
        )


class CreateMirror21Test(BaseTest):
    """
    create mirror: flat repository in subdir
    """
    configOverride = {"max-tries": 1}
    runCmd = "aptly mirror create -keyring=aptlytest.gpg mirror21 http://repo.aptly.info/system-tests/pkg.jenkins.io/debian-stable binary/"
    fixtureGpg = True

    def outputMatchPrepare(self, s):
        return re.sub(r'Signature made .* using', '', s)

    def check(self):
        def removeSHA512(s):
            return re.sub(r"SHA512: .+\n", "", s)

        def removeDates(s):
            return re.sub(r"(Date|Valid-Until): [,0-9:+A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly mirror show mirror21", "mirror_show",
                              match_prepare=lambda s: removeSHA512(removeDates(s)))


class CreateMirror22Test(BaseTest):
    """
    create mirror: mirror with filter
    """
    runCmd = "aptly mirror create -ignore-signatures -filter='nginx | Priority (required)' mirror22 http://repo.aptly.info/system-tests/archive.debian.org/debian-security/ stretch/updates main"

    def check(self):
        def removeDates(s):
            return re.sub(r"(Date|Valid-Until): [,0-9:+A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly mirror show mirror22",
                              "mirror_show", match_prepare=removeDates)


class CreateMirror23Test(BaseTest):
    """
    create mirror: mirror with wrong filter
    """
    runCmd = "aptly mirror create -ignore-signatures -filter='nginx | ' mirror23 http://repo.aptly.info/system-tests/archive.debian.org/debian-security/ stretch/updates main"
    expectedCode = 1


class CreateMirror24Test(BaseTest):
    """
    create mirror: disable config value with option
    """
    runCmd = "aptly mirror create -ignore-signatures=false -keyring=aptlytest.gpg mirror24 http://repo.aptly.info/system-tests/archive.debian.org/debian-security/ stretch/updates main"
    fixtureGpg = True

    def outputMatchPrepare(self, s):
        return re.sub(r'Signature made .* using', '', s)

    configOverride = {
        "gpgDisableVerify": True
    }


class CreateMirror25Test(BaseTest):
    """
    create mirror: mirror with udebs enabled
    """
    runCmd = "aptly -architectures=i386 mirror create -ignore-signatures -with-udebs mirror25 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror25", "mirror_show")


class CreateMirror26Test(BaseTest):
    """
    create mirror: flat mirror with udebs
    """
    runCmd = "aptly mirror create -keyring=aptlytest.gpg -with-udebs mirror26 http://repo.aptly.info/system-tests/pkg.jenkins.io/debian-stable binary/"
    fixtureGpg = True
    expectedCode = 1


class CreateMirror27Test(BaseTest):
    """
    create mirror: component with slashes, no stripping
    """
    runCmd = "aptly mirror create --ignore-signatures mirror27 http://repo.aptly.info/system-tests/mirror.chpc.utah.edu/pub/linux.dell.com/repo/community/ubuntu wheezy openmanage/740"

    def outputMatchPrepare(self, s):
        return self.strip_retry_lines(s)

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror27", "mirror_show")


class CreateMirror29Test(BaseTest):
    """
    create mirror: repo with InRelease verification (internal GPG implementation)
    """
    fixtureCmds = ["gpg --no-default-keyring --keyring aptlytest.gpg --export-options export-minimal --export -o " + os.path.join(
                     os.environ["HOME"], ".gnupg/aptlytest-gpg1.gpg")]
    runCmd = "aptly mirror create --keyring=aptlytest-gpg1.gpg mirror9 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch-backports"
    configOverride = {"gpgProvider": "internal"}
    fixtureGpg = True

    def outputMatchPrepare(self, s):
        return re.sub(r'Signature made .* using', '', s)

    def teardown(self):
        self.run_cmd(["rm", "-f", os.path.join(os.environ["HOME"], ".gnupg/aptlytest-gpg1.gpg")])


class CreateMirror30Test(BaseTest):
    """
    create mirror: repo with InRelease verification, failure  (internal GPG implementation)
    """
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror10 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch"
    configOverride = {"gpgProvider": "internal", "max-tries": 1}
    gold_processor = BaseTest.expand_environ
    fixtureGpg = False
    expectedCode = 1

    def outputMatchPrepare(self, s):
        return re.sub(r'Signature made .* using', '', s)


class CreateMirror31Test(BaseTest):
    """
    create mirror: repo with Release + Release.gpg verification (internal GPG implementation)
    """
    fixtureCmds = ["gpg --no-default-keyring --keyring aptlytest.gpg --export-options export-minimal --export -o " + os.path.join(
                     os.environ["HOME"], ".gnupg/aptlytest-gpg1.gpg")]
    runCmd = "aptly mirror create --keyring=aptlytest-gpg1.gpg mirror11 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch"
    configOverride = {"gpgProvider": "internal", "max-tries": 1}
    fixtureGpg = True
    faketime = True

    def outputMatchPrepare(self, s):
        return re.sub(r'Signature made .* using', '', s)

    def teardown(self):
        self.run_cmd(["rm", "-f", os.path.join(os.environ["HOME"], ".gnupg/aptlytest-gpg1.gpg")])


class CreateMirror32Test(BaseTest):
    """
    create mirror: repo with Release + Release.gpg verification (gpg2)
    """
    configOverride = {"max-tries": 1}
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror32 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch"
    fixtureGpg = True
    requiresGPG2 = True

    def outputMatchPrepare(self, s):
        return \
            re.sub(r'([A-F0-9]{8})[A-F0-9]{8}', r'\1',
                   re.sub(r'^gpgv: (Signature made .+|.+using RSA key.+)\n', '', s, flags=re.MULTILINE))

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror32", "mirror_show")


class CreateMirror33Test(BaseTest):
    """
    create mirror: repo with only InRelease file but no verification
    """
    configOverride = {"max-tries": 1}
    runCmd = "aptly mirror create -ignore-signatures mirror33 http://repo.aptly.info/system-tests/nvidia.github.io/libnvidia-container/stable/ubuntu16.04/amd64 ./"
    fixtureGpg = False
    requiresGPG2 = False

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror33", "mirror_show")


class CreateMirror34Test(BaseTest):
    """
    create mirror error: flat repo with filter but no architectures in InRelease file
    """
    configOverride = {"max-tries": 1}
    runCmd = "aptly mirror create -ignore-signatures -filter \"cuda-12-6 (= 12.6.2-1)\" -filter-with-deps mirror34 http://repo.aptly.info/system-tests/developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64/ ./"


class CreateMirror35Test(BaseTest):
    """
    create mirror: flat repo with filter but no architectures in InRelease file
    """
    configOverride = {"max-tries": 1}
    fixtureCmds = [
        "aptly mirror create -architectures amd64 -ignore-signatures -filter \"cuda-12-6 (= 12.6.2-1)\" -filter-with-deps mirror35 "
        "http://repo.aptly.info/system-tests/developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64/ ./",
    ]
    runCmd = "aptly mirror update -ignore-signatures mirror35"

    # the downloading of the actual packages will return 404 since they don't exist. ignore the errors, the test verifies proper count of filtered packages

    def outputMatchPrepare(self, s):
        s = re.sub(r'Downloading: .*\n', '', s, flags=re.MULTILINE)
        s = re.sub(r'Download Error: .*\n', '', s, flags=re.MULTILINE)
        s = re.sub(r'Retrying .*\n', '', s, flags=re.MULTILINE)
        s = re.sub(r'Error \(retrying\): .*\n', '', s, flags=re.MULTILINE)
        s = re.sub(r'HTTP code 404 while fetching .*\n', '', s, flags=re.MULTILINE)
        s = re.sub(r'ERROR: unable to update: .*\n', '', s, flags=re.MULTILINE)
        return s

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror35", "mirror_show")


class CreateMirror36Test(BaseTest):
    """
    create mirror: mirror with filter read from file
    """
    filterFilePath = os.path.join(os.environ["HOME"], ".aptly-filter.tmp")
    fixtureCmds = [f"bash -c \"echo -n 'nginx | Priority (required)' > {filterFilePath}\""]
    runCmd = f"aptly mirror create -ignore-signatures -filter='@{filterFilePath}' mirror36 http://repo.aptly.info/system-tests/archive.debian.org/debian-security/ stretch/updates main"

    def check(self):
        def removeDates(s):
            return re.sub(r"(Date|Valid-Until): [,0-9:+A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly mirror show mirror36",
                              "mirror_show", match_prepare=removeDates)


class CreateMirror37Test(BaseTest):
    """
    create mirror: mirror with filter read from stdin
    """
    aptly_testing_bin = Path(__file__).parent.parent.parent / "aptly.test"
    # Hack: Normally the test system detects if runCmd is an aptly command and then
    # substitutes the aptly_testing_bin path and deletes the last three lines of output.
    # However, I need to run it in bash to control stdin, so I have to do it manually.
    runCmd = [
        "bash",
        "-c",
        f"echo -n 'nginx | Priority (required)' | {aptly_testing_bin} mirror create " +
        "-ignore-signatures -filter='@-' mirror37 http://repo.aptly.info/system-tests/archive.debian.org/debian-security/ stretch/updates main " +
        "| grep -vE '^(EXIT|PASS|coverage:)'"
    ]

    def check(self):
        def removeDates(s):
            return re.sub(r"(Date|Valid-Until): [,0-9:+A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly mirror show mirror37",
                              "mirror_show", match_prepare=removeDates)
