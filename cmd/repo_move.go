package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
	"sort"
)

func aptlyRepoMoveCopyImport(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 3 {
		cmd.Usage()
		return err
	}

	command := cmd.Name()

	localRepoCollection := debian.NewLocalRepoCollection(context.database)

	dstRepo, err := localRepoCollection.ByName(args[1])
	if err != nil {
		return fmt.Errorf("unable to %s: %s", command, err)
	}

	err = localRepoCollection.LoadComplete(dstRepo)
	if err != nil {
		return fmt.Errorf("unable to %s: %s", command, err)
	}

	var (
		srcRefList *debian.PackageRefList
		srcRepo    *debian.LocalRepo
	)

	if command == "copy" || command == "move" {
		srcRepo, err = localRepoCollection.ByName(args[0])
		if err != nil {
			return fmt.Errorf("unable to %s: %s", command, err)
		}

		if srcRepo.UUID == dstRepo.UUID {
			return fmt.Errorf("unable to %s: source and destination are the same", command)
		}

		err = localRepoCollection.LoadComplete(srcRepo)
		if err != nil {
			return fmt.Errorf("unable to %s: %s", command, err)
		}

		srcRefList = srcRepo.RefList()
	} else if command == "import" {
		repoCollection := debian.NewRemoteRepoCollection(context.database)

		srcRepo, err := repoCollection.ByName(args[0])
		if err != nil {
			return fmt.Errorf("unable to %s: %s", command, err)
		}

		err = repoCollection.LoadComplete(srcRepo)
		if err != nil {
			return fmt.Errorf("unable to %s: %s", command, err)
		}

		if srcRepo.RefList() == nil {
			return fmt.Errorf("unable to %s: mirror not updated", command)
		}

		srcRefList = srcRepo.RefList()
	} else {
		panic("unexpected command")
	}

	context.progress.Printf("Loading packages...\n")

	packageCollection := debian.NewPackageCollection(context.database)
	dstList, err := debian.NewPackageListFromRefList(dstRepo.RefList(), packageCollection)
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	srcList, err := debian.NewPackageListFromRefList(srcRefList, packageCollection)
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	srcList.PrepareIndex()

	var architecturesList []string

	withDeps := cmd.Flag.Lookup("with-deps").Value.Get().(bool)

	if withDeps {
		dstList.PrepareIndex()

		// Calculate architectures
		if len(context.architecturesList) > 0 {
			architecturesList = context.architecturesList
		} else {
			architecturesList = dstList.Architectures(false)
		}

		sort.Strings(architecturesList)

		if len(architecturesList) == 0 {
			return fmt.Errorf("unable to determine list of architectures, please specify explicitly")
		}
	}

	toProcess, err := srcList.Filter(args[2:], withDeps, dstList, context.dependencyOptions, architecturesList)
	if err != nil {
		return fmt.Errorf("unable to %s: %s", command, err)
	}

	var verb string

	if command == "move" {
		verb = "moved"
	} else if command == "copy" {
		verb = "copied"
	} else if command == "import" {
		verb = "imported"
	}

	err = toProcess.ForEach(func(p *debian.Package) error {
		err = dstList.Add(p)
		if err != nil {
			return err
		}

		if command == "move" {
			srcList.Remove(p)
		}
		context.progress.ColoredPrintf("@g[o]@| %s %s", p, verb)
		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to %s: %s", command, err)
	}

	if cmd.Flag.Lookup("dry-run").Value.Get().(bool) {
		context.progress.Printf("\nChanges not saved, as dry run has been requested.\n")
	} else {
		dstRepo.UpdateRefList(debian.NewPackageRefListFromPackageList(dstList))

		err = localRepoCollection.Update(dstRepo)
		if err != nil {
			return fmt.Errorf("unable to save: %s", err)
		}

		if command == "move" {
			srcRepo.UpdateRefList(debian.NewPackageRefListFromPackageList(srcList))

			err = localRepoCollection.Update(srcRepo)
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
		UsageLine: "move <src-name> <dst-name> <package-spec> ...",
		Short:     "move packages between source repos",
		Long: `
Command move moves packages matching <package-spec> from local repo
<src-name> to local repo <dst-name>.

ex:
  $ aptly repo move testing stable 'myapp (=0.1.12)'
`,
		Flag: *flag.NewFlagSet("aptly-repo-move", flag.ExitOnError),
	}

	cmd.Flag.Bool("dry-run", false, "don't move, just show what would be moved")
	cmd.Flag.Bool("with-deps", false, "follow dependencies when processing package-spec")

	return cmd
}
