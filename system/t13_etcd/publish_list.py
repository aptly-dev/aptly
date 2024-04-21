# reuse existing tests:
from t06_publish.list import PublishList1Test, \
                             PublishList2Test, \
                             PublishList3Test, \
                             PublishList4Test, \
                             PublishList5Test

TEST_IGNORE = ["PublishList1Test", "PublishList2Test", "PublishList3Test", "PublishList4Test", "PublishList5Test"]


class PublishList1TestEtcd(PublishList1Test):
    """
    publish list: empty list
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class PublishList2TestEtcd(PublishList2Test):
    """
    publish list: several repos list
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class PublishList3TestEtcd(PublishList3Test):
    """
    publish list: several repos list, raw
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class PublishList4TestEtcd(PublishList4Test):
    """
    publish list json: empty list
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"


class PublishList5TestEtcd(PublishList5Test):
    """
    publish list json: several repos list
    """
    databaseType = "etcd"
    databaseUrl = "127.0.0.1:2379"
