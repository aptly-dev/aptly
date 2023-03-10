import inspect
import os

from api_lib import TASK_SUCCEEDED, APITest

DefaultSigningOptions = {
    "Keyring": os.path.join(os.path.dirname(inspect.getsourcefile(APITest)), "files") + "/aptly.pub",
    "SecretKeyring": os.path.join(os.path.dirname(inspect.getsourcefile(APITest)), "files") + "/aptly.sec",
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

        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], TASK_SUCCEEDED)

        # publishing under prefix, default distribution
        prefix = self.random_name()
        resp = self.post_task(
            "/api/publish/" + prefix,
            json={
                 "SourceKind": "local",
                 "Sources": [{"Name": repo_name}],
                 "Signing": DefaultSigningOptions,
            }
        )
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
            'SkipContents': False,
            'SourceKind': 'local',
            'Sources': [{'Component': 'main', 'Name': repo_name}],
            'Storage': '',
            'Suite': ''}

        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)

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
        resp = self.post_task(
            "/api/publish/:.",
            json={
                 "SourceKind": "local",
                 "Sources": [{"Name": repo_name}],
                 "Signing": DefaultSigningOptions,
                 "Distribution": distribution,
                 "Architectures": ["i386", "amd64"],
            }
        )
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)
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
            'SkipContents': False,
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

        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], TASK_SUCCEEDED)

        self.check_equal(self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot_name}).json()['State'], TASK_SUCCEEDED)

        prefix = self.random_name()
        resp = self.post_task(
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
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)

        _id = resp.json()['ID']
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
            'NotAutomatic': 'yes',
            'ButAutomaticUpgrades': 'yes',
            'Path': prefix + '/' + 'squeeze',
            'Prefix': prefix,
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
        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], TASK_SUCCEEDED)

        prefix = self.random_name()
        resp = self.post_task(
            "/api/publish/" + prefix,
            json={
                "Architectures": ["i386", "source"],
                "SourceKind": "local",
                "Sources": [{"Name": repo_name}],
                "Signing": DefaultSigningOptions,
            }
        )

        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)

        self.check_not_exists(
            "public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_exists("public/" + prefix +
                          "/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc")

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)
        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], TASK_SUCCEEDED)

        self.check_equal(self.delete_task("/api/repos/" + repo_name + "/packages/",
                         json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89']}).json()['State'], TASK_SUCCEEDED)

        # Update and switch AcquireByHash on.
        resp = self.put_task(
            "/api/publish/" + prefix + "/wheezy",
            json={
                "AcquireByHash": True,
                "Signing": DefaultSigningOptions,
            }
        )
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)
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
            'SkipContents': False,
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

        self.check_equal(self.delete_task("/api/publish/" + prefix + "/wheezy").json()['State'], TASK_SUCCEEDED)
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
        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], TASK_SUCCEEDED)

        prefix = self.random_name()
        resp = self.post_task("/api/publish/" + prefix,
                              json={
                                  "Architectures": ["i386", "source"],
                                  "SourceKind": "local",
                                  "Sources": [{"Name": repo_name}],
                                  "Signing": DefaultSigningOptions,
                              })

        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)

        self.check_not_exists(
            "public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_exists("public/" + prefix +
                          "/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc")

        # Publish two repos, so that deleting one while skipping cleanup will
        # not delete the whole prefix.
        resp = self.post_task("/api/publish/" + prefix,
                              json={
                                  "Architectures": ["i386", "source"],
                                  "Distribution": "otherdist",
                                  "SourceKind": "local",
                                  "Sources": [{"Name": repo_name}],
                                  "Signing": DefaultSigningOptions,
                              })

        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "libboost-program-options-dev_1.49.0.1_i386.deb").status_code, 200)
        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], TASK_SUCCEEDED)

        self.check_equal(self.delete_task("/api/repos/" + repo_name + "/packages/",
                         json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89']}).json()['State'], TASK_SUCCEEDED)

        resp = self.put_task("/api/publish/" + prefix + "/wheezy",
                             json={
                                 "Signing": DefaultSigningOptions,
                                 "SkipCleanup": True,
                             })
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)
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
            'SkipContents': False,
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

        self.check_equal(self.delete_task("/api/publish/" + prefix + "/wheezy", params={"SkipCleanup": "1"}).json()['State'], TASK_SUCCEEDED)
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
        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], TASK_SUCCEEDED)

        snapshot1_name = self.random_name()
        self.check_equal(self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot1_name}).json()['State'], TASK_SUCCEEDED)

        prefix = self.random_name()
        resp = self.post_task(
            "/api/publish/" + prefix,
            json={
                "Architectures": ["i386", "source"],
                "SourceKind": "snapshot",
                "Sources": [{"Name": snapshot1_name}],
                "Signing": DefaultSigningOptions,
            })

        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)
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
            'SkipContents': False,
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
        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], TASK_SUCCEEDED)

        self.check_equal(self.delete_task("/api/repos/" + repo_name + "/packages/",
                         json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89']}).json()['State'], TASK_SUCCEEDED)

        snapshot2_name = self.random_name()
        self.check_equal(self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot2_name}).json()['State'], TASK_SUCCEEDED)

        resp = self.put_task(
            "/api/publish/" + prefix + "/wheezy",
            json={
                "Snapshots": [{"Component": "main", "Name": snapshot2_name}],
                "Signing": DefaultSigningOptions,
                "SkipContents": True,
            })
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)
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
            'SkipContents': True,
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

        self.check_equal(self.delete_task("/api/publish/" + prefix + "/wheezy").json()['State'], TASK_SUCCEEDED)
        self.check_not_exists("public/" + prefix + "dists/")


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
        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], TASK_SUCCEEDED)

        snapshot1_name = self.random_name()
        self.check_equal(self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot1_name}).json()['State'], TASK_SUCCEEDED)

        prefix = self.random_name()
        resp = self.post_task("/api/publish/" + prefix,
                              json={
                                  "Architectures": ["i386", "source"],
                                  "SourceKind": "snapshot",
                                  "Sources": [{"Name": snapshot1_name}],
                                  "Signing": DefaultSigningOptions,
                              })

        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)
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
            'SkipContents': False,
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
        resp = self.post_task("/api/publish/" + prefix,
                              json={
                                  "Architectures": ["i386", "source"],
                                  "Distribution": "otherdist",
                                  "SourceKind": "snapshot",
                                  "Sources": [{"Name": snapshot1_name}],
                                  "Signing": DefaultSigningOptions,
                              })

        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)
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
            'SkipContents': False,
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
        self.check_equal(self.post_task("/api/repos/" + repo_name + "/file/" + d).json()['State'], TASK_SUCCEEDED)

        self.check_equal(self.delete_task("/api/repos/" + repo_name + "/packages/",
                         json={"PackageRefs": ['Psource pyspi 0.6.1-1.4 f8f1daa806004e89']}).json()['State'], TASK_SUCCEEDED)

        snapshot2_name = self.random_name()
        self.check_equal(self.post_task("/api/repos/" + repo_name + '/snapshots', json={'Name': snapshot2_name}).json()['State'], TASK_SUCCEEDED)

        resp = self.put_task("/api/publish/" + prefix + "/wheezy",
                             json={
                                 "Snapshots": [{"Component": "main", "Name": snapshot2_name}],
                                 "Signing": DefaultSigningOptions,
                                 "SkipCleanup": True,
                                 "SkipContents": True,
                             })
        self.check_equal(resp.json()['State'], TASK_SUCCEEDED)
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
            'SkipContents': True,
            'SourceKind': 'snapshot',
            'Sources': [{'Component': 'main', 'Name': snapshot2_name}],
            'Storage': '',
            'Suite': ''}

        all_repos = self.get("/api/publish")
        self.check_equal(all_repos.status_code, 200)
        self.check_in(repo_expected, all_repos.json())

        self.check_exists("public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_exists("public/" + prefix + "/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc")

        self.check_equal(self.delete_task("/api/publish/" + prefix + "/wheezy", params={"SkipCleanup": "1"}).json()['State'], TASK_SUCCEEDED)
        self.check_exists("public/" + prefix + "/pool/main/b/boost-defaults/libboost-program-options-dev_1.49.0.1_i386.deb")
        self.check_exists("public/" + prefix + "/pool/main/p/pyspi/pyspi-0.6.1-1.3.stripped.dsc")
