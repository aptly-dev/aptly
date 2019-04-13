from lib import BaseTest


class ListMirror1Test(BaseTest):
    """
    list mirrors: regular list
    """
    fixtureCmds = [
        "aptly mirror create --ignore-signatures mirror1 http://archive.debian.org/debian/ wheezy",
        "aptly mirror create -with-sources --ignore-signatures mirror2 http://archive.debian.org/debian/ wheezy contrib",
        "aptly -architectures=i386 mirror create --ignore-signatures mirror3 http://archive.debian.org/debian/ wheezy non-free",
        "aptly mirror create -ignore-signatures mirror4 http://download.opensuse.org/repositories/Apache:/MirrorBrain/Debian_7.0/ ./",
    ]
    runCmd = "aptly mirror list"


class ListMirror2Test(BaseTest):
    """
    list mirrors: empty list
    """
    runCmd = "aptly mirror list"


class ListMirror3Test(BaseTest):
    """
    list mirrors: raw list
    """
    fixtureDB = True
    runCmd = "aptly -raw mirror list"


class ListMirror4Test(BaseTest):
    """
    list mirrors: raw empty list
    """
    runCmd = "aptly -raw mirror list"
