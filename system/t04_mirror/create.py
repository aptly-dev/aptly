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

    runCmd = "aptly mirror create mirror6 http://mirror.yandex.ru/debian/ suslik"


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
