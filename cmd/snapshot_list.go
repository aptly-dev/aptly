package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/commander"
)

func aptlySnapshotList(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 0 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	raw := cmd.Flag.Lookup("raw").Value.Get().(bool)
	sortMethodString := cmd.Flag.Lookup("sort").Value.Get().(string)

	collection := context.CollectionFactory().SnapshotCollection()
	collection.Sort(sortMethodString)

	if raw {
		collection.ForEach(func(snapshot *deb.Snapshot) error {
			fmt.Printf("%s\n", snapshot.Name)
			return nil
		})
	} else {
		if collection.Len() > 0 {
			fmt.Printf("List of snapshots:\n")

			collection.ForEach(func(snapshot *deb.Snapshot) error {
				fmt.Printf(" * %s\n", snapshot.String())
				return nil
			})

			fmt.Printf("\nTo get more information about snapshot, run `aptly snapshot show <name>`.\n")
		} else {
			fmt.Printf("\nNo snapshots found, create one with `aptly snapshot create...`.\n")
		}
	}

	return err
}

func makeCmdSnapshotList() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotList,
		UsageLine: "list",
		Short:     "list snapshots",
		Long: `
Command list shows full list of snapshots created.

Example:

  $ aptly snapshot list
`,
	}

	cmd.Flag.Bool("raw", false, "display list in machine-readable format")
	cmd.Flag.String("sort", "name", "display list in 'name' or creation 'time' order")

	return cmd
}
