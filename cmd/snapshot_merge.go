package cmd

import (
	"fmt"
	"strings"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
)

func aptlySnapshotMerge(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 2 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	collectionFactory := context.NewCollectionFactory()
	sources := make([]*deb.Snapshot, len(args)-1)

	for i := 0; i < len(args)-1; i++ {
		sources[i], err = collectionFactory.SnapshotCollection().ByName(args[i+1])
		if err != nil {
			return fmt.Errorf("unable to load snapshot: %s", err)
		}

		err = collectionFactory.SnapshotCollection().LoadComplete(sources[i])
		if err != nil {
			return fmt.Errorf("unable to load snapshot: %s", err)
		}
	}

	latest := context.Flags().Lookup("latest").Value.Get().(bool)
	noRemove := context.Flags().Lookup("no-remove").Value.Get().(bool)

	if noRemove && latest {
		return fmt.Errorf("-no-remove and -latest can't be specified together")
	}

	overrideMatching := !latest && !noRemove

	result := sources[0].RefList()
	for i := 1; i < len(sources); i++ {
		result = result.Merge(sources[i].RefList(), overrideMatching, false)
	}

	if latest {
		result.FilterLatestRefs()
	}

	sourceDescription := make([]string, len(sources))
	for i, s := range sources {
		sourceDescription[i] = fmt.Sprintf("'%s'", s.Name)
	}

	// Create <destination> snapshot
	destination := deb.NewSnapshotFromRefList(args[0], sources, result,
		fmt.Sprintf("Merged from sources: %s", strings.Join(sourceDescription, ", ")))

	err = collectionFactory.SnapshotCollection().Add(destination)
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
Merge command merges several <source> snapshots into one <destination> snapshot.
Merge happens from left to right. By default, packages with the same
name-architecture pair are replaced during merge (package from latest snapshot
on the list wins).  If run with only one source snapshot, merge copies <source> into
<destination>.

Example:

    $ aptly snapshot merge wheezy-w-backports wheezy-main wheezy-backports
`,
	}

	cmd.Flag.Bool("latest", false, "use only the latest version of each package")
	cmd.Flag.Bool("no-remove", false, "don't remove duplicate arch/name packages")

	return cmd
}
