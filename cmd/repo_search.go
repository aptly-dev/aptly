package cmd

import (
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func makeCmdRepoSearch() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotMirrorRepoSearch,
		UsageLine: "search <name> [<package-query>]",
		Short:     "search repo for packages matching query",
		Long: `
Command search displays list of packages in local repository that match package query

If query is not specified, all the packages are displayed.

Example:

    $ aptly repo search my-software '$Architecture (i386), Name (% *-dev)'
`,
		Flag: *flag.NewFlagSet("aptly-repo-show", flag.ExitOnError),
	}

	cmd.Flag.Bool("with-deps", false, "include dependencies into search results")
	cmd.Flag.String("format", "", "custom format for result printing")

	return cmd
}
