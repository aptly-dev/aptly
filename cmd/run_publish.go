package cmd

import (
	//"fmt"
	"github.com/smira/commander"
)


func aptlyRunPublish(cmd *commander.Command, args []string) error {

	return nil

}



func makeCmdRunPublish() *commander.Command {
  cmd := &commander.Command{
    Run:       aptlyRunPublish,
    UsageLine: "publish",
    Short:     "publish packages from a configuration file",
    Long: `
Publish packages from repos and mirrors as defined in a configuration file referenced
in aptly.conf.

ex:
  $ aptly run publish
`,
  }
  return cmd
}
