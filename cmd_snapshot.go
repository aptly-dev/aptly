package main

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
)

func aptlySnapshotCreate(cmd *commander.Command, args []string) error {
	var err error

	if len(args) < 4 || args[1] != "from" || args[2] != "mirror" {
		cmd.Usage()
		return err
	}

	repoName, mirrorName := args[3], args[0]

	repoCollection := debian.NewRemoteRepoCollection(context.database)
	repo, err := repoCollection.ByName(repoName)
	if err != nil {
		return fmt.Errorf("unable to create snapshot: %s", err)
	}

	err = repoCollection.LoadComplete(repo)
	if err != nil {
		return fmt.Errorf("unable to create snapshot: %s", err)
	}

	snapshot, err := debian.NewSnapshotFromRepository(mirrorName, repo)
	if err != nil {
		return fmt.Errorf("unable to create snapshot: %s", err)
	}

	snapshotCollection := debian.NewSnapshotCollection(context.database)

	err = snapshotCollection.Add(snapshot)
	if err != nil {
		return fmt.Errorf("unable to add snapshot: %s", err)
	}

	fmt.Printf("\nSnapshot %s successfully created.\nYou can run 'aptly snapshot publish %s' to publish snapshot as Debian repository.\n", snapshot.Name, snapshot.Name)

	return err
}

func aptlySnapshotList(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 0 {
		cmd.Usage()
		return err
	}

	fmt.Printf("List of snapshots:\n")

	snapshotCollection := debian.NewSnapshotCollection(context.database)
	snapshotCollection.ForEach(func(snapshot *debian.Snapshot) {
		fmt.Printf(" * %s\n", snapshot)
	})

	fmt.Printf("\nTo get more information about snapshot, run `aptly snapshot show <name>`.\n")
	return err
}

func aptlySnapshotShow(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return err
	}

	name := args[0]

	snapshotCollection := debian.NewSnapshotCollection(context.database)
	snapshot, err := snapshotCollection.ByName(name)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	err = snapshotCollection.LoadComplete(snapshot)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	fmt.Printf("Name: %s\n", snapshot.Name)
	fmt.Printf("Created At: %s\n", snapshot.CreatedAt.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("Description: %s\n", snapshot.Description)
	fmt.Printf("Number of packages: %d\n", snapshot.NumPackages())

	return err
}

func makeCmdSnapshotCreate() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotCreate,
		UsageLine: "create",
		Short:     "creates snapshot out of any mirror",
		Long: `
Create makes persistent immutable snapshot of repository mirror state in givent moment of time.

ex:
  $ aptly snapshot create <name> from mirror <mirror-name>
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-create", flag.ExitOnError),
	}

	return cmd

}

func makeCmdSnapshotList() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotList,
		UsageLine: "list",
		Short:     "lists snapshots",
		Long: `
list shows full list of snapshots created.

ex:
  $ aptly snapshot list
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-list", flag.ExitOnError),
	}

	return cmd
}

func makeCmdSnapshotShow() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotShow,
		UsageLine: "show",
		Short:     "shows details about snapshot",
		Long: `
shows shows full information about snapshot.

ex:
  $ aptly snapshot show <name>
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-show", flag.ExitOnError),
	}

	return cmd
}

func makeCmdSnapshot() *commander.Command {
	return &commander.Command{
		UsageLine: "snapshot",
		Short:     "manage snapshots of repositories",
		Subcommands: []*commander.Command{
			makeCmdSnapshotCreate(),
			makeCmdSnapshotList(),
			makeCmdSnapshotShow(),
			//makeCmdSnapshotDestroy(),
			//makeCmdSnapshotPublish(),
		},
		Flag: *flag.NewFlagSet("aptly-snapshot", flag.ExitOnError),
	}
}
