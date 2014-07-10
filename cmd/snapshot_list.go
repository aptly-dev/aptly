package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/commander"
	"sort"
)

// Snapshot sorting methods
const (
	SortName = iota
	SortTime
)

type snapshotListToSort struct {
	list       []*deb.Snapshot
	sortMethod int
}

func parseSortMethod(sortMethod string) (int, error) {
	switch sortMethod {
	case "time", "Time":
		return SortTime, nil
	case "name", "Name":
		return SortName, nil
	}

	return -1, fmt.Errorf("sorting method \"%s\" unknown", sortMethod)
}

func (s snapshotListToSort) Swap(i, j int) {
	s.list[i], s.list[j] = s.list[j], s.list[i]
}

func (s snapshotListToSort) Less(i, j int) bool {
	switch s.sortMethod {
	case SortName:
		return s.list[i].Name < s.list[j].Name
	case SortTime:
		return s.list[i].CreatedAt.Before(s.list[j].CreatedAt)
	}
	panic("unknown sort method")
}

func (s snapshotListToSort) Len() int {
	return len(s.list)
}

func aptlySnapshotList(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 0 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	raw := cmd.Flag.Lookup("raw").Value.Get().(bool)
	sortMethodString := cmd.Flag.Lookup("sort").Value.Get().(string)

	snapshotsToSort := &snapshotListToSort{}
	snapshotsToSort.list = make([]*deb.Snapshot, context.CollectionFactory().SnapshotCollection().Len())
	snapshotsToSort.sortMethod, err = parseSortMethod(sortMethodString)
	if err != nil {
		return err
	}

	i := 0
	context.CollectionFactory().SnapshotCollection().ForEach(func(snapshot *deb.Snapshot) error {
		snapshotsToSort.list[i] = snapshot
		i++

		return nil
	})

	sort.Sort(snapshotsToSort)

	if raw {
		for _, snapshot := range snapshotsToSort.list {
			fmt.Printf("%s\n", snapshot.Name)
		}
	} else {
		if len(snapshotsToSort.list) > 0 {
			fmt.Printf("List of snapshots:\n")

			for _, snapshot := range snapshotsToSort.list {
				fmt.Printf(" * %s\n", snapshot.String())
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
	cmd.Flag.String("sort", "name", "display list in 'name' or creation 'time' order")

	return cmd
}
