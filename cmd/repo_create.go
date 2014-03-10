package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
)

func aptlyRepoCreate(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return err
	}

	repo := debian.NewLocalRepo(args[0], cmd.Flag.Lookup("comment").Value.String())

	localRepoCollection := debian.NewLocalRepoCollection(context.database)

	err = localRepoCollection.Add(repo)
	if err != nil {
		return fmt.Errorf("unable to add local repo: %s", err)
	}

	fmt.Printf("\nLocal repo %s successfully added.\nYou can run 'aptly repo add %s ...' to add packages to repository.\n", repo, repo.Name)
	return err
}

func makeCmdRepoCreate() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRepoCreate,
		UsageLine: "create <name>",
		Short:     "create local repository",
		Long: `
Create local package repository. Repository would be empty when
created, packages could be added from files, copied or moved from
another local repository or imported from the mirror.

Example:

  $ aptly repo create testing
`,
		Flag: *flag.NewFlagSet("aptly-repo-create", flag.ExitOnError),
	}

	cmd.Flag.String("comment", "", "any text that would be used to described local repository")

	return cmd
}
