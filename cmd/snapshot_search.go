package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/aptly/query"
	"github.com/smira/commander"
	"github.com/smira/flag"
	"sort"
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

	withDeps := context.Flags().Lookup("with-deps").Value.Get().(bool)
	architecturesList := []string{}

	if withDeps {
		if len(context.ArchitecturesList()) > 0 {
			architecturesList = context.ArchitecturesList()
		} else {
			architecturesList = list.Architectures(false)
		}

		sort.Strings(architecturesList)

		if len(architecturesList) == 0 {
			return fmt.Errorf("unable to determine list of architectures, please specify explicitly")
		}
	}

	result, err := list.Filter([]deb.PackageQuery{q}, withDeps,
		nil, context.DependencyOptions(), architecturesList)
	if err != nil {
		return fmt.Errorf("unable to search: %s", err)
	}

	if result.Len() == 0 {
		return fmt.Errorf("no results")
	}

	format := context.Flags().Lookup("format").Value.String()
	PrintPackageList(result, format)

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
	cmd.Flag.String("format", "", "custom format for result printing")

	return cmd
}
