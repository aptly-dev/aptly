from lib import BaseTest


class RunTask1Test(BaseTest):
  """
  output should match
  """

  runCmd = "aptly task run repo list, repo create local, repo drop local, version"
