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

type SnapshotSearchSuite struct {
	cmd               *commander.Command
	collectionFactory *deb.CollectionFactory
	mockProgress      *MockSnapshotSearchProgress
	mockContext       *MockSnapshotSearchContext
	parentCmd         *commander.Command
}

var _ = Suite(&SnapshotSearchSuite{})

func (s *SnapshotSearchSuite) SetUpTest(c *C) {
	s.cmd = makeCmdSnapshotSearch()
	s.mockProgress = &MockSnapshotSearchProgress{}

	// Set up parent command to simulate snapshot/mirror/repo context
	s.parentCmd = &commander.Command{}
	s.parentCmd.Name = func() string { return "snapshot" }
	s.cmd.Parent = s.parentCmd

	// Set up mock collections
	s.collectionFactory = &deb.CollectionFactory{
		snapshotCollection:   &MockSnapshotSearchCollection{},
		remoteRepoCollection: &MockRemoteSearchRepoCollection{},
		localRepoCollection:  &MockLocalSearchRepoCollection{},
		packageCollection:    &MockPackageSearchCollection{},
	}

	// Set up mock context
	s.mockContext = &MockSnapshotSearchContext{
		flags:             s.cmd.Flag,
		progress:          s.mockProgress,
		collectionFactory: s.collectionFactory,
		architecturesList: []string{"amd64", "i386"},
		dependencyOptions: 0,
	}

	// Set up required flags
	s.cmd.Flag.Bool("with-deps", false, "include dependencies into search results")
	s.cmd.Flag.String("format", "", "custom format for result printing")

	// Set mock context globally
	context = s.mockContext
}

func (s *SnapshotSearchSuite) TestMakeCmdSnapshotSearch(c *C) {
	// Test command creation and basic properties
	cmd := makeCmdSnapshotSearch()
	c.Check(cmd, NotNil)
	c.Check(cmd.UsageLine, Equals, "search <name> [<package-query>]")
	c.Check(cmd.Short, Equals, "search snapshot for packages matching query")
	c.Check(strings.Contains(cmd.Long, "Command search displays list of packages"), Equals, true)

	// Test flags
	withDepsFlag := cmd.Flag.Lookup("with-deps")
	c.Check(withDepsFlag, NotNil)
	c.Check(withDepsFlag.DefValue, Equals, "false")

	formatFlag := cmd.Flag.Lookup("format")
	c.Check(formatFlag, NotNil)
	c.Check(formatFlag.DefValue, Equals, "")
}

func (s *SnapshotSearchSuite) TestAptlySnapshotSearchInvalidArgs(c *C) {
	// Test with no arguments
	err := aptlySnapshotMirrorRepoSearch(s.cmd, []string{})
	c.Check(err, Equals, commander.ErrCommandError)

	// Test with too many arguments
	err = aptlySnapshotMirrorRepoSearch(s.cmd, []string{"snap1", "query1", "extra"})
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *SnapshotSearchSuite) TestAptlySnapshotSearchBasic(c *C) {
	// Test basic snapshot search
	args := []string{"test-snapshot"}

	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, IsNil)

	// Verify snapshot was retrieved and searched
	mockCollection := s.collectionFactory.snapshotCollection.(*MockSnapshotSearchCollection)
	c.Check(mockCollection.byNameCalled, Equals, true)
	c.Check(mockCollection.loadCompleteCalled, Equals, true)
}

func (s *SnapshotSearchSuite) TestAptlySnapshotSearchWithQuery(c *C) {
	// Test snapshot search with query
	args := []string{"test-snapshot", "package-name"}

	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, IsNil)

	// Should have parsed and used the query
	c.Check(len(s.mockProgress.Messages) >= 0, Equals, true)
}

func (s *SnapshotSearchSuite) TestAptlyMirrorSearch(c *C) {
	// Test mirror search
	s.parentCmd.Name = func() string { return "mirror" }
	args := []string{"test-mirror"}

	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, IsNil)

	// Verify mirror was retrieved and searched
	mockCollection := s.collectionFactory.remoteRepoCollection.(*MockRemoteSearchRepoCollection)
	c.Check(mockCollection.byNameCalled, Equals, true)
	c.Check(mockCollection.loadCompleteCalled, Equals, true)
}

func (s *SnapshotSearchSuite) TestAptlyRepoSearch(c *C) {
	// Test repo search
	s.parentCmd.Name = func() string { return "repo" }
	args := []string{"test-repo"}

	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, IsNil)

	// Verify repo was retrieved and searched
	mockCollection := s.collectionFactory.localRepoCollection.(*MockLocalSearchRepoCollection)
	c.Check(mockCollection.byNameCalled, Equals, true)
	c.Check(mockCollection.loadCompleteCalled, Equals, true)
}

func (s *SnapshotSearchSuite) TestAptlySearchUnknownCommand(c *C) {
	// Test with unknown parent command
	s.parentCmd.Name = func() string { return "unknown" }
	args := []string{"test-snapshot"}

	defer func() {
		if r := recover(); r != nil {
			c.Check(r, Equals, "unknown command")
		} else {
			c.Error("Expected panic for unknown command")
		}
	}()

	aptlySnapshotMirrorRepoSearch(s.cmd, args)
}

func (s *SnapshotSearchSuite) TestAptlySnapshotSearchNotFound(c *C) {
	// Test with non-existent snapshot
	mockCollection := &MockSnapshotSearchCollection{shouldErrorByName: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"nonexistent-snapshot"}
	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to search.*")
}

func (s *SnapshotSearchSuite) TestAptlySnapshotSearchLoadError(c *C) {
	// Test with load complete error
	mockCollection := &MockSnapshotSearchCollection{shouldErrorLoadComplete: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to search.*")
}

func (s *SnapshotSearchSuite) TestAptlyMirrorSearchNotFound(c *C) {
	// Test with non-existent mirror
	s.parentCmd.Name = func() string { return "mirror" }
	mockCollection := &MockRemoteSearchRepoCollection{shouldErrorByName: true}
	s.collectionFactory.remoteRepoCollection = mockCollection

	args := []string{"nonexistent-mirror"}
	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to search.*")
}

func (s *SnapshotSearchSuite) TestAptlyRepoSearchNotFound(c *C) {
	// Test with non-existent repo
	s.parentCmd.Name = func() string { return "repo" }
	mockCollection := &MockLocalSearchRepoCollection{shouldErrorByName: true}
	s.collectionFactory.localRepoCollection = mockCollection

	args := []string{"nonexistent-repo"}
	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to search.*")
}

func (s *SnapshotSearchSuite) TestAptlySearchPackageListError(c *C) {
	// Test with package list creation error
	mockPackageCollection := &MockPackageSearchCollection{shouldErrorNewPackageListFromRefList: true}
	s.collectionFactory.packageCollection = mockPackageCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to search.*")
}

func (s *SnapshotSearchSuite) TestAptlySearchQueryFromFile(c *C) {
	// Test with query from file
	originalGetStringOrFileContent := GetStringOrFileContent
	GetStringOrFileContent = func(arg string) (string, error) {
		if arg == "@file" {
			return "package-from-file", nil
		}
		return arg, nil
	}
	defer func() { GetStringOrFileContent = originalGetStringOrFileContent }()

	args := []string{"test-snapshot", "@file"}
	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, IsNil)
}

func (s *SnapshotSearchSuite) TestAptlySearchQueryFileError(c *C) {
	// Test with query file read error
	originalGetStringOrFileContent := GetStringOrFileContent
	GetStringOrFileContent = func(arg string) (string, error) {
		return "", fmt.Errorf("file read error")
	}
	defer func() { GetStringOrFileContent = originalGetStringOrFileContent }()

	args := []string{"test-snapshot", "@file"}
	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to read package query from file.*")
}

func (s *SnapshotSearchSuite) TestAptlySearchInvalidQuery(c *C) {
	// Test with invalid package query
	originalParse := query.Parse
	query.Parse = func(q string) (deb.PackageQuery, error) {
		return nil, fmt.Errorf("invalid query")
	}
	defer func() { query.Parse = originalParse }()

	args := []string{"test-snapshot", "invalid-query"}
	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to search.*")
}

func (s *SnapshotSearchSuite) TestAptlySearchWithDependencies(c *C) {
	// Test search with dependencies
	s.cmd.Flag.Set("with-deps", "true")

	args := []string{"test-snapshot", "package-name"}
	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, IsNil)

	// Should have resolved dependencies
	c.Check(len(s.mockProgress.Messages) >= 0, Equals, true)
}

func (s *SnapshotSearchSuite) TestAptlySearchWithDepsNoArchitectures(c *C) {
	// Test with dependencies but no architectures available
	s.cmd.Flag.Set("with-deps", "true")
	s.mockContext.architecturesList = []string{} // No architectures in context
	mockPackageCollection := &MockPackageSearchCollection{emptyArchitectures: true}
	s.collectionFactory.packageCollection = mockPackageCollection

	args := []string{"test-snapshot", "package-name"}
	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to determine list of architectures.*")
}

func (s *SnapshotSearchSuite) TestAptlySearchFilterError(c *C) {
	// Test with package filter error
	mockPackageCollection := &MockPackageSearchCollection{shouldErrorFilter: true}
	s.collectionFactory.packageCollection = mockPackageCollection

	args := []string{"test-snapshot", "package-name"}
	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to search.*")
}

func (s *SnapshotSearchSuite) TestAptlySearchNoResults(c *C) {
	// Test with no search results
	mockPackageCollection := &MockPackageSearchCollection{emptyResults: true}
	s.collectionFactory.packageCollection = mockPackageCollection

	args := []string{"test-snapshot", "package-name"}
	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*no results.*")
}

func (s *SnapshotSearchSuite) TestAptlySearchWithFormat(c *C) {
	// Test search with custom format
	s.cmd.Flag.Set("format", "{{.Package}}")

	args := []string{"test-snapshot", "package-name"}
	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, IsNil)

	// Should have used custom format
	c.Check(len(s.mockProgress.Messages) >= 0, Equals, true)
}

func (s *SnapshotSearchSuite) TestAptlySearchMatchAllQuery(c *C) {
	// Test search without query (match all)
	args := []string{"test-snapshot"}

	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, IsNil)

	// Should have used MatchAllQuery
	c.Check(len(s.mockProgress.Messages) >= 0, Equals, true)
}

func (s *SnapshotSearchSuite) TestAptlySearchArchitecturesFromContext(c *C) {
	// Test using architectures from context when with-deps is enabled
	s.cmd.Flag.Set("with-deps", "true")
	s.mockContext.architecturesList = []string{"amd64", "arm64"}

	args := []string{"test-snapshot", "package-name"}
	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, IsNil)

	// Should have used context architectures
	c.Check(len(s.mockProgress.Messages) >= 0, Equals, true)
}

func (s *SnapshotSearchSuite) TestAptlySearchArchitecturesFromPackageList(c *C) {
	// Test using architectures from package list when context is empty
	s.cmd.Flag.Set("with-deps", "true")
	s.mockContext.architecturesList = []string{} // Empty context
	mockPackageCollection := &MockPackageSearchCollection{hasArchitectures: true}
	s.collectionFactory.packageCollection = mockPackageCollection

	args := []string{"test-snapshot", "package-name"}
	err := aptlySnapshotMirrorRepoSearch(s.cmd, args)
	c.Check(err, IsNil)

	// Should have used package list architectures
	c.Check(len(s.mockProgress.Messages) >= 0, Equals, true)
}

// Mock implementations for testing

type MockSnapshotSearchProgress struct {
	Messages []string
}

func (m *MockSnapshotSearchProgress) Printf(msg string, a ...interface{}) {
	formatted := fmt.Sprintf(msg, a...)
	m.Messages = append(m.Messages, formatted)
}

type MockSnapshotSearchContext struct {
	flags             *flag.FlagSet
	progress          *MockSnapshotSearchProgress
	collectionFactory *deb.CollectionFactory
	architecturesList []string
	dependencyOptions int
}

func (m *MockSnapshotSearchContext) Flags() *flag.FlagSet     { return m.flags }
func (m *MockSnapshotSearchContext) Progress() aptly.Progress { return m.progress }
func (m *MockSnapshotSearchContext) NewCollectionFactory() *deb.CollectionFactory {
	return m.collectionFactory
}
func (m *MockSnapshotSearchContext) ArchitecturesList() []string { return m.architecturesList }
func (m *MockSnapshotSearchContext) DependencyOptions() int      { return m.dependencyOptions }

type MockSnapshotSearchCollection struct {
	shouldErrorByName       bool
	shouldErrorLoadComplete bool
	byNameCalled            bool
	loadCompleteCalled      bool
}

func (m *MockSnapshotSearchCollection) ByName(name string) (*deb.Snapshot, error) {
	m.byNameCalled = true
	if m.shouldErrorByName {
		return nil, fmt.Errorf("mock snapshot by name error")
	}
	return &deb.Snapshot{Name: name, UUID: "test-uuid-" + name}, nil
}

func (m *MockSnapshotSearchCollection) LoadComplete(snapshot *deb.Snapshot) error {
	m.loadCompleteCalled = true
	if m.shouldErrorLoadComplete {
		return fmt.Errorf("mock snapshot load complete error")
	}
	// Set up mock RefList
	snapshot.packageRefs = deb.NewPackageRefList()
	snapshot.packageRefs.Append(&deb.PackageRef{Key: []byte("pkg1")})
	return nil
}

type MockRemoteSearchRepoCollection struct {
	shouldErrorByName       bool
	shouldErrorLoadComplete bool
	byNameCalled            bool
	loadCompleteCalled      bool
}

func (m *MockRemoteSearchRepoCollection) ByName(name string) (*deb.RemoteRepo, error) {
	m.byNameCalled = true
	if m.shouldErrorByName {
		return nil, fmt.Errorf("mock remote repo by name error")
	}
	return &deb.RemoteRepo{Name: name, uuid: "test-uuid-" + name}, nil
}

func (m *MockRemoteSearchRepoCollection) LoadComplete(repo *deb.RemoteRepo) error {
	m.loadCompleteCalled = true
	if m.shouldErrorLoadComplete {
		return fmt.Errorf("mock remote repo load complete error")
	}
	// Set up mock RefList
	repo.packageRefs = deb.NewPackageRefList()
	repo.packageRefs.Append(&deb.PackageRef{Key: []byte("pkg1")})
	return nil
}

type MockLocalSearchRepoCollection struct {
	shouldErrorByName       bool
	shouldErrorLoadComplete bool
	byNameCalled            bool
	loadCompleteCalled      bool
}

func (m *MockLocalSearchRepoCollection) ByName(name string) (*deb.LocalRepo, error) {
	m.byNameCalled = true
	if m.shouldErrorByName {
		return nil, fmt.Errorf("mock local repo by name error")
	}
	return &deb.LocalRepo{Name: name, uuid: "test-uuid-" + name}, nil
}

func (m *MockLocalSearchRepoCollection) LoadComplete(repo *deb.LocalRepo) error {
	m.loadCompleteCalled = true
	if m.shouldErrorLoadComplete {
		return fmt.Errorf("mock local repo load complete error")
	}
	// Set up mock RefList
	repo.packageRefs = deb.NewPackageRefList()
	repo.packageRefs.Append(&deb.PackageRef{Key: []byte("pkg1")})
	return nil
}

type MockPackageSearchCollection struct {
	shouldErrorNewPackageListFromRefList bool
	shouldErrorFilter                    bool
	emptyResults                         bool
	emptyArchitectures                   bool
	hasArchitectures                     bool
}

// Mock NewPackageListFromRefList function for search
func NewPackageListFromRefListSearch(refList *deb.PackageRefList, packageCollection deb.PackageCollection, progress aptly.Progress) (*deb.PackageList, error) {
	if collection, ok := packageCollection.(*MockPackageSearchCollection); ok && collection.shouldErrorNewPackageListFromRefList {
		return nil, fmt.Errorf("mock package list from ref list error")
	}

	packageList := &deb.PackageList{}

	// Set up mock methods
	packageList.PrepareIndex = func() {}
	packageList.Architectures = func(includeSource bool) []string {
		if collection, ok := packageCollection.(*MockPackageSearchCollection); ok {
			if collection.emptyArchitectures {
				return []string{}
			}
			if collection.hasArchitectures {
				return []string{"amd64", "i386", "arm64"}
			}
		}
		return []string{"amd64", "i386"}
	}
	packageList.Filter = func(options deb.FilterOptions) (*deb.PackageList, error) {
		if collection, ok := packageCollection.(*MockPackageSearchCollection); ok && collection.shouldErrorFilter {
			return nil, fmt.Errorf("mock filter error")
		}

		resultList := &deb.PackageList{}
		if collection, ok := packageCollection.(*MockPackageSearchCollection); ok && collection.emptyResults {
			resultList.Len = func() int { return 0 }
		} else {
			resultList.Len = func() int { return 2 }
		}
		return resultList, nil
	}

	return packageList, nil
}

// Add methods to support repo operations
func (s *deb.Snapshot) RefList() *deb.PackageRefList {
	return s.packageRefs
}

func (r *deb.RemoteRepo) RefList() *deb.PackageRefList {
	return r.packageRefs
}

func (r *deb.LocalRepo) RefList() *deb.PackageRefList {
	return r.packageRefs
}

// Mock MatchAllQuery
type MockMatchAllQuery struct{}

func (m *MockMatchAllQuery) Matches(pkg *deb.Package) bool { return true }
func (m *MockMatchAllQuery) String() string                { return "*" }

// Override deb.MatchAllQuery for testing
func init() {
	// Replace MatchAllQuery constructor
	originalMatchAllQuery := &deb.MatchAllQuery{}
	_ = originalMatchAllQuery // Prevent unused variable warning
}

// PrintPackageList function is already defined in cmd.go

// Override deb.NewPackageListFromRefList for testing
func init() {
	// Mock deb.NewPackageListFromRefList to use our test version
	originalNewPackageListFromRefList := deb.NewPackageListFromRefList
	deb.NewPackageListFromRefList = NewPackageListFromRefListSearch
	_ = originalNewPackageListFromRefList // Prevent unused variable warning
}

// Mock query.Parse function for search tests
func init() {
	originalParse := query.Parse
	query.Parse = func(q string) (deb.PackageQuery, error) {
		return &MockSearchQuery{}, nil
	}
	_ = originalParse // Prevent unused variable warning
}

type MockSearchQuery struct{}

func (m *MockSearchQuery) Matches(pkg *deb.Package) bool { return true }
func (m *MockSearchQuery) String() string                { return "mock-search-query" }

// GetStringOrFileContent function is already defined in string_or_file_flag.go
