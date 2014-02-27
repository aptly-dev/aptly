package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
)

func aptlyRepoRemove(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 2 {
		cmd.Usage()
		return err
	}

	name := args[0]

	localRepoCollection := debian.NewLocalRepoCollection(context.database)
	repo, err := localRepoCollection.ByName(name)
	if err != nil {
		return fmt.Errorf("unable to remove: %s", err)
	}

	err = localRepoCollection.LoadComplete(repo)
	if err != nil {
		return fmt.Errorf("unable to remove: %s", err)
	}

	context.progress.Printf("Loading packages...\n")

	packageCollection := debian.NewPackageCollection(context.database)
	list, err := debian.NewPackageListFromRefList(repo.RefList(), packageCollection)
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	list.PrepareIndex()
	toRemove, err := list.Filter(args[1:], false, nil, 0, nil)
	if err != nil {
		return fmt.Errorf("unable to remove: %s", err)
	}

	toRemove.ForEach(func(p *debian.Package) error {
		list.Remove(p)
		context.progress.ColoredPrintf("@r[-]@| %s removed", p)
		return nil
	})

	if cmd.Flag.Lookup("dry-run").Value.Get().(bool) {
		context.progress.Printf("\nChanges not saved, as dry run has been requested.\n")
	} else {
		repo.UpdateRefList(debian.NewPackageRefListFromPackageList(list))

		err = localRepoCollection.Update(repo)
		if err != nil {
			return fmt.Errorf("unable to save: %s", err)
		}
	}

	return err
}

func makeCmdRepoRemove() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRepoRemove,
		UsageLine: "remove <name> <package-spec> ...",
		Short:     "remove packages from local repository",
		Long: `
Commands removes packages matching specs from local repository. If removed
packages are not referenced by other repos or snapshots, they can be removed
completely (including files) by running 'aptly db cleanup'.

ex:
  $ aptly repo remove testing 'myapp (=0.1.12)'
`,
		Flag: *flag.NewFlagSet("aptly-repo-add", flag.ExitOnError),
	}

	cmd.Flag.Bool("dry-run", false, "don't remove, just show what would be removed")

	return cmd
}
