from api_lib import APITest
from publish import DefaultSigningOptions


class SnapshotsAPITestCreateShowEmpty(APITest):
    """
    GET /api/snapshots/:name, POST /api/snapshots, GET /api/snapshots/:name/packages
    """
    def check(self):
        snapshot_name = self.random_name()
        snapshot_desc = {u'Description': u'fun snapshot',
                         u'Name': snapshot_name}

        # create empty snapshot
        resp = self.post_task("/api/snapshots", json=snapshot_desc)
        self.check_equal(resp.json()['State'], 2)

        self.check_subset(snapshot_desc, self.get("/api/snapshots/" + snapshot_name).json())
        self.check_equal(self.get("/api/snapshots/" + snapshot_name).status_code, 200)

        resp = self.get("/api/snapshots/" + snapshot_name + "/packages")
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json(), [])

        self.check_equal(self.get("/api/snapshots/" + self.random_name()).status_code, 404)

        # create snapshot with duplicate name
        resp = self.post_task("/api/snapshots", json=snapshot_desc)
        self.check_equal(resp.json()['State'], 3)


class SnapshotsAPITestCreateFromRefs(APITest):
    """
    GET /api/snapshots/:name, POST /api/snapshots, GET /api/snapshots/:name/packages,
    GET /api/snapshots
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
        resp = self.post_task("/api/snapshots", json={"Name": empty_snapshot_name})
        self.check_equal(resp.json()['State'], 2)
        self.check_equal(
            self.get("/api/snapshots/" + empty_snapshot_name).json()['Description'], "Created as empty"
        )

        # create and upload package to repo to register package in DB
        repo_name = self.random_name()
        self.check_equal(self.post("/api/repos", json={"Name": repo_name}).status_code, 201)
        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)
        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], 2)

        # create snapshot with empty snapshot as source and package
        snapshot = snapshot_desc.copy()
        snapshot['PackageRefs'] = ["Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378"]
        snapshot['SourceSnapshots'] = [empty_snapshot_name]
        resp = self.post_task("/api/snapshots", json=snapshot)
        self.check_equal(resp.json()['State'], 2)
        snapshot.pop('SourceSnapshots')
        snapshot.pop('PackageRefs')
        resp = self.get("/api/snapshots/" + snapshot_name)
        self.check_equal(resp.status_code, 200)
        self.check_subset(snapshot, resp.json())

        self.check_subset(snapshot, self.get("/api/snapshots/" + snapshot_name).json())
        resp = self.get("/api/snapshots/" + snapshot_name + "/packages")
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json(), ["Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378"])

        # create snapshot with unreferenced package
        resp = self.post_task("/api/snapshots", json={
            "Name": self.random_name(),
            "PackageRefs": ["Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378", "Pamd64 no-such-package 1.2 91"]})
        self.check_equal(resp.json()['State'], 3)

        # list snapshots
        resp = self.get("/api/snapshots", params={"sort": "time"})
        self.check_equal(resp.status_code, 200)
        self.check_equal([s["Name"] for s in resp.json() if s["Name"] in [empty_snapshot_name, snapshot_name]],
                         [empty_snapshot_name, snapshot_name])


class SnapshotsAPITestCreateFromRepo(APITest):
    """
    POST /api/repos, POST /api/repos/:name/snapshots, GET /api/snapshots/:name
    """
    def check(self):
        repo_name = self.random_name()
        snapshot_name = self.random_name()
        self.check_equal(self.post("/api/repos", json={"Name": repo_name}).status_code, 201)

        resp = self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot_name})
        self.check_equal(resp.json()['State'], 2)
        self.check_equal([],
                         self.get("/api/snapshots/" + snapshot_name + "/packages", params={"format": "details"}).json())

        snapshot_name = self.random_name()
        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)

        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], 2)

        resp = self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot_name})
        self.check_equal(resp.json()['State'], 2)
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

        # duplicate snapshot name
        resp = self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot_name})
        self.check_equal(resp.json()['State'], 3)


class SnapshotsAPITestCreateUpdate(APITest):
    """
    POST /api/snapshots, PUT /api/snapshots/:name, GET /api/snapshots/:name
    """
    def check(self):
        snapshot_name = self.random_name()
        snapshot_desc = {u'Description': u'fun snapshot',
                         u'Name': snapshot_name}

        resp = self.post_task("/api/snapshots", json=snapshot_desc)
        self.check_equal(resp.json()['State'], 2)

        new_snapshot_name = self.random_name()
        resp = self.put_task("/api/snapshots/" + snapshot_name, json={'Name': new_snapshot_name,
                                                                      'Description': 'New description'})
        self.check_equal(resp.json()['State'], 2)

        resp = self.get("/api/snapshots/" + new_snapshot_name)
        self.check_equal(resp.status_code, 200)
        self.check_subset({"Name": new_snapshot_name,
                           "Description": "New description"}, resp.json())

        # duplicate name
        resp = self.put_task("/api/snapshots/" + new_snapshot_name, json={'Name': new_snapshot_name,
                                                                          'Description': 'New description'})
        self.check_equal(resp.json()['State'], 3)

        # missing snapshot
        resp = self.put("/api/snapshots/" + snapshot_name, json={})
        self.check_equal(resp.status_code, 404)


class SnapshotsAPITestCreateDelete(APITest):
    """
    POST /api/snapshots, DELETE /api/snapshots/:name, GET /api/snapshots/:name
    """
    def check(self):
        snapshot_name = self.random_name()
        snapshot_desc = {u'Description': u'fun snapshot',
                         u'Name': snapshot_name}

        # deleting unreferenced snapshot
        resp = self.post_task("/api/snapshots", json=snapshot_desc)
        self.check_equal(resp.json()['State'], 2)

        self.check_equal(self.delete_task("/api/snapshots/" + snapshot_name).json()['State'], 2)

        self.check_equal(self.get("/api/snapshots/" + snapshot_name).status_code, 404)

        # deleting referenced snapshot
        snap1, snap2 = self.random_name(), self.random_name()
        self.check_equal(self.post_task("/api/snapshots", json={"Name": snap1}).json()['State'], 2)
        self.check_equal(
            self.post_task(
                "/api/snapshots", json={"Name": snap2, "SourceSnapshots": [snap1]}
            ).json()['State'], 2
        )

        self.check_equal(self.delete_task("/api/snapshots/" + snap1).json()['State'], 3)
        self.check_equal(self.get("/api/snapshots/" + snap1).status_code, 200)
        self.check_equal(self.delete_task("/api/snapshots/" + snap1, params={"force": "1"}).json()['State'], 2)
        self.check_equal(self.get("/api/snapshots/" + snap1).status_code, 404)

        # deleting published snapshot
        resp = self.post_task(
            "/api/publish",
            json={
                 "SourceKind": "snapshot",
                 "Distribution": "trusty",
                 "Architectures": ["i386"],
                 "Sources": [{"Name": snap2}],
                 "Signing": DefaultSigningOptions,
             }
        )
        self.check_equal(resp.json()['State'], 2)

        self.check_equal(self.delete_task("/api/snapshots/" + snap2).json()['State'], 3)
        self.check_equal(self.delete_task("/api/snapshots/" + snap2, params={"force": "1"}).json()['State'], 3)


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

        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], 2)

        resp = self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot_name})
        self.check_equal(resp.json()['State'], 2)

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

        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], 2)

        resp = self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshots[0]})
        self.check_equal(resp.json()['State'], 2)

        resp = self.post_task("/api/snapshots", json={'Name': snapshots[1]})
        self.check_equal(resp.json()['State'], 2)

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
