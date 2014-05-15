package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/commander"
	"sort"
)

func aptlyPublishList(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 0 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	raw := cmd.Flag.Lookup("raw").Value.Get().(bool)

	published := make([]string, 0, context.CollectionFactory().PublishedRepoCollection().Len())

	err = context.CollectionFactory().PublishedRepoCollection().ForEach(func(repo *deb.PublishedRepo) error {
		err := context.CollectionFactory().PublishedRepoCollection().LoadComplete(repo, context.CollectionFactory())
		if err != nil {
			return err
		}

		if raw {
			published = append(published, fmt.Sprintf("%s %s", repo.Prefix, repo.Distribution))
		} else {
			published = append(published, repo.String())
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("unable to load list of repos: %s", err)
	}

	sort.Strings(published)

	if raw {
		for _, info := range published {
			fmt.Printf("%s\n", info)
		}
	} else {
		if len(published) == 0 {
			fmt.Printf("No snapshots/local repos have been published. Publish a snapshot by running `aptly publish snapshot ...`.\n")
			return err
		}

		fmt.Printf("Published repositories:\n")

		for _, description := range published {
			fmt.Printf("  * %s\n", description)
		}
	}

	return err
}

func makeCmdPublishList() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishList,
		UsageLine: "list",
		Short:     "list of published repositories",
		Long: `
Display list of currently published snapshots.

Example:

    $ aptly publish list
`,
	}

	cmd.Flag.Bool("raw", false, "display list in machine-readable format")

	return cmd
}
