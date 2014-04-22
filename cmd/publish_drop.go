package cmd

import (
	"fmt"
	"github.com/smira/commander"
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

	err = context.CollectionFactory().PublishedRepoCollection().Remove(context.PublishedStorage(), prefix, distribution,
		context.CollectionFactory(), context.Progress())
	if err != nil {
		return fmt.Errorf("unable to remove: %s", err)
	}

	context.Progress().Printf("\nPublished repository has been removed successfully.\n")

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
	}

	return cmd
}
