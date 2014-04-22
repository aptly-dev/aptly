package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/commander"
	"github.com/smira/flag"
	"sort"
	"strings"
)

func aptlySnapshotPull(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 4 {
		cmd.Usage()
		return err
	}

	noDeps := context.flags.Lookup("no-deps").Value.Get().(bool)
	noRemove := context.flags.Lookup("no-remove").Value.Get().(bool)

	// Load <name> snapshot
	snapshot, err := context.CollectionFactory().SnapshotCollection().ByName(args[0])
	if err != nil {
		return fmt.Errorf("unable to pull: %s", err)
	}

	err = context.CollectionFactory().SnapshotCollection().LoadComplete(snapshot)
	if err != nil {
		return fmt.Errorf("unable to pull: %s", err)
	}

	// Load <source> snapshot
	source, err := context.CollectionFactory().SnapshotCollection().ByName(args[1])
	if err != nil {
		return fmt.Errorf("unable to pull: %s", err)
	}

	err = context.CollectionFactory().SnapshotCollection().LoadComplete(source)
	if err != nil {
		return fmt.Errorf("unable to pull: %s", err)
	}

	context.Progress().Printf("Dependencies would be pulled into snapshot:\n    %s\nfrom snapshot:\n    %s\nand result would be saved as new snapshot %s.\n",
		snapshot, source, args[2])

	// Convert snapshot to package list
	context.Progress().Printf("Loading packages (%d)...\n", snapshot.RefList().Len()+source.RefList().Len())
	packageList, err := deb.NewPackageListFromRefList(snapshot.RefList(), context.CollectionFactory().PackageCollection(), context.Progress())
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	sourcePackageList, err := deb.NewPackageListFromRefList(source.RefList(), context.CollectionFactory().PackageCollection(), context.Progress())
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

	// Initial dependencies out of arguments
	initialDependencies := make([]deb.Dependency, len(args)-3)
	for i, arg := range args[3:] {
		initialDependencies[i], err = deb.ParseDependency(arg)
		if err != nil {
			return fmt.Errorf("unable to parse argument: %s", err)
		}
	}

	// Perform pull
	for _, arch := range architecturesList {
		dependencies := make([]deb.Dependency, len(initialDependencies), 128)
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
				context.Progress().ColoredPrintf("@y[!]@| @!Dependency %s can't be satisfied with source %s@|", &dep, source)
				continue
			}

			if !noRemove {
				// Remove all packages with the same name and architecture
				for p := packageList.Search(deb.Dependency{Architecture: pkg.Architecture, Pkg: pkg.Name}); p != nil; {
					packageList.Remove(p)
					context.Progress().ColoredPrintf("@r[-]@| %s removed", p)
					p = packageList.Search(deb.Dependency{Architecture: pkg.Architecture, Pkg: pkg.Name})
				}
			}

			// Add new discovered package
			packageList.Add(pkg)
			context.Progress().ColoredPrintf("@g[+]@| %s added", pkg)

			if noDeps {
				continue
			}

			// Find missing dependencies for single added package
			pL := deb.NewPackageList()
			pL.Add(pkg)

			var missing []deb.Dependency
			missing, err = pL.VerifyDependencies(context.DependencyOptions(), []string{arch}, packageList, nil)
			if err != nil {
				context.Progress().ColoredPrintf("@y[!]@| @!Error while verifying dependencies for pkg %s: %s@|", pkg, err)
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

	if context.flags.Lookup("dry-run").Value.Get().(bool) {
		context.Progress().Printf("\nNot creating snapshot, as dry run was requested.\n")
	} else {
		// Create <destination> snapshot
		destination := deb.NewSnapshotFromPackageList(args[2], []*deb.Snapshot{snapshot, source}, packageList,
			fmt.Sprintf("Pulled into '%s' with '%s' as source, pull request was: '%s'", snapshot.Name, source.Name, strings.Join(args[3:], " ")))

		err = context.CollectionFactory().SnapshotCollection().Add(destination)
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
		UsageLine: "pull <name> <source> <destination> <package-name> ...",
		Short:     "pull packages from another snapshot",
		Long: `
Command pull pulls new packages along with its' dependencies to snapshot <name>
from snapshot <source>. Pull can upgrade package version in <name> with
versions from <source> following dependencies. New snapshot <destination>
is created as a result of this process. Packages could be specified simply
as 'package-name' or as dependency 'package-name (>= version)'.

Example:

    $ aptly snapshot pull wheezy-main wheezy-backports wheezy-new-xorg xorg-server-server
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-pull", flag.ExitOnError),
	}

	cmd.Flag.Bool("dry-run", false, "don't create destination snapshot, just show what would be pulled")
	cmd.Flag.Bool("no-deps", false, "don't process dependencies, just pull listed packages")
	cmd.Flag.Bool("no-remove", false, "don't remove other package versions when pulling package")

	return cmd
}
