package cmd

import (
	"fmt"
	"github.com/smira/aptly/debian"
	"github.com/smira/commander"
	"sort"
)

func aptlySnapshotList(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 0 {
		cmd.Usage()
		return err
	}

	if context.collectionFactory.SnapshotCollection().Len() > 0 {
		fmt.Printf("List of snapshots:\n")

		snapshots := make([]string, context.collectionFactory.SnapshotCollection().Len())

		i := 0
		context.collectionFactory.SnapshotCollection().ForEach(func(snapshot *debian.Snapshot) error {
			snapshots[i] = snapshot.String()
			i++
			return nil
		})

		sort.Strings(snapshots)
		for _, snapshot := range snapshots {
			fmt.Printf(" * %s\n", snapshot)
		}

		fmt.Printf("\nTo get more information about snapshot, run `aptly snapshot show <name>`.\n")
	} else {
		fmt.Printf("\nNo snapshots found, create one with `aptly snapshot create...`.\n")
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

	return cmd
}
