package cmd

import (
	"fmt"
	"sort"

	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/query"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyRepoMoveCopyImport(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 3 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	command := cmd.Name()

	collectionFactory := context.NewCollectionFactory()
	dstRepo, err := collectionFactory.LocalRepoCollection().ByName(args[1])
	if err != nil {
		return fmt.Errorf("unable to %s: %s", command, err)
	}

	err = collectionFactory.LocalRepoCollection().LoadComplete(dstRepo, collectionFactory.RefListCollection())
	if err != nil {
		return fmt.Errorf("unable to %s: %s", command, err)
	}

	var (
		srcRefList *deb.SplitRefList
		srcRepo    *deb.LocalRepo
	)

	if command == "copy" || command == "move" { // nolint: goconst
		srcRepo, err = collectionFactory.LocalRepoCollection().ByName(args[0])
		if err != nil {
			return fmt.Errorf("unable to %s: %s", command, err)
		}

		if srcRepo.UUID == dstRepo.UUID {
			return fmt.Errorf("unable to %s: source and destination are the same", command)
		}

		err = collectionFactory.LocalRepoCollection().LoadComplete(srcRepo, collectionFactory.RefListCollection())
		if err != nil {
			return fmt.Errorf("unable to %s: %s", command, err)
		}

		srcRefList = srcRepo.RefList()
	} else if command == "import" { // nolint: goconst
		var srcRemoteRepo *deb.RemoteRepo

		srcRemoteRepo, err = collectionFactory.RemoteRepoCollection().ByName(args[0])
		if err != nil {
			return fmt.Errorf("unable to %s: %s", command, err)
		}

		err = collectionFactory.RemoteRepoCollection().LoadComplete(srcRemoteRepo, collectionFactory.RefListCollection())
		if err != nil {
			return fmt.Errorf("unable to %s: %s", command, err)
		}

		if srcRemoteRepo.RefList().Len() == 0 {
			return fmt.Errorf("unable to %s: mirror not updated", command)
		}

		srcRefList = srcRemoteRepo.RefList()
	} else {
		panic("unexpected command")
	}

	context.Progress().Printf("Loading packages...\n")

	dstList, err := deb.NewPackageListFromRefList(dstRepo.RefList(), collectionFactory.PackageCollection(), context.Progress())
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	srcList, err := deb.NewPackageListFromRefList(srcRefList, collectionFactory.PackageCollection(), context.Progress())
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	srcList.PrepareIndex()

	var architecturesList []string

	withDeps := context.Flags().Lookup("with-deps").Value.Get().(bool)

	if withDeps {
		dstList.PrepareIndex()

		// Calculate architectures
		if len(context.ArchitecturesList()) > 0 {
			architecturesList = context.ArchitecturesList()
		} else {
			architecturesList = dstList.Architectures(false)
		}

		sort.Strings(architecturesList)

		if len(architecturesList) == 0 {
			return fmt.Errorf("unable to determine list of architectures, please specify explicitly")
		}
	}

	queries := make([]deb.PackageQuery, len(args)-2)
	for i := 0; i < len(args)-2; i++ {
		value, err := GetStringOrFileContent(args[i+2])
		if err != nil {
			return fmt.Errorf("unable to read package query from file %s: %w", args[i+2], err)
		}
		queries[i], err = query.Parse(value)
		if err != nil {
			return fmt.Errorf("unable to %s: %s", command, err)
		}
	}

	toProcess, err := srcList.Filter(deb.FilterOptions{
		Queries:           queries,
		WithDependencies:  withDeps,
		Source:            dstList,
		DependencyOptions: context.DependencyOptions(),
		Architectures:     architecturesList,
		Progress:          context.Progress(),
	})
	if err != nil {
		return fmt.Errorf("unable to %s: %s", command, err)
	}

	var verb string

	if command == "move" { // nolint: goconst
		verb = "moved"
	} else if command == "copy" { // nolint: goconst
		verb = "copied"
	} else if command == "import" { // nolint: goconst
		verb = "imported"
	}

	err = toProcess.ForEach(func(p *deb.Package) error {
		err = dstList.Add(p)
		if err != nil {
			return err
		}

		if command == "move" { // nolint: goconst
			srcList.Remove(p)
		}
		context.Progress().ColoredPrintf("@g[o]@| %s %s", p, verb)
		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to %s: %s", command, err)
	}

	if context.Flags().Lookup("dry-run").Value.Get().(bool) {
		context.Progress().Printf("\nChanges not saved, as dry run has been requested.\n")
	} else {
		dstRepo.UpdateRefList(deb.NewSplitRefListFromPackageList(dstList))

		err = collectionFactory.LocalRepoCollection().Update(dstRepo, collectionFactory.RefListCollection())
		if err != nil {
			return fmt.Errorf("unable to save: %s", err)
		}

		if command == "move" { // nolint: goconst
			srcRepo.UpdateRefList(deb.NewSplitRefListFromPackageList(srcList))

			err = collectionFactory.LocalRepoCollection().Update(srcRepo, collectionFactory.RefListCollection())
			if err != nil {
				return fmt.Errorf("unable to save: %s", err)
			}
		}
	}

	return err
}

func makeCmdRepoMove() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRepoMoveCopyImport,
		UsageLine: "move <src-name> <dst-name> <package-query> ...",
		Short:     "move packages between local repositories",
		Long: `
Command move moves packages matching <package-query> from local repo
<src-name> to local repo <dst-name>.

Use '@file' to read package queries from file or '@-' for stdin.

Example:

  $ aptly repo move testing stable 'myapp (=0.1.12)'
`,
		Flag: *flag.NewFlagSet("aptly-repo-move", flag.ExitOnError),
	}

	cmd.Flag.Bool("dry-run", false, "don't move, just show what would be moved")
	cmd.Flag.Bool("with-deps", false, "follow dependencies when processing package-spec")

	return cmd
}
