package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyRepoCreate(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	repo := deb.NewLocalRepo(args[0], context.flags.Lookup("comment").Value.String())
	repo.DefaultDistribution = context.flags.Lookup("distribution").Value.String()
	repo.DefaultComponent = context.flags.Lookup("component").Value.String()

	err = context.CollectionFactory().LocalRepoCollection().Add(repo)
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
	cmd.Flag.String("distribution", "", "default distribution when publishing")
	cmd.Flag.String("component", "main", "default component when publishing")

	return cmd
}
