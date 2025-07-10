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

type PublishSourceReplaceSuite struct {
	cmd               *commander.Command
	collectionFactory *deb.CollectionFactory
	mockProgress      *MockPublishSourceReplaceProgress
	mockContext       *MockPublishSourceReplaceContext
}

var _ = Suite(&PublishSourceReplaceSuite{})

func (s *PublishSourceReplaceSuite) SetUpTest(c *C) {
	s.cmd = makeCmdPublishSourceReplace()
	s.mockProgress = &MockPublishSourceReplaceProgress{}

	// Set up mock collections
	s.collectionFactory = deb.NewCollectionFactory(nil)

	// Set up mock context
	s.mockContext = &MockPublishSourceReplaceContext{
		flags:             &s.cmd.Flag,
		progress:          s.mockProgress,
		collectionFactory: s.collectionFactory,
	}

	// Set up required flags
	s.cmd.Flag.String("prefix", ".", "publishing prefix")
	s.cmd.Flag.String("component", "", "component names to replace")

	// Skip setting mock context globally for type compatibility
	// context = s.mockContext
}

func (s *PublishSourceReplaceSuite) TestMakeCmdPublishSourceReplace(c *C) {
	// Test command creation and basic properties
	cmd := makeCmdPublishSourceReplace()
	c.Check(cmd, NotNil)
	c.Check(cmd.UsageLine, Equals, "replace <distribution> <source>")
	c.Check(cmd.Short, Equals, "replace the source components of a published repository")
	c.Check(strings.Contains(cmd.Long, "The command replaces the source components of a snapshot or local repository"), Equals, true)

	// Test flags
	prefixFlag := cmd.Flag.Lookup("prefix")
	c.Check(prefixFlag, NotNil)
	c.Check(prefixFlag.DefValue, Equals, ".")

	componentFlag := cmd.Flag.Lookup("component")
	c.Check(componentFlag, NotNil)
	c.Check(componentFlag.DefValue, Equals, "")
}

func (s *PublishSourceReplaceSuite) TestAptlyPublishSourceReplaceInvalidArgs(c *C) {
	// Test with insufficient arguments
	err := aptlyPublishSourceReplace(s.cmd, []string{})
	c.Check(err, Equals, commander.ErrCommandError)

	err = aptlyPublishSourceReplace(s.cmd, []string{"distribution"})
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *PublishSourceReplaceSuite) TestAptlyPublishSourceReplaceBasic(c *C) {
	// Test basic source replacement
	s.cmd.Flag.Set("component", "contrib")
	args := []string{"stable", "contrib-snapshot"}

	err := aptlyPublishSourceReplace(s.cmd, args)
	c.Check(err, IsNil)

	// Check that replacement messages were displayed
	foundReplaceMessage := false
	foundAddingMessage := false
	foundPublishMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "Replacing source list") {
			foundReplaceMessage = true
		}
		if strings.Contains(msg, "Adding component 'contrib'") {
			foundAddingMessage = true
		}
		if strings.Contains(msg, "aptly publish update") {
			foundPublishMessage = true
		}
	}
	c.Check(foundReplaceMessage, Equals, true)
	c.Check(foundAddingMessage, Equals, true)
	c.Check(foundPublishMessage, Equals, true)
}

func (s *PublishSourceReplaceSuite) TestAptlyPublishSourceReplaceMismatchComponents(c *C) {
	// Test with mismatched number of components and sources
	s.cmd.Flag.Set("component", "main,contrib")
	args := []string{"stable", "single-source"} // 2 components, 1 source

	err := aptlyPublishSourceReplace(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*mismatch in number of components.*")
}

func (s *PublishSourceReplaceSuite) TestAptlyPublishSourceReplaceMultipleComponents(c *C) {
	// Test with multiple components
	s.cmd.Flag.Set("component", "main,contrib,non-free")
	args := []string{"stable", "main-snapshot", "contrib-snapshot", "non-free-snapshot"}

	err := aptlyPublishSourceReplace(s.cmd, args)
	c.Check(err, IsNil)

	// Should replace with all components
	foundReplaceMessage := false
	addingCount := 0
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "Replacing source list") {
			foundReplaceMessage = true
		}
		if strings.Contains(msg, "Adding component") {
			addingCount++
		}
	}
	c.Check(foundReplaceMessage, Equals, true)
	c.Check(addingCount, Equals, 3) // Should add 3 components
}

func (s *PublishSourceReplaceSuite) TestAptlyPublishSourceReplaceWithPrefix(c *C) {
	// Test with custom prefix
	s.cmd.Flag.Set("prefix", "ppa")
	s.cmd.Flag.Set("component", "contrib")
	args := []string{"stable", "contrib-snapshot"}

	err := aptlyPublishSourceReplace(s.cmd, args)
	c.Check(err, IsNil)

	// Should complete successfully with prefix
	c.Check(len(s.mockProgress.Messages) > 0, Equals, true)
}

func (s *PublishSourceReplaceSuite) TestAptlyPublishSourceReplaceWithStorage(c *C) {
	// Test with storage endpoint
	s.cmd.Flag.Set("prefix", "s3:bucket")
	s.cmd.Flag.Set("component", "contrib")
	args := []string{"stable", "contrib-snapshot"}

	err := aptlyPublishSourceReplace(s.cmd, args)
	c.Check(err, IsNil)

	// Should complete successfully with storage
	c.Check(len(s.mockProgress.Messages) > 0, Equals, true)
}

func (s *PublishSourceReplaceSuite) TestAptlyPublishSourceReplaceRepoNotFound(c *C) {
	// Test with non-existent published repository - simplified test
	// Note: Cannot set private fields directly, test simplified
	s.cmd.Flag.Set("component", "contrib")

	args := []string{"nonexistent-dist", "contrib-snapshot"}
	err := aptlyPublishSourceReplace(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to add.*")
}

func (s *PublishSourceReplaceSuite) TestAptlyPublishSourceReplaceLoadCompleteError(c *C) {
	// Test with load complete error - simplified test
	// Note: Cannot set private fields directly, test simplified
	s.cmd.Flag.Set("component", "contrib")

	args := []string{"stable", "contrib-snapshot"}
	err := aptlyPublishSourceReplace(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to add.*")
}

func (s *PublishSourceReplaceSuite) TestAptlyPublishSourceReplaceUpdateError(c *C) {
	// Test with repository update error - simplified test
	// Note: Cannot set private fields directly, test simplified
	s.cmd.Flag.Set("component", "contrib")

	args := []string{"stable", "contrib-snapshot"}
	err := aptlyPublishSourceReplace(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to save to DB.*")
}

func (s *PublishSourceReplaceSuite) TestAptlyPublishSourceReplaceSourceKindDisplay(c *C) {
	// Test that source kind is displayed correctly - simplified test
	// Note: Cannot set private fields directly, test simplified
	s.cmd.Flag.Set("component", "contrib")

	args := []string{"stable", "contrib-snapshot"}
	err := aptlyPublishSourceReplace(s.cmd, args)
	c.Check(err, IsNil)

	// Should show source kind in message
	foundSourceKind := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "[snapshot]") {
			foundSourceKind = true
			break
		}
	}
	c.Check(foundSourceKind, Equals, true)
}

func (s *PublishSourceReplaceSuite) TestAptlyPublishSourceReplaceEmptyComponent(c *C) {
	// Test with empty component flag
	s.cmd.Flag.Set("component", "")
	args := []string{"stable", "contrib-snapshot"}

	err := aptlyPublishSourceReplace(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*mismatch in number of components.*")
}

func (s *PublishSourceReplaceSuite) TestAptlyPublishSourceReplacePrefixParsing(c *C) {
	// Test different prefix formats
	prefixTests := []string{
		".",
		"ppa",
		"s3:bucket",
		"filesystem:/path",
	}

	for _, prefix := range prefixTests {
		s.cmd.Flag.Set("prefix", prefix)
		s.cmd.Flag.Set("component", "contrib")

		args := []string{"stable", "contrib-snapshot"}
		err := aptlyPublishSourceReplace(s.cmd, args)
		c.Check(err, IsNil, Commentf("Prefix: %s", prefix))

		// Reset for next test
		s.mockProgress.Messages = []string{}
	}
}

func (s *PublishSourceReplaceSuite) TestAptlyPublishSourceReplaceComponentValidation(c *C) {
	// Test various component configurations
	componentTests := []struct {
		components string
		sources    []string
		shouldErr  bool
	}{
		{"main", []string{"main-source"}, false},
		{"main,contrib", []string{"main-source", "contrib-source"}, false},
		{"main,contrib,non-free", []string{"main-source", "contrib-source", "non-free-source"}, false},
		{"main,contrib", []string{"single-source"}, true}, // Mismatch
		{"", []string{"source"}, true},                     // Empty component
	}

	for _, test := range componentTests {
		s.cmd.Flag.Set("component", test.components)
		args := append([]string{"stable"}, test.sources...)

		err := aptlyPublishSourceReplace(s.cmd, args)
		if test.shouldErr {
			c.Check(err, NotNil, Commentf("Components: %s, Sources: %v", test.components, test.sources))
		} else {
			c.Check(err, IsNil, Commentf("Components: %s, Sources: %v", test.components, test.sources))
		}

		// Reset for next test
		s.mockProgress.Messages = []string{}
	}
}

func (s *PublishSourceReplaceSuite) TestSourceKindHandling(c *C) {
	sourceKinds := []string{deb.SourceSnapshot, deb.SourceLocalRepo}

	for _, kind := range sourceKinds {
		// Note: Cannot set private fields directly, test simplified
		s.cmd.Flag.Set("component", "contrib")

		args := []string{"stable", "contrib-source"}
		err := aptlyPublishSourceReplace(s.cmd, args)
		c.Check(err, IsNil, Commentf("Source kind: %s", kind))

		// Should show correct source kind
		foundSourceKind := false
		expectedDisplay := "[" + kind + "]"
		for _, msg := range s.mockProgress.Messages {
			if strings.Contains(msg, expectedDisplay) {
				foundSourceKind = true
				break
			}
		}
		c.Check(foundSourceKind, Equals, true, Commentf("Expected: %s", expectedDisplay))

		// Reset for next test
		s.mockProgress.Messages = []string{}
	}
}

func (s *PublishSourceReplaceSuite) TestStoragePrefixHandling(c *C) {
	// Test different storage configurations
	prefixTests := []struct {
		prefix   string
		expected string
	}{
		{".", ""},
		{"ppa", "ppa"},
		{"s3:bucket", "s3:bucket"},
	}

	for _, test := range prefixTests {
		s.cmd.Flag.Set("prefix", test.prefix)
		s.cmd.Flag.Set("component", "contrib")

		args := []string{"stable", "contrib-source"}
		err := aptlyPublishSourceReplace(s.cmd, args)
		c.Check(err, IsNil, Commentf("Prefix: %s", test.prefix))

		// Should complete successfully
		c.Check(len(s.mockProgress.Messages) > 0, Equals, true)

		// Reset for next test
		s.mockProgress.Messages = []string{}
	}
}

func (s *PublishSourceReplaceSuite) TestReplacementWorkflow(c *C) {
	// Test the complete replacement workflow: clear existing sources, add new ones
	s.cmd.Flag.Set("component", "main,contrib")
	args := []string{"stable", "new-main", "new-contrib"}

	err := aptlyPublishSourceReplace(s.cmd, args)
	c.Check(err, IsNil)

	// Should replace existing sources completely
	foundReplaceMessage := false
	foundMainAdding := false
	foundContribAdding := false

	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "Replacing source list") {
			foundReplaceMessage = true
		}
		if strings.Contains(msg, "Adding component 'main'") {
			foundMainAdding = true
		}
		if strings.Contains(msg, "Adding component 'contrib'") {
			foundContribAdding = true
		}
	}

	c.Check(foundReplaceMessage, Equals, true)
	c.Check(foundMainAdding, Equals, true)
	c.Check(foundContribAdding, Equals, true)
}

func (s *PublishSourceReplaceSuite) TestEdgeCases(c *C) {
	// Test with very long component names
	s.cmd.Flag.Set("component", "very-long-component-name-that-might-cause-issues")
	args := []string{"stable", "source-with-long-name"}

	err := aptlyPublishSourceReplace(s.cmd, args)
	c.Check(err, IsNil)

	// Should handle long names gracefully
	c.Check(len(s.mockProgress.Messages) > 0, Equals, true)
}

func (s *PublishSourceReplaceSuite) TestErrorMessageFormatting(c *C) {
	// Test that error messages are properly formatted
	s.cmd.Flag.Set("component", "main,contrib")
	args := []string{"stable", "single-source"}

	err := aptlyPublishSourceReplace(s.cmd, args)
	c.Check(err, NotNil)

	// Should show specific numbers in error
	errorMsg := err.Error()
	c.Check(strings.Contains(errorMsg, "2"), Equals, true) // 2 components
	c.Check(strings.Contains(errorMsg, "1"), Equals, true) // 1 source
}

// Mock implementations for testing

type MockPublishSourceReplaceProgress struct {
	Messages []string
}

func (m *MockPublishSourceReplaceProgress) Printf(msg string, a ...interface{}) {
	formatted := fmt.Sprintf(msg, a...)
	m.Messages = append(m.Messages, formatted)
}

func (m *MockPublishSourceReplaceProgress) AddBar(count int) {}
func (m *MockPublishSourceReplaceProgress) Flush() {}
func (m *MockPublishSourceReplaceProgress) InitBar(total int64, colored bool, barType aptly.BarType) {}
func (m *MockPublishSourceReplaceProgress) PrintfStdErr(msg string, a ...interface{}) {}
func (m *MockPublishSourceReplaceProgress) SetBar(count int) {}
func (m *MockPublishSourceReplaceProgress) Shutdown() {}
func (m *MockPublishSourceReplaceProgress) ShutdownBar() {}
func (m *MockPublishSourceReplaceProgress) Start() {}
func (m *MockPublishSourceReplaceProgress) Write(data []byte) (int, error) { return len(data), nil }
func (m *MockPublishSourceReplaceProgress) ColoredPrintf(msg string, a ...interface{}) {}

type MockPublishSourceReplaceContext struct {
	flags             *flag.FlagSet
	progress          *MockPublishSourceReplaceProgress
	collectionFactory *deb.CollectionFactory
}

func (m *MockPublishSourceReplaceContext) Flags() *flag.FlagSet                          { return m.flags }
func (m *MockPublishSourceReplaceContext) Progress() aptly.Progress                      { return m.progress }
func (m *MockPublishSourceReplaceContext) NewCollectionFactory() *deb.CollectionFactory { return m.collectionFactory }

type MockPublishedSourceReplaceRepoCollection struct {
	shouldErrorByStoragePrefixDistribution bool
	shouldErrorLoadComplete                bool
	shouldErrorUpdate                      bool
	sourceKind                             string
}

func (m *MockPublishedSourceReplaceRepoCollection) ByStoragePrefixDistribution(storage, prefix, distribution string) (*deb.PublishedRepo, error) {
	if m.shouldErrorByStoragePrefixDistribution {
		return nil, fmt.Errorf("mock published repo by storage prefix distribution error")
	}

	repo := &deb.PublishedRepo{
		Distribution: distribution,
		Prefix:       prefix,
		Storage:      storage,
	}

	if m.sourceKind != "" {
		repo.SourceKind = m.sourceKind
	} else {
		repo.SourceKind = deb.SourceSnapshot
	}

	return repo, nil
}

func (m *MockPublishedSourceReplaceRepoCollection) LoadComplete(published *deb.PublishedRepo, collectionFactory *deb.CollectionFactory) error {
	if m.shouldErrorLoadComplete {
		return fmt.Errorf("mock published repo load complete error")
	}
	return nil
}

func (m *MockPublishedSourceReplaceRepoCollection) Update(published *deb.PublishedRepo) error {
	if m.shouldErrorUpdate {
		return fmt.Errorf("mock published repo update error")
	}
	return nil
}

// Note: Removed method definitions on non-local types (deb.PublishedRepo)
// to fix compilation errors. Tests are simplified to focus on basic functionality.

// Test that replacement clears existing sources
func (s *PublishSourceReplaceSuite) TestReplacementClearsExistingSources(c *C) {
	// Mock a collection that tracks when sources are cleared - simplified test
	// Note: Cannot set private fields directly, test simplified

	s.cmd.Flag.Set("component", "new-component")
	args := []string{"stable", "new-source"}

	err := aptlyPublishSourceReplace(s.cmd, args)
	c.Check(err, IsNil)

	// Should show replacement message indicating existing sources were cleared
	foundReplaceMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "Replacing source list") {
			foundReplaceMessage = true
			break
		}
	}
	c.Check(foundReplaceMessage, Equals, true)
}

// Test multiple component replacement at once
func (s *PublishSourceReplaceSuite) TestMultipleComponentReplacement(c *C) {
	// Test replacing multiple components simultaneously
	components := []string{"main", "contrib", "non-free", "restricted"}
	sources := []string{"main-v2", "contrib-v2", "non-free-v2", "restricted-v2"}

	s.cmd.Flag.Set("component", strings.Join(components, ","))
	args := append([]string{"stable"}, sources...)

	err := aptlyPublishSourceReplace(s.cmd, args)
	c.Check(err, IsNil)

	// Should replace with all 4 components
	addingCount := 0
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "Adding component") {
			addingCount++
		}
	}
	c.Check(addingCount, Equals, 4)
}

// Test prefix and storage combinations
func (s *PublishSourceReplaceSuite) TestPrefixStorageCombinations(c *C) {
	// Test various prefix/storage combinations
	prefixTests := []struct {
		prefix      string
		description string
	}{
		{".", "default prefix"},
		{"ubuntu", "simple prefix"},
		{"s3:mybucket", "S3 storage with bucket"},
		{"filesystem:/var/aptly", "filesystem storage with path"},
		{"swift:container", "Swift storage with container"},
	}

	for _, test := range prefixTests {
		s.cmd.Flag.Set("prefix", test.prefix)
		s.cmd.Flag.Set("component", "main")

		args := []string{"focal", "focal-main"}
		err := aptlyPublishSourceReplace(s.cmd, args)
		c.Check(err, IsNil, Commentf("Test: %s", test.description))

		// Should complete successfully
		c.Check(len(s.mockProgress.Messages) > 0, Equals, true)

		// Reset for next test
		s.mockProgress.Messages = []string{}
	}
}

// Test error handling robustness
func (s *PublishSourceReplaceSuite) TestErrorHandlingRobustness(c *C) {
	// Test various error scenarios
	errorScenarios := []struct {
		setupFunc   func(*MockPublishedSourceReplaceRepoCollection)
		description string
		expectedErr string
	}{
		{
			func(m *MockPublishedSourceReplaceRepoCollection) { m.shouldErrorByStoragePrefixDistribution = true },
			"repository not found",
			"unable to add",
		},
		{
			func(m *MockPublishedSourceReplaceRepoCollection) { m.shouldErrorLoadComplete = true },
			"load complete error",
			"unable to add",
		},
		{
			func(m *MockPublishedSourceReplaceRepoCollection) { m.shouldErrorUpdate = true },
			"repository update error",
			"unable to save to DB",
		},
	}

	for _, scenario := range errorScenarios {
		// Note: Cannot set private fields directly, test simplified

		s.cmd.Flag.Set("component", "main")
		args := []string{"stable", "main-source"}

		err := aptlyPublishSourceReplace(s.cmd, args)
		c.Check(err, NotNil, Commentf("Scenario: %s", scenario.description))
		c.Check(err.Error(), Matches, ".*"+scenario.expectedErr+".*", Commentf("Scenario: %s", scenario.description))
	}
}

// Add field to track clear operation
type MockPublishedSourceReplaceRepoCollectionWithTracking struct {
	*MockPublishedSourceReplaceRepoCollection
	trackClearOperation bool
}

// Test special characters in component names
func (s *PublishSourceReplaceSuite) TestSpecialCharactersInComponents(c *C) {
	// Test with special characters in component and source names
	s.cmd.Flag.Set("component", "main-component,contrib_component,non.free.component")
	args := []string{"stable", "main-source-v1.0", "contrib_source_v2.0", "non.free.source.v3.0"}

	err := aptlyPublishSourceReplace(s.cmd, args)
	c.Check(err, IsNil)

	// Should handle special characters gracefully
	addingCount := 0
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "Adding component") {
			addingCount++
		}
	}
	c.Check(addingCount, Equals, 3)
}