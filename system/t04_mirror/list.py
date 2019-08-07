from lib import BaseTest


class ListMirror1Test(BaseTest):
    """
    list mirrors: regular list
    """
    fixtureCmds = [
        "aptly mirror create --ignore-signatures mirror1 http://cdn-fastly.deb.debian.org/debian/ stretch",
        "aptly mirror create -with-sources --ignore-signatures mirror2 http://cdn-fastly.deb.debian.org/debian/ stretch contrib",
        "aptly -architectures=i386 mirror create --ignore-signatures mirror3 http://cdn-fastly.deb.debian.org/debian/ stretch non-free",
        "aptly mirror create -ignore-signatures mirror4 http://download.opensuse.org/repositories/Apache:/MirrorBrain/Debian_9.0/ ./",
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
