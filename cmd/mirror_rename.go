package cmd

import (
	"fmt"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
)

func aptlyMirrorRename(cmd *commander.Command, args []string) error {
	var (
		err  error
		repo *deb.RemoteRepo
	)

	if len(args) != 2 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	oldName, newName := args[0], args[1]

	collectionFactory := context.NewCollectionFactory()
	repo, err = collectionFactory.RemoteRepoCollection().ByName(oldName)
	if err != nil {
		return fmt.Errorf("unable to rename: %s", err)
	}

	err = repo.CheckLock()
	if err != nil {
		return fmt.Errorf("unable to rename: %s", err)
	}

	_, err = collectionFactory.RemoteRepoCollection().ByName(newName)
	if err == nil {
		return fmt.Errorf("unable to rename: mirror %s already exists", newName)
	}

	repo.Name = newName
	err = collectionFactory.RemoteRepoCollection().Update(repo)
	if err != nil {
		return fmt.Errorf("unable to rename: %s", err)
	}

	fmt.Printf("\nMirror %s -> %s has been successfully renamed.\n", oldName, newName)

	return err
}

func makeCmdMirrorRename() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyMirrorRename,
		UsageLine: "rename <old-name> <new-name>",
		Short:     "renames mirror",
		Long: `
Command changes name of the mirror.Mirror name should be unique.

Example:

  $ aptly mirror rename wheezy-min wheezy-main
`,
	}

	return cmd

}
