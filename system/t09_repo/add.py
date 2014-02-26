import tempfile
import shutil
import os
import inspect
from lib import BaseTest


class AddRepo1Test(BaseTest):
    """
    add package to local repo: .deb file
    """
    fixtureCmds = [
        "aptly repo create -comment=Repo1 repo1",
    ]
    runCmd = "aptly repo add repo1 ${files}/libboost-program-options-dev_1.49.0.1_i386.deb"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo1", "repo_show")

        # check pool
        self.check_exists('pool/00/35/libboost-program-options-dev_1.49.0.1_i386.deb')


class AddRepo2Test(BaseTest):
    """
    add package to local repo: .dsc file
    """
    fixtureCmds = [
        "aptly repo create -comment=Repo2 repo2",
    ]
    runCmd = "aptly repo add repo2 ${files}/pyspi_0.6.1-1.3.dsc ${files}/pyspi-0.6.1-1.3.stripped.dsc"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo2", "repo_show")

        # check pool
        self.check_exists('pool/22/ff/pyspi_0.6.1-1.3.diff.gz')
        self.check_exists('pool/b7/2c/pyspi_0.6.1-1.3.dsc')
        self.check_exists('pool/de/f3/pyspi_0.6.1.orig.tar.gz')
        self.check_exists('pool/2f/5b/pyspi-0.6.1-1.3.stripped.dsc')


class AddRepo3Test(BaseTest):
    """
    add package to local repo: directory
    """
    fixtureCmds = [
        "aptly repo create -comment=Repo3 repo3",
    ]
    runCmd = "aptly repo add repo3 ${files}"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo3", "repo_show")

        # check pool
        self.check_exists('pool/00/35/libboost-program-options-dev_1.49.0.1_i386.deb')
        self.check_exists('pool/22/ff/pyspi_0.6.1-1.3.diff.gz')
        self.check_exists('pool/b7/2c/pyspi_0.6.1-1.3.dsc')
        self.check_exists('pool/de/f3/pyspi_0.6.1.orig.tar.gz')
        self.check_exists('pool/2f/5b/pyspi-0.6.1-1.3.stripped.dsc')


class AddRepo4Test(BaseTest):
    """
    add package to local repo: complex directory + remove
    """
    fixtureCmds = [
        "aptly repo create -comment=Repo4 repo4",
    ]
    runCmd = "aptly repo add -remove-files repo4 "

    def prepare(self):
        super(AddRepo4Test, self).prepare()

        self.tempSrcDir = tempfile.mkdtemp()
        os.makedirs(os.path.join(self.tempSrcDir, "01"), 0755)
        os.makedirs(os.path.join(self.tempSrcDir, "02", "03"), 0755)

        shutil.copy(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "libboost-program-options-dev_1.49.0.1_i386.deb"),
            os.path.join(self.tempSrcDir, "01"))
        shutil.copy(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "pyspi_0.6.1-1.3.dsc"),
            os.path.join(self.tempSrcDir, "02", "03"))
        shutil.copy(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "pyspi_0.6.1.orig.tar.gz"),
            os.path.join(self.tempSrcDir, "02", "03"))
        shutil.copy(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "pyspi_0.6.1-1.3.diff.gz"),
            os.path.join(self.tempSrcDir, "02", "03"))
        shutil.copy(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "pyspi_0.6.1-1.3.diff.gz"),
            os.path.join(self.tempSrcDir, "02", "03", "other.file"))

        self.runCmd += self.tempSrcDir

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo4", "repo_show")

        # check pool
        self.check_exists('pool/00/35/libboost-program-options-dev_1.49.0.1_i386.deb')
        self.check_exists('pool/22/ff/pyspi_0.6.1-1.3.diff.gz')
        self.check_exists('pool/b7/2c/pyspi_0.6.1-1.3.dsc')
        self.check_exists('pool/de/f3/pyspi_0.6.1.orig.tar.gz')

        path = os.path.join(self.tempSrcDir, "01", "libboost-program-options-dev_1.49.0.1_i386.deb")
        if os.path.exists(path):
            raise Exception("path %s shouldn't exist" % (path, ))
        path = os.path.join(self.tempSrcDir, "02", "03", "pyspi_0.6.1.orig.tar.gz")
        if os.path.exists(path):
            raise Exception("path %s shouldn't exist" % (path, ))

        path = os.path.join(self.tempSrcDir, "02", "03", "other.file")
        if not os.path.exists(path):
            raise Exception("path %s doesn't exist" % (path, ))

        shutil.rmtree(self.tempSrcDir)


class AddRepo5Test(BaseTest):
    """
    add package to local repo: some source files missing
    """
    fixtureCmds = [
        "aptly repo create -comment=Repo5 repo5",
    ]
    runCmd = "aptly repo add repo5 "
    outputMatchPrepare = lambda self, s: s.replace(self.tempSrcDir, "")

    def prepare(self):
        super(AddRepo5Test, self).prepare()

        self.tempSrcDir = tempfile.mkdtemp()
        os.makedirs(os.path.join(self.tempSrcDir, "02", "03"), 0755)

        shutil.copy(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "pyspi_0.6.1-1.3.dsc"),
            os.path.join(self.tempSrcDir, "02", "03"))
        shutil.copy(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "pyspi_0.6.1.orig.tar.gz"),
            os.path.join(self.tempSrcDir, "02", "03"))

        self.runCmd += self.tempSrcDir

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show repo5", "repo_show")

        shutil.rmtree(self.tempSrcDir)


class AddRepo6Test(BaseTest):
    """
    add package to local repo: missing file
    """
    fixtureCmds = [
        "aptly repo create -comment=Repo6 repo6",
    ]
    runCmd = "aptly repo add repo6 no-such-file"


class AddRepo7Test(BaseTest):
    """
    add package to local repo: missing repo
    """
    runCmd = "aptly repo add repo7 ${files}"
    expectedCode = 1


class AddRepo8Test(BaseTest):
    """
    add package to local repo: conflict in packages
    """
    fixtureCmds = [
        "aptly repo create -comment=Repo8 repo8",
        "aptly repo add repo8 ${files}/pyspi_0.6.1-1.3.dsc",
    ]
    runCmd = "aptly repo add repo8 ${testfiles}/pyspi_0.6.1-1.3.conflict.dsc"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo8", "repo_show")


class AddRepo9Test(BaseTest):
    """
    add package to local repo: conflict in files
    """
    fixtureCmds = [
        "aptly repo create -comment=Repo9 repo9",
    ]
    runCmd = "aptly repo add repo9 ${files}/pyspi_0.6.1-1.3.dsc"
    outputMatchPrepare = lambda self, s: s.replace(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files"), "")
    gold_processor = BaseTest.expand_environ

    def prepare(self):
        super(AddRepo9Test, self).prepare()

        os.makedirs(os.path.join(os.environ["HOME"], ".aptly", "pool/de/f3/"))
        with open(os.path.join(os.environ["HOME"], ".aptly", "pool/de/f3/pyspi_0.6.1.orig.tar.gz"), "w") as f:
            f.write("abcd")


class AddRepo10Test(BaseTest):
    """
    add package to local repo: double import
    """
    fixtureCmds = [
        "aptly repo create -comment=Repo10 repo10",
        "aptly repo add repo10 ${files}",
    ]
    runCmd = "aptly repo add repo10 ${files}/pyspi_0.6.1-1.3.dsc"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo10", "repo_show")
