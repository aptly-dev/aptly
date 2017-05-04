from api_lib import APITest
from publish import DefaultSigningOptions


class ReposAPITestCreateShow(APITest):
    """
    GET /api/repos/:name, POST /api/repos, GET /api/repos/:name/packages
    """
    def check(self):
        repo_name = self.random_name()
        repo_desc = {u'Comment': u'fun repo',
                     u'DefaultComponent': u'',
                     u'DefaultDistribution': u'',
                     u'Name': repo_name}

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

        self.check_equal(self.delete("/api/repos/" + repo_name).status_code, 200)
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

        resp = self.post("/api/repos/" + repo_name + "/file/" + d)
        self.check_equal(resp.status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo_name + "/snapshots", json={"Name": repo_name}).status_code, 201)

        self.check_equal(self.post("/api/publish",
                         json={
                             "SourceKind": "local",
                             "Sources": [{"Name": repo_name}],
                             "Signing": DefaultSigningOptions,
                         }).status_code, 201)

        # repo is not deletable while it is published
        self.check_equal(self.delete("/api/repos/" + repo_name).status_code, 409)
        self.check_equal(self.delete("/api/repos/" + repo_name, params={"force": "1"}).status_code, 409)

        # drop published
        self.check_equal(self.delete("/api/publish//" + distribution).status_code, 200)
        self.check_equal(self.delete("/api/repos/" + repo_name).status_code, 409)
        self.check_equal(self.delete("/api/repos/" + repo_name, params={"force": "1"}).status_code, 200)
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

        resp = self.post("/api/repos/" + repo_name + "/file/" + d)
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json(), {
            u'FailedFiles': [],
            u'Report': {
                u'Added': [u'pyspi_0.6.1-1.3_source added'],
                u'Removed': [],
                u'Warnings': []}})

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

        self.check_equal(self.post("/api/repos/" + repo_name + "/file/" + d).status_code, 200)
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

        self.check_equal(self.post("/api/repos/" + repo_name + "/file/" + d, params={"noRemove": 1}).status_code, 200)
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

        resp = self.post("/api/repos/" + repo_name + "/file/" + d + "/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json(), {
            u'FailedFiles': [],
            u'Report': {
                u'Added': [u'libboost-program-options-dev_1.49.0.1_i386 added'],
                u'Removed': [],
                u'Warnings': []}})

        self.check_equal(self.get("/api/repos/" + repo_name + "/packages").json(),
                         ['Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378'])

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
        self.check_equal(self.post("/api/repos/" + repo_name + "/file/" + d).status_code, 200)

        self.check_equal(sorted(self.get("/api/repos/" + repo_name + "/packages", params={"q": "pyspi"}).json()),
                         ['Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e', 'Psource pyspi 0.6.1-1.4 f8f1daa806004e89'])

        self.check_equal(sorted(self.get("/api/repos/" + repo_name + "/packages", params={"q": "Version (> 0.6.1-1.4)"}).json()),
                         ['Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378', 'Psource pyspi 0.6.1-1.4 f8f1daa806004e89'])

        self.check_equal(sorted(p['Key'] for p in self.get("/api/repos/" + repo_name + "/packages",
                                                           params={"q": "pyspi", "format": "details"}).json()),
                         ['Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e', 'Psource pyspi 0.6.1-1.4 f8f1daa806004e89'])

        resp = self.get("/api/repos/" + repo_name + "/packages", params={"q": "pyspi)"})
        self.check_equal(resp.status_code, 400)
        self.check_equal(resp.json()[0]["error"], u'parsing failed: unexpected token ): expecting end of query')


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

        self.check_equal(self.post("/api/repos/" + repo_name + "/file/" + d + "/pyspi_0.6.1-1.3.dsc",
                                   params={"noRemove": 1}).status_code, 200)

        self.check_equal(sorted(self.get("/api/repos/" + repo_name + "/packages").json()),
                         ['Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e'])

        self.check_equal(self.post("/api/repos/" + repo_name + "/file/" + d + "/pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)

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

        self.check_equal(self.post("/api/repos/" + repo_name + "/file/" + d).status_code, 200)

        self.check_equal(sorted(self.get("/api/repos/" + repo_name + "/packages").json()),
                         ['Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378',
                          'Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e',
                          'Psource pyspi 0.6.1-1.4 f8f1daa806004e89'])

        self.check_equal(self.post("/api/repos/" + repo_name + "/packages/",
                         json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89']}).status_code, 200)

        self.check_equal(sorted(self.get("/api/repos/" + repo_name + "/packages").json()),
                         ['Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378',
                          'Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e',
                          'Psource pyspi 0.6.1-1.4 f8f1daa806004e89'])

        self.check_equal(self.post("/api/repos/" + repo_name + "/packages/",
                         json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89',
                                               'Psource no-such-package 0.6.1-1.4 f8f1daa806004e89']}).status_code, 404)

        self.check_equal(self.delete("/api/repos/" + repo_name + "/packages/",
                         json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89']}).status_code, 200)

        self.check_equal(sorted(self.get("/api/repos/" + repo_name + "/packages").json()),
                         ['Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378',
                          'Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e'])

        self.check_equal(self.post("/api/repos/" + repo_name + "/packages/",
                         json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89']}).status_code, 200)

        self.check_equal(sorted(self.get("/api/repos/" + repo_name + "/packages").json()),
                         ['Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378',
                          'Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e',
                          'Psource pyspi 0.6.1-1.4 f8f1daa806004e89'])

        repo_name2 = self.random_name()

        self.check_equal(self.post("/api/repos", json={"Name": repo_name2, "Comment": "fun repo"}).status_code, 201)

        self.check_equal(self.post("/api/repos/" + repo_name2 + "/packages/",
                         json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89',
                                               'Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378']}).status_code, 200)

        self.check_equal(sorted(self.get("/api/repos/" + repo_name2 + "/packages").json()),
                         ['Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378',
                          'Psource pyspi 0.6.1-1.4 f8f1daa806004e89'])
