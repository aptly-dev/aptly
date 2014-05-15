package cmd

import (
	"fmt"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyRepoEdit(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	repo, err := context.CollectionFactory().LocalRepoCollection().ByName(args[0])
	if err != nil {
		return fmt.Errorf("unable to edit: %s", err)
	}

	err = context.CollectionFactory().LocalRepoCollection().LoadComplete(repo)
	if err != nil {
		return fmt.Errorf("unable to edit: %s", err)
	}

	if context.flags.Lookup("comment").Value.String() != "" {
		repo.Comment = context.flags.Lookup("comment").Value.String()
	}

	if context.flags.Lookup("distribution").Value.String() != "" {
		repo.DefaultDistribution = context.flags.Lookup("distribution").Value.String()
	}

	if context.flags.Lookup("component").Value.String() != "" {
		repo.DefaultComponent = context.flags.Lookup("component").Value.String()
	}

	err = context.CollectionFactory().LocalRepoCollection().Update(repo)
	if err != nil {
		return fmt.Errorf("unable to edit: %s", err)
	}

	fmt.Printf("Local repo %s successfully updated.\n", repo)
	return err
}

func makeCmdRepoEdit() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRepoEdit,
		UsageLine: "edit <name>",
		Short:     "edit properties of local repository",
		Long: `
Command edit allows to change metadata of local repository:
comment, default distribution and component.

Example:

  $ aptly repo edit -distribution=wheezy testing
`,
		Flag: *flag.NewFlagSet("aptly-repo-edit", flag.ExitOnError),
	}

	cmd.Flag.String("comment", "", "any text that would be used to described local repository")
	cmd.Flag.String("distribution", "", "default distribution when publishing")
	cmd.Flag.String("component", "", "default component when publishing")

	return cmd
}
