from api_lib import APITest


class MirrorsAPITestCreateShow(APITest):
    """
    POST /api/mirrors, GET /api/mirrors/:name/packages
    """
    def check(self):
        mirror_name = self.random_name()
        mirror_desc = {u'Name': mirror_name,
                       u'ArchiveURL': 'http://security.debian.org/',
                       u'Architectures': ['amd64'],
                       u'Components': ['main'],
                       u'Distribution': 'wheezy/updates'}

        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 400)
        self.check_equal({
            'error': 'unable to fetch mirror: verification of detached signature failed: exit status 2',
        }, resp.json())

        mirror_desc[u'IgnoreSignatures'] = True
        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 201)

        resp = self.get("/api/mirrors/" + mirror_name)
        self.check_equal(resp.status_code, 200)
        self.check_subset({u'Name': mirror_name,
                           u'ArchiveRoot': 'http://security.debian.org/',
                           u'Architectures': ['amd64'],
                           u'Components': ['main'],
                           u'Distribution': 'wheezy/updates'}, resp.json())

        resp = self.get("/api/mirrors/" + mirror_desc["Name"] + "/packages")
        self.check_equal(resp.status_code, 404)


class MirrorsAPITestCreateUpdate(APITest):
    """
    POST /api/mirrors, PUT /api/mirrors/:name, GET /api/mirrors/:name/packages
    """
    def check(self):
        mirror_name = self.random_name()
        mirror_desc = {u'Name': mirror_name,
                       u'ArchiveURL': 'https://packagecloud.io/varnishcache/varnish30/debian/',
                       u'Distribution': 'wheezy',
                       u'Components': ['main']}

        mirror_desc[u'IgnoreSignatures'] = True
        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 201)

        resp = self.get("/api/mirrors/" + mirror_name + "/packages")
        self.check_equal(resp.status_code, 404)

        mirror_desc["Name"] = self.random_name()
        resp = self.put_task("/api/mirrors/" + mirror_name, json=mirror_desc)
        self.check_equal(resp.json()["State"], 2)

        _id = resp.json()['ID']
        resp = self.get("/api/tasks/" + str(_id) + "/detail")
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json()['RemainingDownloadSize'], 0)
        self.check_equal(resp.json()['RemainingNumberOfPackages'], 0)

        resp = self.get("/api/mirrors/" + mirror_desc["Name"])
        self.check_equal(resp.status_code, 200)
        self.check_subset({u'Name': mirror_desc["Name"],
                           u'ArchiveRoot': 'https://packagecloud.io/varnishcache/varnish30/debian/',
                           u'Distribution': 'wheezy'}, resp.json())

        resp = self.get("/api/mirrors/" + mirror_desc["Name"] + "/packages")
        self.check_equal(resp.status_code, 200)


class MirrorsAPITestCreateDelete(APITest):
    """
    POST /api/mirrors, DELETE /api/mirrors/:name
    """
    def check(self):
        mirror_name = self.random_name()
        mirror_desc = {u'Name': mirror_name,
                       u'ArchiveURL': 'https://packagecloud.io/varnishcache/varnish30/debian/',
                       u'IgnoreSignatures': True,
                       u'Distribution': 'wheezy',
                       u'Components': ['main']}

        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 201)

        resp = self.delete_task("/api/mirrors/" + mirror_name)
        self.check_equal(resp.json()['State'], 2)


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
                       u'ArchiveURL': 'https://packagecloud.io/varnishcache/varnish30/debian/',
                       u'IgnoreSignatures': True,
                       u'Distribution': 'wheezy',
                       u'Components': ['main']}

        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 201)

        resp = self.get("/api/mirrors")
        self.check_equal(resp.status_code, 200)
        self.check_equal(len(resp.json()), count + 1)
