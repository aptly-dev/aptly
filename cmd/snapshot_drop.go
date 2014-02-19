package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
)

func aptlySnapshotDrop(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return err
	}

	name := args[0]

	snapshotCollection := debian.NewSnapshotCollection(context.database)
	snapshot, err := snapshotCollection.ByName(name)
	if err != nil {
		return fmt.Errorf("unable to drop: %s", err)
	}

	publishedRepoCollection := debian.NewPublishedRepoCollection(context.database)
	published := publishedRepoCollection.BySnapshot(snapshot)

	if len(published) > 0 {
		fmt.Printf("Snapshot `%s` is published currently:\n", snapshot.Name)
		for _, repo := range published {
			err = publishedRepoCollection.LoadComplete(repo, snapshotCollection)
			if err != nil {
				return fmt.Errorf("unable to load published: %s", err)
			}
			fmt.Printf(" * %s\n", repo)
		}

		return fmt.Errorf("unable to drop: snapshot is published")
	}

	force := cmd.Flag.Lookup("force").Value.Get().(bool)
	if !force {
		snapshots := snapshotCollection.BySnapshotSource(snapshot)
		if len(snapshots) > 0 {
			fmt.Printf("Snapshot `%s` was used as a source in following snapshots:\n", snapshot.Name)
			for _, snap := range snapshots {
				fmt.Printf(" * %s\n", snap)
			}

			return fmt.Errorf("won't delete snapshot that was used as source for other snapshots, use -force to override")
		}
	}

	err = snapshotCollection.Drop(snapshot)
	if err != nil {
		return fmt.Errorf("unable to drop: %s", err)
	}

	fmt.Printf("Snapshot `%s` has been dropped.\n", snapshot.Name)

	return err
}

func makeCmdSnapshotDrop() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotDrop,
		UsageLine: "drop <name>",
		Short:     "delete snapshot",
		Long: `
Drop removes information about snapshot. If snapshot is published,
it can't be dropped.

ex.
    $ aptly snapshot drop wheezy-main
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-drop", flag.ExitOnError),
	}

	cmd.Flag.Bool("force", false, "remove snapshot even if it was used as source for other snapshots")

	return cmd
}
