package cmd

import (
	"github.com/smira/commander"
)

func makeCmdScript() *commander.Command {
	return &commander.Command{
		UsageLine: "script",
		Short:     "runs aptly scripts",
		Subcommands: []*commander.Command{
			makeCmdScriptRun(),
		},
	}
}
