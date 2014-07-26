package cmd

import (
	"github.com/smira/commander"
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
			makeCmdRepoRename(),
		},
	}
}
