package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/commander"
	"sort"
)

func aptlySnapshotList(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 0 {
		cmd.Usage()
		return err
	}

	raw := cmd.Flag.Lookup("raw").Value.Get().(bool)

	snapshots := make([]string, context.CollectionFactory().SnapshotCollection().Len())

	i := 0
	context.CollectionFactory().SnapshotCollection().ForEach(func(snapshot *deb.Snapshot) error {
		if raw {
			snapshots[i] = snapshot.Name
		} else {
			snapshots[i] = snapshot.String()
		}
		i++
		return nil
	})

	sort.Strings(snapshots)

	if raw {
		for _, snapshot := range snapshots {
			fmt.Printf("%s\n", snapshot)
		}
	} else {
		if len(snapshots) > 0 {
			fmt.Printf("List of snapshots:\n")

			for _, snapshot := range snapshots {
				fmt.Printf(" * %s\n", snapshot)
			}

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

	return cmd
}
