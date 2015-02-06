from api_lib import APITest


class SnapshotsAPITestCreateShowEmpty(APITest):
    """
    GET /api/snapshots/:name, POST /api/snapshots, GET /api/snapshots/:name/packages
    """
    def check(self):
        snapshot_name = self.random_name()
        snapshot_desc = {u'Description': u'fun snapshot',
                         u'Name': snapshot_name}

        # create empty snapshot
        resp = self.post("/api/snapshots", json=snapshot_desc)
        self.check_subset(snapshot_desc, resp.json())
        self.check_equal(resp.status_code, 201)

        self.check_subset(snapshot_desc, self.get("/api/snapshots/" + snapshot_name).json())
        self.check_equal(self.get("/api/snapshots/" + snapshot_name).status_code, 200)

        resp = self.get("/api/snapshots/" + snapshot_name + "/packages")
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json(), [])

        self.check_equal(self.get("/api/snapshots/" + self.random_name()).status_code, 404)

        # create snapshot with duplicate name
        resp = self.post("/api/snapshots", json=snapshot_desc)
        self.check_equal(resp.status_code, 400)


class SnapshotsAPITestCreateFromRefs(APITest):
    """
    GET /api/snapshots/:name, POST /api/snapshots, GET /api/snapshots/:name/packages
    """
    def check(self):
        snapshot_name = self.random_name()
        snapshot_desc = {u'Description': u'fun snapshot',
                         u'Name': snapshot_name,
                         u'SourceSnapshots': [self.random_name()]}

        # creating snapshot from missing source snapshot
        resp = self.post("/api/snapshots", json=snapshot_desc)
        self.check_equal(resp.status_code, 404)

        # create empty snapshot
        empty_snapshot_name = self.random_name()
        resp = self.post("/api/snapshots", json={"Name": empty_snapshot_name})
        self.check_equal(resp.status_code, 201)
        self.check_equal(resp.json()['Description'], 'Created as empty')

        # create and upload package to repo to register package in DB
        repo_name = self.random_name()
        self.check_equal(self.post("/api/repos", json={"Name": repo_name}).status_code, 201)
        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)
        self.check_equal(self.post("/api/repos/" + repo_name + "/file/" + d).status_code, 200)

        # create snapshot with empty snapshot as source and package
        snapshot = snapshot_desc.copy()
        snapshot['PackageRefs'] = ["Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378"]
        snapshot['SourceSnapshots'] = [empty_snapshot_name]
        resp = self.post("/api/snapshots", json=snapshot)
        self.check_equal(resp.status_code, 201)
        snapshot.pop('SourceSnapshots')
        snapshot.pop('PackageRefs')
        self.check_subset(snapshot, resp.json())

        self.check_subset(snapshot, self.get("/api/snapshots/" + snapshot_name).json())
        resp = self.get("/api/snapshots/" + snapshot_name + "/packages")
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json(), ["Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378"])

        # create snapshot with unreferenced package
        resp = self.post("/api/snapshots", json={
            "Name": self.random_name(),
            "PackageRefs": ["Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378", "Pamd64 no-such-package 1.2 91"]})
        self.check_equal(resp.status_code, 404)


class SnapshotsAPITestCreateFromRepo(APITest):
    """
    POST /api/repos, POST /api/repos/:name/snapshots, GET /api/snapshots/:name
    """
    def check(self):
        repo_name = self.random_name()
        snapshot_name = self.random_name()
        self.check_equal(self.post("/api/repos", json={"Name": repo_name}).status_code, 201)

        resp = self.post("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot_name})
        self.check_equal(resp.status_code, 400)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo_name + "/file/" + d).status_code, 200)

        resp = self.post("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot_name})
        self.check_equal(self.get("/api/snapshots/" + snapshot_name).status_code, 200)

        self.check_subset({u'Architecture': 'i386',
                           u'Package': 'libboost-program-options-dev',
                           u'Version': '1.49.0.1',
                           'FilesHash': '918d2f433384e378'},
                          self.get("/api/snapshots/" + snapshot_name + "/packages", params={"format": "details"}).json()[0])

        self.check_subset({u'Architecture': 'i386',
                           u'Package': 'libboost-program-options-dev',
                           u'Version': '1.49.0.1',
                           'FilesHash': '918d2f433384e378'},
                          self.get("/api/snapshots/" + snapshot_name + "/packages",
                                   params={"format": "details", "q": "Version (> 0.6.1-1.4)"}).json()[0])


class SnapshotsAPITestCreateUpdate(APITest):
    """
    POST /api/snapshots, PUT /api/snapshots/:name, GET /api/snapshots/:name
    """
    def check(self):
        snapshot_name = self.random_name()
        snapshot_desc = {u'Description': u'fun snapshot',
                         u'Name': snapshot_name}

        resp = self.post("/api/snapshots", json=snapshot_desc)
        self.check_equal(resp.status_code, 201)

        new_snapshot_name = self.random_name()
        resp = self.put("/api/snapshots/" + snapshot_name, json={'Name': new_snapshot_name,
                                                                 'Description': 'New description'})
        self.check_equal(resp.status_code, 200)

        resp = self.get("/api/snapshots/" + new_snapshot_name)
        self.check_equal(resp.status_code, 200)
        self.check_subset({"Name": new_snapshot_name,
                           "Description": "New description"}, resp.json())


class SnapshotsAPITestCreateDelete(APITest):
    """
    POST /api/snapshots, DELETE /api/snapshots/:name, GET /api/snapshots/:name
    """
    def check(self):
        snapshot_name = self.random_name()
        snapshot_desc = {u'Description': u'fun snapshot',
                         u'Name': snapshot_name}

        resp = self.post("/api/snapshots", json=snapshot_desc)
        self.check_equal(resp.status_code, 201)

        self.check_equal(self.delete("/api/snapshots/" + snapshot_name).status_code, 200)

        self.check_equal(self.get("/api/snapshots/" + snapshot_name).status_code, 404)


class SnapshotsAPITestSearch(APITest):
    """
    POST /api/snapshots, GET /api/snapshots?sort=name, GET /api/snapshots/:name
    """
    def check(self):

        repo_name = self.random_name()
        self.check_equal(self.post("/api/repos", json={"Name": repo_name}).status_code, 201)

        d = self.random_name()
        snapshot_name = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo_name + "/file/" + d).status_code, 200)

        resp = self.post("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot_name})
        self.check_equal(resp.status_code, 201)

        resp = self.get("/api/snapshots/" + snapshot_name + "/packages",
                        params={"q": "libboost-program-options-dev", "format": "details"})
        self.check_equal(resp.status_code, 200)

        self.check_equal(len(resp.json()), 1)
        self.check_equal(resp.json()[0]["Package"], "libboost-program-options-dev")

        resp = self.get("/api/snapshots/" + snapshot_name + "/packages")
        self.check_equal(resp.status_code, 200)

        self.check_equal(len(resp.json()), 1)
        self.check_equal(resp.json(), ["Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378"])


class SnapshotsAPITestDiff(APITest):
    """
    GET /api/snapshot/:name/diff/:name2
    """
    def check(self):
        repos = [self.random_name() for x in xrange(2)]
        snapshots = [self.random_name() for x in xrange(2)]

        for repo_name in repos:
            self.check_equal(self.post("/api/repos", json={"Name": repo_name}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo_name + "/file/" + d).status_code, 200)

        resp = self.post("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshots[0]})
        self.check_equal(resp.status_code, 201)

        resp = self.post("/api/snapshots", json={'Name': snapshots[1]})
        self.check_equal(resp.status_code, 201)

        resp = self.get("/api/snapshots/" + snapshots[0] + "/diff/" + snapshots[1])
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json(), [{'Left': 'Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378',
                                        'Right': None}])

        resp = self.get("/api/snapshots/" + snapshots[1] + "/diff/" + snapshots[0])
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json(), [{'Right': 'Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378',
                                        'Left': None}])

        resp = self.get("/api/snapshots/" + snapshots[0] + "/diff/" + snapshots[0])
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json(), [])

        resp = self.get("/api/snapshots/" + snapshots[1] + "/diff/" + snapshots[1])
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json(), [])
