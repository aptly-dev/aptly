#!/usr/bin/python

import glob
import importlib
import sys


def run():
    """
    Run system test.
    """
    tests = glob.glob("t*_*")
    fails = []

    for test in tests:
        sys.stdout.write("%s..." % (test, ))

        testModule = importlib.import_module(test)
        t = testModule.Test()

        try:
            t.test()
        except BaseException, e:
            fails.append((test, t, e, testModule))
            sys.stdout.write("FAIL\n")
        else:
            sys.stdout.write("OK\n")

    if len(fails) > 0:
        print "\nFAILURES (%d):" % (len(fails), )

        for (test, t, e, testModule) in fails:
            print "%s: %s" % (test, testModule.__doc__.strip())
            print "ERROR: %s" % (e, )
            print "=" * 60


if __name__ == "__main__":
    run()
