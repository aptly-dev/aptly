package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/query"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlySnapshotFilter(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 3 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	withDeps := context.Flags().Lookup("with-deps").Value.Get().(bool)
	collectionFactory := context.NewCollectionFactory()

	// Load <source> snapshot
	source, err := collectionFactory.SnapshotCollection().ByName(args[0])
	if err != nil {
		return fmt.Errorf("unable to filter: %s", err)
	}

	err = collectionFactory.SnapshotCollection().LoadComplete(source)
	if err != nil {
		return fmt.Errorf("unable to filter: %s", err)
	}

	// Convert snapshot to package list
	context.Progress().Printf("Loading packages (%d)...\n", source.RefList().Len())
	packageList, err := deb.NewPackageListFromRefList(source.RefList(), collectionFactory.PackageCollection(), context.Progress())
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	context.Progress().Printf("Building indexes...\n")
	packageList.PrepareIndex()

	// Calculate architectures
	var architecturesList []string

	if len(context.ArchitecturesList()) > 0 {
		architecturesList = context.ArchitecturesList()
	} else {
		architecturesList = packageList.Architectures(false)
	}

	sort.Strings(architecturesList)

	if len(architecturesList) == 0 && withDeps {
		return fmt.Errorf("unable to determine list of architectures, please specify explicitly")
	}

	// Initial queries out of arguments
	queries := make([]deb.PackageQuery, len(args)-2)
	for i, arg := range args[2:] {
		queries[i], err = query.Parse(arg)
		if err != nil {
			return fmt.Errorf("unable to parse query: %s", err)
		}
	}

	// Filter with dependencies as requested
	result, err := packageList.FilterWithProgress(queries, withDeps, nil, context.DependencyOptions(), architecturesList, context.Progress())
	if err != nil {
		return fmt.Errorf("unable to filter: %s", err)
	}

	// Create <destination> snapshot
	destination := deb.NewSnapshotFromPackageList(args[1], []*deb.Snapshot{source}, result,
		fmt.Sprintf("Filtered '%s', query was: '%s'", source.Name, strings.Join(args[2:], " ")))

	err = collectionFactory.SnapshotCollection().Add(destination)
	if err != nil {
		return fmt.Errorf("unable to create snapshot: %s", err)
	}

	context.Progress().Printf("\nSnapshot %s successfully filtered.\nYou can run 'aptly publish snapshot %s' to publish snapshot as Debian repository.\n", destination.Name, destination.Name)

	return err
}

func makeCmdSnapshotFilter() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotFilter,
		UsageLine: "filter <source> <destination> <package-query> ...",
		Short:     "filter packages in snapshot producing another snapshot",
		Long: `
Command filter does filtering in snapshot <source>, producing another
snapshot <destination>. Packages could be specified simply
as 'package-name' or as package queries.

Example:

    $ aptly snapshot filter wheezy-main wheezy-required 'Priority (required)'
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-filter", flag.ExitOnError),
	}

	cmd.Flag.Bool("with-deps", false, "include dependent packages as well")

	return cmd
}
