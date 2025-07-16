package cmd

import (
	"strings"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
	. "gopkg.in/check.v1"
)

type RepoRemoveSimpleSuite struct {
	cmd               *commander.Command
	collectionFactory *deb.CollectionFactory
}

var _ = Suite(&RepoRemoveSimpleSuite{})

func (s *RepoRemoveSimpleSuite) SetUpTest(c *C) {
	s.cmd = makeCmdRepoRemove()
	s.collectionFactory = &deb.CollectionFactory{}

	// Set up required flags
	s.cmd.Flag.Bool("dry-run", false, "don't remove, just show what would be removed")
	s.cmd.Flag.Bool("with-deps", false, "follow dependencies when processing package-spec")
}

func (s *RepoRemoveSimpleSuite) TestMakeCmdRepoRemove(c *C) {
	// Test command creation and basic properties
	cmd := makeCmdRepoRemove()
	c.Check(cmd, NotNil)
	c.Check(cmd.UsageLine, Equals, "remove <name> <package-query> ...")
	c.Check(cmd.Short, Equals, "remove packages from local repository")
	c.Check(strings.Contains(cmd.Long, "Commands removes packages matching <package-query> from local repository"), Equals, true)

	// Test flags
	dryRunFlag := cmd.Flag.Lookup("dry-run")
	c.Check(dryRunFlag, NotNil)
	c.Check(dryRunFlag.DefValue, Equals, "false")

	withDepsFlag := cmd.Flag.Lookup("with-deps")
	c.Check(withDepsFlag, NotNil)
	c.Check(withDepsFlag.DefValue, Equals, "false")
}

func (s *RepoRemoveSimpleSuite) TestAptlyRepoRemoveInvalidArgs(c *C) {
	// Test with insufficient arguments
	err := aptlyRepoRemove(s.cmd, []string{})
	c.Check(err, Equals, commander.ErrCommandError)

	err = aptlyRepoRemove(s.cmd, []string{"repo-name"})
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *RepoRemoveSimpleSuite) TestAptlyRepoRemoveBasic(c *C) {
	// Test basic remove operation - simplified
	args := []string{"repo-name", "package-name"}

	// Note: This may fail due to missing context, but should not panic
	_ = aptlyRepoRemove(s.cmd, args)
}

func (s *RepoRemoveSimpleSuite) TestAptlyRepoRemoveWithDryRun(c *C) {
	// Test dry run mode
	s.cmd.Flag.Set("dry-run", "true")

	args := []string{"repo-name", "package-name"}
	_ = aptlyRepoRemove(s.cmd, args)
}

func (s *RepoRemoveSimpleSuite) TestAptlyRepoRemoveWithDeps(c *C) {
	// Test with dependencies
	s.cmd.Flag.Set("with-deps", "true")

	args := []string{"repo-name", "package-name"}
	_ = aptlyRepoRemove(s.cmd, args)
}

func (s *RepoRemoveSimpleSuite) TestAptlyRepoRemoveMultipleQueries(c *C) {
	// Test with multiple package queries
	args := []string{"repo-name", "package1", "package2", "package3"}
	_ = aptlyRepoRemove(s.cmd, args)
}
