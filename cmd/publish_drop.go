package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
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

	publishedCollecton := debian.NewPublishedRepoCollection(context.database)

	err = publishedCollecton.Remove(context.publishedStorage, prefix, distribution)
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
		Short:     "removes files of published repository",
		Long: `
Command removes whatever has been published under specified prefix and distribution name.

ex.
    $ aptly publish drop wheezy
`,
		Flag: *flag.NewFlagSet("aptly-publish-drop", flag.ExitOnError),
	}

	return cmd
}
