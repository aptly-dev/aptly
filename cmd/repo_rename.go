package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/commander"
)

func aptlyRepoRename(cmd *commander.Command, args []string) error {
	var (
		err  error
		repo *deb.LocalRepo
	)

	if len(args) != 2 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	oldName, newName := args[0], args[1]

	repo, err = context.CollectionFactory().LocalRepoCollection().ByName(oldName)
	if err != nil {
		return fmt.Errorf("unable to rename: %s", err)
	}

	_, err = context.CollectionFactory().LocalRepoCollection().ByName(newName)
	if err == nil {
		return fmt.Errorf("unable to rename: local repo %s already exists", newName)
	}

	repo.Name = newName
	err = context.CollectionFactory().LocalRepoCollection().Update(repo)
	if err != nil {
		return fmt.Errorf("unable to rename: %s", err)
	}

	fmt.Printf("\nLocal repo %s -> %s has been successfully renamed.\n", oldName, newName)

	return err
}

func makeCmdRepoRename() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRepoRename,
		UsageLine: "rename <old-name> <new-name>",
		Short:     "renames local repository",
		Long: `
Command changes name of the local repo. Local repo name should be unique.

Example:

  $ aptly repo rename wheezy-min wheezy-main
`,
	}

	return cmd

}
