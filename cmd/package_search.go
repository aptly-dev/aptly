package cmd

import (
	"fmt"

	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/query"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyPackageSearch(cmd *commander.Command, args []string) error {
	var (
		err error
		q   deb.PackageQuery
	)

	if len(args) > 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	if len(args) == 1 {
		q, err = query.Parse(args[0])
		if err != nil {
			return fmt.Errorf("unable to search: %s", err)
		}
	} else {
		q = &deb.MatchAllQuery{}
	}

	collectionFactory := context.NewCollectionFactory()
	result := q.Query(collectionFactory.PackageCollection())
	if result.Len() == 0 {
		return fmt.Errorf("no results")
	}

	format := context.Flags().Lookup("format").Value.String()
	PrintPackageList(result, format, "")

	return err
}

func makeCmdPackageSearch() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPackageSearch,
		UsageLine: "search [<package-query>]",
		Short:     "search for packages matching query",
		Long: `
Command search displays list of packages in whole DB that match package query.

If query is not specified, all the packages are displayed.

Example:

    $ aptly package search '$Architecture (i386), Name (% *-dev)'
`,
		Flag: *flag.NewFlagSet("aptly-package-search", flag.ExitOnError),
	}

	cmd.Flag.String("format", "", "custom format for result printing")

	return cmd
}
