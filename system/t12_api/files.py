import inspect
import os
import shutil
import tempfile

from api_lib import APITest
from lib import BaseTest


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
        self.check_equal(self.delete("/api/files/.").status_code, 404)
        self.check_equal(self.delete("/api/files").status_code, 404)
        self.check_equal(self.delete("/api/files/").status_code, 404)
        self.check_equal(self.delete("/api/files/../.").status_code, 404)
        self.check_equal(self.delete("/api/files/./..").status_code, 404)
        self.check_equal(self.delete("/api/files/dir/..").status_code, 404)


class FilesAPITestDputUpload(APITest):
    """
    PUT /api/files/:dir/:file via dput, then POST /api/repos/:name/include/:dir

    Uses the real dput binary to upload a .changes file and all its referenced
    files to the aptly API, then imports them into a local repo via include.
    Skipped if dput is not installed.
    """

    def fixture_available(self):
        return super().fixture_available() and shutil.which("dput") is not None

    def check(self):
        d = self.random_name()
        repo_name = self.random_name()

        # Create target repo
        self.check_equal(
            self.post("/api/repos", json={"Name": repo_name}).status_code, 201)

        changes_dir = os.path.join(
            os.path.dirname(inspect.getsourcefile(BaseTest)), "changes")
        changes_file = os.path.join(changes_dir, "hardlink_0.2.1_amd64.changes")

        # dput strips leading/trailing slashes from 'incoming' then prepends /,
        # producing: PUT http://{fqdn}/api/files/{d}/{filename}
        # fqdn includes host:port so dput connects directly to the test API server.
        dput_cf = (
            "[aptly]\n"
            f"fqdn     = {self.base_url}\n"
            "method   = http\n"
            f"incoming = api/files/{d}\n"
            "login    = *\n"
            "allow_unsigned_uploads = 1\n"
            "allow_dcut = 0\n"
        )

        tmpdir = tempfile.mkdtemp()
        try:
            dput_cf_path = os.path.join(tmpdir, "dput.cf")
            with open(dput_cf_path, "w") as f:
                f.write(dput_cf)

            # dput -U: allow unsigned uploads (skip local GPG check)
            # dput reads the .changes and PUTs every file listed in Files: + the .changes itself
            self.run_cmd(["dput", "-c", dput_cf_path, "-U", "aptly", changes_file])
        finally:
            shutil.rmtree(tmpdir)

        # All files referenced in the .changes must now be present in the upload dir
        self.check_exists(f"upload/{d}/hardlink_0.2.1_amd64.changes")
        self.check_exists(f"upload/{d}/hardlink_0.2.1.dsc")
        self.check_exists(f"upload/{d}/hardlink_0.2.1.tar.gz")
        self.check_exists(f"upload/{d}/hardlink_0.2.1_amd64.deb")

        # Import via the .changes file into the repo
        resp = self.post_task(
            f"/api/repos/{repo_name}/include/{d}",
            params={"ignoreSignature": 1})
        self.check_task(resp)

        output = self.get(f"/api/tasks/{resp.json()['ID']}/output")
        self.check_in(b"Added: hardlink_0.2.1_source added, hardlink_0.2.1_amd64 added", output.content)

        # Packages must be in the repo
        self.check_equal(
            sorted(self.get(f"/api/repos/{repo_name}/packages").json()),
            ["Pamd64 hardlink 0.2.1 daf8fcecbf8210ad", "Psource hardlink 0.2.1 8f72df429d7166e5"])

        # include cleans up the upload dir
        self.check_not_exists(f"upload/{d}")
