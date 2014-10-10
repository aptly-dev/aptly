from api_lib import APITest


class ReposAPITestCreateShow(APITest):
    """
    GET /api/repos/:name, POST /api/repos
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
