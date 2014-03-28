package cmd

import (
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func makeCmdRepo() *commander.Command {
	return &commander.Command{
		UsageLine: "repo",
		Short:     "manage local package repositories",
		Subcommands: []*commander.Command{
			makeCmdRepoAdd(),
			makeCmdRepoCopy(),
			makeCmdRepoCreate(),
			makeCmdRepoDrop(),
			makeCmdRepoEdit(),
			makeCmdRepoImport(),
			makeCmdRepoList(),
			makeCmdRepoMove(),
			makeCmdRepoRemove(),
			makeCmdRepoShow(),
		},
		Flag: *flag.NewFlagSet("aptly-repo", flag.ExitOnError),
	}
}
