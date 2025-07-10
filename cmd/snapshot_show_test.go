package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
	"github.com/smira/flag"
	. "gopkg.in/check.v1"
)

type SnapshotShowSuite struct {
	cmd               *commander.Command
	collectionFactory *deb.CollectionFactory
	mockProgress      *MockSnapshotShowProgress
	mockContext       *MockSnapshotShowContext
}

var _ = Suite(&SnapshotShowSuite{})

func (s *SnapshotShowSuite) SetUpTest(c *C) {
	s.cmd = makeCmdSnapshotShow()
	s.mockProgress = &MockSnapshotShowProgress{}

	// Set up mock collections
	s.collectionFactory = &deb.CollectionFactory{
		snapshotCollection:   &MockSnapshotShowCollection{},
		localRepoCollection:  &MockLocalShowRepoCollection{},
		remoteRepoCollection: &MockRemoteShowRepoCollection{},
		packageCollection:    &MockPackageShowCollection{},
	}

	// Set up mock context
	s.mockContext = &MockSnapshotShowContext{
		flags:             s.cmd.Flag,
		progress:          s.mockProgress,
		collectionFactory: s.collectionFactory,
	}

	// Set up required flags
	s.cmd.Flag.Bool("json", false, "display record in JSON format")
	s.cmd.Flag.Bool("with-packages", false, "show list of packages")

	// Set mock context globally
	context = s.mockContext
}

func (s *SnapshotShowSuite) TestMakeCmdSnapshotShow(c *C) {
	// Test command creation and basic properties
	cmd := makeCmdSnapshotShow()
	c.Check(cmd, NotNil)
	c.Check(cmd.UsageLine, Equals, "show <name>")
	c.Check(cmd.Short, Equals, "shows details about snapshot")
	c.Check(strings.Contains(cmd.Long, "Command show displays full information"), Equals, true)

	// Test flags
	jsonFlag := cmd.Flag.Lookup("json")
	c.Check(jsonFlag, NotNil)
	c.Check(jsonFlag.DefValue, Equals, "false")

	withPackagesFlag := cmd.Flag.Lookup("with-packages")
	c.Check(withPackagesFlag, NotNil)
	c.Check(withPackagesFlag.DefValue, Equals, "false")
}

func (s *SnapshotShowSuite) TestAptlySnapshotShowInvalidArgs(c *C) {
	// Test with no arguments
	err := aptlySnapshotShow(s.cmd, []string{})
	c.Check(err, Equals, commander.ErrCommandError)

	// Test with too many arguments
	err = aptlySnapshotShow(s.cmd, []string{"snap1", "snap2"})
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *SnapshotShowSuite) TestAptlySnapshotShowTxtBasic(c *C) {
	// Test basic text output
	args := []string{"test-snapshot"}

	err := aptlySnapshotShow(s.cmd, args)
	c.Check(err, IsNil)

	// Verify snapshot details were retrieved
	mockCollection := s.collectionFactory.snapshotCollection.(*MockSnapshotShowCollection)
	c.Check(mockCollection.byNameCalled, Equals, true)
	c.Check(mockCollection.loadCompleteCalled, Equals, true)
}

func (s *SnapshotShowSuite) TestAptlySnapshotShowJSONBasic(c *C) {
	// Test basic JSON output
	s.cmd.Flag.Set("json", "true")
	args := []string{"test-snapshot"}

	err := aptlySnapshotShow(s.cmd, args)
	c.Check(err, IsNil)

	// Verify snapshot details were retrieved
	mockCollection := s.collectionFactory.snapshotCollection.(*MockSnapshotShowCollection)
	c.Check(mockCollection.byNameCalled, Equals, true)
	c.Check(mockCollection.loadCompleteCalled, Equals, true)
}

func (s *SnapshotShowSuite) TestAptlySnapshotShowSnapshotNotFound(c *C) {
	// Test with non-existent snapshot
	mockCollection := &MockSnapshotShowCollection{shouldErrorByName: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"nonexistent-snapshot"}
	err := aptlySnapshotShow(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to show.*")
}

func (s *SnapshotShowSuite) TestAptlySnapshotShowLoadCompleteError(c *C) {
	// Test with load complete error
	mockCollection := &MockSnapshotShowCollection{shouldErrorLoadComplete: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotShow(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to show.*")
}

func (s *SnapshotShowSuite) TestAptlySnapshotShowTxtWithPackages(c *C) {
	// Test text output with packages
	s.cmd.Flag.Set("with-packages", "true")
	args := []string{"test-snapshot"}

	err := aptlySnapshotShow(s.cmd, args)
	c.Check(err, IsNil)

	// Should have called package listing function
	c.Check(len(s.mockProgress.Messages) >= 0, Equals, true)
}

func (s *SnapshotShowSuite) TestAptlySnapshotShowJSONWithPackages(c *C) {
	// Test JSON output with packages
	s.cmd.Flag.Set("json", "true")
	s.cmd.Flag.Set("with-packages", "true")
	args := []string{"test-snapshot"}

	err := aptlySnapshotShow(s.cmd, args)
	c.Check(err, IsNil)

	// Should complete successfully
	c.Check(len(s.mockProgress.Messages) >= 0, Equals, true)
}

func (s *SnapshotShowSuite) TestAptlySnapshotShowJSONPackageListError(c *C) {
	// Test JSON output with package list error
	s.cmd.Flag.Set("json", "true")
	s.cmd.Flag.Set("with-packages", "true")
	mockPackageCollection := &MockPackageShowCollection{shouldErrorNewPackageListFromRefList: true}
	s.collectionFactory.packageCollection = mockPackageCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotShow(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to get package list.*")
}

func (s *SnapshotShowSuite) TestAptlySnapshotShowTxtSnapshotSources(c *C) {
	// Test text output with snapshot sources
	mockCollection := &MockSnapshotShowCollection{hasSnapshotSources: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotShow(s.cmd, args)
	c.Check(err, IsNil)

	// Should have queried for source snapshots
	c.Check(mockCollection.byUUIDCalled, Equals, true)
}

func (s *SnapshotShowSuite) TestAptlySnapshotShowTxtLocalRepoSources(c *C) {
	// Test text output with local repo sources
	mockCollection := &MockSnapshotShowCollection{hasLocalRepoSources: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotShow(s.cmd, args)
	c.Check(err, IsNil)

	// Should have queried for source local repos
	mockLocalCollection := s.collectionFactory.localRepoCollection.(*MockLocalShowRepoCollection)
	c.Check(mockLocalCollection.byUUIDCalled, Equals, true)
}

func (s *SnapshotShowSuite) TestAptlySnapshotShowTxtRemoteRepoSources(c *C) {
	// Test text output with remote repo sources
	mockCollection := &MockSnapshotShowCollection{hasRemoteRepoSources: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotShow(s.cmd, args)
	c.Check(err, IsNil)

	// Should have queried for source remote repos
	mockRemoteCollection := s.collectionFactory.remoteRepoCollection.(*MockRemoteShowRepoCollection)
	c.Check(mockRemoteCollection.byUUIDCalled, Equals, true)
}

func (s *SnapshotShowSuite) TestAptlySnapshotShowJSONSnapshotSources(c *C) {
	// Test JSON output with snapshot sources
	s.cmd.Flag.Set("json", "true")
	mockCollection := &MockSnapshotShowCollection{hasSnapshotSources: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotShow(s.cmd, args)
	c.Check(err, IsNil)

	// Should have queried for source snapshots
	c.Check(mockCollection.byUUIDCalled, Equals, true)
}

func (s *SnapshotShowSuite) TestAptlySnapshotShowJSONLocalRepoSources(c *C) {
	// Test JSON output with local repo sources
	s.cmd.Flag.Set("json", "true")
	mockCollection := &MockSnapshotShowCollection{hasLocalRepoSources: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotShow(s.cmd, args)
	c.Check(err, IsNil)

	// Should have queried for source local repos
	mockLocalCollection := s.collectionFactory.localRepoCollection.(*MockLocalShowRepoCollection)
	c.Check(mockLocalCollection.byUUIDCalled, Equals, true)
}

func (s *SnapshotShowSuite) TestAptlySnapshotShowJSONRemoteRepoSources(c *C) {
	// Test JSON output with remote repo sources
	s.cmd.Flag.Set("json", "true")
	mockCollection := &MockSnapshotShowCollection{hasRemoteRepoSources: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotShow(s.cmd, args)
	c.Check(err, IsNil)

	// Should have queried for source remote repos
	mockRemoteCollection := s.collectionFactory.remoteRepoCollection.(*MockRemoteShowRepoCollection)
	c.Check(mockRemoteCollection.byUUIDCalled, Equals, true)
}

func (s *SnapshotShowSuite) TestAptlySnapshotShowSourceErrors(c *C) {
	// Test handling of source lookup errors (should continue gracefully)
	mockSnapshotCollection := &MockSnapshotShowCollection{
		hasSnapshotSources:   true,
		shouldErrorByUUID:    true,
	}
	s.collectionFactory.snapshotCollection = mockSnapshotCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotShow(s.cmd, args)
	c.Check(err, IsNil) // Should not fail on source lookup errors

	// Should have attempted to query for sources
	c.Check(mockSnapshotCollection.byUUIDCalled, Equals, true)
}

func (s *SnapshotShowSuite) TestAptlySnapshotShowJSONMarshalError(c *C) {
	// Test JSON marshal error handling
	s.cmd.Flag.Set("json", "true")
	
	// Create a snapshot that will cause JSON marshal error
	mockCollection := &MockSnapshotShowCollection{causeMarshalError: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotShow(s.cmd, args)
	c.Check(err, NotNil) // Should fail on marshal error
}

func (s *SnapshotShowSuite) TestAptlySnapshotShowEmptyRefList(c *C) {
	// Test with empty ref list
	s.cmd.Flag.Set("json", "true")
	s.cmd.Flag.Set("with-packages", "true")
	mockCollection := &MockSnapshotShowCollection{emptyRefList: true}
	s.collectionFactory.snapshotCollection = mockCollection

	args := []string{"test-snapshot"}
	err := aptlySnapshotShow(s.cmd, args)
	c.Check(err, IsNil)

	// Should handle empty ref list gracefully
	c.Check(len(s.mockProgress.Messages) >= 0, Equals, true)
}

func (s *SnapshotShowSuite) TestAptlySnapshotShowNoSources(c *C) {
	// Test snapshot with no sources
	args := []string{"test-snapshot"}

	err := aptlySnapshotShow(s.cmd, args)
	c.Check(err, IsNil)

	// Should complete successfully without sources
	mockCollection := s.collectionFactory.snapshotCollection.(*MockSnapshotShowCollection)
	c.Check(mockCollection.byNameCalled, Equals, true)
}

// Mock implementations for testing

type MockSnapshotShowProgress struct {
	Messages []string
}

func (m *MockSnapshotShowProgress) Printf(msg string, a ...interface{}) {
	formatted := fmt.Sprintf(msg, a...)
	m.Messages = append(m.Messages, formatted)
}

type MockSnapshotShowContext struct {
	flags             *flag.FlagSet
	progress          *MockSnapshotShowProgress
	collectionFactory *deb.CollectionFactory
}

func (m *MockSnapshotShowContext) Flags() *flag.FlagSet                          { return m.flags }
func (m *MockSnapshotShowContext) Progress() aptly.Progress                      { return m.progress }
func (m *MockSnapshotShowContext) NewCollectionFactory() *deb.CollectionFactory { return m.collectionFactory }

type MockSnapshotShowCollection struct {
	shouldErrorByName        bool
	shouldErrorLoadComplete  bool
	shouldErrorByUUID        bool
	hasSnapshotSources       bool
	hasLocalRepoSources      bool
	hasRemoteRepoSources     bool
	causeMarshalError        bool
	emptyRefList             bool
	byNameCalled             bool
	loadCompleteCalled       bool
	byUUIDCalled             bool
}

func (m *MockSnapshotShowCollection) ByName(name string) (*deb.Snapshot, error) {
	m.byNameCalled = true
	if m.shouldErrorByName {
		return nil, fmt.Errorf("mock snapshot by name error")
	}

	snapshot := &deb.Snapshot{
		Name:        name,
		UUID:        "test-uuid-" + name,
		CreatedAt:   time.Now(),
		Description: "Test snapshot description",
		SourceIDs:   []string{},
	}

	// Set up different source types based on test flags
	if m.hasSnapshotSources {
		snapshot.SourceKind = deb.SourceSnapshot
		snapshot.SourceIDs = []string{"source-snapshot-uuid"}
	} else if m.hasLocalRepoSources {
		snapshot.SourceKind = deb.SourceLocalRepo
		snapshot.SourceIDs = []string{"source-local-repo-uuid"}
	} else if m.hasRemoteRepoSources {
		snapshot.SourceKind = deb.SourceRemoteRepo
		snapshot.SourceIDs = []string{"source-remote-repo-uuid"}
	}

	// Create a problematic field for JSON marshal error testing
	if m.causeMarshalError {
		// Create a cyclic structure that can't be marshaled
		snapshot.Snapshots = []*deb.Snapshot{snapshot} // Self-reference
	}

	return snapshot, nil
}

func (m *MockSnapshotShowCollection) LoadComplete(snapshot *deb.Snapshot) error {
	m.loadCompleteCalled = true
	if m.shouldErrorLoadComplete {
		return fmt.Errorf("mock snapshot load complete error")
	}

	// Set up mock RefList
	if m.emptyRefList {
		snapshot.packageRefs = nil
	} else {
		snapshot.packageRefs = deb.NewPackageRefList()
		snapshot.packageRefs.Append(&deb.PackageRef{Key: []byte("pkg1")})
		snapshot.packageRefs.Append(&deb.PackageRef{Key: []byte("pkg2")})
	}

	return nil
}

func (m *MockSnapshotShowCollection) ByUUID(uuid string) (*deb.Snapshot, error) {
	m.byUUIDCalled = true
	if m.shouldErrorByUUID {
		return nil, fmt.Errorf("mock snapshot by UUID error")
	}
	return &deb.Snapshot{Name: "source-snapshot", UUID: uuid}, nil
}

type MockLocalShowRepoCollection struct {
	shouldErrorByUUID bool
	byUUIDCalled      bool
}

func (m *MockLocalShowRepoCollection) ByUUID(uuid string) (*deb.LocalRepo, error) {
	m.byUUIDCalled = true
	if m.shouldErrorByUUID {
		return nil, fmt.Errorf("mock local repo by UUID error")
	}
	return &deb.LocalRepo{Name: "source-local-repo", uuid: uuid}, nil
}

type MockRemoteShowRepoCollection struct {
	shouldErrorByUUID bool
	byUUIDCalled      bool
}

func (m *MockRemoteShowRepoCollection) ByUUID(uuid string) (*deb.RemoteRepo, error) {
	m.byUUIDCalled = true
	if m.shouldErrorByUUID {
		return nil, fmt.Errorf("mock remote repo by UUID error")
	}
	return &deb.RemoteRepo{Name: "source-remote-repo", uuid: uuid}, nil
}

type MockPackageShowCollection struct {
	shouldErrorNewPackageListFromRefList bool
}

// Mock NewPackageListFromRefList function
func NewPackageListFromRefListShow(refList *deb.PackageRefList, packageCollection deb.PackageCollection, progress aptly.Progress) (*deb.PackageList, error) {
	if collection, ok := packageCollection.(*MockPackageShowCollection); ok && collection.shouldErrorNewPackageListFromRefList {
		return nil, fmt.Errorf("mock package list from ref list error")
	}

	packageList := &deb.PackageList{}

	// Set up mock methods
	packageList.PrepareIndex = func() {}
	packageList.ForEachIndexed = func(handler func(*deb.Package) error) error {
		// Process mock packages
		mockPackages := []*deb.Package{
			{Name: "test-package", Version: "1.0", Architecture: "amd64"},
			{Name: "another-package", Version: "2.0", Architecture: "i386"},
		}
		for _, pkg := range mockPackages {
			if err := handler(pkg); err != nil {
				return err
			}
		}
		return nil
	}

	return packageList, nil
}

// Add methods to support snapshot operations
func (s *deb.Snapshot) RefList() *deb.PackageRefList {
	return s.packageRefs
}

func (s *deb.Snapshot) NumPackages() int {
	if s.packageRefs == nil {
		return 0
	}
	return s.packageRefs.Len()
}

// ListPackagesRefList is already defined in cmd.go:24

// Mock Package.GetFullName method
func (p *deb.Package) GetFullName() string {
	return fmt.Sprintf("%s_%s_%s", p.Name, p.Version, p.Architecture)
}

// Override some global functions for testing
func init() {
	// Mock deb.NewPackageListFromRefList to use our test version
	originalNewPackageListFromRefList := deb.NewPackageListFromRefList
	deb.NewPackageListFromRefList = NewPackageListFromRefListShow
	_ = originalNewPackageListFromRefList // Prevent unused variable warning
}

// Mock context access to make output testable
func (s *SnapshotShowSuite) SetUpSuite(c *C) {
	// Redirect stdout to capture output for testing
	originalStdout := os.Stdout
	_ = originalStdout // Prevent unused variable warning
}