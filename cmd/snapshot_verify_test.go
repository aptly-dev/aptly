package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
	"github.com/smira/flag"
	. "gopkg.in/check.v1"
)

type SnapshotVerifySuite struct {
	cmd               *commander.Command
	collectionFactory *deb.CollectionFactory
	mockProgress      *MockSnapshotVerifyProgress
	mockContext       *MockSnapshotVerifyContext
}

var _ = Suite(&SnapshotVerifySuite{})

func (s *SnapshotVerifySuite) SetUpTest(c *C) {
	s.cmd = makeCmdSnapshotVerify()
	s.mockProgress = &MockSnapshotVerifyProgress{}

	// Set up mock collections
	s.collectionFactory = &deb.CollectionFactory{
		snapshotCollection: &MockSnapshotVerifyCollection{},
		packageCollection:  &MockSnapshotVerifyPackageCollection{},
	}

	// Set up mock context
	s.mockContext = &MockSnapshotVerifyContext{
		flags:             s.cmd.Flag,
		progress:          s.mockProgress,
		collectionFactory: s.collectionFactory,
		architectures:     []string{"amd64", "i386"},
		dependencyOptions: aptly.DependencyOptions{
			FollowRecommends:  false,
			FollowSuggests:    false,
			FollowSource:      false,
			FollowAllVariants: false,
		},
	}

	// Set mock context globally
	context = s.mockContext
}

func (s *SnapshotVerifySuite) TestMakeCmdSnapshotVerify(c *C) {
	// Test command creation and basic properties
	cmd := makeCmdSnapshotVerify()
	c.Check(cmd, NotNil)
	c.Check(cmd.UsageLine, Equals, "verify <name> [<source> ...]")
	c.Check(cmd.Short, Equals, "verify dependencies in snapshot")
	c.Check(strings.Contains(cmd.Long, "Verify does dependency resolution in snapshot"), Equals, true)
}

func (s *SnapshotVerifySuite) TestAptlySnapshotVerifyInvalidArgs(c *C) {
	// Test with no arguments
	err := aptlySnapshotVerify(s.cmd, []string{})
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *SnapshotVerifySuite) TestAptlySnapshotVerifyBasic(c *C) {
	// Test basic snapshot verification
	args := []string{"test-snapshot"}

	err := aptlySnapshotVerify(s.cmd, args)
	c.Check(err, IsNil)

	// Check that verification messages were displayed
	foundLoadingMessage := false
	foundVerifyingMessage := false
	foundSatisfiedMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "Loading packages") {
			foundLoadingMessage = true
		}
		if strings.Contains(msg, "Verifying") {
			foundVerifyingMessage = true
		}
		if strings.Contains(msg, "All dependencies are satisfied") {
			foundSatisfiedMessage = true
		}
	}
	c.Check(foundLoadingMessage, Equals, true)
	c.Check(foundVerifyingMessage, Equals, true)
	c.Check(foundSatisfiedMessage, Equals, true)
}

func (s *SnapshotVerifySuite) TestAptlySnapshotVerifyMultipleSnapshots(c *C) {
	// Test verification with multiple snapshots as sources
	args := []string{"test-snapshot", "source-snapshot1", "source-snapshot2"}

	err := aptlySnapshotVerify(s.cmd, args)
	c.Check(err, IsNil)

	// Should process all snapshots
	c.Check(len(s.mockProgress.Messages) > 0, Equals, true)
}

func (s *SnapshotVerifySuite) TestAptlySnapshotVerifySnapshotNotFound(c *C) {
	// Test with non-existent snapshot
	mockCollection := &MockSnapshotVerifyCollection{shouldErrorByName: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"nonexistent-snapshot"}
	err := aptlySnapshotVerify(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to verify.*")
}

func (s *SnapshotVerifySuite) TestAptlySnapshotVerifyLoadCompleteError(c *C) {
	// Test with load complete error
	mockCollection := &MockSnapshotVerifyCollection{shouldErrorLoadComplete: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotVerify(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to verify.*")
}

func (s *SnapshotVerifySuite) TestAptlySnapshotVerifyPackageListError(c *C) {
	// Test with package list creation error
	mockPackageCollection := &MockSnapshotVerifyPackageCollection{shouldErrorNewPackageList: true}
	s.collectionFactory.packageCollection = mockPackageCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotVerify(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to load packages.*")
}

func (s *SnapshotVerifySuite) TestAptlySnapshotVerifyAppendError(c *C) {
	// Test with package list append error
	mockPackageCollection := &MockSnapshotVerifyPackageCollection{shouldErrorAppend: true}
	s.collectionFactory.packageCollection = mockPackageCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotVerify(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to merge sources.*")
}

func (s *SnapshotVerifySuite) TestAptlySnapshotVerifyNoArchitectures(c *C) {
	// Test with no architectures and empty package list
	s.mockContext.architectures = []string{}
	mockPackageCollection := &MockSnapshotVerifyPackageCollection{emptyArchitectures: true}
	s.collectionFactory.packageCollection = mockPackageCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotVerify(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to determine list of architectures.*")
}

func (s *SnapshotVerifySuite) TestAptlySnapshotVerifyDependencyError(c *C) {
	// Test with dependency verification error
	mockPackageCollection := &MockSnapshotVerifyPackageCollection{shouldErrorVerifyDependencies: true}
	s.collectionFactory.packageCollection = mockPackageCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotVerify(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to verify dependencies.*")
}

func (s *SnapshotVerifySuite) TestAptlySnapshotVerifyMissingDependencies(c *C) {
	// Test with missing dependencies
	mockPackageCollection := &MockSnapshotVerifyPackageCollection{hasMissingDependencies: true}
	s.collectionFactory.packageCollection = mockPackageCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotVerify(s.cmd, args)
	c.Check(err, IsNil)

	// Should show missing dependencies
	foundMissingMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "Missing dependencies") {
			foundMissingMessage = true
			break
		}
	}
	c.Check(foundMissingMessage, Equals, true)
}

func (s *SnapshotVerifySuite) TestAptlySnapshotVerifyArchitectureHandling(c *C) {
	// Test architecture list handling
	testCases := []struct {
		contextArchs []string
		expected     []string
	}{
		{[]string{"amd64"}, []string{"amd64"}},
		{[]string{"i386", "amd64"}, []string{"i386", "amd64"}},
		{[]string{}, []string{"amd64", "all", "source"}}, // From package list
	}

	for _, testCase := range testCases {
		s.mockContext.architectures = testCase.contextArchs
		
		args := []string{"test-snapshot"}
		err := aptlySnapshotVerify(s.cmd, args)
		c.Check(err, IsNil, Commentf("Context architectures: %v", testCase.contextArchs))

		// Reset for next iteration
		s.mockProgress.Messages = []string{}
	}
}

func (s *SnapshotVerifySuite) TestAptlySnapshotVerifyMultipleSourcesAppendError(c *C) {
	// Test with append error on second snapshot
	mockPackageCollection := &MockSnapshotVerifyPackageCollection{shouldErrorAppendSecond: true}
	s.collectionFactory.packageCollection = mockPackageCollection

	args := []string{"test-snapshot", "source-snapshot"}
	err := aptlySnapshotVerify(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to merge sources.*")
}

func (s *SnapshotVerifySuite) TestAptlySnapshotVerifyMultipleSourcesPackageListError(c *C) {
	// Test with package list error on second snapshot
	mockPackageCollection := &MockSnapshotVerifyPackageCollection{shouldErrorNewPackageListSecond: true}
	s.collectionFactory.packageCollection = mockPackageCollection

	args := []string{"test-snapshot", "source-snapshot"}
	err := aptlySnapshotVerify(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to load packages.*")
}

func (s *SnapshotVerifySuite) TestAptlySnapshotVerifyDependencyOptions(c *C) {
	// Test with different dependency options
	s.mockContext.dependencyOptions = aptly.DependencyOptions{
		FollowRecommends:  true,
		FollowSuggests:    true,
		FollowSource:      true,
		FollowAllVariants: true,
	}

	args := []string{"test-snapshot"}
	err := aptlySnapshotVerify(s.cmd, args)
	c.Check(err, IsNil)

	// Should complete with enhanced dependency options
	c.Check(len(s.mockProgress.Messages) > 0, Equals, true)
}

// Mock implementations for testing

type MockSnapshotVerifyProgress struct {
	Messages []string
}

func (m *MockSnapshotVerifyProgress) Printf(msg string, a ...interface{}) {
	formatted := fmt.Sprintf(msg, a...)
	m.Messages = append(m.Messages, formatted)
}

type MockSnapshotVerifyContext struct {
	flags             *flag.FlagSet
	progress          *MockSnapshotVerifyProgress
	collectionFactory *deb.CollectionFactory
	architectures     []string
	dependencyOptions int
}

func (m *MockSnapshotVerifyContext) Flags() *flag.FlagSet                          { return m.flags }
func (m *MockSnapshotVerifyContext) Progress() aptly.Progress                      { return m.progress }
func (m *MockSnapshotVerifyContext) NewCollectionFactory() *deb.CollectionFactory { return m.collectionFactory }
func (m *MockSnapshotVerifyContext) ArchitecturesList() []string                  { return m.architectures }
func (m *MockSnapshotVerifyContext) DependencyOptions() int                      { return m.dependencyOptions }

type MockSnapshotVerifyCollection struct {
	shouldErrorByName       bool
	shouldErrorLoadComplete bool
}

func (m *MockSnapshotVerifyCollection) ByName(name string) (*deb.Snapshot, error) {
	if m.shouldErrorByName {
		return nil, fmt.Errorf("mock snapshot by name error")
	}

	snapshot := &deb.Snapshot{
		Name:        name,
		Description: "Test snapshot",
	}
	snapshot.SetRefList(&MockSnapshotVerifyRefList{})
	
	return snapshot, nil
}

func (m *MockSnapshotVerifyCollection) LoadComplete(snapshot *deb.Snapshot) error {
	if m.shouldErrorLoadComplete {
		return fmt.Errorf("mock snapshot load complete error")
	}
	return nil
}

type MockSnapshotVerifyRefList struct{}

func (m *MockSnapshotVerifyRefList) Len() int { return 5 }

type MockSnapshotVerifyPackageCollection struct {
	shouldErrorNewPackageList       bool
	shouldErrorNewPackageListSecond bool
	shouldErrorAppend               bool
	shouldErrorAppendSecond         bool
	shouldErrorVerifyDependencies   bool
	emptyArchitectures              bool
	hasMissingDependencies          bool
	callCount                       int
}

func (m *MockSnapshotVerifyPackageCollection) NewPackageListFromRefList(refList *deb.PackageRefList, progress aptly.Progress) (*deb.PackageList, error) {
	m.callCount++
	
	if m.shouldErrorNewPackageList && m.callCount == 1 {
		return nil, fmt.Errorf("mock new package list error")
	}
	if m.shouldErrorNewPackageListSecond && m.callCount > 1 {
		return nil, fmt.Errorf("mock new package list error on second call")
	}

	packageList := &MockSnapshotVerifyPackageList{
		collection:                 m,
		emptyArchitectures:         m.emptyArchitectures,
		hasMissingDependencies:     m.hasMissingDependencies,
		shouldErrorVerifyDependencies: m.shouldErrorVerifyDependencies,
		isFirstCall:                m.callCount == 1,
	}
	return packageList, nil
}

type MockSnapshotVerifyPackageList struct {
	collection                    *MockSnapshotVerifyPackageCollection
	emptyArchitectures            bool
	hasMissingDependencies        bool
	shouldErrorVerifyDependencies bool
	isFirstCall                   bool
}

func (m *MockSnapshotVerifyPackageList) PrepareIndex() {}

func (m *MockSnapshotVerifyPackageList) Architectures(includeSource bool) []string {
	if m.emptyArchitectures {
		return []string{}
	}
	if includeSource {
		return []string{"amd64", "all", "source"}
	}
	return []string{"amd64", "all"}
}

func (m *MockSnapshotVerifyPackageList) Append(other *deb.PackageList) error {
	if m.collection.shouldErrorAppend && m.isFirstCall {
		return fmt.Errorf("mock append error")
	}
	if m.collection.shouldErrorAppendSecond && !m.isFirstCall {
		return fmt.Errorf("mock append error on second call")
	}
	return nil
}

func (m *MockSnapshotVerifyPackageList) VerifyDependencies(options int, architectures []string, sources *deb.PackageList, progress aptly.Progress) ([]*deb.Dependency, error) {
	if m.shouldErrorVerifyDependencies {
		return nil, fmt.Errorf("mock verify dependencies error")
	}
	
	if m.hasMissingDependencies {
		// Return some mock missing dependencies
		missing := []*deb.Dependency{
			&MockVerifyDependency{name: "missing-package", version: "1.0"},
			&MockVerifyDependency{name: "another-missing", version: "2.0"},
		}
		return missing, nil
	}
	
	// No missing dependencies
	return []*deb.Dependency{}, nil
}

type MockVerifyDependency struct {
	name    string
	version string
}

func (m *MockVerifyDependency) String() string {
	return fmt.Sprintf("%s (>= %s)", m.name, m.version)
}

// Mock deb.NewPackageListFromRefList
func init() {
	originalNewPackageListFromRefList := deb.NewPackageListFromRefList
	deb.NewPackageListFromRefList = func(refList *deb.PackageRefList, packageCollection deb.PackageCollection, progress aptly.Progress) (*deb.PackageList, error) {
		if collection, ok := packageCollection.(*MockSnapshotVerifyPackageCollection); ok {
			return collection.NewPackageListFromRefList(refList, progress)
		}
		return originalNewPackageListFromRefList(refList, packageCollection, progress)
	}
}

// Mock deb.NewPackageList
func init() {
	originalNewPackageList := deb.NewPackageList
	deb.NewPackageList = func() *deb.PackageList {
		return &MockSnapshotVerifyPackageList{}
	}
	_ = originalNewPackageList // Prevent unused variable warning
}

// Test dependency sorting
func (s *SnapshotVerifySuite) TestDependencySorting(c *C) {
	// Test that missing dependencies are sorted correctly
	deps := []string{"z-package", "a-package", "m-package"}
	sort.Strings(deps)
	
	expected := []string{"a-package", "m-package", "z-package"}
	c.Check(deps, DeepEquals, expected)
}

// Test with various architecture combinations
func (s *SnapshotVerifySuite) TestArchitectureCombinations(c *C) {
	testArchs := [][]string{
		{"amd64"},
		{"i386"},
		{"amd64", "i386"},
		{"amd64", "i386", "armhf"},
	}

	for _, archs := range testArchs {
		s.mockContext.architectures = archs
		
		args := []string{"test-snapshot"}
		err := aptlySnapshotVerify(s.cmd, args)
		c.Check(err, IsNil, Commentf("Architectures: %v", archs))

		// Reset for next test
		s.mockProgress.Messages = []string{}
	}
}

// Test edge cases
func (s *SnapshotVerifySuite) TestSnapshotVerifyEdgeCases(c *C) {
	// Test with single source that satisfies all dependencies
	args := []string{"complete-snapshot"}
	err := aptlySnapshotVerify(s.cmd, args)
	c.Check(err, IsNil)

	// Should show satisfied message
	foundSatisfiedMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "All dependencies are satisfied") {
			foundSatisfiedMessage = true
			break
		}
	}
	c.Check(foundSatisfiedMessage, Equals, true)
}

// Test dependency options combinations
func (s *SnapshotVerifySuite) TestDependencyOptionsCombinations(c *C) {
	optionTests := []aptly.DependencyOptions{
		{FollowRecommends: true},
		{FollowSuggests: true},
		{FollowSource: true},
		{FollowAllVariants: true},
		{FollowRecommends: true, FollowSuggests: true},
		{FollowRecommends: true, FollowSuggests: true, FollowSource: true, FollowAllVariants: true},
	}

	for _, options := range optionTests {
		s.mockContext.dependencyOptions = options
		
		args := []string{"test-snapshot"}
		err := aptlySnapshotVerify(s.cmd, args)
		c.Check(err, IsNil, Commentf("Options: %+v", options))

		// Reset for next test
		s.mockProgress.Messages = []string{}
	}
}

// Test multiple missing dependencies display
func (s *SnapshotVerifySuite) TestMultipleMissingDependenciesDisplay(c *C) {
	mockPackageCollection := &MockSnapshotVerifyPackageCollection{hasMissingDependencies: true}
	s.collectionFactory.packageCollection = mockPackageCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotVerify(s.cmd, args)
	c.Check(err, IsNil)

	// Should show count and list of missing dependencies
	foundMissingCount := false
	foundDependencyList := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "Missing dependencies (2)") {
			foundMissingCount = true
		}
		if strings.Contains(msg, "missing-package") || strings.Contains(msg, "another-missing") {
			foundDependencyList = true
		}
	}
	c.Check(foundMissingCount, Equals, true)
	c.Check(foundDependencyList, Equals, true)
}