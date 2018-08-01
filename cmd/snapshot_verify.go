package cmd

import (
	"fmt"
	"sort"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
)

func aptlySnapshotVerify(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	snapshots := make([]*deb.Snapshot, len(args))
	collectionFactory := context.NewCollectionFactory()
	for i := range snapshots {
		snapshots[i], err = collectionFactory.SnapshotCollection().ByName(args[i])
		if err != nil {
			return fmt.Errorf("unable to verify: %s", err)
		}

		err = collectionFactory.SnapshotCollection().LoadComplete(snapshots[i])
		if err != nil {
			return fmt.Errorf("unable to verify: %s", err)
		}
	}

	context.Progress().Printf("Loading packages...\n")

	packageList, err := deb.NewPackageListFromRefList(snapshots[0].RefList(), collectionFactory.PackageCollection(), context.Progress())
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	sourcePackageList := deb.NewPackageList()
	err = sourcePackageList.Append(packageList)
	if err != nil {
		return fmt.Errorf("unable to merge sources: %s", err)
	}

	var pL *deb.PackageList
	for i := 1; i < len(snapshots); i++ {
		pL, err = deb.NewPackageListFromRefList(snapshots[i].RefList(), collectionFactory.PackageCollection(), context.Progress())
		if err != nil {
			return fmt.Errorf("unable to load packages: %s", err)
		}

		err = sourcePackageList.Append(pL)
		if err != nil {
			return fmt.Errorf("unable to merge sources: %s", err)
		}
	}

	sourcePackageList.PrepareIndex()

	var architecturesList []string

	if len(context.ArchitecturesList()) > 0 {
		architecturesList = context.ArchitecturesList()
	} else {
		architecturesList = packageList.Architectures(true)
	}

	if len(architecturesList) == 0 {
		return fmt.Errorf("unable to determine list of architectures, please specify explicitly")
	}

	context.Progress().Printf("Verifying...\n")

	missing, err := packageList.VerifyDependencies(context.DependencyOptions(), architecturesList, sourcePackageList, context.Progress())
	if err != nil {
		return fmt.Errorf("unable to verify dependencies: %s", err)
	}

	if len(missing) == 0 {
		context.Progress().Printf("All dependencies are satisfied.\n")
	} else {
		context.Progress().Printf("Missing dependencies (%d):\n", len(missing))
		deps := make([]string, len(missing))
		i := 0
		for _, dep := range missing {
			deps[i] = dep.String()
			i++
		}

		sort.Strings(deps)

		for _, dep := range deps {
			context.Progress().Printf("  %s\n", dep)
		}
	}

	return err
}

func makeCmdSnapshotVerify() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotVerify,
		UsageLine: "verify <name> [<source> ...]",
		Short:     "verify dependencies in snapshot",
		Long: `
Verify does dependency resolution in snapshot <name>, possibly using additional
snapshots <source> as dependency sources. All unsatisfied dependencies are
printed.

Example:

    $ aptly snapshot verify wheezy-main wheezy-contrib wheezy-non-free
`,
	}

	return cmd
}
