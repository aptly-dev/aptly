package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/aptly"
)

func aptlyVersion(cmd *commander.Command, args []string) error {
	fmt.Printf("aptly version: %s\n", aptly.Version)
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
