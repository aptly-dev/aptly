package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/aptly/query"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlySnapshotMirrorRepoSearch(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 2 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	name := args[0]
	command := cmd.Parent.Name()

	var reflist *deb.PackageRefList

	if command == "snapshot" {
		snapshot, err := context.CollectionFactory().SnapshotCollection().ByName(name)
		if err != nil {
			return fmt.Errorf("unable to search: %s", err)
		}

		err = context.CollectionFactory().SnapshotCollection().LoadComplete(snapshot)
		if err != nil {
			return fmt.Errorf("unable to search: %s", err)
		}

		reflist = snapshot.RefList()
	} else if command == "mirror" {
		repo, err := context.CollectionFactory().RemoteRepoCollection().ByName(name)
		if err != nil {
			return fmt.Errorf("unable to search: %s", err)
		}

		err = context.CollectionFactory().RemoteRepoCollection().LoadComplete(repo)
		if err != nil {
			return fmt.Errorf("unable to search: %s", err)
		}

		reflist = repo.RefList()
	} else if command == "repo" {
		repo, err := context.CollectionFactory().LocalRepoCollection().ByName(name)
		if err != nil {
			return fmt.Errorf("unable to search: %s", err)
		}

		err = context.CollectionFactory().LocalRepoCollection().LoadComplete(repo)
		if err != nil {
			return fmt.Errorf("unable to search: %s", err)
		}

		reflist = repo.RefList()
	} else {
		panic("unknown command")
	}

	list, err := deb.NewPackageListFromRefList(reflist, context.CollectionFactory().PackageCollection(), context.Progress())
	if err != nil {
		return fmt.Errorf("unable to search: %s", err)
	}

	list.PrepareIndex()

	q, err := query.Parse(args[1])
	if err != nil {
		return fmt.Errorf("unable to search: %s", err)
	}

	result, err := list.Filter([]deb.PackageQuery{q}, context.flags.Lookup("with-deps").Value.Get().(bool),
		nil, context.DependencyOptions(), context.ArchitecturesList())
	if err != nil {
		return fmt.Errorf("unable to search: %s", err)
	}

	result.ForEach(func(p *deb.Package) error {
		context.Progress().Printf("%s\n", p)
		return nil
	})

	return err
}

func makeCmdSnapshotSearch() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotMirrorRepoSearch,
		UsageLine: "search <name> <package-query>",
		Short:     "search snapshot for packages matching query",
		Long: `
Command search displays list of packages in snapshot that match package query

Example:

    $ aptly snapshot search wheezy-main '$Architecture (i386), Name (% *-dev)'
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-search", flag.ExitOnError),
	}

	cmd.Flag.Bool("with-deps", false, "include dependencies into search results")

	return cmd
}
