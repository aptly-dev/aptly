#!/usr/bin/env python

import glob
import importlib
import os
import inspect
import fnmatch
import re
import sys
from tempfile import mkdtemp
import traceback
import random
import subprocess

from lib import BaseTest
from s3_lib import S3Test
from swift_lib import SwiftTest
from api_lib import APITest
from fs_endpoint_lib import FileSystemEndpointTest

try:
    from termcolor import colored
except ImportError:
    def colored(s, **kwargs):
        return s


PYTHON_MINIMUM_VERSION = (3, 9)


def natural_key(string_):
    """See https://blog.codinghorror.com/sorting-for-humans-natural-sort-order/"""
    return [int(s) if s.isdigit() else s for s in re.split(r'(\d+)', string_)]


def walk_modules(package):
    yield importlib.import_module(package)
    for name in sorted(glob.glob(package + "/*.py"), key=natural_key):
        name = os.path.splitext(os.path.basename(name))[0]
        if name == "__init__":
            continue

        yield importlib.import_module(package + "." + name)


def run(include_long_tests=False, capture_results=False, tests=None, filters=None, coverage_dir=None):
    """
    Run system test.
    """
    if not tests:
        tests = sorted(glob.glob("t*_*"), key=natural_key)
    fails = []
    numTests = numFailed = numSkipped = 0
    lastBase = None
    if not coverage_dir:
        coverage_dir = mkdtemp(suffix="aptly-coverage")

    for test in tests:
        for testModule in walk_modules(test):
            for name in sorted(dir(testModule), key=natural_key):
                o = getattr(testModule, name)

                if not (inspect.isclass(o) and issubclass(o, BaseTest) and o is not BaseTest and
                        o is not SwiftTest and o is not S3Test and o is not APITest and o is not FileSystemEndpointTest):
                    continue

                newBase = o.__bases__[0]
                if lastBase is not None and lastBase is not newBase:
                    lastBase.shutdown_class()

                lastBase = newBase

                if filters:
                    matches = False

                    for filt in filters:
                        if fnmatch.fnmatch(o.__name__, filt):
                            matches = True
                            break

                    if not matches:
                        continue

                sys.stdout.write("%s:%s... " % (test, o.__name__))
                sys.stdout.flush()

                t = o()
                if t.longTest and not include_long_tests or not t.fixture_available() or t.skipTest:
                    numSkipped += 1
                    msg = 'SKIP'
                    if t.skipTest and t.skipTest is not True:
                        # If we have a reason to skip, print it
                        msg += ': ' + t.skipTest
                    sys.stdout.write(colored(msg + "\n", color="yellow"))
                    continue

                numTests += 1

                try:
                    t.captureResults = capture_results
                    t.coverage_dir = coverage_dir
                    t.test()
                except Exception:
                    numFailed += 1
                    typ, val, tb = sys.exc_info()
                    fails.append((test, t, typ, val, tb, testModule))
                    traceback.print_exception(typ, val, tb)
                    sys.stdout.write(colored("FAIL\n", color="red"))
                else:
                    sys.stdout.write(colored("OK\n", color="green"))

                t.shutdown()

    if lastBase is not None:
        lastBase.shutdown_class()

    print("COVERAGE_RESULTS: %s" % coverage_dir)

    print("TESTS: %d SUCCESS: %d FAIL: %d SKIP: %d" % (
        numTests, numTests - numFailed, numFailed, numSkipped))

    if len(fails) > 0:
        print("\nFAILURES (%d):" % (len(fails), ))

        for (test, t, typ, val, tb, testModule) in fails:
            doc = t.__doc__ or ''
            print("%s:%s %s" % (test, t.__class__.__name__,
                                testModule.__name__ + ": " + doc.strip()))
            traceback.print_exception(typ, val, tb)
            print("=" * 60)

        sys.exit(1)


if __name__ == "__main__":
    if 'APTLY_VERSION' not in os.environ:
        try:
            os.environ['APTLY_VERSION'] = os.popen(
                "make version").read().strip()
        except BaseException as e:
            print("Failed to capture current version: ", e)

    if sys.version_info < PYTHON_MINIMUM_VERSION:
        raise RuntimeError(f'Tests require Python {PYTHON_MINIMUM_VERSION} or higher.')

    output = subprocess.check_output(['gpg', '--version'], text=True)
    if not output.startswith('gpg (GnuPG) 2'):
        raise RuntimeError('Tests require gpg v2')

    output = subprocess.check_output(['gpgv', '--version'], text=True)
    if not output.startswith('gpgv (GnuPG) 2'):
        raise RuntimeError('Tests require gpgv v2')

    os.chdir(os.path.realpath(os.path.dirname(sys.argv[0])))
    random.seed()
    include_long_tests = False
    capture_results = False
    coverage_dir = None
    tests = None
    args = sys.argv[1:]

    while len(args) > 0 and args[0].startswith("--"):
        if args[0] == "--long":
            include_long_tests = True
        elif args[0] == "--capture":
            capture_results = True
        elif args[0] == "--coverage-dir":
            coverage_dir = args[1]
            args = args[1:]

        args = args[1:]

    tests = []
    filters = []

    for arg in args:
        if arg.startswith('t'):
            tests.append(arg)
        else:
            filters.append(arg)

    run(include_long_tests, capture_results, tests, filters, coverage_dir)
