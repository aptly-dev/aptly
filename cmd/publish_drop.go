package cmd

import (
	"fmt"

	"github.com/smira/aptly/deb"
	"github.com/smira/commander"
)

func aptlyPublishDrop(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 1 || len(args) > 2 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	distribution := args[0]
	param := "."

	if len(args) == 2 {
		param = args[1]
	}

	storage, prefix := deb.ParsePrefix(param)

	err = context.CollectionFactory().PublishedRepoCollection().Remove(context, storage, prefix, distribution,
		context.CollectionFactory(), context.Progress(), context.Flags().Lookup("force-drop").Value.Get().(bool))
	if err != nil {
		return fmt.Errorf("unable to remove: %s", err)
	}

	context.Progress().Printf("\nPublished repository has been removed successfully.\n")

	return err
}

func makeCmdPublishDrop() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishDrop,
		UsageLine: "drop <distribution> [[<endpoint>:]<prefix>]",
		Short:     "remove published repository",
		Long: `
Command removes whatever has been published under specified <prefix>,
publishing <endpoint> and <distribution> name.

Example:

    $ aptly publish drop wheezy
`,
	}

	cmd.Flag.Bool("force-drop", false, "remove published repository even if some files could not be cleaned up")

	return cmd
}
