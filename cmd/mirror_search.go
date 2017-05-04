package cmd

import (
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func makeCmdMirrorSearch() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotMirrorRepoSearch,
		UsageLine: "search <name> [<package-query>]",
		Short:     "search mirror for packages matching query",
		Long: `
Command search displays list of packages in mirror that match package query

If query is not specified, all the packages are displayed.

Example:

    $ aptly mirror search wheezy-main '$Architecture (i386), Name (% *-dev)'
`,
		Flag: *flag.NewFlagSet("aptly-mirror-show", flag.ExitOnError),
	}

	cmd.Flag.Bool("with-deps", false, "include dependencies into search results")
	cmd.Flag.String("format", "", "custom format for result printing")

	return cmd
}
