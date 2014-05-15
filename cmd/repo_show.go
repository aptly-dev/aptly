package cmd

import (
	"fmt"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyRepoShow(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	name := args[0]

	repo, err := context.CollectionFactory().LocalRepoCollection().ByName(name)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	err = context.CollectionFactory().LocalRepoCollection().LoadComplete(repo)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	fmt.Printf("Name: %s\n", repo.Name)
	fmt.Printf("Comment: %s\n", repo.Comment)
	fmt.Printf("Default Distribution: %s\n", repo.DefaultDistribution)
	fmt.Printf("Default Component: %s\n", repo.DefaultComponent)
	fmt.Printf("Number of packages: %d\n", repo.NumPackages())

	withPackages := context.flags.Lookup("with-packages").Value.Get().(bool)
	if withPackages {
		ListPackagesRefList(repo.RefList())
	}

	return err
}

func makeCmdRepoShow() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRepoShow,
		UsageLine: "show <name>",
		Short:     "show details about local repository",
		Long: `
Show command shows full information about local package repository.

ex:
  $ aptly repo show testing
`,
		Flag: *flag.NewFlagSet("aptly-repo-show", flag.ExitOnError),
	}

	cmd.Flag.Bool("with-packages", false, "show list of packages")

	return cmd
}
