from lib import BaseTest


class ListMirror1Test(BaseTest):
    """
    list mirrors: regular list
    """
    fixtureCmds = [
        "aptly mirror create --ignore-signatures mirror1 http://mirror.yandex.ru/debian/ wheezy",
        "aptly mirror create -with-sources --ignore-signatures mirror2 http://mirror.yandex.ru/debian/ squeeze contrib",
        "aptly -architectures=i386 mirror create --ignore-signatures mirror3 http://mirror.yandex.ru/debian/ squeeze non-free",
        "aptly mirror create -ignore-signatures mirror4 http://download.opensuse.org/repositories/home:/DeepDiver1975/xUbuntu_10.04/ ./",
    ]
    runCmd = "aptly mirror list"


class ListMirror2Test(BaseTest):
    """
    list mirrors: empty list
    """
    runCmd = "aptly mirror list"
