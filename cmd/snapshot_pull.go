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

func aptlySnapshotPull(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 4 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	noDeps := context.Flags().Lookup("no-deps").Value.Get().(bool)
	noRemove := context.Flags().Lookup("no-remove").Value.Get().(bool)
	allMatches := context.Flags().Lookup("all-matches").Value.Get().(bool)
	collectionFactory := context.NewCollectionFactory()

	// Load <name> snapshot
	snapshot, err := collectionFactory.SnapshotCollection().ByName(args[0])
	if err != nil {
		return fmt.Errorf("unable to pull: %s", err)
	}

	err = collectionFactory.SnapshotCollection().LoadComplete(snapshot)
	if err != nil {
		return fmt.Errorf("unable to pull: %s", err)
	}

	// Load <source> snapshot
	source, err := collectionFactory.SnapshotCollection().ByName(args[1])
	if err != nil {
		return fmt.Errorf("unable to pull: %s", err)
	}

	err = collectionFactory.SnapshotCollection().LoadComplete(source)
	if err != nil {
		return fmt.Errorf("unable to pull: %s", err)
	}

	context.Progress().Printf("Dependencies would be pulled into snapshot:\n    %s\nfrom snapshot:\n    %s\nand result would be saved as new snapshot %s.\n",
		snapshot, source, args[2])

	// Convert snapshot to package list
	context.Progress().Printf("Loading packages (%d)...\n", snapshot.RefList().Len()+source.RefList().Len())
	packageList, err := deb.NewPackageListFromRefList(snapshot.RefList(), collectionFactory.PackageCollection(), context.Progress())
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	sourcePackageList, err := deb.NewPackageListFromRefList(source.RefList(), collectionFactory.PackageCollection(), context.Progress())
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	context.Progress().Printf("Building indexes...\n")
	packageList.PrepareIndex()
	sourcePackageList.PrepareIndex()

	// Calculate architectures
	var architecturesList []string

	if len(context.ArchitecturesList()) > 0 {
		architecturesList = context.ArchitecturesList()
	} else {
		architecturesList = packageList.Architectures(false)
	}

	sort.Strings(architecturesList)

	if len(architecturesList) == 0 {
		return fmt.Errorf("unable to determine list of architectures, please specify explicitly")
	}

	// Build architecture query: (arch == "i386" | arch == "amd64" | ...)
	var archQuery deb.PackageQuery = &deb.FieldQuery{Field: "$Architecture", Relation: deb.VersionEqual, Value: ""}
	for _, arch := range architecturesList {
		archQuery = &deb.OrQuery{L: &deb.FieldQuery{Field: "$Architecture", Relation: deb.VersionEqual, Value: arch}, R: archQuery}
	}

	// Initial queries out of arguments
	queries := make([]deb.PackageQuery, len(args)-3)
	for i, arg := range args[3:] {
		queries[i], err = query.Parse(arg)
		if err != nil {
			return fmt.Errorf("unable to parse query: %s", err)
		}
		// Add architecture filter
		queries[i] = &deb.AndQuery{L: queries[i], R: archQuery}
	}

	// Filter with dependencies as requested
	result, err := sourcePackageList.FilterWithProgress(queries, !noDeps, packageList, context.DependencyOptions(), architecturesList, context.Progress())
	if err != nil {
		return fmt.Errorf("unable to pull: %s", err)
	}
	result.PrepareIndex()

	alreadySeen := map[string]bool{}

	result.ForEachIndexed(func(pkg *deb.Package) error {
		key := pkg.Architecture + "_" + pkg.Name
		_, seen := alreadySeen[key]

		// If we haven't seen such name-architecture pair and were instructed to remove, remove it
		if !noRemove && !seen {
			// Remove all packages with the same name and architecture
			pS := packageList.Search(deb.Dependency{Architecture: pkg.Architecture, Pkg: pkg.Name}, true)
			for _, p := range pS {
				packageList.Remove(p)
				context.Progress().ColoredPrintf("@r[-]@| %s removed", p)
			}
		}

		// If !allMatches, add only first matching name-arch package
		if !seen || allMatches {
			packageList.Add(pkg)
			context.Progress().ColoredPrintf("@g[+]@| %s added", pkg)
		}

		alreadySeen[key] = true

		return nil
	})
	alreadySeen = nil

	if context.Flags().Lookup("dry-run").Value.Get().(bool) {
		context.Progress().Printf("\nNot creating snapshot, as dry run was requested.\n")
	} else {
		// Create <destination> snapshot
		destination := deb.NewSnapshotFromPackageList(args[2], []*deb.Snapshot{snapshot, source}, packageList,
			fmt.Sprintf("Pulled into '%s' with '%s' as source, pull request was: '%s'", snapshot.Name, source.Name, strings.Join(args[3:], " ")))

		err = collectionFactory.SnapshotCollection().Add(destination)
		if err != nil {
			return fmt.Errorf("unable to create snapshot: %s", err)
		}

		context.Progress().Printf("\nSnapshot %s successfully created.\nYou can run 'aptly publish snapshot %s' to publish snapshot as Debian repository.\n", destination.Name, destination.Name)
	}
	return err
}

func makeCmdSnapshotPull() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotPull,
		UsageLine: "pull <name> <source> <destination> <package-query> ...",
		Short:     "pull packages from another snapshot",
		Long: `
Command pull pulls new packages along with its' dependencies to snapshot <name>
from snapshot <source>. Pull can upgrade package version in <name> with
versions from <source> following dependencies. New snapshot <destination>
is created as a result of this process. Packages could be specified simply
as 'package-name' or as package queries.

Example:

    $ aptly snapshot pull wheezy-main wheezy-backports wheezy-new-xorg xorg-server-server
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-pull", flag.ExitOnError),
	}

	cmd.Flag.Bool("dry-run", false, "don't create destination snapshot, just show what would be pulled")
	cmd.Flag.Bool("no-deps", false, "don't process dependencies, just pull listed packages")
	cmd.Flag.Bool("no-remove", false, "don't remove other package versions when pulling package")
	cmd.Flag.Bool("all-matches", false, "pull all the packages that satisfy the dependency version requirements")

	return cmd
}
