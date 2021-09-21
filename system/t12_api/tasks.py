from api_lib import APITest
from publish import DefaultSigningOptions


class TaskAPITestParallelTasks(APITest):
    """
    GET /api/tasks, GET /api/tasks/:id/wait, GET /api/tasks-wait
    """
    def _create_mirror(self, dist):
        mirror_name = self.random_name()
        mirror_desc = {u'Name': mirror_name,
                       u'ArchiveURL': 'https://packagecloud.io/varnishcache/varnish30/debian/',
                       u'Distribution': dist,
                       u'Components': ['main']}
        mirror_desc[u'IgnoreSignatures'] = True
        resp = self.post("/api/mirrors", json=mirror_desc)
        self.check_equal(resp.status_code, 201)
        resp = self.put("/api/mirrors/" + mirror_name, json=mirror_desc, params={'_async': True})
        self.check_equal(resp.status_code, 202)

        # check that two mirror updates cannot run at the same time
        resp2 = self.put("/api/mirrors/" + mirror_name, json=mirror_desc, params={'_async': True})
        self.check_equal(resp2.status_code, 409)

        return resp.json()['ID'], mirror_name

    def _create_repo(self):
        repo_name = self.random_name()
        distribution = self.random_name()

        self.check_equal(self.post("/api/repos",
                         json={
                             "Name": repo_name,
                             "Comment": "fun repo",
                             "DefaultDistribution": distribution
                         }).status_code, 201)
        d = self.random_name()
        self.check_equal(
            self.upload("/api/files/" + d, "pyspi_0.6.1-1.3.dsc",
                        "pyspi_0.6.1-1.3.diff.gz",
                        "pyspi_0.6.1.orig.tar.gz").status_code, 200)

        resp = self.post("/api/repos/" + repo_name + "/file/" + d, params={'_async': True})
        self.check_equal(resp.status_code, 202)

        return resp.json()['ID'], repo_name

    def _wait_for_task(self, task_id):
        uri = "/api/tasks/%d/wait" % int(task_id)
        resp = self.get(uri)
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json()['State'], 2)

    def _wait_for_all_tasks(self):
        resp = self.get("/api/tasks-wait")
        self.check_equal(resp.status_code, 200)

    def _snapshot(self, res_type, name):
        uri = "/api/%s/%s/snapshots" % (res_type, name)
        resp = self.post(uri, json={"Name": name}, params={'_async': True})
        self.check_equal(resp.status_code, 202)

        return resp.json()['ID']

    def _publish(self, source_kind, name):
        resp = self.post("/api/publish",
                         json={
                             "SourceKind": source_kind,
                             "Sources": [{"Name": name}],
                             "Signing": DefaultSigningOptions,
                         }, params={'_async': True})
        self.check_equal(resp.status_code, 202)
        return resp.json()['ID']

    def check(self):
        publish_task_ids = []
        mirror_task_list = []
        for mirror_dist in ['squeeze', 'jessie']:
            mirror_task_id, mirror_name = self._create_mirror(mirror_dist)
            mirror_task_list.append((mirror_task_id, mirror_name))
        repo_task_id, repo_name = self._create_repo()

        self._wait_for_task(repo_task_id)

        resp = self.delete("/api/tasks/%d" % repo_task_id)
        self.check_equal(resp.status_code, 200)
        resp = self.get("/api/tasks/%d" % repo_task_id)
        self.check_equal(resp.status_code, 404)

        repo_snap_task_id = self._snapshot('repos', repo_name)
        self._wait_for_task(repo_snap_task_id)
        publish_task_ids.append(self._publish('snapshot', repo_name))

        for mirror_task_id, mirror_name in mirror_task_list:
            self._wait_for_task(mirror_task_id)
            mirror_snap_task_id = self._snapshot('mirrors', mirror_name)

            self._wait_for_task(mirror_snap_task_id)
            publish_task_ids.append(self._publish('snapshot', mirror_name))

        self._wait_for_all_tasks()

        for publish_task_id in publish_task_ids:
            resp = self.get("/api/tasks/%d" % publish_task_id)
            self.check_equal(resp.status_code, 200)
            self.check_equal(resp.json()['State'], 2)
