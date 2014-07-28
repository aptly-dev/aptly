package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/aptly/query"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyRepoRemove(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 2 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	name := args[0]

	repo, err := context.CollectionFactory().LocalRepoCollection().ByName(name)
	if err != nil {
		return fmt.Errorf("unable to remove: %s", err)
	}

	err = context.CollectionFactory().LocalRepoCollection().LoadComplete(repo)
	if err != nil {
		return fmt.Errorf("unable to remove: %s", err)
	}

	context.Progress().Printf("Loading packages...\n")

	list, err := deb.NewPackageListFromRefList(repo.RefList(), context.CollectionFactory().PackageCollection(), context.Progress())
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	queries := make([]deb.PackageQuery, len(args)-1)
	for i := 0; i < len(args)-1; i++ {
		queries[i], err = query.Parse(args[i+1])
		if err != nil {
			return fmt.Errorf("unable to remove: %s", err)
		}
	}

	list.PrepareIndex()
	toRemove, err := list.Filter(queries, false, nil, 0, nil)
	if err != nil {
		return fmt.Errorf("unable to remove: %s", err)
	}

	toRemove.ForEach(func(p *deb.Package) error {
		list.Remove(p)
		context.Progress().ColoredPrintf("@r[-]@| %s removed", p)
		return nil
	})

	if context.flags.Lookup("dry-run").Value.Get().(bool) {
		context.Progress().Printf("\nChanges not saved, as dry run has been requested.\n")
	} else {
		repo.UpdateRefList(deb.NewPackageRefListFromPackageList(list))

		err = context.CollectionFactory().LocalRepoCollection().Update(repo)
		if err != nil {
			return fmt.Errorf("unable to save: %s", err)
		}
	}

	return err
}

func makeCmdRepoRemove() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRepoRemove,
		UsageLine: "remove <name> <package-query> ...",
		Short:     "remove packages from local repository",
		Long: `
Commands removes packages matching <package-query> from local repository
<name>. If removed packages are not referenced by other repos or
snapshots, they can be removed completely (including files) by running
'aptly db cleanup'.

Example:

  $ aptly repo remove testing 'myapp (=0.1.12)'
`,
		Flag: *flag.NewFlagSet("aptly-repo-add", flag.ExitOnError),
	}

	cmd.Flag.Bool("dry-run", false, "don't remove, just show what would be removed")

	return cmd
}
