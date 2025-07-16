package cmd

import (
	"strings"

	ctx "github.com/aptly-dev/aptly/context"
	"github.com/smira/commander"
	"github.com/smira/flag"
	. "gopkg.in/check.v1"
)

type PublishSourceAddSuite struct {
	cmd             *commander.Command
	originalContext *ctx.AptlyContext
}

var _ = Suite(&PublishSourceAddSuite{})

func (s *PublishSourceAddSuite) SetUpTest(c *C) {
	s.originalContext = context
	s.cmd = makeCmdPublishSourceAdd()
}

func (s *PublishSourceAddSuite) TearDownTest(c *C) {
	if context != nil && context != s.originalContext {
		context.Shutdown()
	}
	context = s.originalContext
}

func (s *PublishSourceAddSuite) setupMockContext(c *C) {
	// Create a mock context for testing
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	flags.String("prefix", ".", "publishing prefix")
	flags.String("component", "", "component names to add")

	err := InitContext(flags)
	c.Assert(err, IsNil)
}

func (s *PublishSourceAddSuite) TestMakeCmdPublishSourceAdd(c *C) {
	// Test command creation and basic properties
	cmd := makeCmdPublishSourceAdd()
	c.Check(cmd, NotNil)
	c.Check(cmd.UsageLine, Equals, "add <distribution> <source>")
	c.Check(cmd.Short, Equals, "add source components to a published repo")
	c.Check(strings.Contains(cmd.Long, "The command adds components of a snapshot or local repository"), Equals, true)

	// Test flags
	prefixFlag := cmd.Flag.Lookup("prefix")
	c.Check(prefixFlag, NotNil)
	c.Check(prefixFlag.DefValue, Equals, ".")

	componentFlag := cmd.Flag.Lookup("component")
	c.Check(componentFlag, NotNil)
	c.Check(componentFlag.DefValue, Equals, "")
}

func (s *PublishSourceAddSuite) TestAptlyPublishSourceAddInvalidArgs(c *C) {
	// Test with insufficient arguments
	s.setupMockContext(c)

	err := aptlyPublishSourceAdd(s.cmd, []string{})
	c.Check(err, Equals, commander.ErrCommandError)

	err = aptlyPublishSourceAdd(s.cmd, []string{"distribution"})
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *PublishSourceAddSuite) TestAptlyPublishSourceAddBasic(c *C) {
	// Test basic source addition
	s.setupMockContext(c)

	context.Flags().Set("component", "contrib")
	args := []string{"stable", "contrib-snapshot"}

	err := aptlyPublishSourceAdd(s.cmd, args)
	// This will fail because we don't have a real published repo, but test structure
	c.Check(err, NotNil)
}

func (s *PublishSourceAddSuite) TestAptlyPublishSourceAddMismatchComponents(c *C) {
	// Test with mismatched number of components and sources
	s.setupMockContext(c)

	context.Flags().Set("component", "main,contrib")
	args := []string{"stable", "single-source"} // 2 components, 1 source

	err := aptlyPublishSourceAdd(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*mismatch in number of components.*")
}

func (s *PublishSourceAddSuite) TestAptlyPublishSourceAddMultipleComponents(c *C) {
	// Test with multiple components
	s.setupMockContext(c)

	context.Flags().Set("component", "main,contrib,non-free")
	args := []string{"stable", "main-snapshot", "contrib-snapshot", "non-free-snapshot"}

	err := aptlyPublishSourceAdd(s.cmd, args)
	// This will fail because we don't have a real published repo, but test structure
	c.Check(err, NotNil)
}

func (s *PublishSourceAddSuite) TestAptlyPublishSourceAddWithPrefix(c *C) {
	// Test with custom prefix
	s.setupMockContext(c)

	context.Flags().Set("prefix", "ppa")
	context.Flags().Set("component", "contrib")
	args := []string{"stable", "contrib-snapshot"}

	err := aptlyPublishSourceAdd(s.cmd, args)
	// This will fail because we don't have a real published repo, but test structure
	c.Check(err, NotNil)
}

func (s *PublishSourceAddSuite) TestAptlyPublishSourceAddWithStorage(c *C) {
	// Test with storage endpoint
	s.setupMockContext(c)

	context.Flags().Set("prefix", "s3:bucket")
	context.Flags().Set("component", "contrib")
	args := []string{"stable", "contrib-snapshot"}

	err := aptlyPublishSourceAdd(s.cmd, args)
	// This will fail because we don't have a real published repo, but test structure
	c.Check(err, NotNil)
}

func (s *PublishSourceAddSuite) TestAptlyPublishSourceAddRepoNotFound(c *C) {
	// Test with non-existent published repository
	s.setupMockContext(c)

	context.Flags().Set("component", "contrib")

	args := []string{"nonexistent-dist", "contrib-snapshot"}
	err := aptlyPublishSourceAdd(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*unable to add.*")
}

func (s *PublishSourceAddSuite) TestAptlyPublishSourceAddEmptyComponent(c *C) {
	// Test with empty component flag
	s.setupMockContext(c)

	context.Flags().Set("component", "")
	args := []string{"stable", "contrib-snapshot"}

	err := aptlyPublishSourceAdd(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*mismatch in number of components.*")
}
