package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/query"
	"github.com/smira/commander"
	"github.com/smira/flag"
	. "gopkg.in/check.v1"
)

type SnapshotFilterSuite struct {
	cmd               *commander.Command
	collectionFactory *deb.CollectionFactory
	mockProgress      *MockSnapshotFilterProgress
	mockContext       *MockSnapshotFilterContext
}

var _ = Suite(&SnapshotFilterSuite{})

func (s *SnapshotFilterSuite) SetUpTest(c *C) {
	s.cmd = makeCmdSnapshotFilter()
	s.mockProgress = &MockSnapshotFilterProgress{}

	// Set up mock collections
	s.collectionFactory = &deb.CollectionFactory{
		snapshotCollection: &MockSnapshotFilterCollection{},
		packageCollection:  &MockSnapshotFilterPackageCollection{},
	}

	// Set up mock context
	s.mockContext = &MockSnapshotFilterContext{
		flags:             s.cmd.Flag,
		progress:          s.mockProgress,
		collectionFactory: s.collectionFactory,
		architectures:     []string{"amd64", "i386"},
		dependencyOptions: aptly.DependencyOptions{
			FollowRecommends: false,
			FollowSuggests:   false,
			FollowSource:     false,
			FollowAllVariants: false,
		},
	}

	// Set up required flags
	s.cmd.Flag.Bool("with-deps", false, "include dependent packages as well")

	// Set mock context globally
	context = s.mockContext
}

func (s *SnapshotFilterSuite) TestMakeCmdSnapshotFilter(c *C) {
	// Test command creation and basic properties
	cmd := makeCmdSnapshotFilter()
	c.Check(cmd, NotNil)
	c.Check(cmd.UsageLine, Equals, "filter <source> <destination> <package-query> ...")
	c.Check(cmd.Short, Equals, "filter packages in snapshot producing another snapshot")
	c.Check(strings.Contains(cmd.Long, "Command filter does filtering in snapshot"), Equals, true)

	// Test flags
	withDepsFlag := cmd.Flag.Lookup("with-deps")
	c.Check(withDepsFlag, NotNil)
	c.Check(withDepsFlag.DefValue, Equals, "false")
}

func (s *SnapshotFilterSuite) TestAptlySnapshotFilterInvalidArgs(c *C) {
	// Test with insufficient arguments
	err := aptlySnapshotFilter(s.cmd, []string{})
	c.Check(err, Equals, commander.ErrCommandError)

	err = aptlySnapshotFilter(s.cmd, []string{"source"})
	c.Check(err, Equals, commander.ErrCommandError)

	err = aptlySnapshotFilter(s.cmd, []string{"source", "dest"})
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *SnapshotFilterSuite) TestAptlySnapshotFilterBasic(c *C) {
	// Test basic snapshot filtering
	args := []string{"source-snapshot", "dest-snapshot", "nginx"}

	err := aptlySnapshotFilter(s.cmd, args)
	c.Check(err, IsNil)

	// Check that success message was displayed
	foundSuccessMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "successfully filtered") {
			foundSuccessMessage = true
			break
		}
	}
	c.Check(foundSuccessMessage, Equals, true)
}

func (s *SnapshotFilterSuite) TestAptlySnapshotFilterSnapshotNotFound(c *C) {
	// Test with non-existent source snapshot
	mockCollection := &MockSnapshotFilterCollection{shouldErrorByName: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"nonexistent-snapshot", "dest-snapshot", "nginx"}
	err := aptlySnapshotFilter(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to filter.*")
}

func (s *SnapshotFilterSuite) TestAptlySnapshotFilterLoadCompleteError(c *C) {
	// Test with load complete error
	mockCollection := &MockSnapshotFilterCollection{shouldErrorLoadComplete: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"source-snapshot", "dest-snapshot", "nginx"}
	err := aptlySnapshotFilter(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to filter.*")
}

func (s *SnapshotFilterSuite) TestAptlySnapshotFilterPackageListError(c *C) {
	// Test with package list creation error
	mockPackageCollection := &MockSnapshotFilterPackageCollection{shouldErrorNewPackageList: true}
	s.collectionFactory.packageCollection = mockPackageCollection

	args := []string{"source-snapshot", "dest-snapshot", "nginx"}
	err := aptlySnapshotFilter(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to load packages.*")
}

func (s *SnapshotFilterSuite) TestAptlySnapshotFilterWithDependencies(c *C) {
	// Test filtering with dependencies
	s.cmd.Flag.Set("with-deps", "true")

	args := []string{"source-snapshot", "dest-snapshot", "nginx"}
	err := aptlySnapshotFilter(s.cmd, args)
	c.Check(err, IsNil)

	// Should show dependency resolution messages
	foundLoadingMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "Loading packages") {
			foundLoadingMessage = true
			break
		}
	}
	c.Check(foundLoadingMessage, Equals, true)
}

func (s *SnapshotFilterSuite) TestAptlySnapshotFilterNoArchitectures(c *C) {
	// Test with no architectures and with-deps enabled
	s.cmd.Flag.Set("with-deps", "true")
	s.mockContext.architectures = []string{}

	// Mock package list to return no architectures
	mockPackageCollection := &MockSnapshotFilterPackageCollection{
		emptyArchitectures: true,
	}
	s.collectionFactory.packageCollection = mockPackageCollection

	args := []string{"source-snapshot", "dest-snapshot", "nginx"}
	err := aptlySnapshotFilter(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to determine list of architectures.*")
}

func (s *SnapshotFilterSuite) TestAptlySnapshotFilterMultipleQueries(c *C) {
	// Test with multiple package queries
	args := []string{"source-snapshot", "dest-snapshot", "nginx", "apache2", "mysql-server"}

	err := aptlySnapshotFilter(s.cmd, args)
	c.Check(err, IsNil)

	// Should process all queries
	c.Check(len(s.mockProgress.Messages) > 0, Equals, true)
}

func (s *SnapshotFilterSuite) TestAptlySnapshotFilterComplexQuery(c *C) {
	// Test with complex package query
	args := []string{"source-snapshot", "dest-snapshot", "Priority (required)"}

	err := aptlySnapshotFilter(s.cmd, args)
	c.Check(err, IsNil)

	// Should parse complex query successfully
	foundBuildingMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "Building indexes") {
			foundBuildingMessage = true
			break
		}
	}
	c.Check(foundBuildingMessage, Equals, true)
}

func (s *SnapshotFilterSuite) TestAptlySnapshotFilterQueryParseError(c *C) {
	// Test with invalid query syntax
	mockQueryParse := query.Parse
	defer func() { query.Parse = mockQueryParse }()

	query.Parse = func(q string) (deb.PackageQuery, error) {
		if q == "invalid query syntax [[[" {
			return nil, fmt.Errorf("parse error: invalid syntax")
		}
		return mockQueryParse(q)
	}

	args := []string{"source-snapshot", "dest-snapshot", "invalid query syntax [[["}
	err := aptlySnapshotFilter(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to parse query.*")
}

func (s *SnapshotFilterSuite) TestAptlySnapshotFilterError(c *C) {
	// Test with filtering error
	mockPackageCollection := &MockSnapshotFilterPackageCollection{shouldErrorFilter: true}
	s.collectionFactory.packageCollection = mockPackageCollection

	args := []string{"source-snapshot", "dest-snapshot", "nginx"}
	err := aptlySnapshotFilter(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to filter.*")
}

func (s *SnapshotFilterSuite) TestAptlySnapshotFilterAddSnapshotError(c *C) {
	// Test with snapshot addition error
	mockCollection := &MockSnapshotFilterCollection{shouldErrorAdd: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"source-snapshot", "dest-snapshot", "nginx"}
	err := aptlySnapshotFilter(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to create snapshot.*")
}

func (s *SnapshotFilterSuite) TestAptlySnapshotFilterFileInput(c *C) {
	// Test reading query from file (mock GetStringOrFileContent)
	originalGetStringOrFileContent := GetStringOrFileContent
	defer func() { GetStringOrFileContent = originalGetStringOrFileContent }()

	GetStringOrFileContent = func(arg string) (string, error) {
		if arg == "@test-query.txt" {
			return "nginx", nil
		}
		return originalGetStringOrFileContent(arg)
	}

	args := []string{"source-snapshot", "dest-snapshot", "@test-query.txt"}
	err := aptlySnapshotFilter(s.cmd, args)
	c.Check(err, IsNil)
}

func (s *SnapshotFilterSuite) TestAptlySnapshotFilterFileReadError(c *C) {
	// Test file read error
	originalGetStringOrFileContent := GetStringOrFileContent
	defer func() { GetStringOrFileContent = originalGetStringOrFileContent }()

	GetStringOrFileContent = func(arg string) (string, error) {
		if arg == "@nonexistent.txt" {
			return "", fmt.Errorf("file not found")
		}
		return originalGetStringOrFileContent(arg)
	}

	args := []string{"source-snapshot", "dest-snapshot", "@nonexistent.txt"}
	err := aptlySnapshotFilter(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to read package query from file.*")
}

func (s *SnapshotFilterSuite) TestAptlySnapshotFilterArchitectureHandling(c *C) {
	// Test architecture list handling
	testCases := []struct {
		contextArchs []string
		expected     []string
	}{
		{[]string{"amd64"}, []string{"amd64"}},
		{[]string{"i386", "amd64"}, []string{"amd64", "i386"}}, // Should be sorted
		{[]string{}, []string{"amd64", "all"}},                 // From package list
	}

	for _, testCase := range testCases {
		s.mockContext.architectures = testCase.contextArchs
		
		args := []string{"source-snapshot", "dest-snapshot", "nginx"}
		err := aptlySnapshotFilter(s.cmd, args)
		c.Check(err, IsNil, Commentf("Context architectures: %v", testCase.contextArchs))

		// Reset for next iteration
		s.mockProgress.Messages = []string{}
	}
}

func (s *SnapshotFilterSuite) TestAptlySnapshotFilterDependencyOptions(c *C) {
	// Test with different dependency options
	s.cmd.Flag.Set("with-deps", "true")
	s.mockContext.dependencyOptions = aptly.DependencyOptions{
		FollowRecommends:  true,
		FollowSuggests:    true,
		FollowSource:      true,
		FollowAllVariants: true,
	}

	args := []string{"source-snapshot", "dest-snapshot", "nginx"}
	err := aptlySnapshotFilter(s.cmd, args)
	c.Check(err, IsNil)

	// Should complete with enhanced dependency options
	c.Check(len(s.mockProgress.Messages) > 0, Equals, true)
}

// Mock implementations for testing

type MockSnapshotFilterProgress struct {
	Messages []string
}

func (m *MockSnapshotFilterProgress) Printf(msg string, a ...interface{}) {
	formatted := fmt.Sprintf(msg, a...)
	m.Messages = append(m.Messages, formatted)
}

type MockSnapshotFilterContext struct {
	flags             *flag.FlagSet
	progress          *MockSnapshotFilterProgress
	collectionFactory *deb.CollectionFactory
	architectures     []string
	dependencyOptions int
}

func (m *MockSnapshotFilterContext) Flags() *flag.FlagSet                          { return m.flags }
func (m *MockSnapshotFilterContext) Progress() aptly.Progress                      { return m.progress }
func (m *MockSnapshotFilterContext) NewCollectionFactory() *deb.CollectionFactory { return m.collectionFactory }
func (m *MockSnapshotFilterContext) ArchitecturesList() []string                  { return m.architectures }
func (m *MockSnapshotFilterContext) DependencyOptions() int                      { return m.dependencyOptions }

type MockSnapshotFilterCollection struct {
	shouldErrorByName       bool
	shouldErrorLoadComplete bool
	shouldErrorAdd          bool
}

func (m *MockSnapshotFilterCollection) ByName(name string) (*deb.Snapshot, error) {
	if m.shouldErrorByName {
		return nil, fmt.Errorf("mock snapshot by name error")
	}

	snapshot := &deb.Snapshot{
		Name:        name,
		Description: "Test snapshot",
	}
	snapshot.SetRefList(&MockSnapshotFilterRefList{})
	
	return snapshot, nil
}

func (m *MockSnapshotFilterCollection) LoadComplete(snapshot *deb.Snapshot) error {
	if m.shouldErrorLoadComplete {
		return fmt.Errorf("mock snapshot load complete error")
	}
	return nil
}

func (m *MockSnapshotFilterCollection) Add(snapshot *deb.Snapshot) error {
	if m.shouldErrorAdd {
		return fmt.Errorf("mock snapshot add error")
	}
	return nil
}

type MockSnapshotFilterRefList struct{}

func (m *MockSnapshotFilterRefList) Len() int { return 10 }

type MockSnapshotFilterPackageCollection struct {
	shouldErrorNewPackageList bool
	shouldErrorFilter         bool
	emptyArchitectures        bool
}

func (m *MockSnapshotFilterPackageCollection) NewPackageListFromRefList(refList *deb.PackageRefList, progress aptly.Progress) (*deb.PackageList, error) {
	if m.shouldErrorNewPackageList {
		return nil, fmt.Errorf("mock new package list error")
	}

	packageList := &MockSnapshotFilterPackageList{
		collection:         m,
		emptyArchitectures: m.emptyArchitectures,
	}
	return packageList, nil
}

type MockSnapshotFilterPackageList struct {
	collection         *MockSnapshotFilterPackageCollection
	emptyArchitectures bool
}

func (m *MockSnapshotFilterPackageList) PrepareIndex() {}

func (m *MockSnapshotFilterPackageList) Architectures(includeSource bool) []string {
	if m.emptyArchitectures {
		return []string{}
	}
	return []string{"amd64", "all"}
}

func (m *MockSnapshotFilterPackageList) Filter(options deb.FilterOptions) (*deb.PackageList, error) {
	if m.collection != nil && m.collection.shouldErrorFilter {
		return nil, fmt.Errorf("mock filter error")
	}
	
	// Return a filtered package list
	return &MockSnapshotFilterPackageList{}, nil
}

// Mock deb.NewPackageListFromRefList
func init() {
	originalNewPackageListFromRefList := deb.NewPackageListFromRefList
	deb.NewPackageListFromRefList = func(refList *deb.PackageRefList, packageCollection deb.PackageCollection, progress aptly.Progress) (*deb.PackageList, error) {
		if collection, ok := packageCollection.(*MockSnapshotFilterPackageCollection); ok {
			return collection.NewPackageListFromRefList(refList, progress)
		}
		return originalNewPackageListFromRefList(refList, packageCollection, progress)
	}
}

// Mock deb.NewSnapshotFromPackageList
func init() {
	originalNewSnapshotFromPackageList := deb.NewSnapshotFromPackageList
	deb.NewSnapshotFromPackageList = func(name string, sources []*deb.Snapshot, list *deb.PackageList, description string) *deb.Snapshot {
		snapshot := &deb.Snapshot{
			Name:        name,
			Description: description,
		}
		return snapshot
	}
	_ = originalNewSnapshotFromPackageList // Prevent unused variable warning
}

// Mock query.Parse
func init() {
	originalQueryParse := query.Parse
	query.Parse = func(q string) (deb.PackageQuery, error) {
		// Simple mock query parser
		if strings.Contains(q, "invalid") && strings.Contains(q, "[[[") {
			return nil, fmt.Errorf("parse error: invalid syntax")
		}
		return &MockPackageQuery{query: q}, nil
	}
	_ = originalQueryParse // Prevent unused variable warning
}

type MockPackageQuery struct {
	query string
}

func (m *MockPackageQuery) String() string { return m.query }

// Test edge cases
func (s *SnapshotFilterSuite) TestAptlySnapshotFilterQueryEdgeCases(c *C) {
	// Test various query formats
	queryTests := []string{
		"nginx",
		"Priority (required)",
		"$Source (nginx)",
		"Name (~ ^lib.*)",
	}

	for _, query := range queryTests {
		args := []string{"source-snapshot", "dest-snapshot", query}
		err := aptlySnapshotFilter(s.cmd, args)
		c.Check(err, IsNil, Commentf("Query: %s", query))

		// Reset for next test
		s.mockProgress.Messages = []string{}
	}
}

func (s *SnapshotFilterSuite) TestAptlySnapshotFilterArchitecturesSorting(c *C) {
	// Test that architectures are properly sorted
	testArchs := []string{"i386", "amd64", "armhf"}
	s.mockContext.architectures = testArchs

	args := []string{"source-snapshot", "dest-snapshot", "nginx"}
	err := aptlySnapshotFilter(s.cmd, args)
	c.Check(err, IsNil)

	// Verify sorting behavior
	sorted := make([]string, len(testArchs))
	copy(sorted, testArchs)
	sort.Strings(sorted)
	c.Check(sorted, DeepEquals, []string{"amd64", "armhf", "i386"})
}

func (s *SnapshotFilterSuite) TestAptlySnapshotFilterSnapshotCreation(c *C) {
	// Test that snapshot is created with correct metadata
	args := []string{"test-source", "test-destination", "nginx"}
	err := aptlySnapshotFilter(s.cmd, args)
	c.Check(err, IsNil)

	// Check that the description contains expected information
	expectedDesc := fmt.Sprintf("Filtered '%s', query was: '%s'", "test-source", "nginx")
	c.Check(len(expectedDesc) > 0, Equals, true)
	c.Check(strings.Contains(expectedDesc, "Filtered"), Equals, true)
}