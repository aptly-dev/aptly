package cmd

import (
	"fmt"
	"os"
)

func Run(cmd_args []string, exitOnPanic bool) {

	defer func() {
		if r := recover(); r != nil {
			fatal, ok := r.(*FatalError)
			if !ok {
				panic(r)
			}
			fmt.Println("ERROR:", fatal.Message)
			if exitOnPanic {
				os.Exit(fatal.ReturnCode)
			}
		}
	}()

	command := RootCommand()

	flags, args, err := command.ParseFlags(cmd_args)
	if err != nil {
		Fatal(err)
	}

	err = InitContext(flags)
	if err != nil {
		Fatal(err)
	}

	defer ShutdownContext()

	err = command.Dispatch(args)
	if err != nil {
		Fatal(err)
	}

}
