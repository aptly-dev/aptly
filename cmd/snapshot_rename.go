package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/commander"
)

func aptlySnapshotRename(cmd *commander.Command, args []string) error {
	var (
		err      error
		snapshot *deb.Snapshot
	)

	if len(args) != 2 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	oldName, newName := args[0], args[1]

	snapshot, err = context.CollectionFactory().SnapshotCollection().ByName(oldName)
	if err != nil {
		return fmt.Errorf("unable to rename: %s", err)
	}

	_, err = context.CollectionFactory().SnapshotCollection().ByName(newName)
	if err == nil {
		return fmt.Errorf("unable to rename: snapshot %s already exists", newName)
	}

	snapshot.Name = newName
	err = context.CollectionFactory().SnapshotCollection().Update(snapshot)
	if err != nil {
		return fmt.Errorf("unable to rename: %s", err)
	}

	fmt.Printf("\nSnapshot %s -> %s has been successfully renamed.\n", oldName, newName)

	return err
}

func makeCmdSnapshotRename() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotRename,
		UsageLine: "rename <old-name> <new-name>",
		Short:     "renames snapshot",
		Long: `
Command changes name of the snapshot. Snapshot name should be unique.

Example:

  $ aptly snapshot rename wheezy-min wheezy-main
`,
	}

	return cmd

}
