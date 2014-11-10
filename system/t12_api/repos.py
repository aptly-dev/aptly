from api_lib import APITest


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
            u'failedFiles': [],
            u'report': {
                u'added': [u'pyspi_0.6.1-1.3_source added'],
                u'removed': [],
                u'warnings': []}})

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
            u'failedFiles': [],
            u'report': {
                u'added': [u'libboost-program-options-dev_1.49.0.1_i386 added'],
                u'removed': [],
                u'warnings': []}})

        self.check_equal(self.get("/api/repos/" + repo_name + "/packages").json(),
                         ['Pi386 libboost-program-options-dev 1.49.0.1 918d2f433384e378'])

        self.check_not_exists("upload/" + d)
