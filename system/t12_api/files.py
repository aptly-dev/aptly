from api_lib import APITest


class FilesAPITestUpload(APITest):
    """
    POST /files/:dir
    """

    def check(self):
        d = self.random_name()
        resp = self.upload("/api/files/" + d, "pyspi_0.6.1-1.3.dsc")
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json(), [d + '/pyspi_0.6.1-1.3.dsc'])
        self.check_exists("upload/" + d + '/pyspi_0.6.1-1.3.dsc')


class FilesAPITestUploadMulti(APITest):
    """
    POST /files/:dir, GET /files/:dir multi files
    """

    def check(self):
        d = self.random_name()

        self.check_equal(self.get("/api/files/" + d).status_code, 404)

        resp = self.upload("/api/files/" + d, "pyspi_0.6.1-1.3.dsc", "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz")
        self.check_equal(resp.status_code, 200)
        self.check_equal(sorted(resp.json()),
                         [d + '/pyspi_0.6.1-1.3.diff.gz', d + '/pyspi_0.6.1-1.3.dsc', d + '/pyspi_0.6.1.orig.tar.gz'])
        self.check_exists("upload/" + d + '/pyspi_0.6.1-1.3.dsc')
        self.check_exists("upload/" + d + '/pyspi_0.6.1-1.3.diff.gz')
        self.check_exists("upload/" + d + '/pyspi_0.6.1.orig.tar.gz')

        resp = self.get("/api/files/" + d)
        self.check_equal(resp.status_code, 200)
        self.check_equal(sorted(resp.json()),
                         ['pyspi_0.6.1-1.3.diff.gz', 'pyspi_0.6.1-1.3.dsc', 'pyspi_0.6.1.orig.tar.gz'])


class FilesAPITestList(APITest):
    """
    GET /files/
    """

    def check(self):
        d1, d2, d3 = self.random_name(), self.random_name(), self.random_name()

        resp = self.get("/api/files")
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json(), [])

        self.check_equal(self.upload("/api/files/" + d1, "pyspi_0.6.1-1.3.dsc").status_code, 200)
        self.check_equal(self.upload("/api/files/" + d2, "pyspi_0.6.1-1.3.dsc").status_code, 200)
        self.check_equal(self.upload("/api/files/" + d3, "pyspi_0.6.1-1.3.dsc").status_code, 200)

        resp = self.get("/api/files")
        self.check_equal(resp.status_code, 200)
        self.check_equal(sorted(resp.json()), sorted([d1, d2, d3]))


class FilesAPITestDelete(APITest):
    """
    DELETE /files/:dir, DELETE /files/:dir/:name
    """

    def check(self):
        d1, d2 = self.random_name(), self.random_name()

        self.check_equal(self.get("/api/files").json(), [])
        self.check_equal(self.delete("/api/files/" + d1).status_code, 200)
        self.check_equal(self.delete("/api/files/" + d1 + "/" + "pyspi_0.6.1-1.3.dsc").status_code, 200)

        self.check_equal(self.upload("/api/files/" + d1, "pyspi_0.6.1-1.3.dsc").status_code, 200)
        self.check_equal(self.upload("/api/files/" + d2, "pyspi_0.6.1-1.3.dsc").status_code, 200)

        self.check_equal(self.delete("/api/files/" + d1).status_code, 200)
        self.check_equal(self.get("/api/files").json(), [d2])

        self.check_equal(self.delete("/api/files/" + d2 + "/" + "no-such-file").status_code, 200)
        self.check_equal(self.get("/api/files/" + d2).json(), ["pyspi_0.6.1-1.3.dsc"])

        self.check_equal(self.delete("/api/files/" + d2 + "/" + "pyspi_0.6.1-1.3.dsc").status_code, 200)
        self.check_equal(self.get("/api/files").json(), [d2])
        self.check_equal(self.get("/api/files/" + d2).json(), [])


class FilesAPITestSecurity(APITest):
    """
    delete & upload security
    """

    def check(self):
        self.check_equal(self.delete("/api/files/.").status_code, 400)
        self.check_equal(self.delete("/api/files").status_code, 405)
        self.check_equal(self.delete("/api/files/").status_code, 404)
        self.check_equal(self.delete("/api/files/../.").status_code, 400)
        self.check_equal(self.delete("/api/files/./..").status_code, 400)
        self.check_equal(self.delete("/api/files/dir/..").status_code, 400)
