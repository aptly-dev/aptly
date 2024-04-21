# reuse existing tests:
from t04_mirror.update import UpdateMirror1Test, \
        UpdateMirror2Test, \
        UpdateMirror3Test, \
        UpdateMirror4Test, \
        UpdateMirror5Test, \
        UpdateMirror6Test, \
        UpdateMirror7Test, \
        UpdateMirror8Test, \
        UpdateMirror9Test, \
        UpdateMirror10Test, \
        UpdateMirror11Test, \
        UpdateMirror12Test, \
        UpdateMirror13Test, \
        UpdateMirror14Test, \
        UpdateMirror17Test, \
        UpdateMirror18Test, \
        UpdateMirror19Test, \
        UpdateMirror20Test, \
        UpdateMirror21Test, \
        UpdateMirror22Test, \
        UpdateMirror23Test, \
        UpdateMirror24Test, \
        UpdateMirror25Test


TEST_IGNORE = ["UpdateMirror1Test",
               "UpdateMirror2Test",
               "UpdateMirror3Test",
               "UpdateMirror4Test",
               "UpdateMirror5Test",
               "UpdateMirror6Test",
               "UpdateMirror7Test",
               "UpdateMirror8Test",
               "UpdateMirror9Test",
               "UpdateMirror10Test",
               "UpdateMirror11Test",
               "UpdateMirror12Test",
               "UpdateMirror13Test",
               "UpdateMirror14Test",
               "UpdateMirror17Test",
               "UpdateMirror18Test",
               "UpdateMirror19Test",
               "UpdateMirror20Test",
               "UpdateMirror21Test",
               "UpdateMirror22Test",
               "UpdateMirror23Test",
               "UpdateMirror24Test",
               "UpdateMirror25Test"
               ]


class UpdateMirror1TestEtcd(UpdateMirror1Test):
    """
    update mirrors: regular update
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror2TestEtcd(UpdateMirror2Test):
    """
    update mirrors: no such repo
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror3TestEtcd(UpdateMirror3Test):
    """
    update mirrors: wrong checksum in release file
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror4TestEtcd(UpdateMirror4Test):
    """
    update mirrors: wrong checksum in release file, but ignore
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror5TestEtcd(UpdateMirror5Test):
    """
    update mirrors: wrong checksum in package
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror6TestEtcd(UpdateMirror6Test):
    """
    update mirrors: wrong checksum in package, but ignore
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror7TestEtcd(UpdateMirror7Test):
    """
    update mirrors: flat repository
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror8TestEtcd(UpdateMirror8Test):
    """
    update mirrors: with sources (already in pool)
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror9TestEtcd(UpdateMirror9Test):
    """
    update mirrors: flat repository + sources
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror10TestEtcd(UpdateMirror10Test):
    """
    update mirrors: filtered
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror11TestEtcd(UpdateMirror11Test):
    """
    update mirrors: update over FTP
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror12TestEtcd(UpdateMirror12Test):
    """
    update mirrors: update with udebs
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror13TestEtcd(UpdateMirror13Test):
    """
    update mirrors: regular update with --skip-existing-packages option
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror14TestEtcd(UpdateMirror14Test):
    """
    update mirrors: regular update with --skip-existing-packages option
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror17TestEtcd(UpdateMirror17Test):
    """
    update mirrors: update for mirror but with file in pool on legacy MD5 location
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror18TestEtcd(UpdateMirror18Test):
    """
    update mirrors: update for mirror but with file in pool on legacy MD5 location and disabled legacy path support
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror19TestEtcd(UpdateMirror19Test):
    """
    update mirrors: correct matching of Release checksums
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror20TestEtcd(UpdateMirror20Test):
    """
    update mirrors: flat repository (internal GPG implementation)
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror21TestEtcd(UpdateMirror21Test):
    """
    update mirrors: correct matching of Release checksums (internal pgp implementation)
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror22TestEtcd(UpdateMirror22Test):
    """
    update mirrors: SHA512 checksums only
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror23TestEtcd(UpdateMirror23Test):
    """
    update mirrors: update with installer
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror24TestEtcd(UpdateMirror24Test):
    """
    update mirrors: update with installer with separate gpg file
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class UpdateMirror25TestEtcd(UpdateMirror25Test):
    """
    update mirrors: mirror with / in distribution
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"
