from api_lib import TASK_FAILED, TASK_SUCCEEDED, APITest

from .publish import DefaultSigningOptions


class SnapshotsAPITestCreateShowEmpty(APITest):
    """
    GET /api/snapshots/:name, POST /api/snapshots, GET /api/snapshots/:name/packages
    """
    def check(self):
        snapshot_name = self.random_name()
        snapshot_desc = {'Description': 'fun snapshot',
                         'Name': snapshot_name}

        # create empty snapshot
        resp = self.post_task("/api/snapshots", json=snapshot_desc)
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)

        self.check_subset(snapshot_desc, self.get("/api/snapshots/" + snapshot_name).json())
        self.check_equal(self.get("/api/snapshots/" + snapshot_name).status_code, 200)

        resp = self.get("/api/snapshots/" + snapshot_name + "/packages")
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json(), [])

        self.check_equal(self.get("/api/snapshots/" + self.random_name()).status_code, 404)

        # create snapshot with duplicate name
        resp = self.post_task("/api/snapshots", json=snapshot_desc)
        self.check_equal(resp.json()['State'], TASK_FAILED)


class SnapshotsAPITestCreateFromRefs(APITest):
    """
    GET /api/snapshots/:name, POST /api/snapshots, GET /api/snapshots/:name/packages,
    GET /api/snapshots
    """
    def check(self):
        snapshot_name = self.random_name()
        snapshot_desc = {'Description': 'fun snapshot',
                         'Name': snapshot_name,
                         'SourceSnapshots': [self.random_name()]}

        # creating snapshot from missing source snapshot
        resp = self.post("/api/snapshots", json=snapshot_desc)
        self.check_equal(resp.status_code, 404)

        # create empty snapshot
        empty_snapshot_name = self.random_name()
        resp = self.post_task("/api/snapshots", json={"Name": empty_snapshot_name})
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)
        self.check_equal(
            self.get("/api/snapshots/" + empty_snapshot_name).json()['Description'], "Created as empty"
        )

        # create and upload package to repo to register package in DB
        repo_name = self.random_name()
        self.check_equal(self.post("/api/repos", json={"Name": repo_name}).status_code, 201)
        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)
        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], TASK_SUCCEEDED)

        # create snapshot with empty snapshot as source and package
        snapshot = snapshot_desc.copy()
        snapshot['PackageRefs'] = ["Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378"]
        snapshot['SourceSnapshots'] = [empty_snapshot_name]
        resp = self.post_task("/api/snapshots", json=snapshot)
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)
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
        self.check_equal(resp.json()['State'], TASK_FAILED)

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
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)
        self.check_equal([],
                         self.get("/api/snapshots/" + snapshot_name + "/packages", params={"format": "details"}).json())

        snapshot_name = self.random_name()
        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)

        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], TASK_SUCCEEDED)

        resp = self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot_name})
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)
        self.check_equal(self.get("/api/snapshots/" + snapshot_name).status_code, 200)

        self.check_subset({'Architecture': 'i386',
                           'Package': 'libboost-program-options-dev',
                           'Version': '1.49.0.1',
                           'FilesHash': '918d2f433384e378'},
                          self.get("/api/snapshots/" + snapshot_name + "/packages", params={"format": "details"}).json()[0])

        self.check_subset({'Architecture': 'i386',
                           'Package': 'libboost-program-options-dev',
                           'Version': '1.49.0.1',
                           'FilesHash': '918d2f433384e378'},
                          self.get("/api/snapshots/" + snapshot_name + "/packages",
                                   params={"format": "details", "q": "Version (> 0.6.1-1.4)"}).json()[0])

        # duplicate snapshot name
        resp = self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot_name})
        self.check_equal(resp.json()['State'], TASK_FAILED)


class SnapshotsAPITestCreateUpdate(APITest):
    """
    POST /api/snapshots, PUT /api/snapshots/:name, GET /api/snapshots/:name
    """
    def check(self):
        snapshot_name = self.random_name()
        snapshot_desc = {'Description': 'fun snapshot',
                         'Name': snapshot_name}

        resp = self.post_task("/api/snapshots", json=snapshot_desc)
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)

        new_snapshot_name = self.random_name()
        resp = self.put_task("/api/snapshots/" + snapshot_name, json={'Name': new_snapshot_name,
                                                                      'Description': 'New description'})
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)

        resp = self.get("/api/snapshots/" + new_snapshot_name)
        self.check_equal(resp.status_code, 200)
        self.check_subset({"Name": new_snapshot_name,
                           "Description": "New description"}, resp.json())

        # duplicate name
        resp = self.put_task("/api/snapshots/" + new_snapshot_name, json={'Name': new_snapshot_name,
                                                                          'Description': 'New description'})
        self.check_equal(resp.json()['State'], TASK_FAILED)

        # missing snapshot
        resp = self.put("/api/snapshots/" + snapshot_name, json={})
        self.check_equal(resp.status_code, 404)


class SnapshotsAPITestCreateDelete(APITest):
    """
    POST /api/snapshots, DELETE /api/snapshots/:name, GET /api/snapshots/:name
    """
    def check(self):
        snapshot_name = self.random_name()
        snapshot_desc = {'Description': 'fun snapshot',
                         'Name': snapshot_name}

        # deleting unreferenced snapshot
        resp = self.post_task("/api/snapshots", json=snapshot_desc)
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)

        self.check_equal(self.delete_task("/api/snapshots/" + snapshot_name).json()['State'], TASK_SUCCEEDED)

        self.check_equal(self.get("/api/snapshots/" + snapshot_name).status_code, 404)

        # deleting referenced snapshot
        snap1, snap2 = self.random_name(), self.random_name()
        self.check_equal(self.post_task("/api/snapshots", json={"Name": snap1}).json()['State'], TASK_SUCCEEDED)
        self.check_equal(
            self.post_task(
                "/api/snapshots", json={"Name": snap2, "SourceSnapshots": [snap1]}
            ).json()['State'], 2
        )

        self.check_equal(self.delete_task("/api/snapshots/" + snap1).json()['State'], TASK_FAILED)
        self.check_equal(self.get("/api/snapshots/" + snap1).status_code, 200)
        self.check_equal(self.delete_task("/api/snapshots/" + snap1, params={"force": "1"}).json()['State'], TASK_SUCCEEDED)
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
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)

        self.check_equal(self.delete_task("/api/snapshots/" + snap2).json()['State'], TASK_FAILED)
        self.check_equal(self.delete_task("/api/snapshots/" + snap2, params={"force": "1"}).json()['State'], TASK_FAILED)


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

        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], TASK_SUCCEEDED)

        resp = self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot_name})
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)

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
        repos = [self.random_name() for x in range(2)]
        snapshots = [self.random_name() for x in range(2)]

        for repo_name in repos:
            self.check_equal(self.post("/api/repos", json={"Name": repo_name}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)

        self.check_equal(self.post_task("/api/repos/" + repos[-1] + "/file/" + d).json()['State'], TASK_SUCCEEDED)

        resp = self.post_task("/api/repos/" + repos[-1] + '/snapshots', json={'Name': snapshots[0]})
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)

        resp = self.post_task("/api/snapshots", json={'Name': snapshots[1]})
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)

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


class SnapshotsAPITestMerge(APITest):
    """
    POST /api/snapshots, GET /api/snapshots/merge, GET /api/snapshots/:name, DELETE /api/snapshots/:name
    """

    def check(self):
        sources = [
            {"Description": "fun snapshot", "Name": self.random_name()}
            for _ in range(2)
        ]

        # create source snapshots
        for source in sources:
            resp = self.post_task("/api/snapshots", json=source)
            self.check_equal(resp.json()["State"], TASK_SUCCEEDED)

        # create merge snapshot
        merged_name = self.random_name()
        resp = self.post_task(
            "/api/snapshots/merge",
            json={
                "Destination": merged_name,
                "Sources": [source["Name"] for source in sources],
            },
        )
        self.check_equal(resp.json()["State"], TASK_SUCCEEDED)

        # check merge snapshot
        resp = self.get(f"/api/snapshots/{merged_name}")
        self.check_equal(resp.status_code, 200)
        source_list = ", ".join(f"'{source['Name']}'" for source in sources)
        self.check_subset(
            {
                "Name": merged_name,
                "Description": f"Merged from sources: {source_list}",
            },
            resp.json(),
        )

        # remove merge snapshot
        self.check_equal(
            self.delete_task(f"/api/snapshots/{merged_name}").json()["State"], TASK_SUCCEEDED
        )

        # create merge snapshot without sources
        merged_name = self.random_name()
        resp = self.post(
            "/api/snapshots/merge", json={"Destination": merged_name, "Sources": []}
        )
        self.check_equal(resp.status_code, 400)
        self.check_equal(
            resp.json()["error"], "At least one source snapshot is required"
        )
        self.check_equal(self.get(f"/api/snapshots/{merged_name}").status_code, 404)

        # create merge snapshot with non-existing source
        merged_name = self.random_name()
        non_existing_source = self.random_name()
        resp = self.post(
            "/api/snapshots/merge",
            json={"Destination": merged_name, "Sources": [non_existing_source]},
        )
        self.check_equal(
            resp.json()["error"], f"snapshot with name {non_existing_source} not found"
        )
        self.check_equal(resp.status_code, 404)

        self.check_equal(self.get(f"/api/snapshots/{merged_name}").status_code, 404)

        # create merge snapshot with used name
        merged_name = sources[0]["Name"]
        resp = self.post(
            "/api/snapshots/merge",
            json={"Destination": merged_name, "Sources": [source["Name"] for source in sources]},
        )
        self.check_equal(
            resp.json()["error"],
            f"unable to create snapshot: snapshot with name {sources[0]['Name']} already exists",
        )
        self.check_equal(resp.status_code, 500)

        # create merge snapshot with "latest" and "no-remove" flags (should fail)
        merged_name = self.random_name()
        resp = self.post(
            "/api/snapshots/merge",
            json={
                "Destination": merged_name,
                "Sources": [source["Name"] for source in sources],
            },
            params={"latest": "1", "no-remove": "1"},
        )
        self.check_equal(
            resp.json()["error"], "no-remove and latest are mutually exclusive"
        )
        self.check_equal(resp.status_code, 400)
