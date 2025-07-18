package cmd

import (
	ctx "github.com/aptly-dev/aptly/context"
	"github.com/smira/flag"
)

var context *ctx.AptlyContext

// ShutdownContext shuts context down
func ShutdownContext() {
	if context != nil {
		context.Shutdown()
	}
}

// CleanupContext does partial shutdown of context
func CleanupContext() {
	if context != nil {
		context.Cleanup()
	}
}

// InitContext initializes context with default settings
func InitContext(flags *flag.FlagSet) error {
	var err error

	if context != nil {
		panic("context already initialized")
	}

	context, err = ctx.NewContext(flags)

	return err
}

// GetContext gives access to the context
func GetContext() *ctx.AptlyContext {
	return context
}
