import time
from api_lib import APITest

class MirrorsAPITestCreateShow(APITest):
    """
    POST /api/mirrors, GET /api/mirrors/:name/packages
    """
    def check(self):
        mirror_name = self.random_name()
        mirror_desc = {u'Name': mirror_name,
                       u'ArchiveURL': 'http://mirror.yandex.ru/debian/',
                       u'Architectures': ['amd64'],
                       u'Components': ['non-free'],
                       u'Distribution': 'oldstable-proposed-updates'}

        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 403)
        self.check_equal([{'error':'unable to fetch mirror: verification of detached signature failed: exit status 2',
                          'meta':'Operation aborted'}], resp.json())

        mirror_desc[u'IgnoreSignatures'] = True
        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 201)

        resp = self.get("/api/mirrors/" + mirror_name)
        self.check_equal(resp.status_code, 200)
        self.check_subset({u'Name': mirror_name,
                           u'ArchiveRoot': 'http://mirror.yandex.ru/debian/',
                           u'Architectures': ['amd64'],
                           u'Components': ['non-free'],
                           u'Distribution': 'oldstable-proposed-updates'}, resp.json())

        resp = self.get("/api/mirrors/" + mirror_desc["Name"] + "/packages")
        self.check_equal(resp.status_code, 403)


class MirrorsAPITestCreateUpdate(APITest):
    """
    POST /api/mirrors, PUT /api/mirrors/:name, GET /api/mirrors/:name/packages
    """
    def check(self):
        mirror_name = self.random_name()
        mirror_desc = {u'Name': mirror_name,
                       u'ArchiveURL': 'http://repo.varnish-cache.org/debian/',
                       u'Distribution': 'wheezy',
                       u'Components': ['varnish-3.0']}

        mirror_desc[u'IgnoreSignatures'] = True
        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 201)

        resp = self.get("/api/mirrors/" + mirror_name + "/packages")
        self.check_equal(resp.status_code, 403)

        mirror_desc["Name"] = self.random_name()
        resp = self.put("/api/mirrors/" + mirror_name, json=mirror_desc)
        self.check_equal(resp.status_code, 200)
        self.check_subset({u'Name': mirror_desc["Name"],
                           u'ArchiveRoot':'http://repo.varnish-cache.org/debian/',
                           u'Distribution': 'wheezy'}, resp.json())

        for x in xrange(5):
            if resp.json()["LastDownloadDate"] != "0001-01-01T00:00:00Z": break
            time.sleep(3)
            resp = self.get("/api/mirrors/" + mirror_desc["Name"])

        resp = self.get("/api/mirrors/" + mirror_desc["Name"] + "/packages")
        self.check_equal(resp.status_code, 200)


class MirrorsAPITestCreateDelete(APITest):
    """
    POST /api/mirrors, DELETE /api/mirrors/:name
    """
    def check(self):
        mirror_name = self.random_name()
        mirror_desc = {u'Name': mirror_name,
                       u'ArchiveURL': 'http://repo.varnish-cache.org/debian/',
                       u'IgnoreSignatures': True,
                       u'Distribution': 'wheezy',
                       u'Components': ['varnish-3.0']}

        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 201)

        resp = self.delete("/api/mirrors/" + mirror_name)
        self.check_equal(resp.status_code, 200)


class MirrorsAPITestCreateList(APITest):
    """
    GET /api/mirrors, POST /api/mirrors, GET /api/mirrors
    """
    def check(self):
        resp = self.get("/api/mirrors")
        self.check_equal(resp.status_code, 200)
        count = len(resp.json())

        mirror_name = self.random_name()
        mirror_desc = {u'Name': mirror_name,
                       u'ArchiveURL': 'http://repo.varnish-cache.org/debian/',
                       u'IgnoreSignatures': True,
                       u'Distribution': 'wheezy',
                       u'Components': ['varnish-3.0']}

        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 201)

        resp = self.get("/api/mirrors")
        self.check_equal(resp.status_code, 200)
        self.check_equal(len(resp.json()), count + 1)
