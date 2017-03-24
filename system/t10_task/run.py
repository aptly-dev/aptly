from lib import BaseTest


class RunTask1Test(BaseTest):
    """
    task run: simple commands, 1-word command
    """
    gold_processor = BaseTest.expand_environ

    runCmd = "aptly task run repo list, repo create local, repo drop local, version"


class RunTask2Test(BaseTest):
    """
    task run: commands with args
    """
    runCmd = "aptly task run -- repo list -raw, repo create local, repo list"


class RunTask3Test(BaseTest):
    """
    task run: failure
    """
    expectedCode = 1
    runCmd = "aptly task run -- repo show a, repo create local, repo list"


class RunTask4Test(BaseTest):
    """
    task run: from file
    """
    runCmd = "aptly task run -filename=${testfiles}/task"


class RunTask5Test(BaseTest):
    """
    task run: from file not found
    """
    expectedCode = 1
    runCmd = "aptly task run -filename=not_found"
