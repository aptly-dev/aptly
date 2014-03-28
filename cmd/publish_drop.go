package cmd

import (
	"fmt"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyPublishDrop(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 1 || len(args) > 2 {
		cmd.Usage()
		return err
	}

	distribution := args[0]
	prefix := "."

	if len(args) == 2 {
		prefix = args[1]
	}

	err = context.collectionFactory.PublishedRepoCollection().Remove(context.publishedStorage, prefix, distribution)
	if err != nil {
		return fmt.Errorf("unable to remove: %s", err)
	}

	fmt.Printf("\nPublished repositroy has been removed successfully.\n")

	return err
}

func makeCmdPublishDrop() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishDrop,
		UsageLine: "drop <distribution> [<prefix>]",
		Short:     "remove published repository",
		Long: `
Command removes whatever has been published under specified <prefix> and
<distribution> name.

Example:

    $ aptly publish drop wheezy
`,
		Flag: *flag.NewFlagSet("aptly-publish-drop", flag.ExitOnError),
	}

	return cmd
}
