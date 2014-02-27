package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
)

func aptlySnapshotCreate(cmd *commander.Command, args []string) error {
	var (
		err      error
		snapshot *debian.Snapshot
	)

	if len(args) == 4 && args[1] == "from" && args[2] == "mirror" {
		// aptly snapshot create snap from mirror mirror
		repoName, snapshotName := args[3], args[0]

		repoCollection := debian.NewRemoteRepoCollection(context.database)
		repo, err := repoCollection.ByName(repoName)
		if err != nil {
			return fmt.Errorf("unable to create snapshot: %s", err)
		}

		err = repoCollection.LoadComplete(repo)
		if err != nil {
			return fmt.Errorf("unable to create snapshot: %s", err)
		}

		snapshot, err = debian.NewSnapshotFromRepository(snapshotName, repo)
		if err != nil {
			return fmt.Errorf("unable to create snapshot: %s", err)
		}
	} else if len(args) == 4 && args[1] == "from" && args[2] == "repo" {
		// aptly snapshot create snap from repo repo
		localRepoName, snapshotName := args[3], args[0]

		localRepoCollection := debian.NewLocalRepoCollection(context.database)
		repo, err := localRepoCollection.ByName(localRepoName)
		if err != nil {
			return fmt.Errorf("unable to create snapshot: %s", err)
		}

		err = localRepoCollection.LoadComplete(repo)
		if err != nil {
			return fmt.Errorf("unable to create snapshot: %s", err)
		}

		snapshot, err = debian.NewSnapshotFromLocalRepo(snapshotName, repo)
		if err != nil {
			return fmt.Errorf("unable to create snapshot: %s", err)
		}
	} else if len(args) == 2 && args[1] == "empty" {
		// aptly snapshot create snap empty
		snapshotName := args[0]

		packageList := debian.NewPackageList()

		snapshot = debian.NewSnapshotFromPackageList(snapshotName, nil, packageList, "Created as empty")
	} else {
		cmd.Usage()
		return err
	}

	snapshotCollection := debian.NewSnapshotCollection(context.database)

	err = snapshotCollection.Add(snapshot)
	if err != nil {
		return fmt.Errorf("unable to add snapshot: %s", err)
	}

	fmt.Printf("\nSnapshot %s successfully created.\nYou can run 'aptly publish snapshot %s' to publish snapshot as Debian repository.\n", snapshot.Name, snapshot.Name)

	return err
}

func makeCmdSnapshotCreate() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotCreate,
		UsageLine: "create <name> from mirror <mirror-name> | from repo <repo-name> | create <name> empty",
		Short:     "creates immutable snapshot of mirror (local repo) contents",
		Long: `
Command create .. from mirror makes persistent immutable snapshot of remote repository mirror. Snapshot could be
published or further modified using merge, pull and other aptly features.

Command create .. from repo makes persistent immutable snapshot of local repository. Snapshot could be processed
as mirror snapshots, and mixed with snapshots of remote mirrors.

Command create .. empty creates empty snapshot that could be used as a basis for snapshot pull operations, for example.
As snapshots are immutable, creating one empty snapshot should be enough.

ex.
  $ aptly snapshot create wheezy-main-today from mirror wheezy-main
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-create", flag.ExitOnError),
	}

	return cmd

}
