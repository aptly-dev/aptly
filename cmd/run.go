package cmd

import (
	"fmt"
	"github.com/smira/commander"
)

// Run runs single command starting from root cmd with args, optionally initializing context
func Run(cmd *commander.Command, cmdArgs []string, initContext bool) (returnCode int) {
	defer func() {
		if r := recover(); r != nil {
			fatal, ok := r.(*FatalError)
			if !ok {
				panic(r)
			}
			fmt.Println("ERROR:", fatal.Message)
			returnCode = fatal.ReturnCode
		}
	}()

	returnCode = 0

	flags, args, err := cmd.ParseFlags(cmdArgs)
	if err != nil {
		Fatal(err)
	}

	if initContext {
		err = InitContext(flags)
		if err != nil {
			Fatal(err)
		}
		defer ShutdownContext()
	}

	context.UpdateFlags(flags)

	err = cmd.Dispatch(args)
	if err != nil {
		Fatal(err)
	}

	return
}
