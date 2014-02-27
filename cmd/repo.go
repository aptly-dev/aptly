package cmd

import (
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
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
			makeCmdRepoImport(),
			makeCmdRepoList(),
			makeCmdRepoMove(),
			makeCmdRepoRemove(),
			makeCmdRepoShow(),
		},
		Flag: *flag.NewFlagSet("aptly-repo", flag.ExitOnError),
	}
}
