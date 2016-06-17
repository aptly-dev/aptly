package cmd

import (
	"fmt"
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

	collection := context.CollectionFactory().SnapshotCollection()

	snapshot, err := collection.ByName(name)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	err = collection.LoadComplete(snapshot)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}
	descr, err := snapshot.DescriptionWithSources(collection)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	fmt.Printf("Name: %s\n", snapshot.Name)
	fmt.Printf("Created At: %s\n", snapshot.CreatedAt.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("Description: %s\n", descr)
	fmt.Printf("Number of packages: %d\n", snapshot.NumPackages())

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
