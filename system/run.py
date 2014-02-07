#!/usr/local/bin/python

import glob
import importlib
import os
import inspect
import sys

from lib import BaseTest

try:
    from termcolor import colored
except ImportError:
    def colored(s, **kwargs):
        return s


def run(include_long_tests=False, tests=None):
    """
    Run system test.
    """
    if tests is None:
        tests = glob.glob("t*_*")
    fails = []
    numTests = numFailed = numSkipped = 0

    for test in tests:

        testModule = importlib.import_module(test)

        for name in dir(testModule):
            o = getattr(testModule, name)

            if not (inspect.isclass(o) and issubclass(o, BaseTest) and o is not BaseTest):
                continue

            t = o()
            if t.longTest and not include_long_tests or not t.fixture_available():
                numSkipped += 1
                continue

            numTests += 1

            sys.stdout.write("%s:%s... " % (test, o.__name__))

            try:
                t.test()
            except BaseException, e:
                numFailed += 1
                fails.append((test, t, e, testModule))
                sys.stdout.write(colored("FAIL\n", color="red"))
            else:
                sys.stdout.write(colored("OK\n", color="green"))

            t.shutdown()

    print "TESTS: %d SUCCESS: %d FAIL: %d SKIP: %d" % (numTests, numTests - numFailed, numFailed, numSkipped)

    if len(fails) > 0:
        print "\nFAILURES (%d):" % (len(fails), )

        for (test, t, e, testModule) in fails:
            print "%s:%s %s" % (test, t.__class__.__name__, testModule.__doc__.strip() + ": " + t.__doc__.strip())
            print "ERROR: %s" % (e, )
            print "=" * 60

        sys.exit(1)

if __name__ == "__main__":
    os.chdir(os.path.realpath(os.path.dirname(sys.argv[0])))
    include_long_tests = False
    tests = None
    if len(sys.argv) > 1:
        if sys.argv[1] == "--long":
            include_long_tests = True
        else:
            tests = sys.argv[1:]
    run(include_long_tests, tests)
