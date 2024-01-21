import re

from lib import BaseTest


class ListMirror1Test(BaseTest):
    """
    list mirrors: regular list
    """
    fixtureCmds = [
        "aptly mirror create --ignore-signatures mirror1 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch",
        "aptly mirror create -with-sources --ignore-signatures mirror2 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch contrib",
        "aptly -architectures=i386 mirror create --ignore-signatures mirror3 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch non-free",
        "aptly mirror create -ignore-signatures mirror4 http://repo.aptly.info/system-tests/download.opensuse.org/repositories/Apache:/MirrorBrain/Debian_9.0/ ./",
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


class ListMirror5Test(BaseTest):
    """
    list mirrors: json empty list
    """
    runCmd = "aptly mirror list -json"


class ListMirror6Test(BaseTest):
    """
    list mirrors: regular list
    """
    fixtureCmds = [
        "aptly mirror create --ignore-signatures mirror1 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch",
        "aptly mirror create -with-sources --ignore-signatures mirror2 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch contrib",
        "aptly -architectures=i386 mirror create --ignore-signatures mirror3 http://repo.aptly.info/system-tests/archive.debian.org/debian-archive/debian/ stretch non-free",
        "aptly mirror create -ignore-signatures mirror4 http://repo.aptly.info/system-tests/download.opensuse.org/repositories/Apache:/MirrorBrain/Debian_9.0/ ./",
    ]
    runCmd = "aptly mirror list -json"

    def outputMatchPrepare(_, s):
        return re.sub(r'[ ]*"UUID": "[\w-]+",?\n', '', s)
