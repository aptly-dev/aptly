from api_lib import APITest


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

        resp = self.put_task("/api/mirrors/" + mirror_name, json=mirror_desc)
        self.check_task(resp)
        _id = resp.json()['ID']

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
        self.check_task(resp)


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


class MirrorsAPITestSkipArchitectureCheck(APITest):
    """
    GET /api/mirrors, POST /api/mirrors, GET /api/mirrors

    This tests SkipArchitectureCheck and IgnoreSignatures via API.
    The repo to be mirrored requires the SkipArchitectureCheck and SkipComponentCheck in order to be mirrored.
    """
    def check(self):
        resp = self.get("/api/mirrors")
        self.check_equal(resp.status_code, 200)
        count = len(resp.json())

        mirror_name = self.random_name()
        mirror_desc = {'Name': mirror_name,
                       'ArchiveURL': 'http://repo.aptly.info/system-tests/pkg.duosecurity.com/Debian',
                       'Architectures': ['amd64', 'i386'],
                       'SkipArchitectureCheck': True,
                       'SkipComponentCheck': True,
                       'IgnoreSignatures': True,
                       'Distribution': 'bookworm',
                       'Components': ['main']}

        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 201)

        resp = self.get("/api/mirrors")
        self.check_equal(resp.status_code, 200)
        self.check_equal(len(resp.json()), count + 1)

        mirror_desc = {'Name': mirror_name,
                       'IgnoreSignatures': True}
        resp = self.put_task("/api/mirrors/" + mirror_name, json=mirror_desc)
        self.check_task(resp)


class MirrorsAPITestEdit(APITest):
    """
    POST /api/mirrors/{name} - Edit mirror configuration
    """
    def check(self):
        # Create a mirror first
        mirror_name = self.random_name()
        mirror_desc = {'Name': mirror_name,
                       'ArchiveURL': 'http://repo.aptly.info/system-tests/packagecloud.io/varnishcache/varnish30/debian/',
                       'IgnoreSignatures': True,
                       'Distribution': 'wheezy',
                       'Components': ['main'],
                       'Architectures': ['amd64']}

        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 201)

        # Test editing basic properties (Filter, FilterWithDeps, Download options)
        edit_params = {
            'Filter': 'varnish',
            'FilterWithDeps': True,
            'DownloadSources': True,
            'DownloadInstaller': False,
            'DownloadUdebs': False
        }

        resp = self.post("/api/mirrors/" + mirror_name, json=edit_params)
        self.check_equal(resp.status_code, 200)
        self.check_subset({
            'Name': mirror_name,
            'Filter': 'varnish',
            'FilterWithDeps': True,
            'DownloadSources': True
        }, resp.json())

        # Verify the changes persisted
        resp = self.get("/api/mirrors/" + mirror_name)
        self.check_equal(resp.status_code, 200)
        self.check_subset({
            'Filter': 'varnish',
            'FilterWithDeps': True,
            'DownloadSources': True
        }, resp.json())

        # Test editing with empty filter to clear it
        edit_params = {'Filter': ''}
        resp = self.post("/api/mirrors/" + mirror_name, json=edit_params)
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json()['Filter'], '')


class MirrorsAPITestEditNotFound(APITest):
    """
    POST /api/mirrors/{name} - Edit non-existent mirror
    """
    def check(self):
        resp = self.post("/api/mirrors/non-existent-mirror", json={'Filter': 'test'})
        self.check_equal(resp.status_code, 404)
        self.check_in('unable to edit', resp.json()['error'])


class MirrorsAPITestEditArchitectures(APITest):
    """
    POST /api/mirrors/{name} - Edit mirror architectures (triggers fetch)
    """
    def check(self):
        # Create a mirror
        mirror_name = self.random_name()
        mirror_desc = {'Name': mirror_name,
                       'ArchiveURL': 'http://repo.aptly.info/system-tests/security.debian.org/debian-security/',
                       'IgnoreSignatures': True,
                       'Distribution': 'buster/updates',
                       'Components': ['main'],
                       'Architectures': ['amd64']}

        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 201)

        # Edit architectures (should trigger a fetch)
        edit_params = {
            'Architectures': ['amd64', 'i386'],
            'IgnoreSignatures': True
        }

        resp = self.post("/api/mirrors/" + mirror_name, json=edit_params)
        self.check_equal(resp.status_code, 200)

        # Verify architectures were updated
        resp = self.get("/api/mirrors/" + mirror_name)
        self.check_equal(resp.status_code, 200)
        architectures = resp.json()['Architectures']
        self.check_equal(sorted(architectures), ['amd64', 'i386'])


class MirrorsAPITestEditArchiveURL(APITest):
    """
    POST /api/mirrors/{name} - Edit mirror archive URL (triggers fetch)
    """
    def check(self):
        # Create a mirror
        mirror_name = self.random_name()
        mirror_desc = {'Name': mirror_name,
                       'ArchiveURL': 'http://repo.aptly.info/system-tests/ftp.ru.debian.org/debian',
                       'IgnoreSignatures': True,
                       'Distribution': 'bookworm',
                       'Components': ['main'],
                       'Architectures': ['amd64']}

        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 201)

        # Edit archive URL (should trigger a fetch)
        edit_params = {
            'ArchiveURL': 'http://repo.aptly.info/system-tests/ftp.ch.debian.org/debian',
            'IgnoreSignatures': True
        }

        resp = self.post("/api/mirrors/" + mirror_name, json=edit_params)
        self.check_equal(resp.status_code, 200)

        # Verify URL was updated
        resp = self.get("/api/mirrors/" + mirror_name)
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json()['ArchiveRoot'], 'http://repo.aptly.info/system-tests/ftp.ch.debian.org/debian/')


class MirrorsAPITestRename(APITest):
    """
    PUT /api/mirrors/:name - Rename mirror via update endpoint
    """
    def check(self):
        mirror_name = self.random_name()
        mirror_desc = {'Name': mirror_name,
                       'ArchiveURL': 'http://repo.aptly.info/system-tests/packagecloud.io/varnishcache/varnish30/debian/',
                       'IgnoreSignatures': True,
                       'Distribution': 'wheezy',
                       'Components': ['main'],
                       'Architectures': ['amd64']}

        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 201)

        # Rename the mirror via PUT
        new_name = self.random_name()
        resp = self.put_task("/api/mirrors/" + mirror_name, json={'Name': new_name, 'IgnoreSignatures': True})
        self.check_task(resp)

        # Old name should no longer exist
        resp = self.get("/api/mirrors/" + mirror_name)
        self.check_equal(resp.status_code, 404)

        # New name should exist with correct data
        resp = self.get("/api/mirrors/" + new_name)
        self.check_equal(resp.status_code, 200)
        self.check_subset({'Name': new_name,
                           'ArchiveRoot': 'http://repo.aptly.info/system-tests/packagecloud.io/varnishcache/varnish30/debian/',
                           'Distribution': 'wheezy'}, resp.json())


class MirrorsAPITestRenameConflict(APITest):
    """
    PUT /api/mirrors/:name - Rename mirror to an already existing name should fail with 409
    """
    def check(self):
        mirror_name_a = self.random_name()
        mirror_name_b = self.random_name()

        mirror_desc_a = {'Name': mirror_name_a,
                         'ArchiveURL': 'http://repo.aptly.info/system-tests/packagecloud.io/varnishcache/varnish30/debian/',
                         'IgnoreSignatures': True,
                         'Distribution': 'wheezy',
                         'Components': ['main'],
                         'Architectures': ['amd64']}

        mirror_desc_b = {'Name': mirror_name_b,
                         'ArchiveURL': 'http://repo.aptly.info/system-tests/packagecloud.io/varnishcache/varnish30/debian/',
                         'IgnoreSignatures': True,
                         'Distribution': 'wheezy',
                         'Components': ['main'],
                         'Architectures': ['amd64']}

        resp = self.post("/api/mirrors", json=mirror_desc_a)
        self.check_equal(resp.status_code, 201)

        resp = self.post("/api/mirrors", json=mirror_desc_b)
        self.check_equal(resp.status_code, 201)

        # Try to rename mirror A to mirror B's name — should fail with 409
        resp = self.put("/api/mirrors/" + mirror_name_a, json={'Name': mirror_name_b, 'IgnoreSignatures': True})
        self.check_equal(resp.status_code, 409)
        self.check_in('unable to rename', resp.json()['error'])

        # Mirror A should still exist under its original name
        resp = self.get("/api/mirrors/" + mirror_name_a)
        self.check_equal(resp.status_code, 200)
        self.check_subset({'Name': mirror_name_a}, resp.json())


class MirrorsAPITestEditFlatMirrorUdebs(APITest):
    """
    POST /api/mirrors/{name} - Edit flat mirror with udebs (should fail)
    """
    def check(self):
        # Create a flat mirror
        mirror_name = self.random_name()
        mirror_desc = {'Name': mirror_name,
                       'ArchiveURL': 'http://repo.aptly.info/system-tests/cloud.r-project.org/bin/linux/debian/bullseye-cran40/',
                       'IgnoreSignatures': True,
                       'Architectures': ['amd64']}

        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 201)

        # Try to enable udebs on a flat mirror (should fail)
        edit_params = {'DownloadUdebs': True}

        resp = self.post("/api/mirrors/" + mirror_name, json=edit_params)
        self.check_equal(resp.status_code, 400)
        self.check_in("flat mirrors don't support udebs", resp.json()['error'])
