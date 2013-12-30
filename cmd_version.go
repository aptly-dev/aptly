package main

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func aptlyVersion(cmd *commander.Command, args []string) error {
	fmt.Printf("aptly version: %s\n", Version)
	return nil
}

func makeCmdVersion() *commander.Command {
	return &commander.Command{
		Run:       aptlyVersion,
		UsageLine: "version",
		Short:     "display version",
		Long: `
Shows aptly version.

ex:
  $ aptly version
`,
		Flag: *flag.NewFlagSet("aptly-version", flag.ExitOnError),
	}
}
