package cmd

import (
	"fmt"
	"strings"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyPublishSourceRemove(cmd *commander.Command, args []string) error {
	if len(args) < 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	distribution := args[0]
	components := strings.Split(context.Flags().Lookup("component").Value.String(), ",")

	if len(components) == 0 {
		return fmt.Errorf("unable to remove: missing components, specify at least one component")
	}

	prefix := context.Flags().Lookup("prefix").Value.String()
	storage, prefix := deb.ParsePrefix(prefix)

	collectionFactory := context.NewCollectionFactory()
	published, err := collectionFactory.PublishedRepoCollection().ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		return fmt.Errorf("unable to remove: %s", err)
	}

	err = collectionFactory.PublishedRepoCollection().LoadComplete(published, collectionFactory)
	if err != nil {
		return fmt.Errorf("unable to remove: %s", err)
	}

	revision := published.ObtainRevision()
	sources := revision.Sources

	for _, component := range components {
		name, exists := sources[component]
		if !exists {
			return fmt.Errorf("unable to remove: component '%s' does not exist", component)
		}
		context.Progress().Printf("Removing component '%s' with source '%s' [%s]...\n", component, name, published.SourceKind)

		delete(sources, component)
	}

	err = collectionFactory.PublishedRepoCollection().Update(published)
	if err != nil {
		return fmt.Errorf("unable to save to DB: %s", err)
	}

	context.Progress().Printf("\nYou can run 'aptly publish update %s %s' to update the content of the published repository.\n",
		distribution, published.StoragePrefix())

	return err
}

func makeCmdPublishSourceRemove() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishSourceRemove,
		UsageLine: "remove <distribution> [[<endpoint>:]<prefix>] <source>",
		Short:     "remove source from staged source list of published repository",
		Long: `
The command removes sources from the staged source list of the published repository.

The flag -component is mandatory. Use a comma-separated list of components, if
multiple components should be removed, e.g.:

Example:

	$ aptly publish remove -component=contrib,non-free wheezy filesystem:symlink:debian
`,
		Flag: *flag.NewFlagSet("aptly-publish-remove", flag.ExitOnError),
	}
	cmd.Flag.String("prefix", ".", "publishing prefix in the form of [<endpoint>:]<prefix>")
	cmd.Flag.String("component", "", "component names to remove (for multi-component publishing, separate components with commas)")

	return cmd
}
