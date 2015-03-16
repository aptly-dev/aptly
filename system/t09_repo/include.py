import tempfile
import shutil
import os
import inspect
import re
from lib import BaseTest

gpgRemove = lambda _, s: re.sub(r'Signature made .* using|gpgv: keyblock resource .*$|gpgv: Can\'t check signature: .*$', '', s, flags=re.MULTILINE)
changesRemove = lambda _, s: s.replace(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "changes"), "")


class IncludeRepo1Test(BaseTest):
    """
    incldue packages to local repo: .changes file from directory
    """
    fixtureCmds = [
        "aptly repo create unstable",
    ]
    runCmd = "aptly repo include -no-remove-files -keyring=${files}/aptly.pub ${changes}"
    outputMatchPrepare = gpgRemove

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages unstable", "repo_show")

        # check pool
        self.check_exists('pool//20/81/hardlink_0.2.1_amd64.deb')
        self.check_exists('pool/4e/fc/hardlink_0.2.1.dsc')
        self.check_exists('pool/8e/2c/hardlink_0.2.1.tar.gz')


class IncludeRepo2Test(BaseTest):
    """
    incldue packages to local repo: .changes file from file + custom repo
    """
    fixtureCmds = [
        "aptly repo create my-unstable",
        "aptly repo add my-unstable ${files}",
    ]
    runCmd = "aptly repo include -no-remove-files -keyring=${files}/aptly.pub -repo=my-{{.Distribution}} ${changes}/hardlink_0.2.1_amd64.changes"
    outputMatchPrepare = gpgRemove

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages my-unstable", "repo_show")

        # check pool
        self.check_exists('pool//20/81/hardlink_0.2.1_amd64.deb')
        self.check_exists('pool/4e/fc/hardlink_0.2.1.dsc')
        self.check_exists('pool/8e/2c/hardlink_0.2.1.tar.gz')


class IncludeRepo3Test(BaseTest):
    """
    incldue packages to local repo: broken repo flag
    """
    fixtureCmds = [
    ]
    runCmd = "aptly repo include -no-remove-files -keyring=${files}/aptly.pub -repo=my-{{.Distribution} ${changes}"
    expectedCode = 1


class IncludeRepo4Test(BaseTest):
    """
    incldue packages to local repo: missing repo
    """
    fixtureCmds = [
    ]
    runCmd = "aptly repo include -no-remove-files -ignore-signatures -keyring=${files}/aptly.pub ${changes}"
    outputMatchPrepare = changesRemove
    expectedCode = 1
