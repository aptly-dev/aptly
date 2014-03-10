package cmd

import (
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func makeCmdDb() *commander.Command {
	return &commander.Command{
		UsageLine: "db",
		Short:     "manage aptly's internal database and package pool",
		Subcommands: []*commander.Command{
			makeCmdDbCleanup(),
		},
		Flag: *flag.NewFlagSet("aptly-db", flag.ExitOnError),
	}
}
