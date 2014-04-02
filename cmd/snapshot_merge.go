package cmd

import (
	"fmt"
	"github.com/smira/aptly/debian"
	"github.com/smira/commander"
	"strings"
)

func aptlySnapshotMerge(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 2 {
		cmd.Usage()
		return err
	}

	sources := make([]*debian.Snapshot, len(args)-1)

	for i := 0; i < len(args)-1; i++ {
		sources[i], err = context.collectionFactory.SnapshotCollection().ByName(args[i+1])
		if err != nil {
			return fmt.Errorf("unable to load snapshot: %s", err)
		}

		err = context.collectionFactory.SnapshotCollection().LoadComplete(sources[i])
		if err != nil {
			return fmt.Errorf("unable to load snapshot: %s", err)
		}
	}

	result := sources[0].RefList()

	for i := 1; i < len(sources); i++ {
		result = result.Merge(sources[i].RefList(), true)
	}

	sourceDescription := make([]string, len(sources))
	for i, s := range sources {
		sourceDescription[i] = fmt.Sprintf("'%s'", s.Name)
	}

	// Create <destination> snapshot
	destination := debian.NewSnapshotFromRefList(args[0], sources, result,
		fmt.Sprintf("Merged from sources: %s", strings.Join(sourceDescription, ", ")))

	err = context.collectionFactory.SnapshotCollection().Add(destination)
	if err != nil {
		return fmt.Errorf("unable to create snapshot: %s", err)
	}

	fmt.Printf("\nSnapshot %s successfully created.\nYou can run 'aptly publish snapshot %s' to publish snapshot as Debian repository.\n", destination.Name, destination.Name)

	return err
}

func makeCmdSnapshotMerge() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotMerge,
		UsageLine: "merge <destination> <source> [<source>...]",
		Short:     "merges snapshots",
		Long: `
Merge merges several <source> snapshots into one <destination> snapshot.
Merge happens from left to right. Packages with the same name-architecture
pair are replaced during merge (package from latest snapshot on the list
wins). If run with only one source snapshot, merge copies <source> into
<destination>.

Example:

    $ aptly snapshot merge wheezy-w-backports wheezy-main wheezy-backports
`,
	}

	return cmd
}
