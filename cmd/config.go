package cmd

import (
	"github.com/smira/commander"
)

func makeCmdConfig() *commander.Command {
	return &commander.Command{
		UsageLine: "config",
		Short:     "manage aptly configuration",
		Subcommands: []*commander.Command{
			makeCmdConfigShow(),
		},
	}
}
