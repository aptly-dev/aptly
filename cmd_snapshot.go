package main

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
	"strings"
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

	fmt.Printf("\nSnapshot %s successfully created.\nYou can run 'aptly publish snapshot %s' to publish snapshot as Debian repository.\n", snapshot.Name, snapshot.Name)

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
	snapshotCollection.ForEach(func(snapshot *debian.Snapshot) error {
		fmt.Printf(" * %s\n", snapshot)
		return nil
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
	fmt.Printf("Packages:\n")

	packageCollection := debian.NewPackageCollection(context.database)

	err = snapshot.RefList().ForEach(func(key []byte) error {
		p, err := packageCollection.ByKey(key)
		if err != nil {
			return err
		}
		fmt.Printf("  %s\n", p)
		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	return err
}

func aptlySnapshotVerify(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 1 {
		cmd.Usage()
		return err
	}

	snapshotCollection := debian.NewSnapshotCollection(context.database)
	packageCollection := debian.NewPackageCollection(context.database)

	snapshots := make([]*debian.Snapshot, len(args))
	for i := range snapshots {
		snapshots[i], err = snapshotCollection.ByName(args[i])
		if err != nil {
			return fmt.Errorf("unable to verify: %s", err)
		}

		err = snapshotCollection.LoadComplete(snapshots[i])
		if err != nil {
			return fmt.Errorf("unable to verify: %s", err)
		}
	}

	packageList, err := debian.NewPackageListFromRefList(snapshots[0].RefList(), packageCollection)
	if err != nil {
		fmt.Errorf("unable to load packages: %s", err)
	}

	packageIndexedList := debian.NewPackageIndexedList()
	packageIndexedList.Append(packageList)

	for i := 1; i < len(snapshots); i++ {
		pL, err := debian.NewPackageListFromRefList(snapshots[i].RefList(), packageCollection)
		if err != nil {
			fmt.Errorf("unable to load packages: %s", err)
		}

		packageIndexedList.Append(pL)
	}

	packageIndexedList.PrepareIndex()

	var architecturesList []string

	architectures := cmd.Flag.Lookup("architectures").Value.String()
	if architectures != "" {
		architecturesList = strings.Split(architectures, ",")
	} else {
		architecturesList = packageList.Architectures()
	}

	if len(architecturesList) == 0 {
		return fmt.Errorf("unable to determine list of architectures, please specify explicitly")
	}

	missing, err := packageList.VerifyDependencies(0, architecturesList, packageIndexedList)
	if err != nil {
		return fmt.Errorf("unable to verify dependencies: %s", err)
	}

	if len(missing) == 0 {
		fmt.Printf("All dependencies are satisfied.\n")
	} else {
		fmt.Printf("Missing dependencies (%d):\n", len(missing))
		for _, dep := range missing {
			fmt.Printf("  %s\n", dep.String())
		}
	}

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

func makeCmdSnapshotVerify() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotVerify,
		UsageLine: "verify",
		Short:     "verifies that dependencies are satisfied in snapshot",
		Long: `
Verify does depenency resolution in snapshot, possibly using additional snapshots as dependency sources.
All unsatisfied dependencies are returned.

ex:
  $ aptly snapshot verify <name> [<source> ...]
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-verify", flag.ExitOnError),
	}

	cmd.Flag.String("architectures", "", "list of architectures to publish (comma-separated)")

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
			makeCmdSnapshotVerify(),
			//makeCmdSnapshotDestroy(),
		},
		Flag: *flag.NewFlagSet("aptly-snapshot", flag.ExitOnError),
	}
}
