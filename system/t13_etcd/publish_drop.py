# reuse existing tests:
from t06_publish.drop import PublishDrop1Test, \
                             PublishDrop2Test, \
                             PublishDrop3Test, \
                             PublishDrop4Test, \
                             PublishDrop5Test, \
                             PublishDrop6Test, \
                             PublishDrop7Test, \
                             PublishDrop8Test, \
                             PublishDrop9Test

TEST_IGNORE = ["PublishDrop1Test",
               "PublishDrop2Test",
               "PublishDrop3Test",
               "PublishDrop4Test",
               "PublishDrop5Test",
               "PublishDrop6Test",
               "PublishDrop7Test",
               "PublishDrop8Test",
               "PublishDrop9Test"]


class PublishDrop1TestEtcd(PublishDrop1Test):
    """
    publish drop: existing snapshot
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class PublishDrop2TestEtcd(PublishDrop2Test):
    """
    publish drop: under prefix
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class PublishDrop3TestEtcd(PublishDrop3Test):
    """
    publish drop: drop one distribution
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class PublishDrop4TestEtcd(PublishDrop4Test):
    """
    publish drop: drop one of components
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class PublishDrop5TestEtcd(PublishDrop5Test):
    """
    publish drop: component cleanup
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class PublishDrop6TestEtcd(PublishDrop6Test):
    """
    publish drop: no publish
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class PublishDrop7TestEtcd(PublishDrop7Test):
    """
    publish drop: under prefix with trailing & leading slashes
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class PublishDrop8TestEtcd(PublishDrop8Test):
    """
    publish drop: skip component cleanup
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class PublishDrop9TestEtcd(PublishDrop9Test):
    """
    publish drop: component cleanup after first cleanup skipped
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"
