import re

from lib import BaseTest


class CreateMirror1Test(BaseTest):
    """
    create mirror: all architectures + all components
    """
    runCmd = "aptly mirror create --ignore-signatures mirror1 http://archive.debian.org/debian-archive/debian/ stretch"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror1", "mirror_show")


class CreateMirror2Test(BaseTest):
    """
    create mirror: all architectures and 1 component
    """
    runCmd = "aptly mirror create --ignore-signatures mirror2  http://archive.debian.org/debian-archive/debian/ stretch main"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror2", "mirror_show")


class CreateMirror3Test(BaseTest):
    """
    create mirror: some architectures and 2 components
    """
    runCmd = "aptly -architectures=i386,amd64 mirror create --ignore-signatures mirror3 http://archive.debian.org/debian-archive/debian/ stretch main contrib"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror3", "mirror_show")


class CreateMirror4Test(BaseTest):
    """
    create mirror: missing component
    """
    expectedCode = 1

    runCmd = "aptly -architectures=i386,amd64 mirror create --ignore-signatures mirror4 http://archive.debian.org/debian-archive/debian/ stretch life"


class CreateMirror5Test(BaseTest):
    """
    create mirror: missing architecture
    """
    expectedCode = 1

    runCmd = "aptly -architectures=i386,nano68 mirror create --ignore-signatures mirror5 http://archive.debian.org/debian-archive/debian/ stretch"


class CreateMirror6Test(BaseTest):
    """
    create mirror: missing release
    """
    expectedCode = 1
    requiresGPG1 = True

    configOverride = {"max-tries": 1}
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror6 http://archive.debian.org/debian-archive/debian/ suslik"


class CreateMirror7Test(BaseTest):
    """
    create mirror: architectures fixed via config file
    """
    runCmd = "aptly mirror create --ignore-signatures mirror7 http://archive.debian.org/debian-archive/debian/ stretch main contrib"
    configOverride = {"architectures": ["i386", "amd64"]}

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror7", "mirror_show")


class CreateMirror8Test(BaseTest):
    """
    create mirror: already exists
    """
    fixtureCmds = [
        "aptly mirror create --ignore-signatures mirror8 http://archive.debian.org/debian-archive/debian/ stretch main contrib"
    ]
    runCmd = "aptly mirror create --ignore-signatures mirror8 http://archive.debian.org/debian-archive/debian/ stretch main contrib"
    expectedCode = 1


class CreateMirror9Test(BaseTest):
    """
    create mirror: repo with InRelease verification
    """
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror9 http://archive.debian.org/debian-archive/debian/ stretch-backports"
    fixtureGpg = True
    requiresGPG1 = True

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
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror10 http://archive.debian.org/debian-archive/debian/ stretch-backports"
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
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror11 http://archive.debian.org/debian-archive/debian/ stretch"
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
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror12 http://archive.debian.org/debian-archive/debian/ stretch"
    fixtureGpg = False
    gold_processor = BaseTest.expand_environ
    expectedCode = 1

    def outputMatchPrepare(self, s):
        return re.sub(r'Signature made .* using|gpgv: keyblock resource .*$|gpgv: Can\'t check signature: .*$', '', s, flags=re.MULTILINE)


class CreateMirror13Test(BaseTest):
    """
    create mirror: skip verification using config file
    """
    runCmd = "aptly mirror create mirror13 http://archive.debian.org/debian-archive/debian/ stretch"
    configOverride = {"gpgDisableVerify": True}

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror13", "mirror_show")


class CreateMirror14Test(BaseTest):
    """
    create mirror: flat repository
    """
    runCmd = "aptly mirror create -keyring=aptlytest.gpg mirror14  https://cloud.r-project.org/bin/linux/debian jessie-cran35/"
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
    runCmd = "aptly mirror create -keyring=aptlytest.gpg mirror14  https://cloud.r-project.org/bin/linux/debian jessie-cran35/ main"
    expectedCode = 1


class CreateMirror16Test(BaseTest):
    """
    create mirror: there's no "source" architecture
    """
    expectedCode = 1

    runCmd = "aptly -architectures=source mirror create -ignore-signatures mirror16 http://archive.debian.org/debian-archive/debian/ stretch"


class CreateMirror17Test(BaseTest):
    """
    create mirror: mirror with sources enabled
    """
    runCmd = "aptly -architectures=i386 mirror create -ignore-signatures -with-sources mirror17 http://archive.debian.org/debian-archive/debian/ stretch"

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

    runCmd = "aptly mirror create -keyring=aptlytest.gpg mirror18 ppa:gladky-anton/gnuplot"

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

    runCmd = "aptly -architectures='i386' mirror create -keyring=aptlytest.gpg -with-sources mirror19 http://archive.debian.org/debian-security/ stretch/updates main"

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

    runCmd = "aptly -architectures='i386' mirror create -keyring=aptlytest.gpg -with-sources mirror20 http://archive.debian.org/debian-security/ stretch/updates main"
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
            'Get http://archive.debian.org/debian-security/dists/stretch/updates/Release:',
            'Get "http://archive.debian.org/debian-security/dists/stretch/updates/Release":'
        )


class CreateMirror21Test(BaseTest):
    """
    create mirror: flat repository in subdir
    """
    configOverride = {"max-tries": 1}
    runCmd = "aptly mirror create -keyring=aptlytest.gpg mirror21 http://pkg.jenkins-ci.org/debian-stable binary/"
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
    runCmd = "aptly mirror create -ignore-signatures -filter='nginx | Priority (required)' mirror22 http://archive.debian.org/debian-security/ stretch/updates main"

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
    runCmd = "aptly mirror create -ignore-signatures -filter='nginx | ' mirror23 http://archive.debian.org/debian-security/ stretch/updates main"
    expectedCode = 1


class CreateMirror24Test(BaseTest):
    """
    create mirror: disable config value with option
    """
    runCmd = "aptly mirror create -ignore-signatures=false -keyring=aptlytest.gpg mirror24 http://archive.debian.org/debian-security/ stretch/updates main"
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
    runCmd = "aptly -architectures=i386 mirror create -ignore-signatures -with-udebs mirror25 http://archive.debian.org/debian-archive/debian/ stretch"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror25", "mirror_show")


class CreateMirror26Test(BaseTest):
    """
    create mirror: flat mirror with udebs
    """
    runCmd = "aptly mirror create -keyring=aptlytest.gpg -with-udebs mirror26 http://pkg.jenkins-ci.org/debian-stable binary/"
    fixtureGpg = True
    expectedCode = 1


class CreateMirror27Test(BaseTest):
    """
    create mirror: component with slashes, no stripping
    """
    runCmd = "aptly mirror create --ignore-signatures mirror27 https://mirror.chpc.utah.edu/pub/linux.dell.com/repo/community/ubuntu wheezy openmanage/740"

    def outputMatchPrepare(self, s):
        return self.strip_retry_lines(s)

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror27", "mirror_show")


class CreateMirror29Test(BaseTest):
    """
    create mirror: repo with InRelease verification (internal GPG implementation)
    """
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror9 http://archive.debian.org/debian-archive/debian/ stretch-backports"
    configOverride = {"gpgProvider": "internal"}
    fixtureGpg = True

    def outputMatchPrepare(self, s):
        return re.sub(r'Signature made .* using', '', s)


class CreateMirror30Test(BaseTest):
    """
    create mirror: repo with InRelease verification, failure  (internal GPG implementation)
    """
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror10 http://archive.debian.org/debian-archive/debian/ stretch"
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
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror11 http://archive.debian.org/debian-archive/debian/ stretch"
    configOverride = {"gpgProvider": "internal", "max-tries": 1}
    fixtureGpg = True

    def outputMatchPrepare(self, s):
        return re.sub(r'Signature made .* using', '', s)


class CreateMirror32Test(BaseTest):
    """
    create mirror: repo with Release + Release.gpg verification (gpg2)
    """
    configOverride = {"max-tries": 1}
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror32 http://archive.debian.org/debian-archive/debian/ stretch"
    fixtureGpg = True
    requiresGPG2 = True

    def outputMatchPrepare(self, s):
        return \
            re.sub(r'([A-F0-9]{8})[A-F0-9]{8}', r'\1',
                   re.sub(r'^gpgv: (Signature made .+|.+using RSA key.+)\n', '', s, flags=re.MULTILINE))

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror32", "mirror_show")
