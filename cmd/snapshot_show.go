package cmd

import (
	"fmt"

	"github.com/smira/aptly/deb"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlySnapshotShow(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	name := args[0]

	snapshot, err := context.CollectionFactory().SnapshotCollection().ByName(name)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	err = context.CollectionFactory().SnapshotCollection().LoadComplete(snapshot)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	fmt.Printf("Name: %s\n", snapshot.Name)
	fmt.Printf("Created At: %s\n", snapshot.CreatedAt.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("Description: %s\n", snapshot.Description)
	fmt.Printf("Number of packages: %d\n", snapshot.NumPackages())
	if len(snapshot.SourceIDs) > 0 {
		fmt.Printf("Sources:\n")
		for _, sourceID := range snapshot.SourceIDs {
			var name string
			if snapshot.SourceKind == deb.SourceSnapshot {
				var source *deb.Snapshot
				source, err = context.CollectionFactory().SnapshotCollection().ByUUID(sourceID)
				if err != nil {
					continue
				}
				name = source.Name
			} else if snapshot.SourceKind == deb.SourceLocalRepo {
				var source *deb.LocalRepo
				source, err = context.CollectionFactory().LocalRepoCollection().ByUUID(sourceID)
				if err != nil {
					continue
				}
				name = source.Name
			} else if snapshot.SourceKind == deb.SourceRemoteRepo {
				var source *deb.RemoteRepo
				source, err = context.CollectionFactory().RemoteRepoCollection().ByUUID(sourceID)
				if err != nil {
					continue
				}
				name = source.Name
			}

			if name != "" {
				fmt.Printf("  %s [%s]\n", name, snapshot.SourceKind)
			}
		}
	}

	withPackages := context.Flags().Lookup("with-packages").Value.Get().(bool)
	if withPackages {
		ListPackagesRefList(snapshot.RefList())
	}

	return err
}

func makeCmdSnapshotShow() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotShow,
		UsageLine: "show <name>",
		Short:     "shows details about snapshot",
		Long: `
Command show displays full information about a snapshot.

Example:

    $ aptly snapshot show wheezy-main
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-show", flag.ExitOnError),
	}

	cmd.Flag.Bool("with-packages", false, "show list of packages")

	return cmd
}
