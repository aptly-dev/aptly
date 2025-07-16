package cmd

import (
	"bytes"
	"os"
	"strings"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
	. "gopkg.in/check.v1"
)

type SnapshotCreateSuite struct {
	cmd        *commander.Command
	origStdout *os.File
}

var _ = Suite(&SnapshotCreateSuite{})

func (s *SnapshotCreateSuite) SetUpTest(c *C) {
	s.cmd = makeCmdSnapshotCreate()
	s.origStdout = os.Stdout
}

func (s *SnapshotCreateSuite) TearDownTest(c *C) {
	os.Stdout = s.origStdout
}

func (s *SnapshotCreateSuite) TestSnapshotCreateEmpty(c *C) {
	// Test creating empty snapshot
	args := []string{"empty-snapshot", "empty"}

	var buf bytes.Buffer
	os.Stdout = &buf

	err := aptlySnapshotCreate(s.cmd, args)
	c.Check(err, IsNil)

	output := buf.String()
	c.Check(strings.Contains(output, "Snapshot empty-snapshot successfully created"), Equals, true)
	c.Check(strings.Contains(output, "aptly publish snapshot empty-snapshot"), Equals, true)
}

func (s *SnapshotCreateSuite) TestSnapshotCreateFromMirror(c *C) {
	// Test creating snapshot from mirror (will fail due to no context/mirror)
	args := []string{"mirror-snapshot", "from", "mirror", "test-mirror"}

	err := aptlySnapshotCreate(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to create snapshot.*")
}

func (s *SnapshotCreateSuite) TestSnapshotCreateFromRepo(c *C) {
	// Test creating snapshot from local repo (will fail due to no context/repo)
	args := []string{"repo-snapshot", "from", "repo", "test-repo"}

	err := aptlySnapshotCreate(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to create snapshot.*")
}

func (s *SnapshotCreateSuite) TestSnapshotCreateInvalidArgs(c *C) {
	// Test with no arguments
	err := aptlySnapshotCreate(s.cmd, []string{})
	c.Check(err, Equals, commander.ErrCommandError)

	// Test with only name
	err = aptlySnapshotCreate(s.cmd, []string{"test"})
	c.Check(err, Equals, commander.ErrCommandError)

	// Test with wrong syntax
	err = aptlySnapshotCreate(s.cmd, []string{"test", "invalid"})
	c.Check(err, Equals, commander.ErrCommandError)

	// Test with incomplete "from mirror"
	err = aptlySnapshotCreate(s.cmd, []string{"test", "from", "mirror"})
	c.Check(err, Equals, commander.ErrCommandError)

	// Test with incomplete "from repo"
	err = aptlySnapshotCreate(s.cmd, []string{"test", "from", "repo"})
	c.Check(err, Equals, commander.ErrCommandError)

	// Test with wrong "from" syntax
	err = aptlySnapshotCreate(s.cmd, []string{"test", "from", "invalid", "source"})
	c.Check(err, Equals, commander.ErrCommandError)

	// Test with too many arguments
	err = aptlySnapshotCreate(s.cmd, []string{"test", "from", "mirror", "source", "extra"})
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *SnapshotCreateSuite) TestSnapshotCreateValidSyntaxFromMirror(c *C) {
	// Test that valid "from mirror" syntax passes argument validation
	args := []string{"valid-mirror-snapshot", "from", "mirror", "test-mirror"}

	// This will fail at mirror loading but pass argument validation
	err := aptlySnapshotCreate(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to create snapshot.*")
	// Verify it's not a command error (argument validation passed)
	c.Check(err, Not(Equals), commander.ErrCommandError)
}

func (s *SnapshotCreateSuite) TestSnapshotCreateValidSyntaxFromRepo(c *C) {
	// Test that valid "from repo" syntax passes argument validation
	args := []string{"valid-repo-snapshot", "from", "repo", "test-repo"}

	// This will fail at repo loading but pass argument validation
	err := aptlySnapshotCreate(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to create snapshot.*")
	// Verify it's not a command error (argument validation passed)
	c.Check(err, Not(Equals), commander.ErrCommandError)
}

func (s *SnapshotCreateSuite) TestSnapshotCreateMultipleEmpty(c *C) {
	// Test creating multiple empty snapshots
	emptySnapshots := []string{
		"empty-snapshot-1",
		"empty-snapshot-2",
		"empty-snapshot-3",
	}

	for _, name := range emptySnapshots {
		var buf bytes.Buffer
		os.Stdout = &buf

		args := []string{name, "empty"}
		err := aptlySnapshotCreate(s.cmd, args)
		c.Check(err, IsNil, Commentf("Failed for snapshot: %s", name))

		output := buf.String()
		c.Check(strings.Contains(output, "Snapshot "+name+" successfully created"), Equals, true,
			Commentf("Output check failed for snapshot: %s", name))
	}
}

func (s *SnapshotCreateSuite) TestSnapshotCreateSpecialCharacters(c *C) {
	// Test creating snapshots with special characters in names
	testNames := []string{
		"snapshot-with-dashes",
		"snapshot_with_underscores",
		"snapshot.with.dots",
		"snapshot123",
		"UPPERCASESNAPSHOT",
	}

	for _, name := range testNames {
		var buf bytes.Buffer
		os.Stdout = &buf

		args := []string{name, "empty"}
		err := aptlySnapshotCreate(s.cmd, args)
		c.Check(err, IsNil, Commentf("Failed for snapshot name: %s", name))

		output := buf.String()
		c.Check(strings.Contains(output, "Snapshot "+name+" successfully created"), Equals, true,
			Commentf("Output check failed for snapshot name: %s", name))
	}
}

func (s *SnapshotCreateSuite) TestSnapshotCreateEmptyName(c *C) {
	// Test creating snapshot with empty name
	args := []string{"", "empty"}

	var buf bytes.Buffer
	os.Stdout = &buf

	err := aptlySnapshotCreate(s.cmd, args)
	c.Check(err, IsNil) // Empty name is technically valid

	output := buf.String()
	c.Check(strings.Contains(output, "successfully created"), Equals, true)
}

func (s *SnapshotCreateSuite) TestMakeCmdSnapshotCreate(c *C) {
	// Test command creation and configuration
	cmd := makeCmdSnapshotCreate()

	c.Check(cmd.Run, NotNil)
	c.Check(cmd.UsageLine, Equals, "create <name> (from mirror <mirror-name> | from repo <repo-name> | empty)")
	c.Check(cmd.Short, Equals, "creates snapshot of mirror (local repository) contents")

	// Test long description content
	c.Check(strings.Contains(cmd.Long, "Command create <name> from mirror"), Equals, true)
	c.Check(strings.Contains(cmd.Long, "Command create <name> from repo"), Equals, true)
	c.Check(strings.Contains(cmd.Long, "Command create <name> empty"), Equals, true)
	c.Check(strings.Contains(cmd.Long, "aptly snapshot create wheezy-main-today"), Equals, true)
}

func (s *SnapshotCreateSuite) TestSnapshotCreateLongDescription(c *C) {
	// Test detailed long description content
	cmd := makeCmdSnapshotCreate()

	c.Check(strings.Contains(cmd.Long, "persistent immutable snapshot"), Equals, true)
	c.Check(strings.Contains(cmd.Long, "Snapshot could be published"), Equals, true)
	c.Check(strings.Contains(cmd.Long, "merge, pull and other aptly features"), Equals, true)
	c.Check(strings.Contains(cmd.Long, "mixed with snapshots of remote mirrors"), Equals, true)
	c.Check(strings.Contains(cmd.Long, "basis for snapshot pull operations"), Equals, true)
	c.Check(strings.Contains(cmd.Long, "As snapshots are immutable"), Equals, true)
}

func (s *SnapshotCreateSuite) TestSnapshotCreateArgumentCombinations(c *C) {
	// Test various argument combination edge cases

	// Valid combinations that should pass argument validation
	validCombinations := [][]string{
		{"test", "empty"},
		{"test", "from", "mirror", "mirror-name"},
		{"test", "from", "repo", "repo-name"},
	}

	for _, args := range validCombinations {
		err := aptlySnapshotCreate(s.cmd, args)
		// These should pass argument validation (not return commander.ErrCommandError)
		// but may fail later due to missing context/repos/mirrors
		if err == commander.ErrCommandError {
			c.Fatalf("Argument validation failed for valid combination: %v", args)
		}
	}

	// Invalid combinations that should fail argument validation
	invalidCombinations := [][]string{
		{},                                    // No arguments
		{"test"},                              // Missing type
		{"test", "from"},                      // Incomplete from
		{"test", "from", "mirror"},            // Missing mirror name
		{"test", "from", "repo"},              // Missing repo name
		{"test", "from", "invalid", "source"}, // Invalid source type
		{"test", "invalid"},                   // Invalid type
		{"test", "empty", "extra"},            // Extra arguments
	}

	for _, args := range invalidCombinations {
		err := aptlySnapshotCreate(s.cmd, args)
		c.Check(err, Equals, commander.ErrCommandError,
			Commentf("Expected command error for invalid combination: %v", args))
	}
}

func (s *SnapshotCreateSuite) TestSnapshotCreateCaseSensitivity(c *C) {
	// Test that keywords are case sensitive

	// These should fail because keywords must be exact
	invalidCases := [][]string{
		{"test", "Empty"},                  // Capital E
		{"test", "EMPTY"},                  // All caps
		{"test", "from", "Mirror", "test"}, // Capital M
		{"test", "from", "Repo", "test"},   // Capital R
		{"test", "From", "mirror", "test"}, // Capital F
	}

	for _, args := range invalidCases {
		err := aptlySnapshotCreate(s.cmd, args)
		c.Check(err, Equals, commander.ErrCommandError,
			Commentf("Expected case sensitivity failure for: %v", args))
	}
}

func (s *SnapshotCreateSuite) TestSnapshotCreateOutputFormat(c *C) {
	// Test the specific output format for empty snapshots
	args := []string{"format-test", "empty"}

	var buf bytes.Buffer
	os.Stdout = &buf

	err := aptlySnapshotCreate(s.cmd, args)
	c.Check(err, IsNil)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have exactly 2 lines of output
	c.Check(len(lines), Equals, 2)

	// First line should contain success message
	c.Check(lines[0], Matches, "Snapshot format-test successfully created\\.")

	// Second line should contain publish instruction
	c.Check(lines[1], Matches, "You can run 'aptly publish snapshot format-test' to publish snapshot as Debian repository\\.")
}
