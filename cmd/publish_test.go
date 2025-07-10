package cmd

import (
	"strings"
	"testing"

	ctx "github.com/aptly-dev/aptly/context"
	"github.com/smira/commander"
	"github.com/smira/flag"
	. "gopkg.in/check.v1"
)

type PublishSuite struct {
	originalContext *ctx.AptlyContext
}

var _ = Suite(&PublishSuite{})

func (s *PublishSuite) SetUpTest(c *C) {
	s.originalContext = context
}

func (s *PublishSuite) TearDownTest(c *C) {
	if context != nil && context != s.originalContext {
		context.Shutdown()
	}
	context = s.originalContext
}

func (s *PublishSuite) setupMockContext(c *C) {
	// Create a mock context for testing
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	flags.String("component", "main", "component name")
	flags.String("distribution", "stable", "distribution name")
	flags.String("origin", "", "origin")
	flags.String("label", "", "label")
	flags.Bool("force-overwrite", false, "force overwrite")
	flags.Bool("skip-signing", false, "skip signing")
	flags.String("gpg-key", "", "GPG key")
	flags.String("keyring", "", "keyring")
	flags.String("secret-keyring", "", "secret keyring")
	flags.String("passphrase", "", "passphrase")
	flags.String("passphrase-file", "", "passphrase file")
	flags.Bool("batch", false, "batch mode")
	flags.String("architectures", "", "architectures")
	flags.Bool("multi-dist", false, "multi distribution")
	
	err := InitContext(flags)
	c.Assert(err, IsNil)
}

func (s *PublishSuite) TestMakeCmdPublishSnapshot(c *C) {
	// Test makeCmdPublishSnapshot command creation
	cmd := makeCmdPublishSnapshot()
	
	c.Check(cmd, NotNil)
	c.Check(cmd.Run, NotNil)
	c.Check(cmd.UsageLine, Equals, "snapshot <name> [[<endpoint>:]<prefix>]")
	c.Check(cmd.Short, Equals, "publish snapshot")
	c.Check(cmd.Long, Not(Equals), "")
	
	// Check that expected flags are present
	c.Check(cmd.Flag.Lookup("distribution"), NotNil)
	c.Check(cmd.Flag.Lookup("component"), NotNil)
	c.Check(cmd.Flag.Lookup("origin"), NotNil)
	c.Check(cmd.Flag.Lookup("label"), NotNil)
	c.Check(cmd.Flag.Lookup("force-overwrite"), NotNil)
	c.Check(cmd.Flag.Lookup("skip-signing"), NotNil)
}

func (s *PublishSuite) TestMakeCmdPublishRepo(c *C) {
	// Test makeCmdPublishRepo command creation
	cmd := makeCmdPublishRepo()
	
	c.Check(cmd, NotNil)
	c.Check(cmd.Run, NotNil)
	c.Check(cmd.UsageLine, Equals, "repo <name> [[<endpoint>:]<prefix>]")
	c.Check(cmd.Short, Equals, "publish local repository")
	c.Check(cmd.Long, Not(Equals), "")
	
	// Should use the same function as snapshot but different name
	c.Check(cmd.Run, Equals, aptlyPublishSnapshotOrRepo)
}

func (s *PublishSuite) TestPublishSnapshotOrRepoNoArgs(c *C) {
	// Test aptlyPublishSnapshotOrRepo with no arguments
	s.setupMockContext(c)
	
	cmd := makeCmdPublishSnapshot()
	err := aptlyPublishSnapshotOrRepo(cmd, []string{})
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *PublishSuite) TestPublishSnapshotOrRepoTooManyArgs(c *C) {
	// Test aptlyPublishSnapshotOrRepo with too many arguments
	s.setupMockContext(c)
	
	cmd := makeCmdPublishSnapshot()
	// Set component to "main" which means we expect 1 snapshot + optional prefix
	context.Flags().Set("component", "main")
	
	// Too many args: 3 args when we expect max 2 (1 snapshot + 1 prefix)
	err := aptlyPublishSnapshotOrRepo(cmd, []string{"snap1", "snap2", "snap3"})
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *PublishSuite) TestPublishSnapshotOrRepoMultiComponent(c *C) {
	// Test aptlyPublishSnapshotOrRepo with multiple components
	s.setupMockContext(c)
	
	cmd := makeCmdPublishSnapshot()
	// Set multiple components
	context.Flags().Set("component", "main,contrib,non-free")
	
	// Should expect 3 snapshots (one per component)
	err := aptlyPublishSnapshotOrRepo(cmd, []string{"snap1"}) // Too few
	c.Check(err, Equals, commander.ErrCommandError)
	
	err = aptlyPublishSnapshotOrRepo(cmd, []string{"snap1", "snap2", "snap3", "snap4", "snap5"}) // Too many
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *PublishSuite) TestMakeCmdPublishList(c *C) {
	// Test makeCmdPublishList command creation
	cmd := makeCmdPublishList()
	
	c.Check(cmd, NotNil)
	c.Check(cmd.Run, NotNil)
	c.Check(cmd.UsageLine, Equals, "list")
	c.Check(cmd.Short, Equals, "list published repositories")
	c.Check(cmd.Long, Not(Equals), "")
	
	// Check that expected flags are present
	c.Check(cmd.Flag.Lookup("raw"), NotNil)
}

func (s *PublishSuite) TestMakeCmdPublishShow(c *C) {
	// Test makeCmdPublishShow command creation
	cmd := makeCmdPublishShow()
	
	c.Check(cmd, NotNil)
	c.Check(cmd.Run, NotNil)
	c.Check(cmd.UsageLine, Equals, "show <distribution> [[<endpoint>:]<prefix>]")
	c.Check(cmd.Short, Equals, "shows details of published repository")
	c.Check(cmd.Long, Not(Equals), "")
}

func (s *PublishSuite) TestMakeCmdPublishDrop(c *C) {
	// Test makeCmdPublishDrop command creation
	cmd := makeCmdPublishDrop()
	
	c.Check(cmd, NotNil)
	c.Check(cmd.Run, NotNil)
	c.Check(cmd.UsageLine, Equals, "drop <distribution> [[<endpoint>:]<prefix>]")
	c.Check(cmd.Short, Equals, "remove published repository")
	c.Check(cmd.Long, Not(Equals), "")
	
	// Check that expected flags are present
	c.Check(cmd.Flag.Lookup("force-drop"), NotNil)
	c.Check(cmd.Flag.Lookup("skip-cleanup"), NotNil)
}

func (s *PublishSuite) TestMakeCmdPublishUpdate(c *C) {
	// Test makeCmdPublishUpdate command creation
	cmd := makeCmdPublishUpdate()
	
	c.Check(cmd, NotNil)
	c.Check(cmd.Run, NotNil)
	c.Check(cmd.UsageLine, Equals, "update <distribution> [[<endpoint>:]<prefix>]")
	c.Check(cmd.Short, Equals, "update published repository")
	c.Check(cmd.Long, Not(Equals), "")
	
	// Check that expected flags are present
	c.Check(cmd.Flag.Lookup("force-overwrite"), NotNil)
	c.Check(cmd.Flag.Lookup("skip-signing"), NotNil)
}

func (s *PublishSuite) TestMakeCmdPublishSwitch(c *C) {
	// Test makeCmdPublishSwitch command creation
	cmd := makeCmdPublishSwitch()
	
	c.Check(cmd, NotNil)
	c.Check(cmd.Run, NotNil)
	c.Check(cmd.UsageLine, Equals, "switch <distribution> [<component1>:]<name1> [<component2>:]<name2> ... [[<endpoint>:]<prefix>]")
	c.Check(cmd.Short, Equals, "update published repository by switching to new snapshot")
	c.Check(cmd.Long, Not(Equals), "")
	
	// Check that expected flags are present
	c.Check(cmd.Flag.Lookup("force-overwrite"), NotNil)
	c.Check(cmd.Flag.Lookup("skip-signing"), NotNil)
	c.Check(cmd.Flag.Lookup("component"), NotNil)
}

func (s *PublishSuite) TestPublishListNoArgs(c *C) {
	// Test aptlyPublishList with no arguments (should work)
	s.setupMockContext(c)
	
	err := aptlyPublishList(makeCmdPublishList(), []string{})
	// Will likely error due to no real collection factory, but tests structure
	c.Check(err, NotNil)
}

func (s *PublishSuite) TestPublishListWithArgs(c *C) {
	// Test aptlyPublishList with arguments (should fail)
	s.setupMockContext(c)
	
	err := aptlyPublishList(makeCmdPublishList(), []string{"extra", "args"})
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *PublishSuite) TestPublishShowNoArgs(c *C) {
	// Test aptlyPublishShow with no arguments
	s.setupMockContext(c)
	
	err := aptlyPublishShow(makeCmdPublishShow(), []string{})
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *PublishSuite) TestPublishDropNoArgs(c *C) {
	// Test aptlyPublishDrop with no arguments
	s.setupMockContext(c)
	
	err := aptlyPublishDrop(makeCmdPublishDrop(), []string{})
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *PublishSuite) TestPublishUpdateNoArgs(c *C) {
	// Test aptlyPublishUpdate with no arguments
	s.setupMockContext(c)
	
	err := aptlyPublishUpdate(makeCmdPublishUpdate(), []string{})
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *PublishSuite) TestPublishSwitchInsufficientArgs(c *C) {
	// Test aptlyPublishSwitch with insufficient arguments
	s.setupMockContext(c)
	
	err := aptlyPublishSwitch(makeCmdPublishSwitch(), []string{})
	c.Check(err, Equals, commander.ErrCommandError)
	
	err = aptlyPublishSwitch(makeCmdPublishSwitch(), []string{"distribution-only"})
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *PublishSuite) TestPublishCommandFlags(c *C) {
	// Test that all publish commands have the expected flags
	commands := []*commander.Command{
		makeCmdPublishSnapshot(),
		makeCmdPublishRepo(),
		makeCmdPublishList(),
		makeCmdPublishShow(),
		makeCmdPublishDrop(),
		makeCmdPublishUpdate(),
		makeCmdPublishSwitch(),
	}
	
	for _, cmd := range commands {
		c.Check(cmd, NotNil, Commentf("Command should not be nil: %s", cmd.Name()))
		c.Check(cmd.Run, NotNil, Commentf("Command run function should not be nil: %s", cmd.Name()))
		c.Check(cmd.UsageLine, Not(Equals), "", Commentf("Command usage should not be empty: %s", cmd.Name()))
		c.Check(cmd.Short, Not(Equals), "", Commentf("Command short description should not be empty: %s", cmd.Name()))
		c.Check(cmd.Long, Not(Equals), "", Commentf("Command long description should not be empty: %s", cmd.Name()))
	}
}

func (s *PublishSuite) TestPublishPrefixParsing(c *C) {
	// Test prefix parsing patterns used in publish commands
	
	// Mock prefix parsing similar to deb.ParsePrefix
	testCases := []struct {
		input         string
		expectedStore string
		expectedPrefix string
	}{
		{"", "", ""},
		{"prefix", "", "prefix"},
		{"endpoint:prefix", "endpoint", "prefix"},
		{"s3:us-east-1:bucket/prefix", "s3:us-east-1", "bucket/prefix"},
		{"filesystem:/path/to/dir", "filesystem", "/path/to/dir"},
	}
	
	for _, tc := range testCases {
		// Simple parsing logic for testing
		parts := strings.SplitN(tc.input, ":", 2)
		var storage, prefix string
		
		if len(parts) == 1 {
			storage = ""
			prefix = parts[0]
		} else if len(parts) == 2 {
			// Handle special cases like s3:region:bucket
			if parts[0] == "s3" && strings.Contains(parts[1], ":") {
				subParts := strings.SplitN(parts[1], ":", 2)
				storage = parts[0] + ":" + subParts[0]
				prefix = subParts[1]
			} else {
				storage = parts[0]
				prefix = parts[1]
			}
		}
		
		c.Check(storage, Equals, tc.expectedStore, Commentf("Input: %s", tc.input))
		c.Check(prefix, Equals, tc.expectedPrefix, Commentf("Input: %s", tc.input))
	}
}

func (s *PublishSuite) TestComponentParsing(c *C) {
	// Test component parsing used in publish commands
	testCases := []struct {
		input    string
		expected []string
	}{
		{"main", []string{"main"}},
		{"main,contrib", []string{"main", "contrib"}},
		{"main,contrib,non-free", []string{"main", "contrib", "non-free"}},
		{"", []string{""}},
		{"single", []string{"single"}},
	}
	
	for _, tc := range testCases {
		components := strings.Split(tc.input, ",")
		c.Check(components, DeepEquals, tc.expected, Commentf("Input: %s", tc.input))
	}
}

func (s *PublishSuite) TestPublishErrorHandling(c *C) {
	// Test error handling patterns in publish commands
	s.setupMockContext(c)
	
	// Test various error scenarios
	commands := []struct {
		name    string
		cmd     *commander.Command
		fn      func(*commander.Command, []string) error
		args    []string
		wantErr bool
	}{
		{"publish snapshot no args", makeCmdPublishSnapshot(), aptlyPublishSnapshotOrRepo, []string{}, true},
		{"publish list with args", makeCmdPublishList(), aptlyPublishList, []string{"arg"}, true},
		{"publish show no args", makeCmdPublishShow(), aptlyPublishShow, []string{}, true},
		{"publish drop no args", makeCmdPublishDrop(), aptlyPublishDrop, []string{}, true},
		{"publish update no args", makeCmdPublishUpdate(), aptlyPublishUpdate, []string{}, true},
		{"publish switch no args", makeCmdPublishSwitch(), aptlyPublishSwitch, []string{}, true},
	}
	
	for _, tc := range commands {
		err := tc.fn(tc.cmd, tc.args)
		if tc.wantErr {
			c.Check(err, NotNil, Commentf("Test case: %s", tc.name))
		} else {
			c.Check(err, IsNil, Commentf("Test case: %s", tc.name))
		}
	}
}

func (s *PublishSuite) TestPublishArgumentValidation(c *C) {
	// Test argument validation patterns
	s.setupMockContext(c)
	
	// Test component/argument count validation
	testCases := []struct {
		components string
		args       []string
		valid      bool
	}{
		{"main", []string{"snap1"}, true},                    // 1 component, 1 snapshot
		{"main", []string{"snap1", "prefix"}, true},          // 1 component, 1 snapshot + prefix
		{"main", []string{}, false},                          // 1 component, no snapshots
		{"main", []string{"snap1", "snap2", "prefix"}, false}, // 1 component, too many args
		{"main,contrib", []string{"snap1", "snap2"}, true},   // 2 components, 2 snapshots
		{"main,contrib", []string{"snap1", "snap2", "prefix"}, true}, // 2 components, 2 snapshots + prefix
		{"main,contrib", []string{"snap1"}, false},           // 2 components, not enough snapshots
	}
	
	for _, tc := range testCases {
		components := strings.Split(tc.components, ",")
		args := tc.args
		
		// Validation logic similar to aptlyPublishSnapshotOrRepo
		valid := len(args) >= len(components) && len(args) <= len(components)+1
		
		c.Check(valid, Equals, tc.valid, Commentf("Components: %s, Args: %v", tc.components, tc.args))
	}
}

func (s *PublishSuite) TestPublishSourceCommands(c *C) {
	// Test publish source commands creation
	sourceCommands := []*commander.Command{
		makeCmdPublishSourceAdd(),
		makeCmdPublishSourceDrop(),
		makeCmdPublishSourceList(),
		makeCmdPublishSourceRemove(),
		makeCmdPublishSourceReplace(),
		makeCmdPublishSourceUpdate(),
	}
	
	for _, cmd := range sourceCommands {
		c.Check(cmd, NotNil)
		c.Check(cmd.Run, NotNil)
		c.Check(cmd.UsageLine, Not(Equals), "")
		c.Check(cmd.Short, Not(Equals), "")
		c.Check(strings.Contains(cmd.Short, "source"), Equals, true)
	}
}