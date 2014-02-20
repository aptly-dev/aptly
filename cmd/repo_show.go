package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
)

func aptlyRepoShow(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return err
	}

	name := args[0]

	localRepoCollection := debian.NewLocalRepoCollection(context.database)
	repo, err := localRepoCollection.ByName(name)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	err = localRepoCollection.LoadComplete(repo)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	fmt.Printf("Name: %s\n", repo.Name)
	fmt.Printf("Comment: %s\n", repo.Comment)
	fmt.Printf("Number of packages: %d\n", repo.NumPackages())

	withPackages := cmd.Flag.Lookup("with-packages").Value.Get().(bool)
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
Show shows full information about local package repository.

ex:
  $ aptly repo show testing
`,
		Flag: *flag.NewFlagSet("aptly-repo-show", flag.ExitOnError),
	}

	cmd.Flag.Bool("with-packages", false, "show list of packages")

	return cmd
}
