import urllib
from api_lib import APITest


class PackagesAPITestShow(APITest):
    """
    GET /api/packages/:key
    """
    def check(self):
        # upload package to repo to create it
        repo_name = self.random_name()
        self.check_equal(self.post("/api/repos", json={"Name": repo_name, "Comment": "fun repo"}).status_code, 201)

        d = self.random_name()
        self.check_equal(self.upload("/api/files/" + d,
                         "pyspi_0.6.1-1.3.dsc", "pyspi_0.6.1-1.3.diff.gz", "pyspi_0.6.1.orig.tar.gz").status_code, 200)

        resp = self.post_task("/api/repos/" + repo_name + "/file/" + d)
        self.check_equal(resp.json()['State'], 2)

        # get information about package
        resp = self.get("/api/packages/" + urllib.quote('Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e'))
        self.check_equal(resp.status_code, 200)
        self.check_equal(resp.json(), {
            'Architecture': 'any',
            'Binary': 'python-at-spi',
            'Build-Depends': 'debhelper (>= 5), cdbs, libatspi-dev, python-pyrex, python-support (>= 0.4), python-all-dev, libx11-dev',  # noqa
            'Checksums-Sha1': ' 95a2468e4bbce730ba286f2211fa41861b9f1d90 3456 pyspi_0.6.1-1.3.diff.gz\n 56c8a9b1f4ab636052be8966690998cbe865cd6c 1782 pyspi_0.6.1-1.3.dsc\n 9694b80acc171c0a5bc99f707933864edfce555e 29063 pyspi_0.6.1.orig.tar.gz\n',  # noqa
            'Checksums-Sha256': ' 2e770b28df948f3197ed0b679bdea99f3f2bf745e9ddb440c677df9c3aeaee3c 3456 pyspi_0.6.1-1.3.diff.gz\n d494aaf526f1ec6b02f14c2f81e060a5722d6532ddc760ec16972e45c2625989 1782 pyspi_0.6.1-1.3.dsc\n 64069ee828c50b1c597d10a3fefbba279f093a4723965388cdd0ac02f029bfb9 29063 pyspi_0.6.1.orig.tar.gz\n',  # noqa
            'Checksums-Sha512': ' 384b5e94b4113262e41bda1a2563f4f439cb8c97f43e2caefe16d7626718c21b36d3145b915eed24053eaa7fe3b6186494a87a3fcf9627f6e653b54bb3caa897 3456 pyspi_0.6.1-1.3.diff.gz\n fde06b7dc5762a04986d0669420822f6a1e82b195322ae9cbd2dae40bda557c57ad77fe3546007ea645f801c4cd30ef4eb0e96efb2dee6b71c4c9a187d643683 1782 pyspi_0.6.1-1.3.dsc\n c278f52953203292bcc828bcf05aee456b160f91716f51ec1a1dbbcdb8b08fc29183d0a1135629fc0ebe86a3e84cedc685c3aa1714b70cc5db8877d40e754d7f 29063 pyspi_0.6.1.orig.tar.gz\n',  # noqa
            'Files': ' 22ff26db69b73d3438fdde21ab5ba2f1 3456 pyspi_0.6.1-1.3.diff.gz\n b72cb94699298a117b7c82641c68b6fd 1782 pyspi_0.6.1-1.3.dsc\n def336bd566ea688a06ec03db7ccf1f4 29063 pyspi_0.6.1.orig.tar.gz\n',  # noqa
            'FilesHash': '3a8b37cbd9a3559e',
            'Format': '1.0',
            'Homepage': 'http://people.redhat.com/zcerza/dogtail',
            'Key': 'Psource pyspi 0.6.1-1.3 3a8b37cbd9a3559e',
            'Maintainer': 'Jose Carlos Garcia Sogo <jsogo@debian.org>',
            'Package': 'pyspi',
            'ShortKey': 'Psource pyspi 0.6.1-1.3',
            'Standards-Version': '3.7.3',
            'Vcs-Svn': 'svn://svn.tribulaciones.org/srv/svn/pyspi/trunk',
            'Version': '0.6.1-1.3'})

        resp = self.get("/api/packages/" + urllib.quote('Pamd64 no-such-package 1.0 3a8b37cbd9a3559e'))
        self.check_equal(resp.status_code, 404)
