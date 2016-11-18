package cmd

import (
	"fmt"
	"sort"

	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/query"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlySnapshotMirrorRepoSearch(cmd *commander.Command, args []string) error {
	var (
		err error
		q   deb.PackageQuery
	)

	if len(args) < 1 || len(args) > 2 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	name := args[0]
	command := cmd.Parent.Name()
	collectionFactory := context.NewCollectionFactory()

	var reflist *deb.PackageRefList

	if command == "snapshot" { // nolint: goconst
		var snapshot *deb.Snapshot
		snapshot, err = collectionFactory.SnapshotCollection().ByName(name)
		if err != nil {
			return fmt.Errorf("unable to search: %s", err)
		}

		err = collectionFactory.SnapshotCollection().LoadComplete(snapshot)
		if err != nil {
			return fmt.Errorf("unable to search: %s", err)
		}

		reflist = snapshot.RefList()
	} else if command == "mirror" {
		var repo *deb.RemoteRepo
		repo, err = collectionFactory.RemoteRepoCollection().ByName(name)
		if err != nil {
			return fmt.Errorf("unable to search: %s", err)
		}

		err = collectionFactory.RemoteRepoCollection().LoadComplete(repo)
		if err != nil {
			return fmt.Errorf("unable to search: %s", err)
		}

		reflist = repo.RefList()
	} else if command == "repo" { // nolint: goconst
		var repo *deb.LocalRepo
		repo, err = collectionFactory.LocalRepoCollection().ByName(name)
		if err != nil {
			return fmt.Errorf("unable to search: %s", err)
		}

		err = collectionFactory.LocalRepoCollection().LoadComplete(repo)
		if err != nil {
			return fmt.Errorf("unable to search: %s", err)
		}

		reflist = repo.RefList()
	} else {
		panic("unknown command")
	}

	list, err := deb.NewPackageListFromRefList(reflist, collectionFactory.PackageCollection(), context.Progress())
	if err != nil {
		return fmt.Errorf("unable to search: %s", err)
	}

	list.PrepareIndex()

	if len(args) == 2 {
		q, err = query.Parse(args[1])
		if err != nil {
			return fmt.Errorf("unable to search: %s", err)
		}
	} else {
		q = &deb.MatchAllQuery{}
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

	result, err := list.FilterWithProgress([]deb.PackageQuery{q}, withDeps,
		nil, context.DependencyOptions(), architecturesList, context.Progress())
	if err != nil {
		return fmt.Errorf("unable to search: %s", err)
	}

	if result.Len() == 0 {
		return fmt.Errorf("no results")
	}

	format := context.Flags().Lookup("format").Value.String()
	PrintPackageList(result, format, "")

	return err
}

func makeCmdSnapshotSearch() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotMirrorRepoSearch,
		UsageLine: "search <name> [<package-query>]",
		Short:     "search snapshot for packages matching query",
		Long: `
Command search displays list of packages in snapshot that match package query

If query is not specified, all the packages are displayed.

Example:

    $ aptly snapshot search wheezy-main '$Architecture (i386), Name (% *-dev)'
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-search", flag.ExitOnError),
	}

	cmd.Flag.Bool("with-deps", false, "include dependencies into search results")
	cmd.Flag.String("format", "", "custom format for result printing")

	return cmd
}
