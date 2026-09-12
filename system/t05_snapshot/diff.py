import re
import tempfile
from pathlib import Path
from lib import BaseTest


def trimTrailingWhitespace(_, s):
    return re.sub(r'\s*$', '', s, flags=re.MULTILINE)


class DiffSnapshot1Test(BaseTest):
    """
    diff two snapshots: normal diff
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-backports",
        "aptly snapshot pull snap1 snap2 snap3 'rsyslog (>= 7.4.4)'"
    ]
    runCmd = "aptly snapshot diff snap1 snap3"
    outputMatchPrepare = trimTrailingWhitespace


class DiffSnapshot2Test(BaseTest):
    """
    diff two snapshots: normal diff II
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-backports",
    ]
    runCmd = "aptly snapshot diff snap1 snap2"
    outputMatchPrepare = trimTrailingWhitespace


class DiffSnapshot3Test(BaseTest):
    """
    diff two snapshots: normal diff II + only-matching
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-backports",
    ]
    runCmd = "aptly snapshot diff -only-matching snap1 snap2"
    outputMatchPrepare = trimTrailingWhitespace


class DiffSnapshot4Test(BaseTest):
    """
    diff two snapshots: doesn't exist
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
    ]
    runCmd = "aptly snapshot diff -only-matching snap1 snap-no"
    expectedCode = 1


class DiffSnapshot5Test(BaseTest):
    """
    diff two snapshots: doesn't exist
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap2 from mirror wheezy-main",
    ]
    runCmd = "aptly snapshot diff -only-matching snap-no snap2"
    expectedCode = 1


class DiffSnapshot6Test(BaseTest):
    """
    diff two snapshots: identical snapshots
    """
    fixtureDB = True
    fixtureCmds = [
        "aptly snapshot create snap1 from mirror wheezy-main",
        "aptly snapshot create snap2 from mirror wheezy-main",
    ]
    runCmd = "aptly snapshot diff snap1 snap2"


class DiffSnapshotArchitectureVariantTest(BaseTest):
    """
    snapshot diff: display base and variant architectures independently
    """
    runCmd = "aptly snapshot diff variant-empty variant-normal"

    def prepare_fixture(self):
        super().prepare_fixture()
        self.run_cmd("aptly repo create variant-display")
        self.run_cmd("aptly snapshot create variant-empty empty")
        with tempfile.TemporaryDirectory(prefix="aptly-diff-variant-") as tmp:
            for arch in ("amd64", "amd64v3"):
                package_dir = Path(tmp) / arch
                control_dir = package_dir / "DEBIAN"
                control_dir.mkdir(parents=True)
                control = (
                    "Package: test-package\n"
                    "Version: 1.0\n"
                    "Architecture: amd64\n"
                    "Maintainer: Aptly Test <test@example.com>\n"
                    "Description: Snapshot diff display fixture\n"
                )
                if arch == "amd64v3":
                    control += "Architecture-Variant: amd64v3\n"
                (control_dir / "control").write_text(control)
                package_file = str(Path(tmp) / (
                    "test-package_1.0_" + arch + ".deb"))
                self.run_cmd(["dpkg-deb", "--build", str(package_dir), package_file])
                self.run_cmd(["aptly", "repo", "add", "variant-display", package_file])
                snapshot = "variant-normal" if arch == "amd64" else "variant-both"
                self.run_cmd([
                    "aptly", "snapshot", "create", snapshot,
                    "from", "repo", "variant-display",
                ])

    def check(self):
        def check_row(output, expected):
            lines = output.strip().splitlines()
            self.check_equal(len(lines), 2)
            self.check_equal(" ".join(lines[0].split()),
                             "Arch | Package | Version in A | Version in B")
            self.check_equal(" ".join(lines[1].split()), expected)

        check_row(self.output, "+ amd64 | test-package | - | 1.0")
        for left, right, expected in (
            ("variant-normal", "variant-both",
             "+ amd64v3 | test-package | - | 1.0"),
            ("variant-both", "variant-normal",
             "- amd64v3 | test-package | 1.0 | -"),
            ("variant-normal", "variant-empty",
             "- amd64 | test-package | 1.0 | -"),
        ):
            check_row(self.run_cmd(["aptly", "snapshot", "diff", left, right]), expected)
