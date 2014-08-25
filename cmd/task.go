package cmd

import (
	"github.com/smira/commander"
)

func makeCmdTask() *commander.Command {
	return &commander.Command{
		UsageLine: "task",
		Short:     "manage aptly tasks",
		Subcommands: []*commander.Command{
			makeCmdTaskRun(),
		},
	}
}
