import tempfile
import shutil
import os
import inspect
import re
from lib import BaseTest

gpgRemove = lambda _, s: re.sub(r'Signature made .* using|gpgv: keyblock resource .*$|gpgv: Can\'t check signature: .*$', '', s, flags=re.MULTILINE)
changesRemove = lambda _, s: s.replace(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "changes"), "")
tempDirRemove = lambda self, s: s.replace(self.tempSrcDir, "")


class IncludeRepo1Test(BaseTest):
    """
    include packages to local repo: .changes file from directory
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
    include packages to local repo: .changes file from file + custom repo
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
    include packages to local repo: broken repo flag
    """
    fixtureCmds = [
    ]
    runCmd = "aptly repo include -no-remove-files -keyring=${files}/aptly.pub -repo=my-{{.Distribution} ${changes}"
    expectedCode = 1
    outputMatchPrepare = lambda _, s: s.replace('; missing space?', '')


class IncludeRepo4Test(BaseTest):
    """
    include packages to local repo: missing repo
    """
    fixtureCmds = [
    ]
    runCmd = "aptly repo include -no-remove-files -ignore-signatures -keyring=${files}/aptly.pub ${changes}"
    outputMatchPrepare = changesRemove
    expectedCode = 1


class IncludeRepo5Test(BaseTest):
    """
    include packages to local repo: remove files being added
    """
    fixtureCmds = [
        "aptly repo create unstable",
    ]
    runCmd = "aptly repo include -keyring=${files}/aptly.pub "
    outputMatchPrepare = gpgRemove

    def prepare(self):
        super(IncludeRepo5Test, self).prepare()

        self.tempSrcDir = tempfile.mkdtemp()

        shutil.copytree(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "changes"), os.path.join(self.tempSrcDir, "01"))

        shutil.copy(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "pyspi_0.6.1-1.3.diff.gz"),
                    os.path.join(self.tempSrcDir, "01", "pyspi_0.6.1-1.3.diff.gz"))

        self.runCmd += self.tempSrcDir

    def check(self):
        try:
            self.check_output()
            self.check_cmd_output("aptly repo show -with-packages unstable", "repo_show")

            # check pool
            self.check_exists('pool//20/81/hardlink_0.2.1_amd64.deb')
            self.check_exists('pool/4e/fc/hardlink_0.2.1.dsc')
            self.check_exists('pool/8e/2c/hardlink_0.2.1.tar.gz')

            for path in ["hardlink_0.2.1.dsc", "hardlink_0.2.1.tar.gz", "hardlink_0.2.1_amd64.changes", "hardlink_0.2.1_amd64.deb"]:
                path = os.path.join(self.tempSrcDir, "01", path)
                if os.path.exists(path):
                    raise Exception("path %s shouldn't exist" % (path, ))

            path = os.path.join(self.tempSrcDir, "01", "pyspi_0.6.1-1.3.diff.gz")
            if not os.path.exists(path):
                raise Exception("path %s doesn't exist" % (path, ))

        finally:
            shutil.rmtree(self.tempSrcDir)


class IncludeRepo6Test(BaseTest):
    """
    include packages to local repo: missing files
    """
    fixtureCmds = [
        "aptly repo create unstable",
    ]
    runCmd = "aptly repo include -keyring=${files}/aptly.pub "
    outputMatchPrepare = lambda self, s: gpgRemove(self, tempDirRemove(self, s))
    expectedCode = 1

    def prepare(self):
        super(IncludeRepo6Test, self).prepare()

        self.tempSrcDir = tempfile.mkdtemp()
        os.makedirs(os.path.join(self.tempSrcDir, "01"), 0755)

        for path in ["hardlink_0.2.1.dsc", "hardlink_0.2.1_amd64.changes", "hardlink_0.2.1_amd64.deb"]:
            shutil.copy(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "changes", path),
                        os.path.join(self.tempSrcDir, "01", path))

        self.runCmd += self.tempSrcDir

    def check(self):
        try:
            super(IncludeRepo6Test, self).check()

            for path in ["hardlink_0.2.1.dsc", "hardlink_0.2.1_amd64.changes", "hardlink_0.2.1_amd64.deb"]:
                path = os.path.join(self.tempSrcDir, "01", path)
                if not os.path.exists(path):
                    raise Exception("path %s doesn't exist" % (path, ))
        finally:
            shutil.rmtree(self.tempSrcDir)


class IncludeRepo7Test(BaseTest):
    """
    include packages to local repo: wrong checksum
    """
    fixtureCmds = [
        "aptly repo create unstable",
    ]
    runCmd = "aptly repo include -keyring=${files}/aptly.pub "
    outputMatchPrepare = lambda self, s: gpgRemove(self, tempDirRemove(self, s))
    expectedCode = 1

    def prepare(self):
        super(IncludeRepo7Test, self).prepare()

        self.tempSrcDir = tempfile.mkdtemp()

        shutil.copytree(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "changes"), os.path.join(self.tempSrcDir, "01"))

        with open(os.path.join(self.tempSrcDir, "01", "hardlink_0.2.1.dsc"), "w") as f:
            f.write("A" * 949)  # file size

        self.runCmd += self.tempSrcDir

    def check(self):
        try:
            super(IncludeRepo7Test, self).check()
        finally:
            shutil.rmtree(self.tempSrcDir)


class IncludeRepo8Test(BaseTest):
    """
    include packages to local repo: wrong signature
    """
    fixtureCmds = [
        "aptly repo create unstable",
    ]
    runCmd = "aptly repo include -keyring=${files}/aptly.pub "
    outputMatchPrepare = lambda self, s: gpgRemove(self, tempDirRemove(self, s))
    expectedCode = 1

    def prepare(self):
        super(IncludeRepo8Test, self).prepare()

        self.tempSrcDir = tempfile.mkdtemp()

        shutil.copytree(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "changes"), os.path.join(self.tempSrcDir, "01"))

        with open(os.path.join(self.tempSrcDir, "01", "hardlink_0.2.1_amd64.changes"), "r+") as f:
            contents = f.read()
            f.seek(0, 0)
            f.write(contents.replace('Julian', 'Andrey'))
            f.truncate()

        self.runCmd += self.tempSrcDir

    def check(self):
        try:
            super(IncludeRepo8Test, self).check()
        finally:
            shutil.rmtree(self.tempSrcDir)


class IncludeRepo9Test(BaseTest):
    """
    include packages to local repo: unsigned
    """
    fixtureCmds = [
        "aptly repo create unstable",
    ]
    runCmd = "aptly repo include -keyring=${files}/aptly.pub "
    outputMatchPrepare = lambda self, s: gpgRemove(self, tempDirRemove(self, s))
    expectedCode = 1

    def prepare(self):
        super(IncludeRepo9Test, self).prepare()

        self.tempSrcDir = tempfile.mkdtemp()

        shutil.copytree(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "changes"), os.path.join(self.tempSrcDir, "01"))

        with open(os.path.join(self.tempSrcDir, "01", "hardlink_0.2.1_amd64.changes"), "r+") as f:
            contents = f.readlines()
            contents = contents[3:31]
            f.seek(0, 0)
            f.write("".join(contents))
            f.truncate()

        self.runCmd += self.tempSrcDir

    def check(self):
        try:
            super(IncludeRepo9Test, self).check()
        finally:
            shutil.rmtree(self.tempSrcDir)


class IncludeRepo10Test(BaseTest):
    """
    include packages to local repo: wrong signature + -ignore-signatures
    """
    fixtureCmds = [
        "aptly repo create unstable",
    ]
    runCmd = "aptly repo include -ignore-signatures "
    outputMatchPrepare = lambda self, s: gpgRemove(self, tempDirRemove(self, s))

    def prepare(self):
        super(IncludeRepo10Test, self).prepare()

        self.tempSrcDir = tempfile.mkdtemp()

        shutil.copytree(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "changes"), os.path.join(self.tempSrcDir, "01"))

        with open(os.path.join(self.tempSrcDir, "01", "hardlink_0.2.1_amd64.changes"), "r+") as f:
            contents = f.read()
            f.seek(0, 0)
            f.write(contents.replace('Julian', 'Andrey'))
            f.truncate()

        self.runCmd += self.tempSrcDir

    def check(self):
        try:
            super(IncludeRepo10Test, self).check()
        finally:
            shutil.rmtree(self.tempSrcDir)


class IncludeRepo11Test(BaseTest):
    """
    include packages to local repo: unsigned + -accept-unsigned
    """
    fixtureCmds = [
        "aptly repo create unstable",
    ]
    runCmd = "aptly repo include -accept-unsigned -keyring=${files}/aptly.pub "
    outputMatchPrepare = lambda self, s: gpgRemove(self, tempDirRemove(self, s))

    def prepare(self):
        super(IncludeRepo11Test, self).prepare()

        self.tempSrcDir = tempfile.mkdtemp()

        shutil.copytree(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "changes"), os.path.join(self.tempSrcDir, "01"))

        with open(os.path.join(self.tempSrcDir, "01", "hardlink_0.2.1_amd64.changes"), "r+") as f:
            contents = f.readlines()
            contents = contents[3:31]
            f.seek(0, 0)
            f.write("".join(contents))
            f.truncate()

        self.runCmd += self.tempSrcDir

    def check(self):
        try:
            super(IncludeRepo11Test, self).check()
        finally:
            shutil.rmtree(self.tempSrcDir)


class IncludeRepo12Test(BaseTest):
    """
    include packages to local repo: unsigned + -accept-unsigned + restriction breakage
    """
    fixtureCmds = [
        "aptly repo create unstable",
    ]
    runCmd = "aptly repo include -accept-unsigned -keyring=${files}/aptly.pub "
    outputMatchPrepare = lambda self, s: gpgRemove(self, tempDirRemove(self, s))
    expectedCode = 1

    def prepare(self):
        super(IncludeRepo12Test, self).prepare()

        self.tempSrcDir = tempfile.mkdtemp()

        shutil.copytree(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "changes"), os.path.join(self.tempSrcDir, "01"))

        with open(os.path.join(self.tempSrcDir, "01", "hardlink_0.2.1_amd64.changes"), "r+") as f:
            contents = f.readlines()
            contents = contents[3:31]
            contents[3] = "Binary: hardlink-dbg\n"
            f.seek(0, 0)
            f.write("".join(contents))
            f.truncate()

        self.runCmd += self.tempSrcDir

    def check(self):
        try:
            super(IncludeRepo12Test, self).check()
        finally:
            shutil.rmtree(self.tempSrcDir)


class IncludeRepo13Test(BaseTest):
    """
    include packages to local repo: with denying uploaders.json
    """
    fixtureCmds = [
        "aptly repo create unstable",
    ]
    runCmd = "aptly repo include -uploaders-file=${changes}/uploaders1.json -no-remove-files -keyring=${files}/aptly.pub ${changes}"
    outputMatchPrepare = lambda _, s: changesRemove(_, gpgRemove(_, s))
    expectedCode = 1


class IncludeRepo14Test(BaseTest):
    """
    include packages to local repo: allow with uploaders.json
    """
    fixtureCmds = [
        "aptly repo create unstable",
    ]
    runCmd = "aptly repo include -uploaders-file=${changes}/uploaders2.json -no-remove-files -keyring=${files}/aptly.pub ${changes}"
    outputMatchPrepare = lambda _, s: changesRemove(_, gpgRemove(_, s))


class IncludeRepo15Test(BaseTest):
    """
    include packages to local repo: no uploaders.json
    """
    fixtureCmds = [
        "aptly repo create unstable",
    ]
    runCmd = "aptly repo include -uploaders-file=${changes}/uploaders-404.json -no-remove-files -keyring=${files}/aptly.pub ${changes}"
    outputMatchPrepare = lambda _, s: changesRemove(_, gpgRemove(_, s))
    expectedCode = 1


class IncludeRepo16Test(BaseTest):
    """
    include packages to local repo: malformed JSON
    """
    fixtureCmds = [
        "aptly repo create unstable",
    ]
    runCmd = "aptly repo include -uploaders-file=${changes}/uploaders3.json -no-remove-files -keyring=${files}/aptly.pub ${changes}"
    outputMatchPrepare = lambda _, s: changesRemove(_, gpgRemove(_, s))
    expectedCode = 1


class IncludeRepo17Test(BaseTest):
    """
    include packages to local repo: malformed rule
    """
    fixtureCmds = [
        "aptly repo create unstable",
    ]
    runCmd = "aptly repo include -uploaders-file=${changes}/uploaders4.json -no-remove-files -keyring=${files}/aptly.pub ${changes}"
    outputMatchPrepare = lambda _, s: changesRemove(_, gpgRemove(_, s))
    expectedCode = 1


class IncludeRepo18Test(BaseTest):
    """
    include packages to local repo: repo uploaders.json + global uploaders.json
    """
    fixtureCmds = [
        "aptly repo create -uploaders-file=${changes}/uploaders2.json unstable",
    ]
    runCmd = "aptly repo include -uploaders-file=${changes}/uploaders1.json -no-remove-files -keyring=${files}/aptly.pub ${changes}"
    outputMatchPrepare = lambda _, s: changesRemove(_, gpgRemove(_, s))


class IncludeRepo19Test(BaseTest):
    """
    include packages to local repo: per-repo uploaders.json
    """
    fixtureCmds = [
        "aptly repo create -uploaders-file=${changes}/uploaders1.json unstable",
    ]
    runCmd = "aptly repo include -no-remove-files -keyring=${files}/aptly.pub ${changes}"
    outputMatchPrepare = lambda _, s: changesRemove(_, gpgRemove(_, s))
    expectedCode = 1
