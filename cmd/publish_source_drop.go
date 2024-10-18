package cmd

import (
	"fmt"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyPublishSourceDrop(cmd *commander.Command, args []string) error {
	if len(args) != 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	prefix := context.Flags().Lookup("prefix").Value.String()
	distribution := args[0]
	storage, prefix := deb.ParsePrefix(prefix)

	collectionFactory := context.NewCollectionFactory()
	published, err := collectionFactory.PublishedRepoCollection().ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		return fmt.Errorf("unable to drop: %s", err)
	}

	err = collectionFactory.PublishedRepoCollection().LoadComplete(published, collectionFactory)
	if err != nil {
		return fmt.Errorf("unable to drop: %s", err)
	}

	published.DropRevision()

	err = collectionFactory.PublishedRepoCollection().Update(published)
	if err != nil {
		return fmt.Errorf("unable to save to DB: %s", err)
	}

	context.Progress().Printf("Source changes have been removed successfully.\n")

	return err
}

func makeCmdPublishSourceDrop() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishSourceDrop,
		UsageLine: "drop <distribution>",
		Short:     "drops staged source changes of published repository",
		Long: `
Command drops the staged source changes of the published repository.

Example:

    $ aptly publish source drop wheezy
`,
		Flag: *flag.NewFlagSet("aptly-publish-source-drop", flag.ExitOnError),
	}
	cmd.Flag.String("prefix", ".", "publishing prefix in the form of [<endpoint>:]<prefix>")
	cmd.Flag.String("component", "", "component names to add (for multi-component publishing, separate components with commas)")

	return cmd
}
