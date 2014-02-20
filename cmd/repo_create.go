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
		Short:     "create new local package repository",
		Long: `
Creates new empty local package repository.

ex:
  $ aptly repo create testing
`,
		Flag: *flag.NewFlagSet("aptly-repo-create", flag.ExitOnError),
	}

	cmd.Flag.String("comment", "", "comment for the repository")

	return cmd
}
