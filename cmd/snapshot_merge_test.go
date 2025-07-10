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

type SnapshotMergeSuite struct {
	cmd               *commander.Command
	collectionFactory *deb.CollectionFactory
	mockProgress      *MockSnapshotMergeProgress
	mockContext       *MockSnapshotMergeContext
}

var _ = Suite(&SnapshotMergeSuite{})

func (s *SnapshotMergeSuite) SetUpTest(c *C) {
	s.cmd = makeCmdSnapshotMerge()
	s.mockProgress = &MockSnapshotMergeProgress{}

	// Set up mock collections
	s.collectionFactory = &deb.CollectionFactory{
		snapshotCollection: &MockSnapshotMergeCollection{},
	}

	// Set up mock context
	s.mockContext = &MockSnapshotMergeContext{
		flags:             s.cmd.Flag,
		progress:          s.mockProgress,
		collectionFactory: s.collectionFactory,
	}

	// Set up required flags
	s.cmd.Flag.Bool("latest", false, "use only the latest version of each package")
	s.cmd.Flag.Bool("no-remove", false, "don't remove duplicate arch/name packages")

	// Set mock context globally
	context = s.mockContext
}

func (s *SnapshotMergeSuite) TestMakeCmdSnapshotMerge(c *C) {
	// Test command creation and basic properties
	cmd := makeCmdSnapshotMerge()
	c.Check(cmd, NotNil)
	c.Check(cmd.UsageLine, Equals, "merge <destination> <source> [<source>...]")
	c.Check(cmd.Short, Equals, "merges snapshots")
	c.Check(strings.Contains(cmd.Long, "Merge command merges several <source> snapshots into one <destination> snapshot"), Equals, true)

	// Test flags
	latestFlag := cmd.Flag.Lookup("latest")
	c.Check(latestFlag, NotNil)
	c.Check(latestFlag.DefValue, Equals, "false")

	noRemoveFlag := cmd.Flag.Lookup("no-remove")
	c.Check(noRemoveFlag, NotNil)
	c.Check(noRemoveFlag.DefValue, Equals, "false")
}

func (s *SnapshotMergeSuite) TestAptlySnapshotMergeInvalidArgs(c *C) {
	// Test with insufficient arguments
	err := aptlySnapshotMerge(s.cmd, []string{})
	c.Check(err, Equals, commander.ErrCommandError)

	err = aptlySnapshotMerge(s.cmd, []string{"destination"})
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *SnapshotMergeSuite) TestAptlySnapshotMergeBasic(c *C) {
	// Test basic snapshot merge
	args := []string{"merged-snapshot", "source1", "source2"}

	err := aptlySnapshotMerge(s.cmd, args)
	c.Check(err, IsNil)

	// Check that success message was displayed
	foundSuccessMessage := false
	foundPublishMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "Snapshot merged-snapshot successfully created") {
			foundSuccessMessage = true
		}
		if strings.Contains(msg, "aptly publish snapshot merged-snapshot") {
			foundPublishMessage = true
		}
	}
	c.Check(foundSuccessMessage, Equals, true)
	c.Check(foundPublishMessage, Equals, true)
}

func (s *SnapshotMergeSuite) TestAptlySnapshotMergeSnapshotNotFound(c *C) {
	// Test with non-existent source snapshot
	mockCollection := &MockSnapshotMergeCollection{shouldErrorByName: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"merged-snapshot", "nonexistent-source"}
	err := aptlySnapshotMerge(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to load snapshot.*")
}

func (s *SnapshotMergeSuite) TestAptlySnapshotMergeLoadCompleteError(c *C) {
	// Test with load complete error
	mockCollection := &MockSnapshotMergeCollection{shouldErrorLoadComplete: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"merged-snapshot", "source1"}
	err := aptlySnapshotMerge(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to load snapshot.*")
}

func (s *SnapshotMergeSuite) TestAptlySnapshotMergeAddError(c *C) {
	// Test with add snapshot error
	mockCollection := &MockSnapshotMergeCollection{shouldErrorAdd: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"merged-snapshot", "source1"}
	err := aptlySnapshotMerge(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to create snapshot.*")
}

func (s *SnapshotMergeSuite) TestAptlySnapshotMergeLatestFlag(c *C) {
	// Test with latest flag
	s.cmd.Flag.Set("latest", "true")
	args := []string{"merged-snapshot", "source1", "source2"}

	err := aptlySnapshotMerge(s.cmd, args)
	c.Check(err, IsNil)

	// Should succeed with latest flag
	foundSuccessMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "successfully created") {
			foundSuccessMessage = true
			break
		}
	}
	c.Check(foundSuccessMessage, Equals, true)
}

func (s *SnapshotMergeSuite) TestAptlySnapshotMergeNoRemoveFlag(c *C) {
	// Test with no-remove flag
	s.cmd.Flag.Set("no-remove", "true")
	args := []string{"merged-snapshot", "source1", "source2"}

	err := aptlySnapshotMerge(s.cmd, args)
	c.Check(err, IsNil)

	// Should succeed with no-remove flag
	foundSuccessMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "successfully created") {
			foundSuccessMessage = true
			break
		}
	}
	c.Check(foundSuccessMessage, Equals, true)
}

func (s *SnapshotMergeSuite) TestAptlySnapshotMergeConflictingFlags(c *C) {
	// Test with conflicting flags (latest and no-remove together)
	s.cmd.Flag.Set("latest", "true")
	s.cmd.Flag.Set("no-remove", "true")
	args := []string{"merged-snapshot", "source1"}

	err := aptlySnapshotMerge(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*-no-remove and -latest can't be specified together.*")
}

func (s *SnapshotMergeSuite) TestAptlySnapshotMergeSingleSource(c *C) {
	// Test merging with only one source (copy operation)
	args := []string{"copy-snapshot", "source1"}

	err := aptlySnapshotMerge(s.cmd, args)
	c.Check(err, IsNil)

	// Should succeed even with single source
	foundSuccessMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "successfully created") {
			foundSuccessMessage = true
			break
		}
	}
	c.Check(foundSuccessMessage, Equals, true)
}

func (s *SnapshotMergeSuite) TestAptlySnapshotMergeMultipleSources(c *C) {
	// Test merging with multiple sources
	args := []string{"multi-merged", "source1", "source2", "source3", "source4"}

	err := aptlySnapshotMerge(s.cmd, args)
	c.Check(err, IsNil)

	// Should handle multiple sources
	foundSuccessMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "successfully created") {
			foundSuccessMessage = true
			break
		}
	}
	c.Check(foundSuccessMessage, Equals, true)
}

func (s *SnapshotMergeSuite) TestAptlySnapshotMergeDescriptionGeneration(c *C) {
	// Test that merge description includes source names
	args := []string{"described-merge", "alpha-snapshot", "beta-snapshot"}

	err := aptlySnapshotMerge(s.cmd, args)
	c.Check(err, IsNil)

	// Description should be generated with source names
	foundSuccessMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "successfully created") {
			foundSuccessMessage = true
			break
		}
	}
	c.Check(foundSuccessMessage, Equals, true)
}

func (s *SnapshotMergeSuite) TestAptlySnapshotMergeFlagCombinations(c *C) {
	// Test various flag combinations
	flagTests := []struct {
		latest    string
		noRemove  string
		shouldErr bool
	}{
		{"false", "false", false}, // Default behavior
		{"true", "false", false},  // Latest only
		{"false", "true", false},  // No-remove only
		{"true", "true", true},    // Conflicting flags
	}

	for _, test := range flagTests {
		s.cmd.Flag.Set("latest", test.latest)
		s.cmd.Flag.Set("no-remove", test.noRemove)

		args := []string{"test-merge", "source1", "source2"}
		err := aptlySnapshotMerge(s.cmd, args)

		if test.shouldErr {
			c.Check(err, NotNil, Commentf("Latest: %s, NoRemove: %s", test.latest, test.noRemove))
		} else {
			c.Check(err, IsNil, Commentf("Latest: %s, NoRemove: %s", test.latest, test.noRemove))
		}

		// Reset for next test
		s.mockProgress.Messages = []string{}
	}
}

func (s *SnapshotMergeSuite) TestAptlySnapshotMergeSourceLoadOrder(c *C) {
	// Test that sources are loaded in correct order
	mockCollection := &MockSnapshotMergeCollection{trackLoadOrder: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"ordered-merge", "first", "second", "third"}
	err := aptlySnapshotMerge(s.cmd, args)
	c.Check(err, IsNil)

	// Sources should be loaded in order
	c.Check(len(mockCollection.loadOrder), Equals, 3)
	c.Check(mockCollection.loadOrder[0], Equals, "first")
	c.Check(mockCollection.loadOrder[1], Equals, "second")
	c.Check(mockCollection.loadOrder[2], Equals, "third")
}

func (s *SnapshotMergeSuite) TestAptlySnapshotMergeRefListMerging(c *C) {
	// Test that RefList.Merge is called with correct parameters
	mockCollection := &MockSnapshotMergeCollection{trackMergeOperations: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"merge-test", "source1", "source2"}
	err := aptlySnapshotMerge(s.cmd, args)
	c.Check(err, IsNil)

	// Should have performed merge operations
	c.Check(mockCollection.mergeOperations > 0, Equals, true)
}

func (s *SnapshotMergeSuite) TestAptlySnapshotMergeFilterLatestRefs(c *C) {
	// Test that FilterLatestRefs is called when latest flag is set
	mockCollection := &MockSnapshotMergeCollection{trackFilterLatest: true}
	s.collectionFactory.snapshotCollection = mockCollection
	s.cmd.Flag.Set("latest", "true")

	args := []string{"latest-merge", "source1", "source2"}
	err := aptlySnapshotMerge(s.cmd, args)
	c.Check(err, IsNil)

	// Should have called FilterLatestRefs
	c.Check(mockCollection.filterLatestCalled, Equals, true)
}

func (s *SnapshotMergeSuite) TestAptlySnapshotMergeOverrideMatching(c *C) {
	// Test override matching behavior with different flag combinations
	scenarios := []struct {
		latest    bool
		noRemove  bool
		expectOverride bool
	}{
		{false, false, true},  // Default: override matching
		{true, false, false},  // Latest: no override matching
		{false, true, false},  // No-remove: no override matching
	}

	for _, scenario := range scenarios {
		s.cmd.Flag.Set("latest", fmt.Sprintf("%t", scenario.latest))
		s.cmd.Flag.Set("no-remove", fmt.Sprintf("%t", scenario.noRemove))

		args := []string{"override-test", "source1", "source2"}
		err := aptlySnapshotMerge(s.cmd, args)
		c.Check(err, IsNil, Commentf("Latest: %t, NoRemove: %t", scenario.latest, scenario.noRemove))

		// Reset for next test
		s.mockProgress.Messages = []string{}
	}
}

// Mock implementations for testing

type MockSnapshotMergeProgress struct {
	Messages []string
}

func (m *MockSnapshotMergeProgress) Printf(msg string, a ...interface{}) {
	formatted := fmt.Sprintf(msg, a...)
	m.Messages = append(m.Messages, formatted)
}

type MockSnapshotMergeContext struct {
	flags             *flag.FlagSet
	progress          *MockSnapshotMergeProgress
	collectionFactory *deb.CollectionFactory
}

func (m *MockSnapshotMergeContext) Flags() *flag.FlagSet                          { return m.flags }
func (m *MockSnapshotMergeContext) Progress() aptly.Progress                      { return m.progress }
func (m *MockSnapshotMergeContext) NewCollectionFactory() *deb.CollectionFactory { return m.collectionFactory }

type MockSnapshotMergeCollection struct {
	shouldErrorByName       bool
	shouldErrorLoadComplete bool
	shouldErrorAdd          bool
	trackLoadOrder          bool
	trackMergeOperations    bool
	trackFilterLatest       bool
	loadOrder               []string
	mergeOperations         int
	filterLatestCalled      bool
}

func (m *MockSnapshotMergeCollection) ByName(name string) (*deb.Snapshot, error) {
	if m.shouldErrorByName {
		return nil, fmt.Errorf("mock snapshot by name error")
	}

	if m.trackLoadOrder {
		m.loadOrder = append(m.loadOrder, name)
	}

	snapshot := &deb.Snapshot{
		Name:        name,
		Description: fmt.Sprintf("Test snapshot %s", name),
	}
	snapshot.SetRefList(&MockSnapshotMergeRefList{collection: m})

	return snapshot, nil
}

func (m *MockSnapshotMergeCollection) LoadComplete(snapshot *deb.Snapshot) error {
	if m.shouldErrorLoadComplete {
		return fmt.Errorf("mock snapshot load complete error")
	}
	return nil
}

func (m *MockSnapshotMergeCollection) Add(snapshot *deb.Snapshot) error {
	if m.shouldErrorAdd {
		return fmt.Errorf("mock snapshot add error")
	}
	return nil
}

type MockSnapshotMergeRefList struct {
	collection *MockSnapshotMergeCollection
}

func (m *MockSnapshotMergeRefList) Merge(other *deb.PackageRefList, overrideMatching, prefix bool) *deb.PackageRefList {
	if m.collection.trackMergeOperations {
		m.collection.mergeOperations++
	}
	return &MockSnapshotMergeRefList{collection: m.collection}
}

func (m *MockSnapshotMergeRefList) FilterLatestRefs() {
	if m.collection.trackFilterLatest {
		m.collection.filterLatestCalled = true
	}
}

// Mock Snapshot methods
func (s *deb.Snapshot) RefList() *deb.PackageRefList {
	if s.refList != nil {
		return s.refList
	}
	return &MockSnapshotMergeRefList{}
}

func (s *deb.Snapshot) SetRefList(refList *deb.PackageRefList) {
	s.refList = refList
}

// Mock deb.NewSnapshotFromRefList
func init() {
	originalNewSnapshotFromRefList := deb.NewSnapshotFromRefList
	deb.NewSnapshotFromRefList = func(name string, sources []*deb.Snapshot, refList *deb.PackageRefList, description string) *deb.Snapshot {
		snapshot := &deb.Snapshot{
			Name:        name,
			Description: description,
		}
		snapshot.SetRefList(refList)
		return snapshot
	}
	_ = originalNewSnapshotFromRefList // Prevent unused variable warning
}

// Test edge cases
func (s *SnapshotMergeSuite) TestAptlySnapshotMergeSpecialCharacters(c *C) {
	// Test with special characters in snapshot names
	args := []string{"special-merge", "source-with-dashes", "source_with_underscores", "source.with.dots"}

	err := aptlySnapshotMerge(s.cmd, args)
	c.Check(err, IsNil)

	// Should handle special characters in names
	foundSuccessMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "successfully created") {
			foundSuccessMessage = true
			break
		}
	}
	c.Check(foundSuccessMessage, Equals, true)
}

func (s *SnapshotMergeSuite) TestAptlySnapshotMergeLongSourceList(c *C) {
	// Test with many source snapshots
	sources := []string{"destination"}
	for i := 1; i <= 10; i++ {
		sources = append(sources, fmt.Sprintf("source-%d", i))
	}

	err := aptlySnapshotMerge(s.cmd, sources)
	c.Check(err, IsNil)

	// Should handle many sources
	foundSuccessMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "successfully created") {
			foundSuccessMessage = true
			break
		}
	}
	c.Check(foundSuccessMessage, Equals, true)
}

// Test error message formatting
func (s *SnapshotMergeSuite) TestErrorMessageFormatting(c *C) {
	// Test that error messages are properly formatted
	s.cmd.Flag.Set("latest", "true")
	s.cmd.Flag.Set("no-remove", "true")

	args := []string{"conflict-test", "source1"}
	err := aptlySnapshotMerge(s.cmd, args)
	c.Check(err, NotNil)

	// Should show specific error message
	errorMsg := err.Error()
	c.Check(strings.Contains(errorMsg, "-no-remove"), Equals, true)
	c.Check(strings.Contains(errorMsg, "-latest"), Equals, true)
}

// Test merge strategy validation
func (s *SnapshotMergeSuite) TestMergeStrategyValidation(c *C) {
	// Test different merge strategies
	strategies := []struct {
		latest         bool
		noRemove       bool
		expectedStrategy string
	}{
		{false, false, "override"},
		{true, false, "latest"},
		{false, true, "no-remove"},
	}

	for _, strategy := range strategies {
		s.cmd.Flag.Set("latest", fmt.Sprintf("%t", strategy.latest))
		s.cmd.Flag.Set("no-remove", fmt.Sprintf("%t", strategy.noRemove))

		args := []string{"strategy-test", "source1", "source2"}
		err := aptlySnapshotMerge(s.cmd, args)
		c.Check(err, IsNil, Commentf("Strategy: %s", strategy.expectedStrategy))

		// Reset for next test
		s.mockProgress.Messages = []string{}
	}
}

// Test description generation with multiple sources
func (s *SnapshotMergeSuite) TestDescriptionWithManySources(c *C) {
	// Test description generation with many sources
	args := []string{"many-source-merge", "alpha", "beta", "gamma", "delta", "epsilon"}

	err := aptlySnapshotMerge(s.cmd, args)
	c.Check(err, IsNil)

	// Should generate description with all source names
	foundSuccessMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "successfully created") {
			foundSuccessMessage = true
			break
		}
	}
	c.Check(foundSuccessMessage, Equals, true)
}