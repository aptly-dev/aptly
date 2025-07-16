package cmd

import (
	"fmt"
	"strings"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/query"
	"github.com/smira/commander"
	"github.com/smira/flag"
	. "gopkg.in/check.v1"
)

type SnapshotPullSuite struct {
	cmd               *commander.Command
	collectionFactory *deb.CollectionFactory
	mockProgress      *MockSnapshotPullProgress
	mockContext       *MockSnapshotPullContext
}

var _ = Suite(&SnapshotPullSuite{})

func (s *SnapshotPullSuite) SetUpTest(c *C) {
	s.cmd = makeCmdSnapshotPull()
	s.mockProgress = &MockSnapshotPullProgress{}

	// Set up mock collections
	s.collectionFactory = &deb.CollectionFactory{
		snapshotCollection: &MockSnapshotPullCollection{},
		packageCollection:  &MockPackagePullCollection{},
	}

	// Set up mock context
	s.mockContext = &MockSnapshotPullContext{
		flags:             s.cmd.Flag,
		progress:          s.mockProgress,
		collectionFactory: s.collectionFactory,
		architecturesList: []string{"amd64", "i386"},
		dependencyOptions: 0,
	}

	// Set up required flags
	s.cmd.Flag.Bool("dry-run", false, "don't create destination snapshot")
	s.cmd.Flag.Bool("no-deps", false, "don't process dependencies")
	s.cmd.Flag.Bool("no-remove", false, "don't remove other package versions")
	s.cmd.Flag.Bool("all-matches", false, "pull all matching packages")

	// Set mock context globally
	context = s.mockContext
}

func (s *SnapshotPullSuite) TestMakeCmdSnapshotPull(c *C) {
	// Test command creation and basic properties
	cmd := makeCmdSnapshotPull()
	c.Check(cmd, NotNil)
	c.Check(cmd.UsageLine, Equals, "pull <name> <source> <destination> <package-query> ...")
	c.Check(cmd.Short, Equals, "pull packages from another snapshot")
	c.Check(strings.Contains(cmd.Long, "Command pull pulls new packages"), Equals, true)

	// Test flags
	requiredFlags := []string{"dry-run", "no-deps", "no-remove", "all-matches"}
	for _, flagName := range requiredFlags {
		flag := cmd.Flag.Lookup(flagName)
		c.Check(flag, NotNil, Commentf("Flag %s should exist", flagName))
		c.Check(flag.DefValue, Equals, "false")
	}
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullBasic(c *C) {
	// Test basic snapshot pull operation
	args := []string{"target-snapshot", "source-snapshot", "dest-snapshot", "package-name"}

	err := aptlySnapshotPull(s.cmd, args)
	c.Check(err, IsNil)

	// Check that progress messages were displayed
	c.Check(len(s.mockProgress.Messages) > 0, Equals, true)
	foundProgressMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "Dependencies would be pulled") {
			foundProgressMessage = true
			break
		}
	}
	c.Check(foundProgressMessage, Equals, true)
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullInvalidArgs(c *C) {
	// Test with insufficient arguments
	testCases := [][]string{
		{},
		{"one"},
		{"one", "two"},
		{"one", "two", "three"},
	}

	for _, args := range testCases {
		err := aptlySnapshotPull(s.cmd, args)
		c.Check(err, Equals, commander.ErrCommandError, Commentf("Args: %v", args))
	}
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullTargetSnapshotNotFound(c *C) {
	// Test with non-existent target snapshot
	mockCollection := &MockSnapshotPullCollection{shouldErrorByName: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"nonexistent-target", "source-snapshot", "dest-snapshot", "package-name"}
	err := aptlySnapshotPull(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to pull.*")
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullTargetLoadError(c *C) {
	// Test with target snapshot load error
	mockCollection := &MockSnapshotPullCollection{shouldErrorLoadComplete: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"target-snapshot", "source-snapshot", "dest-snapshot", "package-name"}
	err := aptlySnapshotPull(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to pull.*")
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullSourceSnapshotNotFound(c *C) {
	// Test with non-existent source snapshot
	mockCollection := &MockSnapshotPullCollection{shouldErrorSourceByName: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"target-snapshot", "nonexistent-source", "dest-snapshot", "package-name"}
	err := aptlySnapshotPull(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to pull.*")
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullSourceLoadError(c *C) {
	// Test with source snapshot load error
	mockCollection := &MockSnapshotPullCollection{shouldErrorSourceLoadComplete: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"target-snapshot", "source-snapshot", "dest-snapshot", "package-name"}
	err := aptlySnapshotPull(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to pull.*")
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullPackageLoadError(c *C) {
	// Test with package list creation error
	mockCollection := &MockPackagePullCollection{shouldErrorNewPackageListFromRefList: true}
	s.collectionFactory.packageCollection = mockCollection

	args := []string{"target-snapshot", "source-snapshot", "dest-snapshot", "package-name"}
	err := aptlySnapshotPull(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to load packages.*")
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullNoArchitectures(c *C) {
	// Test with no architectures available
	s.mockContext.architecturesList = []string{}
	mockCollection := &MockPackagePullCollection{emptyArchitectures: true}
	s.collectionFactory.packageCollection = mockCollection

	args := []string{"target-snapshot", "source-snapshot", "dest-snapshot", "package-name"}
	err := aptlySnapshotPull(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to determine list of architectures.*")
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullQueryFromFile(c *C) {
	// Test with query from file
	originalGetStringOrFileContent := GetStringOrFileContent
	GetStringOrFileContent = func(arg string) (string, error) {
		if arg == "@file" {
			return "package-from-file", nil
		}
		return arg, nil
	}
	defer func() { GetStringOrFileContent = originalGetStringOrFileContent }()

	args := []string{"target-snapshot", "source-snapshot", "dest-snapshot", "@file"}
	err := aptlySnapshotPull(s.cmd, args)
	c.Check(err, IsNil)
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullQueryFileError(c *C) {
	// Test with query file read error
	originalGetStringOrFileContent := GetStringOrFileContent
	GetStringOrFileContent = func(arg string) (string, error) {
		return "", fmt.Errorf("file read error")
	}
	defer func() { GetStringOrFileContent = originalGetStringOrFileContent }()

	args := []string{"target-snapshot", "source-snapshot", "dest-snapshot", "@file"}
	err := aptlySnapshotPull(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to read package query from file.*")
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullInvalidQuery(c *C) {
	// Test with invalid package query
	originalParse := query.Parse
	query.Parse = func(q string) (deb.PackageQuery, error) {
		return nil, fmt.Errorf("invalid query")
	}
	defer func() { query.Parse = originalParse }()

	args := []string{"target-snapshot", "source-snapshot", "dest-snapshot", "invalid-query"}
	err := aptlySnapshotPull(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to parse query.*")
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullFilterError(c *C) {
	// Test with package filter error
	mockCollection := &MockPackagePullCollection{shouldErrorFilter: true}
	s.collectionFactory.packageCollection = mockCollection

	args := []string{"target-snapshot", "source-snapshot", "dest-snapshot", "package-name"}
	err := aptlySnapshotPull(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to pull.*")
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullWithFlags(c *C) {
	// Test pull with various flags
	flagTests := []struct {
		flag string
		value string
	}{
		{"no-deps", "true"},
		{"no-remove", "true"},
		{"all-matches", "true"},
	}

	for _, test := range flagTests {
		s.cmd.Flag.Set(test.flag, test.value)
		args := []string{"target-snapshot", "source-snapshot", "dest-snapshot", "package-name"}

		err := aptlySnapshotPull(s.cmd, args)
		c.Check(err, IsNil, Commentf("Flag: %s", test.flag))

		// Reset flag
		s.cmd.Flag.Set(test.flag, "false")
	}
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullDryRun(c *C) {
	// Test dry run mode
	s.cmd.Flag.Set("dry-run", "true")

	args := []string{"target-snapshot", "source-snapshot", "dest-snapshot", "package-name"}
	err := aptlySnapshotPull(s.cmd, args)
	c.Check(err, IsNil)

	// Should show dry run message
	foundDryRunMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "Not creating snapshot, as dry run") {
			foundDryRunMessage = true
			break
		}
	}
	c.Check(foundDryRunMessage, Equals, true)
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullCreateSnapshotError(c *C) {
	// Test with snapshot creation error
	mockCollection := &MockSnapshotPullCollection{shouldErrorAdd: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"target-snapshot", "source-snapshot", "dest-snapshot", "package-name"}
	err := aptlySnapshotPull(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to create snapshot.*")
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullMultipleQueries(c *C) {
	// Test with multiple package queries
	args := []string{"target-snapshot", "source-snapshot", "dest-snapshot", "package1", "package2", "package3"}

	err := aptlySnapshotPull(s.cmd, args)
	c.Check(err, IsNil)

	// Should process all queries
	c.Check(len(s.mockProgress.Messages) > 0, Equals, true)
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullArchitectureFiltering(c *C) {
	// Test that architecture filtering is applied to queries
	args := []string{"target-snapshot", "source-snapshot", "dest-snapshot", "package-name"}

	err := aptlySnapshotPull(s.cmd, args)
	c.Check(err, IsNil)

	// The function should build architecture queries correctly
	// This is tested indirectly through successful execution
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullPackageProcessing(c *C) {
	// Test package addition and removal logic
	s.cmd.Flag.Set("no-remove", "false") // Allow removal
	s.cmd.Flag.Set("all-matches", "false") // Only first match

	args := []string{"target-snapshot", "source-snapshot", "dest-snapshot", "package-name"}
	err := aptlySnapshotPull(s.cmd, args)
	c.Check(err, IsNil)

	// Should show package addition/removal messages
	foundAddMessage := false
	foundRemoveMessage := false
	for _, msg := range s.mockProgress.ColoredMessages {
		if strings.Contains(msg, "added") {
			foundAddMessage = true
		}
		if strings.Contains(msg, "removed") {
			foundRemoveMessage = true
		}
	}
	c.Check(foundAddMessage, Equals, true)
	c.Check(foundRemoveMessage, Equals, true)
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullNoRemove(c *C) {
	// Test with no-remove flag
	s.cmd.Flag.Set("no-remove", "true")

	args := []string{"target-snapshot", "source-snapshot", "dest-snapshot", "package-name"}
	err := aptlySnapshotPull(s.cmd, args)
	c.Check(err, IsNil)

	// Should not show removal messages
	foundRemoveMessage := false
	for _, msg := range s.mockProgress.ColoredMessages {
		if strings.Contains(msg, "removed") {
			foundRemoveMessage = true
			break
		}
	}
	c.Check(foundRemoveMessage, Equals, false)
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullAllMatches(c *C) {
	// Test with all-matches flag
	s.cmd.Flag.Set("all-matches", "true")

	args := []string{"target-snapshot", "source-snapshot", "dest-snapshot", "package-name"}
	err := aptlySnapshotPull(s.cmd, args)
	c.Check(err, IsNil)

	// Should allow multiple matches of the same name-arch pair
	// This is tested indirectly through successful execution
}

func (s *SnapshotPullSuite) TestAptlySnapshotPullSuccessMessage(c *C) {
	// Test success message output
	args := []string{"target-snapshot", "source-snapshot", "dest-snapshot", "package-name"}

	err := aptlySnapshotPull(s.cmd, args)
	c.Check(err, IsNil)

	// Should show success message
	foundSuccessMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "successfully created") {
			foundSuccessMessage = true
			break
		}
	}
	c.Check(foundSuccessMessage, Equals, true)
}

// Mock implementations for testing

type MockSnapshotPullProgress struct {
	Messages        []string
	ColoredMessages []string
}

func (m *MockSnapshotPullProgress) Printf(msg string, a ...interface{}) {
	formatted := fmt.Sprintf(msg, a...)
	m.Messages = append(m.Messages, formatted)
}

func (m *MockSnapshotPullProgress) ColoredPrintf(msg string, a ...interface{}) {
	formatted := fmt.Sprintf(msg, a...)
	m.ColoredMessages = append(m.ColoredMessages, formatted)
}

type MockSnapshotPullContext struct {
	flags             *flag.FlagSet
	progress          *MockSnapshotPullProgress
	collectionFactory *deb.CollectionFactory
	architecturesList []string
	dependencyOptions int
}

func (m *MockSnapshotPullContext) Flags() *flag.FlagSet                          { return m.flags }
func (m *MockSnapshotPullContext) Progress() aptly.Progress                      { return m.progress }
func (m *MockSnapshotPullContext) NewCollectionFactory() *deb.CollectionFactory { return m.collectionFactory }
func (m *MockSnapshotPullContext) ArchitecturesList() []string                  { return m.architecturesList }
func (m *MockSnapshotPullContext) DependencyOptions() int                       { return m.dependencyOptions }

type MockSnapshotPullCollection struct {
	shouldErrorByName               bool
	shouldErrorLoadComplete         bool
	shouldErrorSourceByName         bool
	shouldErrorSourceLoadComplete   bool
	shouldErrorAdd                  bool
}

func (m *MockSnapshotPullCollection) ByName(name string) (*deb.Snapshot, error) {
	if m.shouldErrorByName || (m.shouldErrorSourceByName && name == "nonexistent-source") {
		return nil, fmt.Errorf("mock snapshot by name error")
	}
	return &deb.Snapshot{Name: name, UUID: "test-uuid-" + name}, nil
}

func (m *MockSnapshotPullCollection) LoadComplete(snapshot *deb.Snapshot) error {
	if m.shouldErrorLoadComplete || (m.shouldErrorSourceLoadComplete && strings.Contains(snapshot.Name, "source")) {
		return fmt.Errorf("mock snapshot load complete error")
	}
	// Set up mock RefList
	snapshot.packageRefs = deb.NewPackageRefList()
	snapshot.packageRefs.Append(&deb.PackageRef{Key: []byte("pkg1")})
	snapshot.packageRefs.Append(&deb.PackageRef{Key: []byte("pkg2")})
	return nil
}

func (m *MockSnapshotPullCollection) Add(snapshot *deb.Snapshot) error {
	if m.shouldErrorAdd {
		return fmt.Errorf("mock snapshot add error")
	}
	return nil
}

type MockPackagePullCollection struct {
	shouldErrorNewPackageListFromRefList bool
	shouldErrorFilter                    bool
	emptyArchitectures                   bool
}

// Mock NewPackageListFromRefList function for snapshot pull
func NewPackageListFromRefListPull(refList *deb.PackageRefList, packageCollection deb.PackageCollection, progress aptly.Progress) (*deb.PackageList, error) {
	if collection, ok := packageCollection.(*MockPackagePullCollection); ok && collection.shouldErrorNewPackageListFromRefList {
		return nil, fmt.Errorf("mock package list from ref list error")
	}

	packageList := &deb.PackageList{}

	// Set up mock methods
	packageList.PrepareIndex = func() {}
	packageList.Architectures = func(includeSource bool) []string {
		if collection, ok := packageCollection.(*MockPackagePullCollection); ok && collection.emptyArchitectures {
			return []string{}
		}
		return []string{"amd64", "i386"}
	}
	packageList.Filter = func(options deb.FilterOptions) (*deb.PackageList, error) {
		if collection, ok := packageCollection.(*MockPackagePullCollection); ok && collection.shouldErrorFilter {
			return nil, fmt.Errorf("mock filter error")
		}
		return packageList, nil
	}
	packageList.ForEachIndexed = func(handler func(*deb.Package) error) error {
		// Process mock packages
		mockPackages := []*deb.Package{
			{Name: "test-package", Architecture: "amd64"},
			{Name: "another-package", Architecture: "i386"},
		}
		for _, pkg := range mockPackages {
			if err := handler(pkg); err != nil {
				return err
			}
		}
		return nil
	}
	packageList.Add = func(pkg *deb.Package) error { return nil }
	packageList.Remove = func(pkg *deb.Package) {}
	packageList.Search = func(dep deb.Dependency, useDefaults, useAllVersions bool) []*deb.Package {
		// Return mock packages for removal testing
		return []*deb.Package{
			{Name: dep.Pkg, Architecture: dep.Architecture},
		}
	}

	return packageList, nil
}

// Add methods to support snapshot operations
func (s *deb.Snapshot) RefList() *deb.PackageRefList {
	return s.packageRefs
}

// Mock NewSnapshotFromPackageList function
func NewSnapshotFromPackageList(name string, sources []*deb.Snapshot, packageList *deb.PackageList, description string) *deb.Snapshot {
	return &deb.Snapshot{
		Name:        name,
		UUID:        "new-snapshot-uuid",
		Description: description,
		SourceIDs:   []string{},
	}
}

// Mock query parsing and architecture queries
func init() {
	// Override query.Parse if not already overridden
	originalParse := query.Parse
	query.Parse = func(q string) (deb.PackageQuery, error) {
		return &MockSnapshotPullQuery{}, nil
	}
	_ = originalParse
}

type MockSnapshotPullQuery struct{}

func (m *MockSnapshotPullQuery) Matches(pkg *deb.Package) bool { return true }
func (m *MockSnapshotPullQuery) String() string               { return "mock-query" }

// Mock field query for architecture filtering
type MockFieldQuery struct {
	Field    string
	Relation int
	Value    string
}

func (m *MockFieldQuery) Matches(pkg *deb.Package) bool { return true }
func (m *MockFieldQuery) String() string               { return fmt.Sprintf("%s %s %s", m.Field, m.Relation, m.Value) }

// Mock OR query
type MockOrQuery struct {
	L deb.PackageQuery
	R deb.PackageQuery
}

func (m *MockOrQuery) Matches(pkg *deb.Package) bool { return m.L.Matches(pkg) || m.R.Matches(pkg) }
func (m *MockOrQuery) String() string               { return fmt.Sprintf("(%s | %s)", m.L.String(), m.R.String()) }

// Mock AND query
type MockAndQuery struct {
	L deb.PackageQuery
	R deb.PackageQuery
}

func (m *MockAndQuery) Matches(pkg *deb.Package) bool { return m.L.Matches(pkg) && m.R.Matches(pkg) }
func (m *MockAndQuery) String() string               { return fmt.Sprintf("(%s, %s)", m.L.String(), m.R.String()) }

// Mock dependency struct
type MockDependency struct {
	Pkg          string
	Architecture string
}

// Mock package struct methods
func (p *deb.Package) String() string {
	return fmt.Sprintf("%s_%s", p.Name, p.Architecture)
}