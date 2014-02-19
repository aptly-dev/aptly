package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
	"sort"
)

func aptlySnapshotList(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 0 {
		cmd.Usage()
		return err
	}

	snapshotCollection := debian.NewSnapshotCollection(context.database)

	if snapshotCollection.Len() > 0 {
		fmt.Printf("List of snapshots:\n")

		snapshots := make([]string, snapshotCollection.Len())

		i := 0
		snapshotCollection.ForEach(func(snapshot *debian.Snapshot) error {
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
		Short:     "lists snapshots",
		Long: `
Command list shows full list of snapshots created.

ex:
  $ aptly snapshot list
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-list", flag.ExitOnError),
	}

	return cmd
}
