package cmd

import (
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func makeCmdDbCleanup() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyDbCleanup,
		UsageLine: "cleanup",
		Short:     "remove unused entries in DB and unreferenced files in the pool",
		Long: `
Database cleanup removes information about unreferenced packages and removes
files in the package pool that aren't used by packages anymore

ex:
  $ aptly db cleanup
`,
		Flag: *flag.NewFlagSet("aptly-db-cleanup", flag.ExitOnError),
	}

	return cmd
}

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
