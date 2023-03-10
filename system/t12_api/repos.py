from api_lib import TASK_FAILED, TASK_SUCCEEDED, APITest

from .publish import DefaultSigningOptions


class ReposAPITestCreateShow(APITest):
    """
    GET /api/repos/:name, POST /api/repos, GET /api/repos/:name/packages
    """
    def check(self):
        repo_name = self.random_name()
        repo_desc = {'Comment': 'fun repo',
                     'DefaultComponent': '',
                     'DefaultDistribution': '',
                     'Name': repo_name}

        resp = self.post("/api/repos", json={"Name": repo_name, "Comment": "fun repo"})
        self.check_equal(resp.json(), repo_desc)
        self.check_equal(resp.status_code, 201)

        self.check_equal(self.get("/api/repos/" + repo_name).json(), repo_desc)
        self.check_equal(self.get("/api/repos/" + repo_name).status_code, 200)

        resp = self.get("/api/repos/" + repo_name + "/packages")
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json(), [])

        self.check_equal(self.get("/api/repos/" + self.random_name()).status_code, 404)


class ReposAPITestCreateIndexDelete(APITest):
    """
    GET /api/repos, POST /api/repos, DELETE /api/repos/:name
    """
    def check(self):
        repo_name = self.random_name()

        self.check_equal(self.post("/api/repos", json={"Name": repo_name, "Comment": "fun repo"}).status_code, 201)

        repos = self.get("/api/repos/").json()
        names = [repo["Name"] for repo in repos]
        assert repo_name in names

        self.check_equal(self.delete_task("/api/repos/" + repo_name).json()['State'], TASK_SUCCEEDED)
        self.check_equal(self.delete("/api/repos/" + repo_name).status_code, 404)

        self.check_equal(self.get("/api/repos/" + repo_name).status_code, 404)

        self.check_equal(self.delete("/api/repos/" + self.random_name()).status_code, 404)

        # create once again
        distribution = self.random_name()
        self.check_equal(self.post("/api/repos",
                         json={
                             "Name": repo_name,
                             "Comment": "fun repo",
                             "DefaultDistribution": distribution
                         }).status_code, 201)
        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "pyspi_0.6.1-1.3.dsc", "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz").status_code, 200)

        resp = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)

        self.check_equal(self.post_task("/api/repos/" + repo_name + "/snapshots", json={"Name": repo_name}).json()['State'], TASK_SUCCEEDED)

        self.check_equal(self.post_task("/api/publish",
                         json={
                             "SourceKind": "local",
                             "Sources": [{"Name": repo_name}],
                             "Signing": DefaultSigningOptions,
                         }).json()['State'], TASK_SUCCEEDED)

        # repo is not deletable while it is published
        self.check_equal(self.delete_task("/api/repos/" + repo_name).json()['State'], TASK_FAILED)
        self.check_equal(self.delete_task("/api/repos/" + repo_name, params={"force": "1"}).json()['State'], TASK_FAILED)

        # drop published
        self.check_equal(self.delete_task("/api/publish//" + distribution).json()['State'], TASK_SUCCEEDED)
        self.check_equal(self.delete_task("/api/repos/" + repo_name).json()['State'], TASK_FAILED)
        self.check_equal(self.delete_task("/api/repos/" + repo_name, params={"force": "1"}).json()['State'], TASK_SUCCEEDED)
        self.check_equal(self.get("/api/repos/" + repo_name).status_code, 404)


class ReposAPITestAdd(APITest):
    """
    POST /api/repos/:name/file/:dir, GET /api/repos/:name/packages
    """
    def check(self):
        repo_name = self.random_name()

        self.check_equal(self.post("/api/repos", json={"Name": repo_name, "Comment": "fun repo"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "pyspi_0.6.1-1.3.dsc", "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz").status_code, 200)

        resp = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)

        resp = self.get("/api/tasks/" + str(resp.json()['ID']) + "/output")
        self.check_equal(resp.status_code, 200)

        self.check_in(b"Added: pyspi_0.6.1-1.3_source added", resp.content)
        self.check_not_in(b"Removed: ", resp.content)
        self.check_not_in(b"Failed files: ", resp.content)
        self.check_not_in(b"Warnings: ", resp.content)

        self.check_equal(self.get("/api/repos/" + repo_name + "/packages").json(), ['Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e'])

        self.check_not_exists("upload/" + d)


class ReposAPITestAddNotFullRemove(APITest):
    """
    POST /api/repos/:name/file/:dir not all files removed
    """
    def check(self):
        repo_name = self.random_name()

        self.check_equal(self.post("/api/repos", json={"Name": repo_name, "Comment": "fun repo"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "pyspi_0.6.1-1.3.dsc", "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz", "aptly.pub").status_code, 200)

        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], TASK_SUCCEEDED)
        self.check_equal(self.get("/api/repos/" + repo_name + "/packages").json(), ['Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e'])

        self.check_exists("upload/" + d + "/aptly.pub")
        self.check_not_exists("upload/" + d + "/pyspi_0.6.1-1.3.dsc")


class ReposAPITestAddNoRemove(APITest):
    """
    POST /api/repos/:name/file/:dir no remove
    """
    def check(self):
        repo_name = self.random_name()

        self.check_equal(self.post("/api/repos", json={"Name": repo_name, "Comment": "fun repo"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "pyspi_0.6.1-1.3.dsc", "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz").status_code, 200)

        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d, params={"noRemove": 1}).json()['State'], TASK_SUCCEEDED)
        self.check_equal(self.get("/api/repos/" + repo_name + "/packages").json(), ['Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e'])

        self.check_exists("upload/" + d + "/pyspi_0.6.1-1.3.dsc")


class ReposAPITestAddFile(APITest):
    """
    POST /api/repos/:name/file/:dir/:file
    """
    def check(self):
        repo_name = self.random_name()

        self.check_equal(self.post("/api/repos", json={"Name": repo_name, "Comment": "fun repo"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)

        resp = self.post_task("/api/repos/" + repo_name + "/file/" + d + "/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)

        resp = self.get("/api/tasks/" + str(resp.json()['ID']) + "/output")
        self.check_equal(resp.status_code, 200)

        self.check_in(b"Added: libboost-program-options-dev_1.49.0.1_i386 added", resp.content)
        self.check_not_in(b"Removed: ", resp.content)
        self.check_not_in(b"Failed files: ", resp.content)
        self.check_not_in(b"Warnings: ", resp.content)

        self.check_equal(self.get("/api/repos/" + repo_name + "/packages").json(),
                         ['Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378'])

        self.check_not_exists("upload/" + d)


class ReposAPITestInclude(APITest):
    """
    POST /api/repos/:name/include/:dir, GET /api/repos/:name/packages
    """
    def check(self):
        repo_name = self.random_name()

        self.check_equal(self.post("/api/repos", json={"Name": repo_name, "Comment": "fun repo"}).status_code, 201)

        d = self.random_name()
        resp = self.upload("/api/files/" + d, "hardlink_0.2.1_amd64.changes",
                           "hardlink_0.2.1.dsc", "hardlink_0.2.1.tar.gz",
                           "hardlink_0.2.1_amd64.deb", directory='changes')
        self.check_equal(resp.status_code, 200)

        resp = self.post_task("/api/repos/" + repo_name + "/include/" + d, params={"ignoreSignature": 1})
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)

        resp = self.get("/api/tasks/" + str(resp.json()['ID']) + "/output")
        self.check_equal(resp.status_code, 200)
        self.check_in(b"Added: hardlink_0.2.1_source added, hardlink_0.2.1_amd64 added", resp.content)
        self.check_equal(
            sorted(self.get("/api/repos/" + repo_name + "/packages").json()),
            ['Pamd64 hardlink 0.2.1 daf8fcecbf8210ad', 'Psource hardlink 0.2.1 8f72df429d7166e5']
        )

        self.check_not_exists("upload/" + d)


class ReposAPITestShowQuery(APITest):
    """
    GET /api/repos/:name/packages?q=query
    """
    def check(self):
        repo_name = self.random_name()

        self.check_equal(self.post("/api/repos", json={"Name": repo_name, "Comment": "fun repo"}).status_code, 201)

        d = self.random_name()
        self.check_equal(
            self.upload("/api/files/" + d,
                        "libboost-program-options-dev_1.49.0.1_i386.deb", "pyspi_0.6.1-1.3.dsc",
                        "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                        "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)
        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], TASK_SUCCEEDED)

        self.check_equal(sorted(self.get("/api/repos/" + repo_name + "/packages", params={"q": "pyspi"}).json()),
                         ['Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e', 'Psource pyspi 0.6.1-1.4 f8f1daa806004e89'])

        self.check_equal(sorted(self.get("/api/repos/" + repo_name + "/packages", params={"q": "Version (> 0.6.1-1.4)"}).json()),
                         ['Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378', 'Psource pyspi 0.6.1-1.4 f8f1daa806004e89'])

        self.check_equal(sorted(p['Key'] for p in self.get("/api/repos/" + repo_name + "/packages",
                                                           params={"q": "pyspi", "format": "details"}).json()),
                         ['Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e', 'Psource pyspi 0.6.1-1.4 f8f1daa806004e89'])

        resp = self.get("/api/repos/" + repo_name + "/packages", params={"q": "pyspi)"})
        self.check_equal(resp.status_code, 400)
        self.check_equal(resp.json()["error"], 'parsing failed: unexpected token ): expecting end of query')


class ReposAPITestAddMultiple(APITest):
    """
    POST /api/repos/:name/file/:dir/:file multiple
    """
    def check(self):
        repo_name = self.random_name()

        self.check_equal(self.post("/api/repos", json={"Name": repo_name, "Comment": "fun repo"}).status_code, 201)

        d = self.random_name()
        self.check_equal(
            self.upload("/api/files/" + d,
                        "libboost-program-options-dev_1.49.0.1_i386.deb", "pyspi_0.6.1-1.3.dsc",
                        "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                        "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)

        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d + "/pyspi_0.6.1-1.3.dsc",
                                        params={"noRemove": 1}).json()['State'], TASK_SUCCEEDED)

        self.check_equal(sorted(self.get("/api/repos/" + repo_name + "/packages").json()),
                         ['Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e'])

        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d + "/pyspi-0.6.1-1.3.stripped.dsc").json()['State'], TASK_SUCCEEDED)

        self.check_equal(sorted(self.get("/api/repos/" + repo_name + "/packages").json()),
                         ['Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e', 'Psource pyspi 0.6.1-1.4 f8f1daa806004e89'])


class ReposAPITestPackagesAddDelete(APITest):
    """
    POST/DELETE /api/repos/:name/packages
    """
    def check(self):
        repo_name = self.random_name()

        self.check_equal(self.post("/api/repos", json={"Name": repo_name, "Comment": "fun repo"}).status_code, 201)

        d = self.random_name()
        self.check_equal(
            self.upload("/api/files/" + d,
                        "libboost-program-options-dev_1.49.0.1_i386.deb", "pyspi_0.6.1-1.3.dsc",
                        "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                        "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)

        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], TASK_SUCCEEDED)

        self.check_equal(sorted(self.get("/api/repos/" + repo_name + "/packages").json()),
                         ['Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378',
                          'Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e',
                          'Psource pyspi 0.6.1-1.4 f8f1daa806004e89'])

        self.check_equal(self.post_task("/api/repos/" + repo_name + "/packages/",
                         json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89']}).json()['State'], TASK_SUCCEEDED)

        self.check_equal(sorted(self.get("/api/repos/" + repo_name + "/packages").json()),
                         ['Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378',
                          'Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e',
                          'Psource pyspi 0.6.1-1.4 f8f1daa806004e89'])

        self.check_equal(self.post_task("/api/repos/" + repo_name + "/packages/",
                         json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89',
                                               'Psource no-such-package 0.6.1-1.4 f8f1daa806004e89']}).json()['State'], TASK_FAILED)

        self.check_equal(self.delete_task("/api/repos/" + repo_name + "/packages/",
                         json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89']}).json()['State'], TASK_SUCCEEDED)

        self.check_equal(sorted(self.get("/api/repos/" + repo_name + "/packages").json()),
                         ['Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378',
                          'Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e'])

        self.check_equal(self.post_task("/api/repos/" + repo_name + "/packages/",
                         json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89']}).json()['State'], TASK_SUCCEEDED)

        self.check_equal(sorted(self.get("/api/repos/" + repo_name + "/packages").json()),
                         ['Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378',
                          'Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e',
                          'Psource pyspi 0.6.1-1.4 f8f1daa806004e89'])

        repo_name2 = self.random_name()

        self.check_equal(self.post("/api/repos", json={"Name": repo_name2, "Comment": "fun repo"}).status_code, 201)

        self.check_equal(self.post_task("/api/repos/" + repo_name2 + "/packages/",
                         json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89',
                                               'Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378']}).json()['State'], TASK_SUCCEEDED)

        self.check_equal(sorted(self.get("/api/repos/" + repo_name2 + "/packages").json()),
                         ['Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378',
                          'Psource pyspi 0.6.1-1.4 f8f1daa806004e89'])
