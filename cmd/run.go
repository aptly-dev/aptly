package cmd

import (
	"fmt"
	"os"

	ctx "github.com/smira/aptly/context"
	"github.com/smira/commander"
)

// Run runs single command starting from root cmd with args, optionally initializing context
func Run(cmd *commander.Command, cmdArgs []string, initContext bool) (returnCode int) {
	defer func() {
		if r := recover(); r != nil {
			fatal, ok := r.(*ctx.FatalError)
			if !ok {
				panic(r)
			}
			fmt.Fprintln(os.Stderr, "ERROR:", fatal.Message)
			returnCode = fatal.ReturnCode
		}
	}()

	returnCode = 0

	flags, args, err := cmd.ParseFlags(cmdArgs)
	if err != nil {
		ctx.Fatal(err)
	}

	if initContext {
		err = InitContext(flags)
		if err != nil {
			ctx.Fatal(err)
		}
		defer ShutdownContext()
	}

	context.UpdateFlags(flags)

	err = cmd.Dispatch(args)
	if err != nil {
		ctx.Fatal(err)
	}

	return
}
