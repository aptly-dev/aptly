package main

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
	"github.com/wsxiaoys/terminal/color"
	"sort"
	"strings"
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

func aptlySnapshotList(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 0 {
		cmd.Usage()
		return err
	}

	snapshotCollection := debian.NewSnapshotCollection(context.database)

	if snapshotCollection.Len() > 0 {
		fmt.Printf("List of snapshots:\n")

		snapshots := make(sort.StringSlice, snapshotCollection.Len())

		i := 0
		snapshotCollection.ForEach(func(snapshot *debian.Snapshot) error {
			snapshots[i] = snapshot.String()
			i++
			return nil
		})

		sort.Strings(snapshots)
		for _, snapshot := range snapshots {
			fmt.Printf(" * %s\n", snapshot)
		}

		fmt.Printf("\nTo get more information about snapshot, run `aptly snapshot show <name>`.\n")
	} else {
		fmt.Printf("\nNo snapshots found, create one with `aptly snapshot create...`.\n")
	}
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

	withPackages := cmd.Flag.Lookup("with-packages").Value.Get().(bool)
	if withPackages {
		ListPackagesRefList(snapshot.RefList())
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

	sourcePackageList := debian.NewPackageList()
	err = sourcePackageList.Append(packageList)
	if err != nil {
		fmt.Errorf("unable to merge sources: %s", err)
	}

	for i := 1; i < len(snapshots); i++ {
		pL, err := debian.NewPackageListFromRefList(snapshots[i].RefList(), packageCollection)
		if err != nil {
			fmt.Errorf("unable to load packages: %s", err)
		}

		err = sourcePackageList.Append(pL)
		if err != nil {
			fmt.Errorf("unable to merge sources: %s", err)
		}
	}

	sourcePackageList.PrepareIndex()

	var architecturesList []string

	if len(context.architecturesList) > 0 {
		architecturesList = context.architecturesList
	} else {
		architecturesList = packageList.Architectures()
	}

	if len(architecturesList) == 0 {
		return fmt.Errorf("unable to determine list of architectures, please specify explicitly")
	}

	missing, err := packageList.VerifyDependencies(context.dependencyOptions, architecturesList, sourcePackageList)
	if err != nil {
		return fmt.Errorf("unable to verify dependencies: %s", err)
	}

	if len(missing) == 0 {
		fmt.Printf("All dependencies are satisfied.\n")
	} else {
		fmt.Printf("Missing dependencies (%d):\n", len(missing))
		deps := make(sort.StringSlice, len(missing))
		i := 0
		for _, dep := range missing {
			deps[i] = dep.String()
			i++
		}

		sort.Strings(deps)

		for _, dep := range deps {
			fmt.Printf("  %s\n", dep)
		}
	}

	return err
}

func aptlySnapshotPull(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 4 {
		cmd.Usage()
		return err
	}

	noDeps := cmd.Flag.Lookup("no-deps").Value.Get().(bool)

	snapshotCollection := debian.NewSnapshotCollection(context.database)
	packageCollection := debian.NewPackageCollection(context.database)

	// Load <name> snapshot
	snapshot, err := snapshotCollection.ByName(args[0])
	if err != nil {
		return fmt.Errorf("unable to pull: %s", err)
	}

	err = snapshotCollection.LoadComplete(snapshot)
	if err != nil {
		return fmt.Errorf("unable to pull: %s", err)
	}

	// Load <source> snapshot
	source, err := snapshotCollection.ByName(args[1])
	if err != nil {
		return fmt.Errorf("unable to pull: %s", err)
	}

	err = snapshotCollection.LoadComplete(source)
	if err != nil {
		return fmt.Errorf("unable to pull: %s", err)
	}

	fmt.Printf("Dependencies would be pulled into snapshot:\n    %s\nfrom snapshot:\n    %s\nand result would be saved as new snapshot %s.\n",
		snapshot, source, args[2])

	// Convert snapshot to package list
	fmt.Printf("Loading packages (%d)...\n", snapshot.RefList().Len()+source.RefList().Len())
	packageList, err := debian.NewPackageListFromRefList(snapshot.RefList(), packageCollection)
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	sourcePackageList, err := debian.NewPackageListFromRefList(source.RefList(), packageCollection)
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	fmt.Printf("Building indexes...\n")
	packageList.PrepareIndex()
	sourcePackageList.PrepareIndex()

	// Calculate architectures
	var architecturesList []string

	if len(context.architecturesList) > 0 {
		architecturesList = context.architecturesList
	} else {
		architecturesList = packageList.Architectures()
	}

	if len(architecturesList) == 0 {
		return fmt.Errorf("unable to determine list of architectures, please specify explicitly")
	}

	// Initial dependencies out of arguments
	initialDependencies := make([]debian.Dependency, len(args)-3)
	for i, arg := range args[3:] {
		initialDependencies[i], err = debian.ParseDependency(arg)
		if err != nil {
			return fmt.Errorf("unable to parse argument: %s", err)
		}
	}

	// Perform pull
	for _, arch := range architecturesList {
		dependencies := make([]debian.Dependency, len(initialDependencies), 128)
		for i := range dependencies {
			dependencies[i] = initialDependencies[i]
			dependencies[i].Architecture = arch
		}

		// Go over list of initial dependencies + list of dependencies found
		for i := 0; i < len(dependencies); i++ {
			dep := dependencies[i]

			// Search for package that can satisfy dependencies
			pkg := sourcePackageList.Search(dep)
			if pkg == nil {
				color.Printf("@y[!]@| @!Dependency %s can't be satisfied with source %s@|", &dep, source)
				fmt.Printf("\n")
				continue
			}

			// Remove all packages with the same name and architecture
			for p := packageList.Search(debian.Dependency{Architecture: arch, Pkg: pkg.Name}); p != nil; {
				packageList.Remove(p)
				color.Printf("@r[-]@| %s removed", p)
				fmt.Printf("\n")
				p = packageList.Search(debian.Dependency{Architecture: arch, Pkg: pkg.Name})
			}

			// Add new discovered package
			packageList.Add(pkg)
			color.Printf("@g[+]@| %s added", pkg)
			fmt.Printf("\n")

			if noDeps {
				continue
			}

			// Find missing dependencies for single added package
			pL := debian.NewPackageList()
			pL.Add(pkg)

			missing, err := pL.VerifyDependencies(context.dependencyOptions, []string{arch}, packageList)
			if err != nil {
				color.Printf("@y[!]@| @!Error while verifying dependencies for pkg %s: %s@|", pkg, err)
				fmt.Printf("\n")
			}

			// Append missing dependencies to the list of dependencies to satisfy
			for _, misDep := range missing {
				found := false
				for _, d := range dependencies {
					if d == misDep {
						found = true
						break
					}
				}

				if !found {
					dependencies = append(dependencies, misDep)
				}
			}
		}
	}

	if cmd.Flag.Lookup("dry-run").Value.Get().(bool) {
		fmt.Printf("\nNot creating snapshot, as dry run was requested.\n")
	} else {
		// Create <destination> snapshot
		destination := debian.NewSnapshotFromPackageList(args[2], []*debian.Snapshot{snapshot, source}, packageList,
			fmt.Sprintf("Pulled into '%s' with '%s' as source, pull request was: '%s'", snapshot.Name, source.Name, strings.Join(args[3:], " ")))

		err = snapshotCollection.Add(destination)
		if err != nil {
			return fmt.Errorf("unable to create snapshot: %s", err)
		}

		fmt.Printf("\nSnapshot %s successfully created.\nYou can run 'aptly publish snapshot %s' to publish snapshot as Debian repository.\n", destination.Name, destination.Name)
	}
	return err
}

func aptlySnapshotDiff(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 2 {
		cmd.Usage()
		return err
	}

	onlyMatching := cmd.Flag.Lookup("only-matching").Value.Get().(bool)

	snapshotCollection := debian.NewSnapshotCollection(context.database)
	packageCollection := debian.NewPackageCollection(context.database)

	// Load <name-a> snapshot
	snapshotA, err := snapshotCollection.ByName(args[0])
	if err != nil {
		return fmt.Errorf("unable to load snapshot A: %s", err)
	}

	err = snapshotCollection.LoadComplete(snapshotA)
	if err != nil {
		return fmt.Errorf("unable to load snapshot A: %s", err)
	}

	// Load <name-b> snapshot
	snapshotB, err := snapshotCollection.ByName(args[1])
	if err != nil {
		return fmt.Errorf("unable to load snapshot B: %s", err)
	}

	err = snapshotCollection.LoadComplete(snapshotB)
	if err != nil {
		return fmt.Errorf("unable to load snapshot B: %s", err)
	}

	// Calculate diff
	diff, err := snapshotA.RefList().Diff(snapshotB.RefList(), packageCollection)
	if err != nil {
		return fmt.Errorf("unable to calculate diff: %s", err)
	}

	if len(diff) == 0 {
		fmt.Printf("Snapshots are identical.\n")
	} else {
		fmt.Printf("  Arch   | Package                                  | Version in A                             | Version in B\n")
		for _, pdiff := range diff {
			if onlyMatching && (pdiff.Left == nil || pdiff.Right == nil) {
				continue
			}

			var verA, verB, pkg, arch, code string

			if pdiff.Left == nil {
				verA = "-"
				verB = pdiff.Right.Version
				pkg = pdiff.Right.Name
				arch = pdiff.Right.Architecture
			} else {
				pkg = pdiff.Left.Name
				arch = pdiff.Left.Architecture
				verA = pdiff.Left.Version
				if pdiff.Right == nil {
					verB = "-"
				} else {
					verB = pdiff.Right.Version
				}
			}

			if pdiff.Left == nil {
				code = "@g+@|"
			} else {
				if pdiff.Right == nil {
					code = "@r-@|"
				} else {
					code = "@y!@|"
				}
			}

			color.Printf(code+" %-6s | %-40s | %-40s | %-40s\n", arch, pkg, verA, verB)
		}
	}

	return err
}

func aptlySnapshotMerge(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 2 {
		cmd.Usage()
		return err
	}

	snapshotCollection := debian.NewSnapshotCollection(context.database)

	sources := make([]*debian.Snapshot, len(args)-1)

	for i := 0; i < len(args)-1; i++ {
		sources[i], err = snapshotCollection.ByName(args[i+1])
		if err != nil {
			return fmt.Errorf("unable to load snapshot: %s", err)
		}

		err = snapshotCollection.LoadComplete(sources[i])
		if err != nil {
			return fmt.Errorf("unable to load snapshot: %s", err)
		}
	}

	result := sources[0].RefList()

	for i := 1; i < len(sources); i++ {
		result = result.Merge(sources[i].RefList())
	}

	sourceDescription := make([]string, len(sources))
	for i, s := range sources {
		sourceDescription[i] = fmt.Sprintf("'%s'", s.Name)
	}

	// Create <destination> snapshot
	destination := debian.NewSnapshotFromRefList(args[0], sources, result,
		fmt.Sprintf("Merged from sources: %s", strings.Join(sourceDescription, ", ")))

	err = snapshotCollection.Add(destination)
	if err != nil {
		return fmt.Errorf("unable to create snapshot: %s", err)
	}

	fmt.Printf("\nSnapshot %s successfully created.\nYou can run 'aptly publish snapshot %s' to publish snapshot as Debian repository.\n", destination.Name, destination.Name)

	return err
}

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

func makeCmdSnapshotCreate() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotCreate,
		UsageLine: "create <name> from mirror <mirror-name>",
		Short:     "creates snapshot out of any mirror",
		Long: `
Command create makes persistent immutable snapshot of remote repository mirror. Snapshot could be
published or further modified using merge, pull and other aptly features.

ex.
  $ aptly snapshot create wheezy-main-today from mirror wheezy-main
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
Command list shows full list of snapshots created.

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
		UsageLine: "show <name>",
		Short:     "shows details about snapshot",
		Long: `
Command show displays full information about snapshot.

ex.
	$ aptly snapshot show wheezy-main
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-show", flag.ExitOnError),
	}

	cmd.Flag.Bool("with-packages", false, "show list of packages")

	return cmd
}

func makeCmdSnapshotVerify() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotVerify,
		UsageLine: "verify <name> [<source> ...]",
		Short:     "verifies that dependencies are satisfied in snapshot",
		Long: `
Verify does depenency resolution in snapshot, possibly using additional snapshots as dependency sources.
All unsatisfied dependencies are returned.

ex.
	$ aptly snapshot verify wheezy-main wheezy-contrib wheezy-non-free
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-verify", flag.ExitOnError),
	}

	return cmd
}

func makeCmdSnapshotPull() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotPull,
		UsageLine: "pull <name> <source> <destination> <package-name> ...",
		Short:     "performs partial upgrades (pulls new packages) from another snapshot",
		Long: `
Command pull pulls new packages along with its dependencies in <name> snapshot
from <source> snapshot. Also can upgrade package version from one snapshot into
another, once again along with dependencies. New snapshot <destination> is created as result of this
process. Packages could be specified simply as 'package-name' or as dependency 'package-name (>= version)'.

ex.
	$ aptly snapshot pull wheezy-main wheezy-backports wheezy-new-xorg xorg-server-server
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-pull", flag.ExitOnError),
	}

	cmd.Flag.Bool("dry-run", false, "don't create destination snapshot, just show what would be pulled")
	cmd.Flag.Bool("no-deps", false, "don't process dependencies, just pull listed packages")

	return cmd
}

func makeCmdSnapshotDiff() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotDiff,
		UsageLine: "diff <name-a> <name-b>",
		Short:     "calculates difference in packages between two snapshots",
		Long: `
Command diff shows list of missing and new packages, difference in package versions between two snapshots.

ex.
	$ aptly snapshot diff -only-matching wheezy-main wheezy-backports
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-diff", flag.ExitOnError),
	}

	cmd.Flag.Bool("only-matching", false, "display diff only for matching packages (don't display missing packages)")

	return cmd
}

func makeCmdSnapshotMerge() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotMerge,
		UsageLine: "merge <destination> <source> [<source>...]",
		Short:     "merges snapshots into one, replacing matching packages",
		Long: `
Merge merges several snapshots into one. Merge happens from left to right. Packages with the same
name-architecture pair are replaced during merge (package from latest snapshot on the list wins).
If run with only one source snapshot, merge copies source into destination.

ex.
	$ aptly snapshot merge wheezy-w-backports wheezy-main wheezy-backports
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-merge", flag.ExitOnError),
	}

	return cmd
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

func makeCmdSnapshot() *commander.Command {
	return &commander.Command{
		UsageLine: "snapshot",
		Short:     "manage snapshots of repositories",
		Subcommands: []*commander.Command{
			makeCmdSnapshotCreate(),
			makeCmdSnapshotList(),
			makeCmdSnapshotShow(),
			makeCmdSnapshotVerify(),
			makeCmdSnapshotPull(),
			makeCmdSnapshotDiff(),
			makeCmdSnapshotMerge(),
			makeCmdSnapshotDrop(),
		},
		Flag: *flag.NewFlagSet("aptly-snapshot", flag.ExitOnError),
	}
}
