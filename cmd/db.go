package cmd

import (
	"github.com/smira/commander"
)

func makeCmdDB() *commander.Command {
	return &commander.Command{
		UsageLine: "db",
		Short:     "manage aptly's internal database and package pool",
		Subcommands: []*commander.Command{
			makeCmdDBCleanup(),
			makeCmdDBRecover(),
		},
	}
}
