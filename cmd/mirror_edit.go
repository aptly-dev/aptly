package cmd

import (
	"fmt"
	"github.com/smira/aptly/query"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyMirrorEdit(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	repo, err := context.CollectionFactory().RemoteRepoCollection().ByName(args[0])
	if err != nil {
		return fmt.Errorf("unable to edit: %s", err)
	}

	context.flags.Visit(func(flag *flag.Flag) {
		switch flag.Name {
		case "filter":
			repo.Filter = flag.Value.String()
		case "filter-with-deps":
			repo.FilterWithDeps = flag.Value.Get().(bool)
		}
	})

	if repo.Filter != "" {
		_, err = query.Parse(repo.Filter)
		if err != nil {
			return fmt.Errorf("unable to edit: %s", err)
		}
	}

	err = context.CollectionFactory().RemoteRepoCollection().Update(repo)
	if err != nil {
		return fmt.Errorf("unable to edit: %s", err)
	}

	fmt.Printf("Mirror %s successfully updated.\n", repo)
	return err
}

func makeCmdMirrorEdit() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyMirrorEdit,
		UsageLine: "edit <name>",
		Short:     "edit properties of mirorr",
		Long: `
Command edit allows to change settings of mirror:
filters.

Example:

  $ aptly mirror edit -filter=nginx -filter-with-deps some-mirror
`,
		Flag: *flag.NewFlagSet("aptly-mirror-edit", flag.ExitOnError),
	}

	cmd.Flag.String("filter", "", "filter packages in mirror")
	cmd.Flag.Bool("filter-with-deps", false, "when filtering, include dependencies of matching packages as well")

	return cmd
}
