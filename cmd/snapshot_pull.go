package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
	"github.com/wsxiaoys/terminal/color"
	"sort"
	"strings"
)

func aptlySnapshotPull(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 4 {
		cmd.Usage()
		return err
	}

	noDeps := cmd.Flag.Lookup("no-deps").Value.Get().(bool)
	noRemove := cmd.Flag.Lookup("no-remove").Value.Get().(bool)

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
		architecturesList = packageList.Architectures(false)
	}

	sort.Strings(architecturesList)

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

			if !noRemove {
				// Remove all packages with the same name and architecture
				for p := packageList.Search(debian.Dependency{Architecture: pkg.Architecture, Pkg: pkg.Name}); p != nil; {
					packageList.Remove(p)
					color.Printf("@r[-]@| %s removed", p)
					fmt.Printf("\n")
					p = packageList.Search(debian.Dependency{Architecture: pkg.Architecture, Pkg: pkg.Name})
				}
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
	cmd.Flag.Bool("no-remove", false, "don't remove other package versions when pulling package")

	return cmd
}
