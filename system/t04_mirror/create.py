import re

from lib import BaseTest


class CreateMirror1Test(BaseTest):
    """
    create mirror: all architectures + all components
    """
    runCmd = "aptly mirror create --ignore-signatures mirror1 http://mirror.yandex.ru/debian/ wheezy"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror1", "mirror_show")


class CreateMirror2Test(BaseTest):
    """
    create mirror: all architectures and 1 component
    """
    runCmd = "aptly mirror create --ignore-signatures mirror2  http://mirror.yandex.ru/debian/ wheezy main"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror2", "mirror_show")


class CreateMirror3Test(BaseTest):
    """
    create mirror: some architectures and 2 components
    """
    runCmd = "aptly -architectures=i386,amd64 mirror create --ignore-signatures mirror3 http://mirror.yandex.ru/debian/ wheezy main contrib"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror3", "mirror_show")


class CreateMirror4Test(BaseTest):
    """
    create mirror: missing component
    """
    expectedCode = 1

    runCmd = "aptly -architectures=i386,amd64 mirror create --ignore-signatures mirror4 http://mirror.yandex.ru/debian/ wheezy life"


class CreateMirror5Test(BaseTest):
    """
    create mirror: missing architecture
    """
    expectedCode = 1

    runCmd = "aptly -architectures=i386,nano68 mirror create --ignore-signatures mirror5 http://mirror.yandex.ru/debian/ wheezy"


class CreateMirror6Test(BaseTest):
    """
    create mirror: missing release
    """
    expectedCode = 1

    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror6 http://mirror.yandex.ru/debian/ suslik"


class CreateMirror7Test(BaseTest):
    """
    create mirror: architectures fixed via config file
    """
    runCmd = "aptly mirror create --ignore-signatures mirror7 http://mirror.yandex.ru/debian/ wheezy main contrib"
    configOverride = {"architectures": ["i386", "amd64"]}

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror7", "mirror_show")


class CreateMirror8Test(BaseTest):
    """
    create mirror: already exists
    """
    fixtureCmds = [
        "aptly mirror create --ignore-signatures mirror8 http://mirror.yandex.ru/debian/ wheezy main contrib"
    ]
    runCmd = "aptly mirror create --ignore-signatures mirror8 http://mirror.yandex.ru/debian/ wheezy main contrib"
    expectedCode = 1


class CreateMirror9Test(BaseTest):
    """
    create mirror: repo with InRelease verification
    """
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror9 http://mirror.yandex.ru/debian/ wheezy-backports"
    fixtureGpg = True
    outputMatchPrepare = lambda _, s: re.sub(r'Signature made .* using|Warning: using insecure memory!\n', '', s)

    def check(self):
        def removeDates(s):
            return re.sub(r"(Date|Valid-Until): [,0-9:+A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly mirror show mirror9", "mirror_show", match_prepare=removeDates)


class CreateMirror10Test(BaseTest):
    """
    create mirror: repo with InRelease verification, failure
    """
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror10 http://mirror.yandex.ru/debian-backports/ squeeze-backports"
    fixtureGpg = False
    gold_processor = BaseTest.expand_environ
    outputMatchPrepare = lambda _, s: re.sub(r'Signature made .* using|gpgv: keyblock resource .*$|gpgv: Can\'t check signature: .*$', '', s, flags=re.MULTILINE)
    expectedCode = 1


class CreateMirror11Test(BaseTest):
    """
    create mirror: repo with Release + Release.gpg verification
    """
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror11 http://mirror.yandex.ru/debian/ wheezy"
    fixtureGpg = True
    outputMatchPrepare = lambda _, s: re.sub(r'Signature made .* using', '', s)

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror11", "mirror_show")


class CreateMirror12Test(BaseTest):
    """
    create mirror: repo with Release+Release.gpg verification, failure
    """
    runCmd = "aptly mirror create --keyring=aptlytest.gpg mirror12 http://mirror.yandex.ru/debian/ wheezy"
    fixtureGpg = False
    gold_processor = BaseTest.expand_environ
    outputMatchPrepare = lambda _, s: re.sub(r'Signature made .* using|gpgv: keyblock resource .*$|gpgv: Can\'t check signature: .*$', '', s, flags=re.MULTILINE)
    expectedCode = 1


class CreateMirror13Test(BaseTest):
    """
    create mirror: skip verification using config file
    """
    runCmd = "aptly mirror create mirror13 http://mirror.yandex.ru/debian/ wheezy"
    configOverride = {"gpgDisableVerify": True}

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror13", "mirror_show")


class CreateMirror14Test(BaseTest):
    """
    create mirror: flat repository
    """
    runCmd = "aptly mirror create -keyring=aptlytest.gpg mirror14 http://download.opensuse.org/repositories/home:/monkeyiq/Debian_7.0/ ./"
    fixtureGpg = True
    outputMatchPrepare = lambda _, s: re.sub(r'Signature made .* using', '', s)

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror14", "mirror_show")


class CreateMirror15Test(BaseTest):
    """
    create mirror: flat repository + components
    """
    runCmd = "aptly mirror create -keyring=aptlytest.gpg mirror14 http://download.opensuse.org/repositories/home:/monkeyiq/Debian_7.0/ ./ main"
    expectedCode = 1


class CreateMirror16Test(BaseTest):
    """
    create mirror: there's no "source" architecture
    """
    expectedCode = 1

    runCmd = "aptly -architectures=source mirror create -ignore-signatures mirror16 http://mirror.yandex.ru/debian/ wheezy"


class CreateMirror17Test(BaseTest):
    """
    create mirror: mirror with sources enabled
    """
    runCmd = "aptly -architectures=i386 mirror create -ignore-signatures -with-sources mirror17 http://mirror.yandex.ru/debian/ wheezy"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror17", "mirror_show")


class CreateMirror18Test(BaseTest):
    """
    create mirror: mirror with ppa URL
    """
    fixtureGpg = True
    outputMatchPrepare = lambda _, s: re.sub(r'Signature made .* using', '', s)

    configOverride = {
        "ppaDistributorID": "ubuntu",
        "ppaCodename": "maverick",
    }

    runCmd = "aptly mirror create -keyring=aptlytest.gpg mirror18 ppa:gladky-anton/gnuplot"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror18", "mirror_show")


class CreateMirror19Test(BaseTest):
    """
    create mirror: mirror with / in distribution
    """
    fixtureGpg = True
    outputMatchPrepare = lambda _, s: re.sub(r'Signature made .* using', '', s)

    runCmd = "aptly -architectures='i386' mirror create -keyring=aptlytest.gpg -with-sources mirror19 http://security.debian.org/ wheezy/updates main"

    def check(self):
        def removeDates(s):
            return re.sub(r"(Date|Valid-Until): [,0-9:+A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly mirror show mirror19", "mirror_show", match_prepare=removeDates)


class CreateMirror20Test(BaseTest):
    """
    create mirror: using failing HTTP_PROXY
    """
    fixtureGpg = True
    outputMatchPrepare = lambda _, s: s.replace('getsockopt: ', '')

    runCmd = "aptly -architectures='i386' mirror create -keyring=aptlytest.gpg -with-sources mirror20 http://security.debian.org/ wheezy/updates main"
    environmentOverride = {"HTTP_PROXY": "127.0.0.1:3137"}
    expectedCode = 1


class CreateMirror21Test(BaseTest):
    """
    create mirror: flat repository in subdir
    """
    runCmd = "aptly mirror create -keyring=aptlytest.gpg mirror21 http://pkg.jenkins-ci.org/debian-stable binary/"
    fixtureGpg = True
    outputMatchPrepare = lambda _, s: re.sub(r'Signature made .* using', '', s)

    def check(self):
        def removeSHA512(s):
            return re.sub(r"SHA512: .+\n", "", s)

        def removeDates(s):
            return re.sub(r"(Date|Valid-Until): [,0-9:+A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly mirror show mirror21", "mirror_show", match_prepare=lambda s: removeSHA512(removeDates(s)))


class CreateMirror22Test(BaseTest):
    """
    create mirror: mirror with filter
    """
    runCmd = "aptly mirror create -ignore-signatures -filter='nginx | Priority (required)' mirror22 http://security.debian.org/ wheezy/updates main"

    def check(self):
        def removeDates(s):
            return re.sub(r"(Date|Valid-Until): [,0-9:+A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly mirror show mirror22", "mirror_show", match_prepare=removeDates)


class CreateMirror23Test(BaseTest):
    """
    create mirror: mirror with wrong filter
    """
    runCmd = "aptly mirror create -ignore-signatures -filter='nginx | ' mirror23 http://security.debian.org/ wheezy/updates main"
    expectedCode = 1


class CreateMirror24Test(BaseTest):
    """
    create mirror: disable config value with option
    """
    runCmd = "aptly mirror create -ignore-signatures=false -keyring=aptlytest.gpg mirror24 http://security.debian.org/ wheezy/updates main"
    fixtureGpg = True
    outputMatchPrepare = lambda _, s: re.sub(r'Signature made .* using', '', s)

    configOverride = {
        "gpgDisableVerify": True
    }


class CreateMirror25Test(BaseTest):
    """
    create mirror: mirror with udebs enabled
    """
    runCmd = "aptly -architectures=i386 mirror create -ignore-signatures -with-udebs mirror25 http://mirror.yandex.ru/debian/ wheezy"

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
    runCmd = "aptly mirror create --ignore-signatures mirror27 http://linux.dell.com/repo/community/ubuntu wheezy openmanage/740"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly mirror show mirror27", "mirror_show")


class CreateMirror28Test(BaseTest):
    """
    create mirror: -force-components
    """
    runCmd = "aptly mirror create -ignore-signatures -force-components mirror28 http://downloads-distro.mongodb.org/repo/ubuntu-upstart dist 10gen"

    def check(self):
        def removeDates(s):
            return re.sub(r"(Date|Valid-Until): [,0-9:+A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly mirror show mirror28", "mirror_show", match_prepare=removeDates)
