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
        "aptly repo create -comment=Repo1 -distribution=squeeze repo1",
    ]
    runCmd = "aptly repo add repo1 ${files}/libboost-program-options-dev_1.49.0.1_i386.deb"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo1", "repo_show")

        # check pool
        self.check_exists('pool/c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb')


class AddRepo2Test(BaseTest):
    """
    add package to local repo: .dsc file
    """
    fixtureCmds = [
        "aptly repo create -comment=Repo2 -distribution=squeeze repo2",
    ]
    runCmd = "aptly repo add repo2 ${files}/pyspi_0.6.1-1.3.dsc ${files}/pyspi-0.6.1-1.3.stripped.dsc"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo2", "repo_show")

        # check pool
        self.check_exists('pool/2e/77/0b28df948f3197ed0b679bdea99f_pyspi_0.6.1-1.3.diff.gz')
        self.check_exists('pool/d4/94/aaf526f1ec6b02f14c2f81e060a5_pyspi_0.6.1-1.3.dsc')
        self.check_exists('pool/64/06/9ee828c50b1c597d10a3fefbba27_pyspi_0.6.1.orig.tar.gz')
        self.check_exists('pool/28/9d/3aefa970876e9c43686ce2b02f47_pyspi-0.6.1-1.3.stripped.dsc')


class AddRepo3Test(BaseTest):
    """
    add package to local repo: directory
    """
    fixtureCmds = [
        "aptly repo create -comment=Repo3 -distribution=squeeze repo3",
    ]
    runCmd = "aptly repo add repo3 ${files}"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo3", "repo_show")

        # check pool
        self.check_exists('pool/c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb')
        self.check_exists('pool/2e/77/0b28df948f3197ed0b679bdea99f_pyspi_0.6.1-1.3.diff.gz')
        self.check_exists('pool/d4/94/aaf526f1ec6b02f14c2f81e060a5_pyspi_0.6.1-1.3.dsc')
        self.check_exists('pool/64/06/9ee828c50b1c597d10a3fefbba27_pyspi_0.6.1.orig.tar.gz')
        self.check_exists('pool/28/9d/3aefa970876e9c43686ce2b02f47_pyspi-0.6.1-1.3.stripped.dsc')


class AddRepo4Test(BaseTest):
    """
    add package to local repo: complex directory + remove
    """
    fixtureCmds = [
        "aptly repo create -comment=Repo4 -distribution=squeeze repo4",
    ]
    runCmd = "aptly repo add -remove-files repo4 "

    def prepare(self):
        super(AddRepo4Test, self).prepare()

        self.tempSrcDir = tempfile.mkdtemp()
        os.makedirs(os.path.join(self.tempSrcDir, "01"), 0o755)
        os.makedirs(os.path.join(self.tempSrcDir, "02", "03"), 0o755)

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
        self.check_exists('pool/c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb')
        self.check_exists('pool/2e/77/0b28df948f3197ed0b679bdea99f_pyspi_0.6.1-1.3.diff.gz')
        self.check_exists('pool/d4/94/aaf526f1ec6b02f14c2f81e060a5_pyspi_0.6.1-1.3.dsc')
        self.check_exists('pool/64/06/9ee828c50b1c597d10a3fefbba27_pyspi_0.6.1.orig.tar.gz')

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
        "aptly repo create -comment=Repo5 -distribution=squeeze repo5",
    ]
    runCmd = "aptly repo add repo5 "
    expectedCode = 1

    def outputMatchPrepare(self, s):
        return s.replace(self.tempSrcDir, "")

    def prepare(self):
        super(AddRepo5Test, self).prepare()

        self.tempSrcDir = tempfile.mkdtemp()
        os.makedirs(os.path.join(self.tempSrcDir, "02", "03"), 0o755)

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
        "aptly repo create -comment=Repo6 -distribution=squeeze repo6",
    ]
    runCmd = "aptly repo add repo6 no-such-file"
    expectedCode = 1


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
        "aptly repo create -comment=Repo8 -distribution=squeeze repo8",
        "aptly repo add repo8 ${files}/pyspi_0.6.1-1.3.dsc",
    ]
    runCmd = "aptly repo add repo8 ${testfiles}/pyspi_0.6.1-1.3.conflict.dsc"
    expectedCode = 1

    def outputMatchPrepare(self, s):
        return s.replace(os.path.join(os.path.dirname(inspect.getsourcefile(self.__class__)), self.__class__.__name__), ""). \
                         replace(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files"), "")

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo8", "repo_show")


class AddRepo9Test(BaseTest):
    """
    add package to local repo: conflict in files
    """
    fixtureCmds = [
        "aptly repo create -comment=Repo9 -distribution=squeeze repo9",
    ]
    runCmd = "aptly repo add repo9 ${files}/pyspi_0.6.1-1.3.dsc"
    gold_processor = BaseTest.expand_environ
    expectedCode = 1

    def outputMatchPrepare(self, s):
        return s.replace(os.path.join(os.path.dirname(inspect.getsourcefile(self.__class__)), self.__class__.__name__), ""). \
                         replace(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files"), "")

    def prepare(self):
        super(AddRepo9Test, self).prepare()

        os.makedirs(os.path.join(os.environ["HOME"], ".aptly", "pool/64/06/"))
        with open(os.path.join(os.environ["HOME"], ".aptly", "pool/64/06/9ee828c50b1c597d10a3fefbba27_pyspi_0.6.1.orig.tar.gz"), "w") as f:
            f.write("abcd")


class AddRepo10Test(BaseTest):
    """
    add package to local repo: double import
    """
    fixtureCmds = [
        "aptly repo create -comment=Repo10 -distribution=squeeze repo10",
        "aptly repo add repo10 ${files}",
    ]
    runCmd = "aptly repo add repo10 ${files}/pyspi_0.6.1-1.3.dsc"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo10", "repo_show")


class AddRepo11Test(BaseTest):
    """
    add package to local repo: conflict in packages + -force-replace
    """
    fixtureCmds = [
        "aptly repo create -comment=Repo11 -distribution=squeeze repo11",
        "aptly repo add repo11 ${files}/pyspi_0.6.1-1.3.dsc",
    ]
    runCmd = "aptly repo add -force-replace repo11 ${testfiles}/pyspi_0.6.1-1.3.conflict.dsc"

    def outputMatchPrepare(self, s):
        return s.replace(os.path.join(os.path.dirname(inspect.getsourcefile(self.__class__)), self.__class__.__name__), ""). \
                         replace(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files"), "")

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo11", "repo_show")


class AddRepo12Test(BaseTest):
    """
    add package to local repo: .udeb file
    """
    fixtureCmds = [
        "aptly repo create -comment=Repo12 -distribution=squeeze repo12",
    ]
    runCmd = "aptly repo add repo12 ${udebs}/dmraid-udeb_1.0.0.rc16-4.1_amd64.udeb"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo12", "repo_show")

        # check pool
        self.check_exists('pool/ef/ae/69921b97494e40437712053b60a5_dmraid-udeb_1.0.0.rc16-4.1_amd64.udeb')


class AddRepo13Test(BaseTest):
    """
    add package to local repo: .udeb and .deb files
    """
    fixtureCmds = [
        "aptly repo create -comment=Repo13 -distribution=squeeze repo13",
    ]
    runCmd = "aptly repo add repo13 ${udebs} ${files}"

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show -with-packages repo13", "repo_show")

        # check pool
        self.check_exists('pool/ef/ae/69921b97494e40437712053b60a5_dmraid-udeb_1.0.0.rc16-4.1_amd64.udeb')
        self.check_exists('pool/d4/94/aaf526f1ec6b02f14c2f81e060a5_pyspi_0.6.1-1.3.dsc')


class AddRepo14Test(BaseTest):
    """
    add same package to local repo twice and make sure the file doesn't get truncated.
    """
    fixtureCmds = [
        "aptly repo create -comment=Repo14 -distribution=squeeze repo14",
        "aptly repo add repo14 ${files}/libboost-program-options-dev_1.49.0.1_i386.deb",
        "aptly publish repo -distribution=test1 -skip-signing repo14"
    ]
    runCmd = "aptly repo add repo14 $aptlyroot/public/pool/"

    def check(self):
        super(AddRepo14Test, self).check()
        # check pool
        self.check_exists('pool/c7/6b/4bd12fd92e4dfe1b55b18a67a669_libboost-program-options-dev_1.49.0.1_i386.deb')


class AddRepo15Test(BaseTest):
    """
    add package with wrong case in stanza and missing fields
    """
    fixtureCmds = [
        "aptly repo create -comment=Repo15 -distribution=squeeze repo15",
    ]
    runCmd = "aptly repo add repo15 ${testfiles}"
    expectedCode = 1

    def outputMatchPrepare(self, s):
        return s.replace(os.path.join(os.path.dirname(inspect.getsourcefile(self.__class__)), self.__class__.__name__), ""). \
                         replace(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files"), "")


class AddRepo16Test(BaseTest):
    """
    add package to local repo: some source files missing, but already in the pool
    """
    fixtureCmds = [
        "aptly repo create repo1",
        "aptly repo create repo2",
        "aptly repo add repo1 ${files}"
    ]
    runCmd = "aptly repo add repo2 "

    def outputMatchPrepare(self, s):
        return s.replace(self.tempSrcDir, "")

    def prepare(self):
        super(AddRepo16Test, self).prepare()

        self.tempSrcDir = tempfile.mkdtemp()
        os.makedirs(os.path.join(self.tempSrcDir, "02", "03"), 0o755)

        shutil.copy(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "pyspi_0.6.1-1.3.dsc"),
                    os.path.join(self.tempSrcDir, "02", "03"))
        shutil.copy(os.path.join(os.path.dirname(inspect.getsourcefile(BaseTest)), "files", "pyspi_0.6.1.orig.tar.gz"),
                    os.path.join(self.tempSrcDir, "02", "03"))

        self.runCmd += self.tempSrcDir

    def check(self):
        self.check_output()
        self.check_cmd_output("aptly repo show repo2", "repo_show")

        shutil.rmtree(self.tempSrcDir)
