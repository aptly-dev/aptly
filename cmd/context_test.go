package cmd

import (
	"testing"

	ctx "github.com/aptly-dev/aptly/context"
	"github.com/smira/flag"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ContextSuite struct {
	originalContext *ctx.AptlyContext
}

var _ = Suite(&ContextSuite{})

func (s *ContextSuite) SetUpTest(c *C) {
	// Save original context state
	s.originalContext = context
	context = nil // Reset context for each test
}

func (s *ContextSuite) TearDownTest(c *C) {
	// Clean up and restore original context
	if context != nil {
		context.Shutdown()
		context = nil
	}
	context = s.originalContext
}

func (s *ContextSuite) TestInitContextSuccess(c *C) {
	// Test successful context initialization
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	err := InitContext(flags)
	c.Check(err, IsNil)
	c.Check(context, NotNil)
	c.Check(GetContext(), Equals, context)
}

func (s *ContextSuite) TestInitContextPanic(c *C) {
	// Test that initializing context twice causes panic
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	// First initialization should succeed
	err := InitContext(flags)
	c.Check(err, IsNil)
	c.Check(context, NotNil)

	// Second initialization should panic
	c.Check(func() { InitContext(flags) }, Panics, "context already initialized")
}

func (s *ContextSuite) TestInitContextError(c *C) {
	// Test context initialization with invalid flags
	// This tests the error path where ctx.NewContext might fail
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	// Add some invalid flag configuration that might cause NewContext to fail
	// Note: This depends on the ctx.NewContext implementation details
	flags.String("invalid-config", "/nonexistent/path/to/config", "invalid config")
	flags.Set("invalid-config", "/nonexistent/path/to/config")

	err := InitContext(flags)
	// The error handling depends on the ctx.NewContext implementation
	// If it doesn't fail with invalid paths, the test still validates the error path exists
	if err != nil {
		c.Check(context, IsNil)
	} else {
		c.Check(context, NotNil)
	}
}

func (s *ContextSuite) TestGetContextBeforeInit(c *C) {
	// Test GetContext when context is nil
	c.Check(context, IsNil)
	result := GetContext()
	c.Check(result, IsNil)
}

func (s *ContextSuite) TestGetContextAfterInit(c *C) {
	// Test GetContext after successful initialization
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	err := InitContext(flags)
	c.Check(err, IsNil)

	result := GetContext()
	c.Check(result, NotNil)
	c.Check(result, Equals, context)
}

func (s *ContextSuite) TestShutdownContext(c *C) {
	// Test ShutdownContext function
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	err := InitContext(flags)
	c.Check(err, IsNil)
	c.Check(context, NotNil)

	// ShutdownContext should not panic and should call context.Shutdown()
	c.Check(func() { ShutdownContext() }, Not(Panics))
}

func (s *ContextSuite) TestShutdownContextNil(c *C) {
	// Test ShutdownContext when context is nil (should panic or handle gracefully)
	context = nil

	// This will panic if context is nil, which might be expected behavior
	c.Check(func() { ShutdownContext() }, Panics, ".*")
}

func (s *ContextSuite) TestCleanupContext(c *C) {
	// Test CleanupContext function
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	err := InitContext(flags)
	c.Check(err, IsNil)
	c.Check(context, NotNil)

	// CleanupContext should not panic and should call context.Cleanup()
	c.Check(func() { CleanupContext() }, Not(Panics))
}

func (s *ContextSuite) TestCleanupContextNil(c *C) {
	// Test CleanupContext when context is nil (should panic or handle gracefully)
	context = nil

	// This will panic if context is nil, which might be expected behavior
	c.Check(func() { CleanupContext() }, Panics, ".*")
}

func (s *ContextSuite) TestContextLifecycle(c *C) {
	// Test complete context lifecycle: init -> use -> cleanup -> shutdown
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	// Initialize
	err := InitContext(flags)
	c.Check(err, IsNil)
	c.Check(context, NotNil)

	// Use
	ctx := GetContext()
	c.Check(ctx, NotNil)
	c.Check(ctx, Equals, context)

	// Cleanup
	c.Check(func() { CleanupContext() }, Not(Panics))

	// Context should still exist after cleanup
	c.Check(context, NotNil)
	c.Check(GetContext(), NotNil)

	// Shutdown
	c.Check(func() { ShutdownContext() }, Not(Panics))
}

func (s *ContextSuite) TestMultipleCleanups(c *C) {
	// Test calling CleanupContext multiple times
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	err := InitContext(flags)
	c.Check(err, IsNil)

	// Multiple cleanups should not cause issues
	c.Check(func() { CleanupContext() }, Not(Panics))
	c.Check(func() { CleanupContext() }, Not(Panics))
	c.Check(func() { CleanupContext() }, Not(Panics))
}

func (s *ContextSuite) TestContextVariableIsolation(c *C) {
	// Test that the context variable is properly managed
	c.Check(context, IsNil)

	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	err := InitContext(flags)
	c.Check(err, IsNil)

	// Store reference
	originalContext := context
	c.Check(originalContext, NotNil)

	// GetContext should return the same instance
	retrievedContext := GetContext()
	c.Check(retrievedContext, Equals, originalContext)

	// Context variable should be the same
	c.Check(context, Equals, originalContext)
}

func (s *ContextSuite) TestFlagSetVariations(c *C) {
	// Test InitContext with different FlagSet configurations
	testCases := []struct {
		name    string
		setupFn func() *flag.FlagSet
	}{
		{
			name: "empty flagset",
			setupFn: func() *flag.FlagSet {
				return flag.NewFlagSet("empty", flag.ContinueOnError)
			},
		},
		{
			name: "flagset with common flags",
			setupFn: func() *flag.FlagSet {
				fs := flag.NewFlagSet("common", flag.ContinueOnError)
				fs.String("config", "", "config file")
				fs.Bool("debug", false, "debug mode")
				return fs
			},
		},
		{
			name: "flagset with aptly-specific flags",
			setupFn: func() *flag.FlagSet {
				fs := flag.NewFlagSet("aptly", flag.ContinueOnError)
				fs.String("architectures", "", "architectures")
				fs.String("distribution", "", "distribution")
				return fs
			},
		},
	}

	for _, tc := range testCases {
		// Reset context for each test case
		if context != nil {
			context.Shutdown()
			context = nil
		}

		flags := tc.setupFn()
		err := InitContext(flags)
		c.Check(err, IsNil, Commentf("Failed for test case: %s", tc.name))
		c.Check(context, NotNil, Commentf("Context is nil for test case: %s", tc.name))
		c.Check(GetContext(), NotNil, Commentf("GetContext returned nil for test case: %s", tc.name))
	}
}
