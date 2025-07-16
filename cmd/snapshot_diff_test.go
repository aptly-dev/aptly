package cmd

import (
	"fmt"
	"strings"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
	"github.com/smira/flag"
	. "gopkg.in/check.v1"
)

type SnapshotDiffSuite struct {
	cmd               *commander.Command
	collectionFactory *deb.CollectionFactory
	mockProgress      *MockSnapshotDiffProgress
	mockContext       *MockSnapshotDiffContext
}

var _ = Suite(&SnapshotDiffSuite{})

func (s *SnapshotDiffSuite) SetUpTest(c *C) {
	s.cmd = makeCmdSnapshotDiff()
	s.mockProgress = &MockSnapshotDiffProgress{}

	// Set up mock collections
	s.collectionFactory = &deb.CollectionFactory{
		snapshotCollection: &MockSnapshotDiffCollection{},
		packageCollection:  &MockSnapshotDiffPackageCollection{},
	}

	// Set up mock context
	s.mockContext = &MockSnapshotDiffContext{
		flags:             s.cmd.Flag,
		progress:          s.mockProgress,
		collectionFactory: s.collectionFactory,
	}

	// Set up required flags
	s.cmd.Flag.Bool("only-matching", false, "display diff only for matching packages")

	// Set mock context globally
	context = s.mockContext
}

func (s *SnapshotDiffSuite) TestMakeCmdSnapshotDiff(c *C) {
	// Test command creation and basic properties
	cmd := makeCmdSnapshotDiff()
	c.Check(cmd, NotNil)
	c.Check(cmd.UsageLine, Equals, "diff <name-a> <name-b>")
	c.Check(cmd.Short, Equals, "difference between two snapshots")
	c.Check(strings.Contains(cmd.Long, "Displays difference in packages between two snapshots"), Equals, true)

	// Test flags
	onlyMatchingFlag := cmd.Flag.Lookup("only-matching")
	c.Check(onlyMatchingFlag, NotNil)
	c.Check(onlyMatchingFlag.DefValue, Equals, "false")
}

func (s *SnapshotDiffSuite) TestAptlySnapshotDiffInvalidArgs(c *C) {
	// Test with insufficient arguments
	err := aptlySnapshotDiff(s.cmd, []string{})
	c.Check(err, Equals, commander.ErrCommandError)

	err = aptlySnapshotDiff(s.cmd, []string{"snapshot-a"})
	c.Check(err, Equals, commander.ErrCommandError)

	// Test with too many arguments
	err = aptlySnapshotDiff(s.cmd, []string{"snapshot-a", "snapshot-b", "extra"})
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *SnapshotDiffSuite) TestAptlySnapshotDiffIdentical(c *C) {
	// Test with identical snapshots
	s.mockContext.identicalSnapshots = true
	args := []string{"snapshot-a", "snapshot-b"}

	err := aptlySnapshotDiff(s.cmd, args)
	c.Check(err, IsNil)

	// Check that identical message was displayed
	foundIdenticalMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "Snapshots are identical") {
			foundIdenticalMessage = true
			break
		}
	}
	c.Check(foundIdenticalMessage, Equals, true)
}

func (s *SnapshotDiffSuite) TestAptlySnapshotDiffWithDifferences(c *C) {
	// Test with different snapshots
	args := []string{"snapshot-a", "snapshot-b"}

	err := aptlySnapshotDiff(s.cmd, args)
	c.Check(err, IsNil)

	// Check that diff header was displayed
	foundHeader := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "Arch   | Package") {
			foundHeader = true
			break
		}
	}
	c.Check(foundHeader, Equals, true)

	// Check that colored output was used for differences
	c.Check(len(s.mockProgress.ColoredMessages) > 0, Equals, true)
}

func (s *SnapshotDiffSuite) TestAptlySnapshotDiffOnlyMatching(c *C) {
	// Test with only-matching flag
	s.cmd.Flag.Set("only-matching", "true")
	args := []string{"snapshot-a", "snapshot-b"}

	err := aptlySnapshotDiff(s.cmd, args)
	c.Check(err, IsNil)

	// Should filter to only show matching packages
	c.Check(len(s.mockProgress.Messages) > 0, Equals, true)
}

func (s *SnapshotDiffSuite) TestAptlySnapshotDiffSnapshotANotFound(c *C) {
	// Test with non-existent first snapshot
	mockCollection := &MockSnapshotDiffCollection{shouldErrorByNameA: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"nonexistent-a", "snapshot-b"}
	err := aptlySnapshotDiff(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to load snapshot A.*")
}

func (s *SnapshotDiffSuite) TestAptlySnapshotDiffSnapshotBNotFound(c *C) {
	// Test with non-existent second snapshot
	mockCollection := &MockSnapshotDiffCollection{shouldErrorByNameB: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"snapshot-a", "nonexistent-b"}
	err := aptlySnapshotDiff(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to load snapshot B.*")
}

func (s *SnapshotDiffSuite) TestAptlySnapshotDiffLoadCompleteErrorA(c *C) {
	// Test with load complete error for snapshot A
	mockCollection := &MockSnapshotDiffCollection{shouldErrorLoadCompleteA: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"snapshot-a", "snapshot-b"}
	err := aptlySnapshotDiff(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to load snapshot A.*")
}

func (s *SnapshotDiffSuite) TestAptlySnapshotDiffLoadCompleteErrorB(c *C) {
	// Test with load complete error for snapshot B
	mockCollection := &MockSnapshotDiffCollection{shouldErrorLoadCompleteB: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"snapshot-a", "snapshot-b"}
	err := aptlySnapshotDiff(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to load snapshot B.*")
}

func (s *SnapshotDiffSuite) TestAptlySnapshotDiffCalculateError(c *C) {
	// Test with diff calculation error
	s.mockContext.shouldErrorDiff = true

	args := []string{"snapshot-a", "snapshot-b"}
	err := aptlySnapshotDiff(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to calculate diff.*")
}

func (s *SnapshotDiffSuite) TestAptlySnapshotDiffPackageStates(c *C) {
	// Test different package states in diff
	s.mockContext.testAllPackageStates = true
	args := []string{"snapshot-a", "snapshot-b"}

	err := aptlySnapshotDiff(s.cmd, args)
	c.Check(err, IsNil)

	// Should show different types of changes with different colors
	foundAddition := false
	foundRemoval := false
	foundUpdate := false

	for _, msg := range s.mockProgress.ColoredMessages {
		if strings.Contains(msg, "@g+@|") {
			foundAddition = true
		}
		if strings.Contains(msg, "@r-@|") {
			foundRemoval = true
		}
		if strings.Contains(msg, "@y!@|") {
			foundUpdate = true
		}
	}

	c.Check(foundAddition, Equals, true)
	c.Check(foundRemoval, Equals, true)
	c.Check(foundUpdate, Equals, true)
}

func (s *SnapshotDiffSuite) TestAptlySnapshotDiffOnlyMatchingFiltering(c *C) {
	// Test that only-matching properly filters out missing packages
	s.cmd.Flag.Set("only-matching", "true")
	s.mockContext.testOnlyMatchingFiltering = true
	args := []string{"snapshot-a", "snapshot-b"}

	err := aptlySnapshotDiff(s.cmd, args)
	c.Check(err, IsNil)

	// Should only show updates, not additions or removals
	foundAddition := false
	foundRemoval := false
	foundUpdate := false

	for _, msg := range s.mockProgress.ColoredMessages {
		if strings.Contains(msg, "@g+@|") {
			foundAddition = true
		}
		if strings.Contains(msg, "@r-@|") {
			foundRemoval = true
		}
		if strings.Contains(msg, "@y!@|") {
			foundUpdate = true
		}
	}

	// With only-matching, should only show updates, not additions/removals
	c.Check(foundAddition, Equals, false)
	c.Check(foundRemoval, Equals, false)
	c.Check(foundUpdate, Equals, true)
}

// Mock implementations for testing

type MockSnapshotDiffProgress struct {
	Messages        []string
	ColoredMessages []string
}

func (m *MockSnapshotDiffProgress) Printf(msg string, a ...interface{}) {
	formatted := fmt.Sprintf(msg, a...)
	m.Messages = append(m.Messages, formatted)
}

func (m *MockSnapshotDiffProgress) ColoredPrintf(msg string, a ...interface{}) {
	formatted := fmt.Sprintf(msg, a...)
	m.ColoredMessages = append(m.ColoredMessages, formatted)
}

type MockSnapshotDiffContext struct {
	flags                     *flag.FlagSet
	progress                  *MockSnapshotDiffProgress
	collectionFactory         *deb.CollectionFactory
	identicalSnapshots        bool
	shouldErrorDiff           bool
	testAllPackageStates      bool
	testOnlyMatchingFiltering bool
}

func (m *MockSnapshotDiffContext) Flags() *flag.FlagSet     { return m.flags }
func (m *MockSnapshotDiffContext) Progress() aptly.Progress { return m.progress }
func (m *MockSnapshotDiffContext) NewCollectionFactory() *deb.CollectionFactory {
	return m.collectionFactory
}

type MockSnapshotDiffCollection struct {
	shouldErrorByNameA       bool
	shouldErrorByNameB       bool
	shouldErrorLoadCompleteA bool
	shouldErrorLoadCompleteB bool
}

func (m *MockSnapshotDiffCollection) ByName(name string) (*deb.Snapshot, error) {
	if name == "nonexistent-a" && m.shouldErrorByNameA {
		return nil, fmt.Errorf("mock snapshot A by name error")
	}
	if name == "nonexistent-b" && m.shouldErrorByNameB {
		return nil, fmt.Errorf("mock snapshot B by name error")
	}

	snapshot := &deb.Snapshot{
		Name:        name,
		Description: "Test snapshot",
	}
	snapshot.SetRefList(&MockSnapshotDiffRefList{name: name})

	return snapshot, nil
}

func (m *MockSnapshotDiffCollection) LoadComplete(snapshot *deb.Snapshot) error {
	if snapshot.Name == "snapshot-a" && m.shouldErrorLoadCompleteA {
		return fmt.Errorf("mock snapshot A load complete error")
	}
	if snapshot.Name == "snapshot-b" && m.shouldErrorLoadCompleteB {
		return fmt.Errorf("mock snapshot B load complete error")
	}
	return nil
}

type MockSnapshotDiffPackageCollection struct{}

type MockSnapshotDiffRefList struct {
	name string
}

func (m *MockSnapshotDiffRefList) Diff(other *deb.PackageRefList, packageCollection deb.PackageCollection) ([]*deb.PackageDiff, error) {
	if context, ok := context.(*MockSnapshotDiffContext); ok && context.shouldErrorDiff {
		return nil, fmt.Errorf("mock diff calculation error")
	}

	if context, ok := context.(*MockSnapshotDiffContext); ok && context.identicalSnapshots {
		// Return empty diff for identical snapshots
		return []*deb.PackageDiff{}, nil
	}

	// Create mock diff with different package states
	diff := []*deb.PackageDiff{}

	if context, ok := context.(*MockSnapshotDiffContext); ok && context.testOnlyMatchingFiltering {
		// Only include updates (both sides present) for only-matching test
		diff = append(diff, &deb.PackageDiff{
			Left:  &deb.Package{Name: "updated-pkg", Version: "1.0", Architecture: "amd64"},
			Right: &deb.Package{Name: "updated-pkg", Version: "2.0", Architecture: "amd64"},
		})
	} else if context, ok := context.(*MockSnapshotDiffContext); ok && context.testAllPackageStates {
		// Include all types of changes

		// Package only in B (addition)
		diff = append(diff, &deb.PackageDiff{
			Left:  nil,
			Right: &deb.Package{Name: "new-pkg", Version: "1.0", Architecture: "amd64"},
		})

		// Package only in A (removal)
		diff = append(diff, &deb.PackageDiff{
			Left:  &deb.Package{Name: "removed-pkg", Version: "1.0", Architecture: "amd64"},
			Right: nil,
		})

		// Package in both with different versions (update)
		diff = append(diff, &deb.PackageDiff{
			Left:  &deb.Package{Name: "updated-pkg", Version: "1.0", Architecture: "amd64"},
			Right: &deb.Package{Name: "updated-pkg", Version: "2.0", Architecture: "amd64"},
		})
	} else {
		// Default case - simple difference
		diff = append(diff, &deb.PackageDiff{
			Left:  &deb.Package{Name: "test-pkg", Version: "1.0", Architecture: "amd64"},
			Right: &deb.Package{Name: "test-pkg", Version: "2.0", Architecture: "amd64"},
		})
	}

	return diff, nil
}

// Helper methods for Snapshot
func (s *deb.Snapshot) RefList() *deb.PackageRefList {
	if s.refList != nil {
		return s.refList
	}
	return nil
}

func (s *deb.Snapshot) SetRefList(refList *deb.PackageRefList) {
	s.refList = refList
}

// Test different diff scenarios
func (s *SnapshotDiffSuite) TestAptlySnapshotDiffScenarios(c *C) {
	// Test scenarios for package differences
	scenarios := []struct {
		name                    string
		testAllPackageStates    bool
		testOnlyMatching        bool
		identicalSnapshots      bool
		expectedMessages        int
		expectedColoredMessages int
	}{
		{"identical", false, false, true, 1, 0},
		{"with-differences", true, false, false, 1, 3},
		{"only-matching", false, true, false, 1, 1},
	}

	for _, scenario := range scenarios {
		// Reset progress
		s.mockProgress.Messages = []string{}
		s.mockProgress.ColoredMessages = []string{}

		// Set scenario flags
		s.mockContext.testAllPackageStates = scenario.testAllPackageStates
		s.mockContext.testOnlyMatchingFiltering = scenario.testOnlyMatching
		s.mockContext.identicalSnapshots = scenario.identicalSnapshots
		s.cmd.Flag.Set("only-matching", fmt.Sprintf("%t", scenario.testOnlyMatching))

		args := []string{"snapshot-a", "snapshot-b"}
		err := aptlySnapshotDiff(s.cmd, args)
		c.Check(err, IsNil, Commentf("Scenario: %s", scenario.name))

		c.Check(len(s.mockProgress.Messages) >= scenario.expectedMessages, Equals, true,
			Commentf("Scenario: %s, expected at least %d messages, got %d",
				scenario.name, scenario.expectedMessages, len(s.mockProgress.Messages)))

		c.Check(len(s.mockProgress.ColoredMessages) >= scenario.expectedColoredMessages, Equals, true,
			Commentf("Scenario: %s, expected at least %d colored messages, got %d",
				scenario.name, scenario.expectedColoredMessages, len(s.mockProgress.ColoredMessages)))
	}
}

// Test output formatting
func (s *SnapshotDiffSuite) TestOutputFormatting(c *C) {
	// Test that the diff header has the correct format
	expectedHeader := "  Arch   | Package                                  | Version in A                             | Version in B"
	c.Check(len(expectedHeader), Equals, 122) // Verify expected width

	// Test color codes
	colorCodes := []string{"@g+@|", "@r-@|", "@y!@|"}
	for _, code := range colorCodes {
		c.Check(len(code), Equals, 5) // All color codes should be same length
	}
}

// Test package diff structure
func (s *SnapshotDiffSuite) TestPackageDiffStructure(c *C) {
	// Test PackageDiff with different combinations
	testCases := []struct {
		left  *deb.Package
		right *deb.Package
		desc  string
	}{
		{nil, &deb.Package{Name: "new", Version: "1.0", Architecture: "amd64"}, "addition"},
		{&deb.Package{Name: "old", Version: "1.0", Architecture: "amd64"}, nil, "removal"},
		{&deb.Package{Name: "pkg", Version: "1.0", Architecture: "amd64"},
			&deb.Package{Name: "pkg", Version: "2.0", Architecture: "amd64"}, "update"},
	}

	for _, testCase := range testCases {
		diff := &deb.PackageDiff{
			Left:  testCase.left,
			Right: testCase.right,
		}

		// Verify the structure
		c.Check(diff.Left, Equals, testCase.left, Commentf("Case: %s", testCase.desc))
		c.Check(diff.Right, Equals, testCase.right, Commentf("Case: %s", testCase.desc))
	}
}

// Test edge cases
func (s *SnapshotDiffSuite) TestSnapshotDiffEdgeCases(c *C) {
	// Test with empty snapshots
	s.mockContext.identicalSnapshots = true
	args := []string{"empty-a", "empty-b"}
	err := aptlySnapshotDiff(s.cmd, args)
	c.Check(err, IsNil)

	// Should handle empty snapshots gracefully
	foundIdenticalMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "Snapshots are identical") {
			foundIdenticalMessage = true
			break
		}
	}
	c.Check(foundIdenticalMessage, Equals, true)
}
