package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
	"sort"
)

func aptlyPublishList(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 0 {
		cmd.Usage()
		return err
	}

	publishedCollecton := debian.NewPublishedRepoCollection(context.database)
	snapshotCollection := debian.NewSnapshotCollection(context.database)

	if publishedCollecton.Len() == 0 {
		fmt.Printf("No snapshots have been published. Publish a snapshot by running `aptly publish snapshot ...`.\n")
		return err
	}

	published := make([]string, 0, publishedCollecton.Len())

	err = publishedCollecton.ForEach(func(repo *debian.PublishedRepo) error {
		err := publishedCollecton.LoadComplete(repo, snapshotCollection)
		if err != nil {
			return err
		}

		published = append(published, repo.String())
		return nil
	})

	if err != nil {
		return fmt.Errorf("unable to load list of repos: %s", err)
	}

	sort.Strings(published)

	fmt.Printf("Published repositories:\n")

	for _, description := range published {
		fmt.Printf("  * %s\n", description)
	}

	return err
}

func makeCmdPublishList() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishList,
		UsageLine: "list",
		Short:     "displays list of published repositories",
		Long: `
Display command displays list of currently published snapshots with information about published root.

ex.
    $ aptly publish list
`,
		Flag: *flag.NewFlagSet("aptly-publish-list", flag.ExitOnError),
	}

	return cmd
}
