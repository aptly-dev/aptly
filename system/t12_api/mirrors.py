from api_lib import TASK_SUCCEEDED, APITest


class MirrorsAPITestCreateShow(APITest):
    """
    POST /api/mirrors, GET /api/mirrors/:name/packages
    """

    def check(self):
        mirror_name = self.random_name()
        mirror_desc = {'Name': mirror_name,
                       'ArchiveURL': 'http://repo.aptly.info/system-tests/security.debian.org/debian-security/',
                       'Architectures': ['amd64'],
                       'Components': ['main'],
                       'Distribution': 'buster/updates'}

        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 400)
        self.check_equal({
            'error': 'unable to fetch mirror: verification of detached signature failed: exit status 2',
        }, resp.json())

        mirror_desc['IgnoreSignatures'] = True
        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 201)

        resp = self.get("/api/mirrors/" + mirror_name)
        self.check_equal(resp.status_code, 200)
        self.check_subset({'Name': mirror_name,
                           'ArchiveRoot': 'http://repo.aptly.info/system-tests/security.debian.org/debian-security/',
                           'Architectures': ['amd64'],
                           'Components': ['main'],
                           'Distribution': 'buster/updates'}, resp.json())

        resp = self.get("/api/mirrors/" + mirror_desc["Name"] + "/packages")
        self.check_equal(resp.status_code, 404)


class MirrorsAPITestCreateUpdate(APITest):
    """
    POST /api/mirrors, PUT /api/mirrors/:name, GET /api/mirrors/:name/packages
    """
    def check(self):
        mirror_name = self.random_name()
        mirror_desc = {'Name': mirror_name,
                       'ArchiveURL': 'http://repo.aptly.info/system-tests/packagecloud.io/varnishcache/varnish30/debian/',
                       'Distribution': 'wheezy',
                       'Keyrings': ["aptlytest.gpg"],
                       'Architectures': ["amd64"],
                       'Components': ['main']}

        mirror_desc['IgnoreSignatures'] = True
        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 201)

        resp = self.get("/api/mirrors/" + mirror_name + "/packages")
        self.check_equal(resp.status_code, 404)

        mirror_desc["Name"] = self.random_name()
        resp = self.put_task("/api/mirrors/" + mirror_name, json=mirror_desc)
        self.check_equal(resp.status_code, 200)
        _id = resp.json()['ID']
        if resp.json()["State"] != TASK_SUCCEEDED:
            resp = self.get("/api/tasks/" + str(_id) + "/output")
            raise Exception("task failed: " + str(resp.json()))

        resp = self.get("/api/tasks/" + str(_id) + "/detail")
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json()['RemainingDownloadSize'], 0)
        self.check_equal(resp.json()['RemainingNumberOfPackages'], 0)

        resp = self.get("/api/mirrors/" + mirror_desc["Name"])
        self.check_equal(resp.status_code, 200)
        self.check_subset({'Name': mirror_desc["Name"],
                           'ArchiveRoot': 'http://repo.aptly.info/system-tests/packagecloud.io/varnishcache/varnish30/debian/',
                           'Distribution': 'wheezy'}, resp.json())

        resp = self.get("/api/mirrors/" + mirror_desc["Name"] + "/packages")
        self.check_equal(resp.status_code, 200)


class MirrorsAPITestCreateDelete(APITest):
    """
    POST /api/mirrors, DELETE /api/mirrors/:name
    """
    def check(self):
        mirror_name = self.random_name()
        mirror_desc = {'Name': mirror_name,
                       'ArchiveURL': 'http://repo.aptly.info/system-tests/packagecloud.io/varnishcache/varnish30/debian/',
                       'IgnoreSignatures': True,
                       'Distribution': 'wheezy',
                       'Components': ['main']}

        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 201)

        resp = self.delete_task("/api/mirrors/" + mirror_name)
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)


class MirrorsAPITestCreateList(APITest):
    """
    GET /api/mirrors, POST /api/mirrors, GET /api/mirrors
    """
    def check(self):
        resp = self.get("/api/mirrors")
        self.check_equal(resp.status_code, 200)
        count = len(resp.json())

        mirror_name = self.random_name()
        mirror_desc = {'Name': mirror_name,
                       'ArchiveURL': 'http://repo.aptly.info/system-tests/packagecloud.io/varnishcache/varnish30/debian/',
                       'IgnoreSignatures': True,
                       'Distribution': 'wheezy',
                       'Components': ['main']}

        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 201)

        resp = self.get("/api/mirrors")
        self.check_equal(resp.status_code, 200)
        self.check_equal(len(resp.json()), count + 1)
