from azure_lib import AzureTest


class AzureRepoTest(AzureTest):
    """
    Azure: add directory to repo
    """

    fixtureCmds = [
        'aptly repo create -comment=Repo -distribution=squeeze repo',
    ]
    runCmd = 'aptly repo add repo ${files}'

    use_azure_pool = True

    def prepare(self):
        super(AzureRepoTest, self).prepare()

        self.configOverride['packagePoolStorage'] = {
            'azure': self.azure_endpoint,
        }

    def check(self):
        self.check_output()
        self.check_cmd_output('aptly repo show -with-packages repo', 'repo_show')

        # check pool
        self.check_exists_azure_only(
            'c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb'
        )
        self.check_exists_azure_only(
            '2e/77/0b28df948f3197ed0b679bdea99f_pyspi_0.6.1-1.3.diff.gz'
        )
        self.check_exists_azure_only(
            'd4/94/aaf526f1ec6b02f14c2f81e060a5_pyspi_0.6.1-1.3.dsc'
        )
        self.check_exists_azure_only(
            '64/06/9ee828c50b1c597d10a3fefbba27_pyspi_0.6.1.orig.tar.gz'
        )
        self.check_exists_azure_only(
            '28/9d/3aefa970876e9c43686ce2b02f47_pyspi-0.6.1-1.3.stripped.dsc'
        )
