import inspect
import os
import threading

from api_lib import TASK_SUCCEEDED, APITest

DefaultSigningOptions = {
    "Keyring": os.path.join(os.path.dirname(inspect.getsourcefile(APITest)), "files") + "/aptly.pub",
}


class PublishAPITestRepo(APITest):
    """
    POST /publish/:prefix (local repos), GET /publish
    """
    fixtureGpg = True

    def check(self):
        repo_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb", "pyspi_0.6.1-1.3.dsc",
                                     "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                                     "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)

        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        # publishing under prefix, default distribution
        prefix = self.random_name()
        task = self.post_task(
            "/api/publish/" + prefix,
            json={
                 "SourceKind": "local",
                 "Sources": [{"Name": repo_name}],
                 "Signing": DefaultSigningOptions,
            }
        )
        self.check_task(task)
        repo_expected = {
            'AcquireByHash': False,
            'Architectures': ['i386', 'source'],
            'Codename': '',
            'Distribution': 'wheezy',
            'Label': '',
            'Origin': '',
            'NotAutomatic': '',
            'ButAutomaticUpgrades': '',
            'Path': prefix + '/' + 'wheezy',
            'Prefix': prefix,
            'SignedBy': '',
            'SkipContents': False,
            'MultiDist': False,
            'SourceKind': 'local',
            'Sources': [{'Component': 'main', 'Name': repo_name}],
            'Storage': '',
            'Suite': ''}

        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())

        self.check_exists("public/" + prefix + "/dists/wheezy/Release")
        self.check_exists("public/" + prefix +
                          "/dists/wheezy/main/binary-i386/Packages")
        self.check_exists("public/" + prefix +
                          "/dists/wheezy/main/Contents-i386.gz")
        self.check_exists("public/" + prefix +
                          "/dists/wheezy/main/source/Sources")
        self.check_exists(
            "public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")

        # publishing under root, custom distribution, architectures
        distribution = self.random_name()
        task = self.post_task(
            "/api/publish/:.",
            json={
                 "SourceKind": "local",
                 "Sources": [{"Name": repo_name}],
                 "Signing": DefaultSigningOptions,
                 "Distribution": distribution,
                 "Architectures": ["i386", "amd64"],
            }
        )
        self.check_task(task)
        repo2_expected = {
            'AcquireByHash': False,
            'Architectures': ['amd64', 'i386'],
            'Codename': '',
            'Distribution': distribution,
            'Label': '',
            'Origin': '',
            'NotAutomatic': '',
            'ButAutomaticUpgrades': '',
            'Path': './' + distribution,
            'Prefix': ".",
            'SignedBy': '',
            'SkipContents': False,
            'MultiDist': False,
            'SourceKind': 'local',
            'Sources': [{'Component': 'main', 'Name': repo_name}],
            'Storage': '',
            'Suite': ''}
        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())

        self.check_exists("public/dists/" + distribution + "/Release")
        self.check_exists("public/dists/" + distribution +
                          "/main/binary-i386/Packages")
        self.check_exists("public/dists/" + distribution +
                          "/main/Contents-i386.gz")
        self.check_exists("public/dists/" + distribution +
                          "/main/binary-amd64/Packages")
        self.check_not_exists(
            "public/dists/" + distribution + "/main/Contents-amd64.gz")
        self.check_exists(
            "public/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")

        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())
        self.check_in(repo2_expected, all_repos.json())


class PublishAPITestRepoMultiDist(APITest):
    """
    Test MultiDist publishing to subdirectory
    """
    fixtureGpg = True

    def check(self):
        repo_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo_name, "DefaultDistribution": "bookworm"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb", "pyspi_0.6.1-1.3.dsc",
                                     "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                                     "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)

        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        # publishing under prefix, default distribution
        prefix = self.random_name()
        task = self.post_task(
            "/api/publish/" + prefix,
            json={
                 "SourceKind": "local",
                 "MultiDist": True,
                 "Sources": [{"Name": repo_name}],
                 "Signing": DefaultSigningOptions,
            }
        )
        self.check_task(task)
        repo_expected = {
            'AcquireByHash': False,
            'Architectures': ['i386', 'source'],
            'Codename': '',
            'Distribution': 'bookworm',
            'Label': '',
            'Origin': '',
            'NotAutomatic': '',
            'ButAutomaticUpgrades': '',
            'Path': prefix + '/' + 'bookworm',
            'Prefix': prefix,
            'SignedBy': '',
            'SkipContents': False,
            'MultiDist': True,
            'SourceKind': 'local',
            'Sources': [{'Component': 'main', 'Name': repo_name}],
            'Storage': '',
            'Suite': ''}

        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())

        self.check_exists("public/" + prefix + "/dists/bookworm/Release")
        self.check_exists("public/" + prefix +
                          "/dists/bookworm/main/binary-i386/Packages")
        self.check_exists("public/" + prefix +
                          "/dists/bookworm/main/Contents-i386.gz")
        self.check_exists("public/" + prefix +
                          "/dists/bookworm/main/source/Sources")
        self.check_exists(
            "public/" + prefix + "/pool/bookworm/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")


class PublishAPITestRepoSignedBy(APITest):
    """
    POST /publish/:prefix (local repos), GET /publish
    """
    fixtureGpg = True

    def check(self):
        repo_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb", "pyspi_0.6.1-1.3.dsc",
                                     "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                                     "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)

        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        # publishing under prefix, default distribution
        prefix = self.random_name()
        task = self.post_task(
            "/api/publish/" + prefix,
            json={
                 "SourceKind": "local",
                 "Sources": [{"Name": repo_name}],
                 "Signing": DefaultSigningOptions,
                 "SignedBy": "just,a,string"
            }
        )
        self.check_task(task)
        repo_expected = {
            'AcquireByHash': False,
            'Architectures': ['i386', 'source'],
            'Codename': '',
            'Distribution': 'wheezy',
            'Label': '',
            'Origin': '',
            'NotAutomatic': '',
            'ButAutomaticUpgrades': '',
            'Path': prefix + '/' + 'wheezy',
            'Prefix': prefix,
            'SignedBy': 'just,a,string',
            'SkipContents': False,
            'MultiDist': False,
            'SourceKind': 'local',
            'Sources': [{'Component': 'main', 'Name': repo_name}],
            'Storage': '',
            'Suite': ''}

        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())


class PublishSnapshotAPITest(APITest):
    """
    POST /publish/:prefix (snapshots), GET /publish
    """

    def check(self):
        repo_name = self.random_name()
        snapshot_name = self.random_name()
        self.check_equal(
            self.post("/api/repos", json={"Name": repo_name}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)

        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        task = self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot_name})
        self.check_task(task)

        prefix = self.random_name()
        task = self.post_task(
            "/api/publish/" + prefix,
            json={
                "AcquireByHash": True,
                "SourceKind": "snapshot",
                "Sources": [{"Name": snapshot_name}],
                "Signing": DefaultSigningOptions,
                "Distribution": "squeeze",
                "NotAutomatic": "yes",
                "ButAutomaticUpgrades": "yes",
                "Origin": "earth",
                "Label": "fun",
            }
        )
        self.check_task(task)

        _id = task.json()['ID']
        resp = self.get("/api/tasks/" + str(_id) + "/detail")
        self.check_equal(resp.json()['RemainingNumberOfPackages'], 0)
        self.check_equal(resp.json()['TotalNumberOfPackages'], 1)

        repo_expected = {
            'AcquireByHash': True,
            'Architectures': ['i386'],
            'Codename': '',
            'Distribution': 'squeeze',
            'Label': 'fun',
            'Origin': 'earth',
            'MultiDist': False,
            'NotAutomatic': 'yes',
            'ButAutomaticUpgrades': 'yes',
            'Path': prefix + '/' + 'squeeze',
            'Prefix': prefix,
            'SignedBy': '',
            'SkipContents': False,
            'SourceKind': 'snapshot',
            'Sources': [{'Component': 'main', 'Name': snapshot_name}],
            'Storage': '',
            'Suite': '',
        }
        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())

        self.check_exists("public/" + prefix + "/dists/squeeze/Release")
        self.check_exists("public/" + prefix +
                          "/dists/squeeze/main/binary-i386/by-hash")
        self.check_exists("public/" + prefix +
                          "/dists/squeeze/main/binary-i386/Packages")
        self.check_exists("public/" + prefix +
                          "/dists/squeeze/main/Contents-i386.gz")
        self.check_exists(
            "public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")


class PublishSnapshotAPITestSignedBy(APITest):
    """
    POST /publish/:prefix (snapshots), GET /publish
    """

    def check(self):
        repo_name = self.random_name()
        snapshot_name = self.random_name()
        self.check_equal(
            self.post("/api/repos", json={"Name": repo_name}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)

        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        task = self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot_name})
        self.check_task(task)

        prefix = self.random_name()
        task = self.post_task(
            "/api/publish/" + prefix,
            json={
                "AcquireByHash": True,
                "SourceKind": "snapshot",
                "Sources": [{"Name": snapshot_name}],
                "Signing": DefaultSigningOptions,
                "Distribution": "squeeze",
                "NotAutomatic": "yes",
                "ButAutomaticUpgrades": "yes",
                "Origin": "earth",
                "Label": "fun",
                "SignedBy": "just,a,string",
            }
        )
        self.check_task(task)

        _id = task.json()['ID']
        resp = self.get("/api/tasks/" + str(_id) + "/detail")
        self.check_equal(resp.json()['RemainingNumberOfPackages'], 0)
        self.check_equal(resp.json()['TotalNumberOfPackages'], 1)

        repo_expected = {
            'AcquireByHash': True,
            'Architectures': ['i386'],
            'Codename': '',
            'Distribution': 'squeeze',
            'Label': 'fun',
            'Origin': 'earth',
            'MultiDist': False,
            'NotAutomatic': 'yes',
            'ButAutomaticUpgrades': 'yes',
            'Path': prefix + '/' + 'squeeze',
            'Prefix': prefix,
            'SignedBy': 'just,a,string',
            'SkipContents': False,
            'SourceKind': 'snapshot',
            'Sources': [{'Component': 'main', 'Name': snapshot_name}],
            'Storage': '',
            'Suite': '',
        }
        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())


class PublishUpdateAPITestRepo(APITest):
    """
    PUT /publish/:prefix/:distribution (local repos), DELETE /publish/:prefix/:distribution
    """
    fixtureGpg = True

    def check(self):
        repo_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(
            self.upload("/api/files/" + d,
                        "pyspi_0.6.1-1.3.dsc",
                        "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                        "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)
        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        prefix = self.random_name()
        task = self.post_task(
            "/api/publish/" + prefix,
            json={
                "Architectures": ["i386", "source"],
                "SourceKind": "local",
                "Sources": [{"Name": repo_name}],
                "Signing": DefaultSigningOptions,
            }
        )
        self.check_task(task)

        self.check_not_exists(
            "public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_exists("public/" + prefix +
                          "/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc")

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)
        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        task = self.delete_task("/api/repos/" + repo_name + "/packages/",
                                json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89']})
        self.check_task(task)

        # Update and switch AcquireByHash on.
        task = self.put_task(
            "/api/publish/" + prefix + "/wheezy",
            json={
                "AcquireByHash": True,
                "Signing": DefaultSigningOptions,
            }
        )
        self.check_task(task)
        repo_expected = {
            'AcquireByHash': True,
            'Architectures': ['i386', 'source'],
            'Codename': '',
            'Distribution': 'wheezy',
            'Label': '',
            'Origin': '',
            'NotAutomatic': '',
            'ButAutomaticUpgrades': '',
            'Path': prefix + '/' + 'wheezy',
            'Prefix': prefix,
            'SignedBy': '',
            'SkipContents': False,
            'MultiDist': False,
            'SourceKind': 'local',
            'Sources': [{'Component': 'main', 'Name': repo_name}],
            'Storage': '',
            'Suite': ''}

        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())

        self.check_exists("public/" + prefix +
                          "/dists/wheezy/main/binary-i386/by-hash")

        self.check_exists(
            "public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_not_exists(
            "public/" + prefix + "/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc")

        task = self.delete_task("/api/publish/" + prefix + "/wheezy")
        self.check_task(task)
        self.check_not_exists("public/" + prefix + "dists/")


class PublishUpdateAPITestRepoSignedBy(APITest):
    """
    PUT /publish/:prefix/:distribution (local repos), DELETE /publish/:prefix/:distribution
    """
    fixtureGpg = True

    def check(self):
        repo_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(
            self.upload("/api/files/" + d,
                        "pyspi_0.6.1-1.3.dsc",
                        "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                        "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)
        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        prefix = self.random_name()
        task = self.post_task(
            "/api/publish/" + prefix,
            json={
                "Architectures": ["i386", "source"],
                "SourceKind": "local",
                "Sources": [{"Name": repo_name}],
                "Signing": DefaultSigningOptions,
            }
        )
        self.check_task(task)

        self.check_not_exists(
            "public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_exists("public/" + prefix +
                          "/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc")

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)
        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        task = self.delete_task("/api/repos/" + repo_name + "/packages/",
                                json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89']})
        self.check_task(task)

        # Update and specify SignedBy.
        task = self.put_task(
            "/api/publish/" + prefix + "/wheezy",
            json={
                "Signing": DefaultSigningOptions,
                "SignedBy": "just,a,string",
            }
        )
        self.check_task(task)
        repo_expected = {
            'AcquireByHash': False,
            'Architectures': ['i386', 'source'],
            'Codename': '',
            'Distribution': 'wheezy',
            'Label': '',
            'Origin': '',
            'NotAutomatic': '',
            'ButAutomaticUpgrades': '',
            'Path': prefix + '/' + 'wheezy',
            'Prefix': prefix,
            'SignedBy': 'just,a,string',
            'SkipContents': False,
            'MultiDist': False,
            'SourceKind': 'local',
            'Sources': [{'Component': 'main', 'Name': repo_name}],
            'Storage': '',
            'Suite': ''}

        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())


class PublishUpdateAPIMultiDist(APITest):
    """
    Test MultiDist publishing to subdirectory
    """
    fixtureGpg = True

    def check(self):
        repo_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo_name, "DefaultDistribution": "bookworm"}).status_code, 201)

        d = self.random_name()
        self.check_equal(
            self.upload("/api/files/" + d,
                        "pyspi_0.6.1-1.3.dsc",
                        "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                        "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)
        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        prefix = self.random_name()
        task = self.post_task(
            "/api/publish/" + prefix,
            json={
                "Architectures": ["i386", "source"],
                "SourceKind": "local",
                "Sources": [{"Name": repo_name}],
                "Signing": DefaultSigningOptions,
            }
        )
        self.check_task(task)

        self.check_not_exists(
            "public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_exists("public/" + prefix +
                          "/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc")

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)
        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        task = self.delete_task("/api/repos/" + repo_name + "/packages/",
                                json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89']})
        self.check_task(task)

        # Update and switch MultiDist on.
        task = self.put_task(
            "/api/publish/" + prefix + "/bookworm",
            json={
                "MultiDist": True,
                "Signing": DefaultSigningOptions,
            }
        )
        self.check_task(task)
        repo_expected = {
            'AcquireByHash': False,
            'Architectures': ['i386', 'source'],
            'Codename': '',
            'Distribution': 'bookworm',
            'Label': '',
            'Origin': '',
            'NotAutomatic': '',
            'ButAutomaticUpgrades': '',
            'Path': prefix + '/' + 'bookworm',
            'Prefix': prefix,
            'SignedBy': '',
            'SkipContents': False,
            'MultiDist': True,
            'SourceKind': 'local',
            'Sources': [{'Component': 'main', 'Name': repo_name}],
            'Storage': '',
            'Suite': ''}

        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())

        self.check_exists(
            "public/" + prefix + "/pool/bookworm/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_not_exists(
            "public/" + prefix + "/pool/bookworm/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc")

        task = self.delete_task("/api/publish/" + prefix + "/bookworm")
        self.check_task(task)
        self.check_not_exists("public/" + prefix + "dists/")


class PublishConcurrentUpdateAPITestRepo(APITest):
    """
    PUT /publish/:prefix/:distribution (local repos), DELETE /publish/:prefix/:distribution
    """
    fixtureGpg = True

    def check(self):
        repo_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(
            self.upload("/api/files/" + d,
                        "pyspi_0.6.1-1.3.dsc",
                        "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                        "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)
        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        prefix = self.random_name()
        task = self.post_task(
            "/api/publish/" + prefix,
            json={
                "Architectures": ["i386", "source"],
                "SourceKind": "local",
                "Sources": [{"Name": repo_name}],
                "Signing": DefaultSigningOptions,
            }
        )
        self.check_task(task)

        self.check_not_exists(
            "public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_exists("public/" + prefix +
                          "/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc")

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)
        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        task = self.delete_task("/api/repos/" + repo_name + "/packages/",
                                json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89']})
        self.check_task(task)

        def _do_update(result, index):
            resp = self.put_task(
                "/api/publish/" + prefix + "/wheezy",
                json={
                    "AcquireByHash": True,
                    "Signing": DefaultSigningOptions,
                }
            )
            try:
                self.check_equal(resp.json()['State'], TASK_SUCCEEDED)
            except BaseException as e:
                result[index] = e

        n_workers = 10
        worker_results = [None] * n_workers
        tasks = [threading.Thread(target=_do_update, args=(worker_results, i,)) for i in range(n_workers)]
        [task.start() for task in tasks]
        [task.join() for task in tasks]
        for result in worker_results:
            if isinstance(result, BaseException):
                raise result

        repo_expected = {
            'AcquireByHash': True,
            'Architectures': ['i386', 'source'],
            'Codename': '',
            'Distribution': 'wheezy',
            'Label': '',
            'Origin': '',
            'NotAutomatic': '',
            'ButAutomaticUpgrades': '',
            'Path': prefix + '/' + 'wheezy',
            'Prefix': prefix,
            'SignedBy': '',
            'SkipContents': False,
            'MultiDist': False,
            'SourceKind': 'local',
            'Sources': [{'Component': 'main', 'Name': repo_name}],
            'Storage': '',
            'Suite': ''}

        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())

        self.check_exists("public/" + prefix +
                          "/dists/wheezy/main/binary-i386/by-hash")

        self.check_exists(
            "public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_not_exists(
            "public/" + prefix + "/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc")

        task = self.delete_task("/api/publish/" + prefix + "/wheezy")
        self.check_task(task)
        self.check_not_exists("public/" + prefix + "dists/")


class PublishUpdateSkipCleanupAPITestRepo(APITest):
    """
    PUT /publish/:prefix/:distribution (local repos), DELETE /publish/:prefix/:distribution
    """
    fixtureGpg = True

    def check(self):
        repo_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(
            self.upload("/api/files/" + d,
                        "pyspi_0.6.1-1.3.dsc",
                        "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                        "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)
        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        prefix = self.random_name()
        task = self.post_task("/api/publish/" + prefix,
                              json={
                                  "Architectures": ["i386", "source"],
                                  "SourceKind": "local",
                                  "Sources": [{"Name": repo_name}],
                                  "Signing": DefaultSigningOptions,
                              })
        self.check_task(task)

        self.check_not_exists(
            "public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_exists("public/" + prefix +
                          "/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc")

        # Publish two repos, so that deleting one while skipping cleanup will
        # not delete the whole prefix.
        task = self.post_task("/api/publish/" + prefix,
                              json={
                                  "Architectures": ["i386", "source"],
                                  "Distribution": "otherdist",
                                  "SourceKind": "local",
                                  "Sources": [{"Name": repo_name}],
                                  "Signing": DefaultSigningOptions,
                              })
        self.check_task(task)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)
        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        task = self.delete_task("/api/repos/" + repo_name + "/packages/",
                                json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89']})
        self.check_task(task)

        task = self.put_task("/api/publish/" + prefix + "/wheezy",
                             json={
                                 "Signing": DefaultSigningOptions,
                                 "SkipCleanup": True,
                             })
        self.check_task(task)
        repo_expected = {
            'AcquireByHash': False,
            'Architectures': ['i386', 'source'],
            'Codename': '',
            'Distribution': 'wheezy',
            'Label': '',
            'Origin': '',
            'NotAutomatic': '',
            'ButAutomaticUpgrades': '',
            'Path': prefix + '/' + 'wheezy',
            'Prefix': prefix,
            'SignedBy': '',
            'SkipContents': False,
            'MultiDist': False,
            'SourceKind': 'local',
            'Sources': [{'Component': 'main', 'Name': repo_name}],
            'Storage': '',
            'Suite': ''}

        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())

        self.check_exists(
            "public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_exists("public/" + prefix +
                          "/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc")

        task = self.delete_task("/api/publish/" + prefix + "/wheezy", params={"SkipCleanup": "1"})
        self.check_task(task)
        self.check_exists("public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_exists("public/" + prefix + "/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc")


class PublishSwitchAPITestRepo(APITest):
    """
    PUT /publish/:prefix/:distribution (snapshots), DELETE /publish/:prefix/:distribution
    """
    fixtureGpg = True

    def check(self):
        repo_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(
            self.upload("/api/files/" + d,
                        "pyspi_0.6.1-1.3.dsc",
                        "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                        "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)
        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        snapshot1_name = self.random_name()
        task = self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot1_name})
        self.check_task(task)

        prefix = self.random_name()
        task = self.post_task(
            "/api/publish/" + prefix,
            json={
                "Architectures": ["i386", "source"],
                "SourceKind": "snapshot",
                "Sources": [{"Name": snapshot1_name}],
                "Signing": DefaultSigningOptions,
            })
        self.check_task(task)

        repo_expected = {
            'AcquireByHash': False,
            'Architectures': ['i386', 'source'],
            'Codename': '',
            'Distribution': 'wheezy',
            'Label': '',
            'NotAutomatic': '',
            'ButAutomaticUpgrades': '',
            'Origin': '',
            'Path': prefix + '/' + 'wheezy',
            'Prefix': prefix,
            'SignedBy': '',
            'SkipContents': False,
            'MultiDist': False,
            'SourceKind': 'snapshot',
            'Sources': [{'Component': 'main', 'Name': snapshot1_name}],
            'Storage': '',
            'Suite': ''}
        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())

        self.check_not_exists(
            "public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_exists("public/" + prefix +
                          "/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc")

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)
        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        task = self.delete_task("/api/repos/" + repo_name + "/packages/",
                                json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89']})
        self.check_task(task)

        snapshot2_name = self.random_name()
        task = self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot2_name})
        self.check_task(task)

        task = self.put_task(
            "/api/publish/" + prefix + "/wheezy",
            json={
                "Snapshots": [{"Component": "main", "Name": snapshot2_name}],
                "Signing": DefaultSigningOptions,
                "SkipContents": True,
            })
        self.check_task(task)
        repo_expected = {
            'AcquireByHash': False,
            'Architectures': ['i386', 'source'],
            'Codename': '',
            'Distribution': 'wheezy',
            'Label': '',
            'Origin': '',
            'NotAutomatic': '',
            'ButAutomaticUpgrades': '',
            'Path': prefix + '/' + 'wheezy',
            'Prefix': prefix,
            'SignedBy': '',
            'SkipContents': True,
            'MultiDist': False,
            'SourceKind': 'snapshot',
            'Sources': [{'Component': 'main', 'Name': snapshot2_name}],
            'Storage': '',
            'Suite': ''}

        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())

        self.check_exists(
            "public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_not_exists(
            "public/" + prefix + "/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc")

        task = self.delete_task("/api/publish/" + prefix + "/wheezy")
        self.check_task(task)
        self.check_not_exists("public/" + prefix + "dists/")


class PublishSwitchAPITestRepoSignedBy(APITest):
    """
    PUT /publish/:prefix/:distribution (snapshots), DELETE /publish/:prefix/:distribution
    """
    fixtureGpg = True

    def check(self):
        repo_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(
            self.upload("/api/files/" + d,
                        "pyspi_0.6.1-1.3.dsc",
                        "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                        "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)
        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        snapshot1_name = self.random_name()
        task = self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot1_name})
        self.check_task(task)

        prefix = self.random_name()
        task = self.post_task(
            "/api/publish/" + prefix,
            json={
                "Architectures": ["i386", "source"],
                "SourceKind": "snapshot",
                "Sources": [{"Name": snapshot1_name}],
                "Signing": DefaultSigningOptions,
            })
        self.check_task(task)

        repo_expected = {
            'AcquireByHash': False,
            'Architectures': ['i386', 'source'],
            'Codename': '',
            'Distribution': 'wheezy',
            'Label': '',
            'NotAutomatic': '',
            'ButAutomaticUpgrades': '',
            'Origin': '',
            'Path': prefix + '/' + 'wheezy',
            'Prefix': prefix,
            'SignedBy': '',
            'SkipContents': False,
            'MultiDist': False,
            'SourceKind': 'snapshot',
            'Sources': [{'Component': 'main', 'Name': snapshot1_name}],
            'Storage': '',
            'Suite': ''}
        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())

        snapshot2_name = self.random_name()
        task = self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot2_name})
        self.check_task(task)

        task = self.put_task(
            "/api/publish/" + prefix + "/wheezy",
            json={
                "Snapshots": [{"Component": "main", "Name": snapshot2_name}],
                "Signing": DefaultSigningOptions,
                "SkipContents": True,
                "SignedBy": "just,a,string",
            })
        self.check_task(task)
        repo_expected = {
            'AcquireByHash': False,
            'Architectures': ['i386', 'source'],
            'Codename': '',
            'Distribution': 'wheezy',
            'Label': '',
            'Origin': '',
            'NotAutomatic': '',
            'ButAutomaticUpgrades': '',
            'Path': prefix + '/' + 'wheezy',
            'Prefix': prefix,
            'SignedBy': 'just,a,string',
            'SkipContents': True,
            'MultiDist': False,
            'SourceKind': 'snapshot',
            'Sources': [{'Component': 'main', 'Name': snapshot2_name}],
            'Storage': '',
            'Suite': ''}

        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())


class PublishSwitchAPISkipCleanupTestRepo(APITest):
    """
    PUT /publish/:prefix/:distribution (snapshots), DELETE /publish/:prefix/:distribution
    """
    fixtureGpg = True

    def check(self):
        repo_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(
            self.upload("/api/files/" + d,
                        "pyspi_0.6.1-1.3.dsc",
                        "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                        "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)
        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        snapshot1_name = self.random_name()
        task = self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot1_name})
        self.check_task(task)

        prefix = self.random_name()
        task = self.post_task("/api/publish/" + prefix,
                              json={
                                  "Architectures": ["i386", "source"],
                                  "SourceKind": "snapshot",
                                  "Sources": [{"Name": snapshot1_name}],
                                  "Signing": DefaultSigningOptions,
                              })

        self.check_task(task)
        repo_expected = {
            'AcquireByHash': False,
            'Architectures': ['i386', 'source'],
            'Codename': '',
            'Distribution': 'wheezy',
            'Label': '',
            'NotAutomatic': '',
            'ButAutomaticUpgrades': '',
            'Origin': '',
            'Path': prefix + '/' + 'wheezy',
            'Prefix': prefix,
            'SignedBy': '',
            'SkipContents': False,
            'MultiDist': False,
            'SourceKind': 'snapshot',
            'Sources': [{'Component': 'main', 'Name': snapshot1_name}],
            'Storage': '',
            'Suite': ''}
        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())

        self.check_not_exists(
            "public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_exists("public/" + prefix +
                          "/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc")

        # Publish two snapshots, so that deleting one while skipping cleanup will
        # not delete the whole prefix.
        task = self.post_task("/api/publish/" + prefix,
                              json={
                                  "Architectures": ["i386", "source"],
                                  "Distribution": "otherdist",
                                  "SourceKind": "snapshot",
                                  "Sources": [{"Name": snapshot1_name}],
                                  "Signing": DefaultSigningOptions,
                              })

        self.check_task(task)
        repo_expected = {
            'AcquireByHash': False,
            'Architectures': ['i386', 'source'],
            'Codename': '',
            'Distribution': 'otherdist',
            'Label': '',
            'NotAutomatic': '',
            'ButAutomaticUpgrades': '',
            'Origin': '',
            'Path': prefix + '/' + 'otherdist',
            'Prefix': prefix,
            'SignedBy': '',
            'SkipContents': False,
            'MultiDist': False,
            'SourceKind': 'snapshot',
            'Sources': [{'Component': 'main', 'Name': snapshot1_name}],
            'Storage': '',
            'Suite': ''}
        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)
        task = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_task(task)

        task = self.delete_task("/api/repos/" + repo_name + "/packages/",
                                json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89']})
        self.check_task(task)

        snapshot2_name = self.random_name()
        task = self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot2_name})
        self.check_task(task)

        task = self.put_task("/api/publish/" + prefix + "/wheezy",
                             json={
                                 "Snapshots": [{"Component": "main", "Name": snapshot2_name}],
                                 "Signing": DefaultSigningOptions,
                                 "SkipCleanup": True,
                                 "SkipContents": True,
                             })
        self.check_task(task)
        repo_expected = {
            'AcquireByHash': False,
            'Architectures': ['i386', 'source'],
            'Codename': '',
            'Distribution': 'wheezy',
            'Label': '',
            'Origin': '',
            'NotAutomatic': '',
            'ButAutomaticUpgrades': '',
            'Path': prefix + '/' + 'wheezy',
            'Prefix': prefix,
            'SignedBy': '',
            'SkipContents': True,
            'MultiDist': False,
            'SourceKind': 'snapshot',
            'Sources': [{'Component': 'main', 'Name': snapshot2_name}],
            'Storage': '',
            'Suite': ''}

        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())

        self.check_exists("public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_exists("public/" + prefix + "/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc")

        task = self.delete_task("/api/publish/" + prefix + "/wheezy", params={"SkipCleanup": "1"})
        self.check_task(task)
        self.check_exists("public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_exists("public/" + prefix + "/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc")


class PublishShowAPITestRepo(APITest):
    """
    GET /publish/:prefix/:distribution
    """

    def check(self):
        repo1_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo1_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(
            self.upload("/api/files/" + d,
                        "pyspi_0.6.1-1.3.dsc",
                        "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                        "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)
        self.check_equal(self.post_task("/api/repos/" + repo1_name + "/file/" + d).status_code, 200)

        # publishing under prefix, default distribution
        prefix = self.random_name()
        self.check_equal(self.post(
            "/api/publish/" + prefix,
            json={
                 "Architectures": ["i386", "source"],
                 "SourceKind": "local",
                 "Sources": [{"Component": "main", "Name": repo1_name}],
                 "Signing": DefaultSigningOptions,
            }
        ).status_code, 201)

        repo_expected = {
            'AcquireByHash': False,
            'Architectures': ['i386', 'source'],
            'Codename': '',
            'Distribution': 'wheezy',
            'Label': '',
            'NotAutomatic': '',
            'ButAutomaticUpgrades': '',
            'Origin': '',
            'Path': prefix + '/' + 'wheezy',
            'Prefix': prefix,
            'SignedBy': '',
            'SkipContents': False,
            'MultiDist': False,
            'SourceKind': 'local',
            'Sources': [{'Component': 'main', 'Name': repo1_name}],
            'Storage': '',
            'Suite': ''}
        repo = self.get("/api/publish/" + prefix + "/wheezy")
        self.check_equal(repo.status_code, 200)
        self.check_equal(repo_expected, repo.json())


class ServePublishedListTestRepo(APITest):
    """
    GET /repos
    """

    def check(self):
        d = "libboost-program-options-dev_1.62.0.1"
        r = "bar"
        f = "libboost-program-options-dev_1.62.0.1_i386.deb"

        self.check_equal(self.upload("/api/files/" + d, f).status_code, 200)

        self.check_equal(self.post("/api/repos", json={
            "Name": r,
            "Comment": "test repo",
            "DefaultDistribution": r,
            "DefaultComponent": "main"
        }).status_code, 201)

        self.check_equal(self.post(f"/api/repos/{r}/file/{d}").status_code, 200)

        self.check_equal(self.post("/api/publish/filesystem:apiandserve:", json={
            "SourceKind": "local",
            "Sources": [
                {
                    "Component": "main",
                    "Name": r
                }
            ],
            "Distribution": r,
            "Signing":  {
                "Skip": True
            }
        }).status_code, 201)

        get = self.get("/repos")
        expected_content_type = "text/html; charset=utf-8"
        if get.headers['content-type'] != expected_content_type:
            raise Exception(f"Received content-type {get.headers['content-type']} was not: {expected_content_type}")

        excepted_content = b'<pre>\n<a href="apiandserve/">apiandserve</a>\n</pre>'
        if excepted_content != get.content:
            raise Exception(f"Expected content {excepted_content} was not: {get.content}")


class ServePublishedTestRepo(APITest):
    """
    GET /repos/:storage/*pkgPath
    """

    def check(self):
        d = self.random_name()
        r = self.random_name()
        f = "libboost-program-options-dev_1.62.0.1_i386.deb"

        self.check_equal(self.upload("/api/files/" + d, f).status_code, 200)

        self.check_equal(self.post("/api/repos", json={
            "Name": r,
            "Comment": "test repo",
            "DefaultDistribution": r,
            "DefaultComponent": "main"
        }).status_code, 201)

        self.check_equal(self.post(f"/api/repos/{r}/file/{d}").status_code, 200)

        self.check_equal(self.post("/api/publish/filesystem:apiandserve:", json={
            "SourceKind": "local",
            "Sources": [
                {
                    "Component": "main",
                    "Name": r
                }
            ],
            "Distribution": r,
            "Signing":  {
                "Skip": True
            }
        }).status_code, 201)

        get = self.get(f"/repos/apiandserve/pool/main/b/boost-defaults/{f}")
        deb_content_types = [
            "application/x-deb",
            "application/x-debian-package",
            "application/vnd.debian.binary-package"
        ]
        if get.headers['content-type'] not in deb_content_types:
            raise Exception(f"Received content-type {get.headers['content-type']} not one of expected: {deb_content_types}")

        if len(get.content) != 3428:
            raise Exception(f"Expected file size 3428 bytes != {len(get.content)} bytes")


class ServePublishedNotFoundTestRepo(APITest):
    """
    GET /repos/:storage/*pkgPath
    """

    def check(self):
        d = self.random_name()
        r = self.random_name()
        f = "libboost-program-options-dev_1.62.0.1_i386.deb"

        self.check_equal(self.upload("/api/files/" + d, f).status_code, 200)

        self.check_equal(self.post("/api/repos", json={
            "Name": r,
            "Comment": "test repo",
            "DefaultDistribution": r,
            "DefaultComponent": "main"
        }).status_code, 201)

        self.check_equal(self.post(f"/api/repos/{r}/file/{d}").status_code, 200)

        self.check_equal(self.post("/api/publish/filesystem:apiandserve:", json={
            "SourceKind": "local",
            "Sources": [
                {
                    "Component": "main",
                    "Name": r
                }
            ],
            "Distribution": r,
            "Signing":  {
                "Skip": True
            }
        }).status_code, 201)

        get = self.get("/repos/apiandserve/pool/main/b/boost-defaults/i-dont-exist")
        if get.status_code != 404:
            raise Exception(f"Expected status 404 != {get.status_code}")


class PublishSourcesAddAPITestRepo(APITest):
    """
    POST /publish/:prefix/:distribution/sources
    """
    fixtureGpg = True

    def check(self):
        repo1_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo1_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb", "pyspi_0.6.1-1.3.dsc",
                                     "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                                     "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo1_name + "/file/" + d).status_code, 200)

        # publishing under prefix, default distribution
        prefix = self.random_name()
        self.check_equal(self.post(
            "/api/publish/" + prefix,
            json={
                 "SourceKind": "local",
                 "Sources": [{"Component": "main", "Name": repo1_name}],
                 "Signing": DefaultSigningOptions,
            }
        ).status_code, 201)

        repo2_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo2_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo2_name + "/file/" + d).status_code, 200)

        # Actual test
        self.check_equal(self.post(
            "/api/publish/" + prefix + "/wheezy/sources",
            json={"Component": "test", "Name": repo2_name}
        ).status_code, 201)

        sources_expected = [{"Component": "main", "Name": repo1_name}, {"Component": "test", "Name": repo2_name}]
        sources = self.get("/api/publish/" + prefix + "/wheezy/sources")
        self.check_equal(sources.status_code, 200)
        self.check_equal(sources_expected, sources.json())


class PublishSourceUpdateAPITestRepo(APITest):
    """
    PUT /publish/:prefix/:distribution/sources/main
    """
    fixtureGpg = True

    def check(self):
        repo1_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo1_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb", "pyspi_0.6.1-1.3.dsc",
                                     "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                                     "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo1_name + "/file/" + d).status_code, 200)

        # publishing under prefix, default distribution
        prefix = self.random_name()
        self.check_equal(self.post(
            "/api/publish/" + prefix,
            json={
                 "SourceKind": "local",
                 "Sources": [{"Component": "main", "Name": repo1_name}],
                 "Signing": DefaultSigningOptions,
            }
        ).status_code, 201)

        repo2_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo2_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo2_name + "/file/" + d).status_code, 200)

        # Actual test
        self.check_equal(self.put(
            "/api/publish/" + prefix + "/wheezy/sources/main",
            json={"Component": "main", "Name": repo2_name}
        ).status_code, 200)

        sources_expected = [{"Component": "main", "Name": repo2_name}]
        sources = self.get("/api/publish/" + prefix + "/wheezy/sources")
        self.check_equal(sources.status_code, 200)
        self.check_equal(sources_expected, sources.json())


class PublishSourcesUpdateAPITestRepo(APITest):
    """
    PUT /publish/:prefix/:distribution/sources
    """
    fixtureGpg = True

    def check(self):
        repo1_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo1_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb", "pyspi_0.6.1-1.3.dsc",
                                     "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                                     "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo1_name + "/file/" + d).status_code, 200)

        repo2_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo2_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo2_name + "/file/" + d).status_code, 200)

        # publishing under prefix, default distribution
        prefix = self.random_name()
        self.check_equal(self.post(
            "/api/publish/" + prefix,
            json={
                 "SourceKind": "local",
                 "Sources": [{"Component": "main", "Name": repo1_name}],
                 "Signing": DefaultSigningOptions,
            }
        ).status_code, 201)

        # Actual test
        self.check_equal(self.put(
            "/api/publish/" + prefix + "/wheezy/sources",
            json=[{"Component": "test", "Name": repo1_name}, {"Component": "other-test", "Name": repo2_name}]
        ).status_code, 200)

        sources_expected = [{"Component": "other-test", "Name": repo2_name}, {"Component": "test", "Name": repo1_name}]
        sources = self.get("/api/publish/" + prefix + "/wheezy/sources")
        self.check_equal(sources.status_code, 200)
        self.check_equal(sources_expected, sources.json())


class PublishSourceRemoveAPITestRepo(APITest):
    """
    DELETE /publish/:prefix/:distribution/sources/test
    """
    fixtureGpg = True

    def check(self):
        repo1_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo1_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb", "pyspi_0.6.1-1.3.dsc",
                                     "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                                     "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo1_name + "/file/" + d).status_code, 200)

        repo2_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo2_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo2_name + "/file/" + d).status_code, 200)

        # publishing under prefix, default distribution
        prefix = self.random_name()
        self.check_equal(self.post(
            "/api/publish/" + prefix,
            json={
                 "SourceKind": "local",
                 "Sources": [{"Component": "main", "Name": repo1_name}, {"Component": "test", "Name": repo2_name}],
                 "Signing": DefaultSigningOptions,
            }
        ).status_code, 201)

        # Actual test
        self.check_equal(self.delete("/api/publish/" + prefix + "/wheezy/sources/test").status_code, 200)

        sources_expected = [{"Component": "main", "Name": repo1_name}]
        sources = self.get("/api/publish/" + prefix + "/wheezy/sources")
        self.check_equal(sources.status_code, 200)
        self.check_equal(sources_expected, sources.json())


class PublishSourcesDropAPITestRepo(APITest):
    """
    DELETE /publish/:prefix/:distribution/sources
    """
    fixtureGpg = True

    def check(self):
        repo1_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo1_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb", "pyspi_0.6.1-1.3.dsc",
                                     "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                                     "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo1_name + "/file/" + d).status_code, 200)

        repo2_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo2_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo2_name + "/file/" + d).status_code, 200)

        # publishing under prefix, default distribution
        prefix = self.random_name()
        self.check_equal(self.post(
            "/api/publish/" + prefix,
            json={
                 "SourceKind": "local",
                 "Sources": [{"Component": "main", "Name": repo1_name}, {"Component": "test", "Name": repo2_name}],
                 "Signing": DefaultSigningOptions,
            }
        ).status_code, 201)

        self.check_equal(self.delete("/api/publish/" + prefix + "/wheezy/sources/test").status_code, 200)

        # Actual test
        self.check_equal(self.delete("/api/publish/" + prefix + "/wheezy/sources").status_code, 200)

        self.check_equal(self.get("/api/publish/" + prefix + "/wheezy/sources").status_code, 404)


class PublishSourcesListAPITestRepo(APITest):
    """
    GET /publish/:prefix/:distribution/sources
    """
    fixtureGpg = True

    def check(self):
        repo1_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo1_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb", "pyspi_0.6.1-1.3.dsc",
                                     "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                                     "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo1_name + "/file/" + d).status_code, 200)

        repo2_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo2_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo2_name + "/file/" + d).status_code, 200)

        # publishing under prefix, default distribution
        prefix = self.random_name()
        self.check_equal(self.post(
            "/api/publish/" + prefix,
            json={
                 "SourceKind": "local",
                 "Sources": [{"Component": "main", "Name": repo1_name}],
                 "Signing": DefaultSigningOptions,
            }
        ).status_code, 201)

        # Actual test
        self.check_equal(self.post(
            "/api/publish/" + prefix + "/wheezy/sources",
            json={"Component": "test", "Name": repo1_name}
        ).status_code, 201)

        self.check_equal(self.put(
            "/api/publish/" + prefix + "/wheezy/sources/main",
            json={"Component": "main", "Name": repo2_name}
        ).status_code, 200)

        self.check_equal(self.delete("/api/publish/" + prefix + "/wheezy/sources/main").status_code, 200)

        sources_expected = [{"Component": "test", "Name": repo1_name}]
        sources = self.get("/api/publish/" + prefix + "/wheezy/sources")
        self.check_equal(sources.status_code, 200)
        self.check_equal(sources_expected, sources.json())


class PublishSourceReplaceAPITestRepo(APITest):
    """
    PUT /publish/:prefix/:distribution/sources/main
    """
    fixtureGpg = True

    def check(self):
        repo1_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo1_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb", "pyspi_0.6.1-1.3.dsc",
                                     "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                                     "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo1_name + "/file/" + d).status_code, 200)

        repo2_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo2_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo2_name + "/file/" + d).status_code, 200)

        # publishing under prefix, default distribution
        prefix = self.random_name()
        self.check_equal(self.post(
            "/api/publish/" + prefix,
            json={
                 "SourceKind": "local",
                 "Sources": [{"Component": "main", "Name": repo1_name}],
                 "Signing": DefaultSigningOptions,
            }
        ).status_code, 201)

        # Actual test
        self.check_equal(self.put(
            "/api/publish/" + prefix + "/wheezy/sources/main",
            json={"Component": "test", "Name": repo2_name}
        ).status_code, 200)

        sources_expected = [{"Component": "test", "Name": repo2_name}]
        sources = self.get("/api/publish/" + prefix + "/wheezy/sources")
        self.check_equal(sources.status_code, 200)
        self.check_equal(sources_expected, sources.json())


class PublishUpdateSourcesAPITestRepo(APITest):
    """
    POST /publish/:prefix/:distribution/update
    """
    fixtureGpg = True

    def check(self):
        repo1_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo1_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb", "pyspi_0.6.1-1.3.dsc",
                                     "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz",
                                     "pyspi-0.6.1-1.3.stripped.dsc").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo1_name + "/file/" + d).status_code, 200)

        repo2_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo2_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo2_name + "/file/" + d).status_code, 200)

        repo3_name = self.random_name()
        self.check_equal(self.post(
            "/api/repos", json={"Name": repo3_name, "DefaultDistribution": "wheezy"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                                     "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)

        self.check_equal(self.post("/api/repos/" + repo3_name + "/file/" + d).status_code, 200)

        # publishing under prefix, default distribution
        prefix = self.random_name()
        self.check_equal(self.post(
            "/api/publish/" + prefix,
            json={
                 "Signing": DefaultSigningOptions,
                 "SourceKind": "local",
                 "Sources": [{"Component": "main", "Name": repo1_name}, {"Component": "test", "Name": repo2_name}],
            }
        ).status_code, 201)

        # remove 'main' component
        self.check_equal(self.delete("/api/publish/" + prefix + "/wheezy/sources/main").status_code, 200)

        # update 'test' component
        self.check_equal(self.put(
            "/api/publish/" + prefix + "/wheezy/sources/test",
            json={"Component": "test", "Name": repo1_name}
        ).status_code, 200)

        # add 'other-test' component
        self.check_equal(self.post(
            "/api/publish/" + prefix + "/wheezy/sources",
            json={"Component": "other-test", "Name": repo3_name}
        ).status_code, 201)

        sources_expected = [{"Component": "other-test", "Name": repo3_name}, {"Component": "test", "Name": repo1_name}]
        sources = self.get("/api/publish/" + prefix + "/wheezy/sources")
        self.check_equal(sources.status_code, 200)
        self.check_equal(sources_expected, sources.json())

        # update published repository and publish new content
        self.check_equal(self.post(
            "/api/publish/" + prefix + "/wheezy/update",
            json={
                "AcquireByHash": True,
                "MultiDist": False,
                "Signing": DefaultSigningOptions,
                "SkipBz2": True,
                "SkipContents": True,
            }
        ).status_code, 200)

        repo_expected = {
            'AcquireByHash': True,
            'Architectures': ['i386', 'source'],
            'Codename': '',
            'Distribution': 'wheezy',
            'Label': '',
            'Origin': '',
            'NotAutomatic': '',
            'ButAutomaticUpgrades': '',
            'Path': prefix + '/' + 'wheezy',
            'Prefix': prefix,
            'SignedBy': '',
            'SkipContents': True,
            'MultiDist': False,
            'SourceKind': 'local',
            'Sources': [{"Component": "other-test", "Name": repo3_name}, {"Component": "test", "Name": repo1_name}],
            'Storage': '',
            'Suite': ''}

        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())
