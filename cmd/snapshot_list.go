package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
)

func aptlySnapshotList(cmd *commander.Command, args []string) error {
	if len(args) != 0 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	jsonFlag := cmd.Flag.Lookup("json").Value.Get().(bool)

	if jsonFlag {
		return aptlySnapshotListJson(cmd, args)
	}

	return aptlySnapshotListTxt(cmd, args)
}

func aptlySnapshotListTxt(cmd *commander.Command, args []string) error {
	var err error

	raw := cmd.Flag.Lookup("raw").Value.Get().(bool)
	sortMethodString := cmd.Flag.Lookup("sort").Value.Get().(string)

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.SnapshotCollection()

	if raw {
		collection.ForEachSorted(sortMethodString, func(snapshot *deb.Snapshot) error {
			fmt.Printf("%s\n", snapshot.Name)
			return nil
		})
	} else {
		if collection.Len() > 0 {
			fmt.Printf("List of snapshots:\n")

			err = collection.ForEachSorted(sortMethodString, func(snapshot *deb.Snapshot) error {
				fmt.Printf(" * %s\n", snapshot.String())
				return nil
			})

			if err != nil {
				return err
			}

			fmt.Printf("\nTo get more information about snapshot, run `aptly snapshot show <name>`.\n")
		} else {
			fmt.Printf("\nNo snapshots found, create one with `aptly snapshot create...`.\n")
		}
	}

	return err
}

func aptlySnapshotListJson(cmd *commander.Command, args []string) error {
	var err error

	sortMethodString := cmd.Flag.Lookup("sort").Value.Get().(string)

	collection := context.CollectionFactory().SnapshotCollection()

	jsonSnapshots := make([]*deb.Snapshot, collection.Len())
	i := 0
	collection.ForEachSorted(sortMethodString, func(snapshot *deb.Snapshot) error {
		jsonSnapshots[i] = snapshot
		i++
		return nil
	})
	if output, e := json.MarshalIndent(jsonSnapshots, "", "  "); e == nil {
		fmt.Println(string(output))
	} else {
		err = e
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

	cmd.Flag.Bool("json", false, "display list in JSON format")
	cmd.Flag.Bool("raw", false, "display list in machine-readable format")
	cmd.Flag.String("sort", "name", "display list in 'name' or creation 'time' order")

	return cmd
}
