package cmd

import (
	"fmt"

	"github.com/AlekSi/pointer"
	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyRepoEdit(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	collectionFactory := context.NewCollectionFactory()
	repo, err := collectionFactory.LocalRepoCollection().ByName(args[0])
	if err != nil {
		return fmt.Errorf("unable to edit: %s", err)
	}

	err = collectionFactory.LocalRepoCollection().LoadComplete(repo)
	if err != nil {
		return fmt.Errorf("unable to edit: %s", err)
	}

	var uploadersFile *string

	context.Flags().Visit(func(flag *flag.Flag) {
		switch flag.Name {
		case "comment":
			repo.Comment = flag.Value.String()
		case "distribution":
			repo.DefaultDistribution = flag.Value.String()
		case "component":
			repo.DefaultComponent = flag.Value.String()
		case "uploaders-file":
			uploadersFile = pointer.ToString(flag.Value.String())
		}
	})

	if uploadersFile != nil {
		if *uploadersFile != "" {
			repo.Uploaders, err = deb.NewUploadersFromFile(*uploadersFile)
			if err != nil {
				return err
			}
		} else {
			repo.Uploaders = nil
		}
	}

	err = collectionFactory.LocalRepoCollection().Update(repo)
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
Command edit allows one to change metadata of local repository:
comment, default distribution and component.

Example:

  $ aptly repo edit -distribution=wheezy testing
`,
		Flag: *flag.NewFlagSet("aptly-repo-edit", flag.ExitOnError),
	}

	cmd.Flag.String("comment", "", "any text that would be used to described local repository")
	cmd.Flag.String("distribution", "", "default distribution when publishing")
	cmd.Flag.String("component", "", "default component when publishing")
	cmd.Flag.String("uploaders-file", "", "uploaders.json to be used when including .changes into this repository")

	return cmd
}
