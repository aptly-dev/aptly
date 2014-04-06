package main

import (
	"fmt"
	"github.com/smira/aptly/cmd"
	"os"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fatal, ok := r.(*cmd.FatalError)
			if !ok {
				panic(r)
			}
			fmt.Println("ERROR:", fatal.Message)
			os.Exit(fatal.ReturnCode)
		}
	}()

	command := cmd.RootCommand()

	flags, args, err := command.ParseFlags(os.Args[1:])
	if err != nil {
		cmd.Fatal(err)
	}

	err = cmd.InitContext(flags)
	if err != nil {
		cmd.Fatal(err)
	}
	defer cmd.ShutdownContext()

	err = command.Dispatch(args)
	if err != nil {
		cmd.Fatal(err)
	}
}
