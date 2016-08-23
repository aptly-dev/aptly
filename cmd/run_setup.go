package cmd

import (
	//"fmt"
	"github.com/smira/commander"
)

func aptlyRunSetup(cmd *commander.Command, args []string) error {

  return nil
}


func makeCmdRunSetup() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRunSetup,
		UsageLine: "setup",
		Short:     "setup mirrors and repos from a configuration file",
		Long: `
Initialise or update mirrors and repos defined in a configuration file referenced
in aptly.conf.

ex:
  $ aptly run setup
`,
  }
	return cmd
}
