package cmd

import (
	"strings"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
	. "gopkg.in/check.v1"
)

type ServeSimpleSuite struct {
	cmd               *commander.Command
	collectionFactory *deb.CollectionFactory
}

var _ = Suite(&ServeSimpleSuite{})

func (s *ServeSimpleSuite) SetUpTest(c *C) {
	s.cmd = makeCmdServe()
	s.collectionFactory = &deb.CollectionFactory{}

	// Set up required flags
	s.cmd.Flag.String("listen", ":8080", "host:port to listen on")
	s.cmd.Flag.Bool("no-lock", false, "don't lock the database")
}

func (s *ServeSimpleSuite) TestMakeCmdServe(c *C) {
	// Test command creation and basic properties
	cmd := makeCmdServe()
	c.Check(cmd, NotNil)
	c.Check(cmd.UsageLine, Equals, "serve")
	c.Check(cmd.Short, Equals, "start HTTP server to serve published repositories")
	c.Check(strings.Contains(cmd.Long, "Command serve starts embedded HTTP server"), Equals, true)

	// Test flags
	listenFlag := cmd.Flag.Lookup("listen")
	c.Check(listenFlag, NotNil)
	c.Check(listenFlag.DefValue, Equals, ":8080")

	noLockFlag := cmd.Flag.Lookup("no-lock")
	c.Check(noLockFlag, NotNil)
	c.Check(noLockFlag.DefValue, Equals, "false")
}

func (s *ServeSimpleSuite) TestAptlyServeInvalidArgs(c *C) {
	// Test with arguments (should not accept any)
	err := aptlyServe(s.cmd, []string{"invalid", "args"})
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *ServeSimpleSuite) TestAptlyServeBasic(c *C) {
	// Test basic serve operation - simplified
	args := []string{}

	// Note: This may fail due to missing context, but should not panic
	_ = aptlyServe(s.cmd, args)
}

func (s *ServeSimpleSuite) TestAptlyServeWithCustomListen(c *C) {
	// Test with custom listen address
	s.cmd.Flag.Set("listen", "localhost:9090")

	args := []string{}
	_ = aptlyServe(s.cmd, args)
}

func (s *ServeSimpleSuite) TestAptlyServeWithNoLock(c *C) {
	// Test with no-lock flag
	s.cmd.Flag.Set("no-lock", "true")

	args := []string{}
	_ = aptlyServe(s.cmd, args)
}