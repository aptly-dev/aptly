package cmd

import (
	"fmt"
	"strings"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyPublishSourceReplace(cmd *commander.Command, args []string) error {
	if len(args) < 2 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	distribution := args[0]
	names := args[1:]
	components := strings.Split(context.Flags().Lookup("component").Value.String(), ",")

	if len(names) != len(components) {
		return fmt.Errorf("mismatch in number of components (%d) and sources (%d)", len(components), len(names))
	}

	prefix := context.Flags().Lookup("prefix").Value.String()
	storage, prefix := deb.ParsePrefix(prefix)

	collectionFactory := context.NewCollectionFactory()
	published, err := collectionFactory.PublishedRepoCollection().ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		return fmt.Errorf("unable to add: %s", err)
	}

	err = collectionFactory.PublishedRepoCollection().LoadComplete(published, collectionFactory)
	if err != nil {
		return fmt.Errorf("unable to add: %s", err)
	}

	revision := published.ObtainRevision()
	sources := revision.Sources
	context.Progress().Printf("Clearing staged source list...\n")
	clear(sources)

	for i, component := range components {
		name := names[i]
		context.Progress().Printf("Adding component '%s' with source '%s' [%s]...\n", component, name, published.SourceKind)

		sources[component] = name
	}

	err = collectionFactory.PublishedRepoCollection().Update(published)
	if err != nil {
		return fmt.Errorf("unable to save to DB: %s", err)
	}

	context.Progress().Printf("\nYou can run 'aptly publish update %s %s' to update the content of the published repository.\n",
		distribution, published.StoragePrefix())

	return err
}

func makeCmdPublishSourceReplace() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishSourceReplace,
		UsageLine: "replace <distribution> <source>",
		Short:     "replace the source components of a published repository",
		Long: `
The command replaces the source components of a snapshot or local repository to be published.

This does not publish the changes directly, but rather schedules them for a subsequent 'aptly publish update'.

The flag -component is mandatory. Use a comma-separated list of components, if
multiple components should be modified. The number of given components must be
equal to the number of given sources, e.g.:

	aptly publish source replace -component=main,contrib wheezy wheezy-main wheezy-contrib

Example:

	$ aptly publish source replace -component=contrib wheezy ppa wheezy-contrib
`,
		Flag: *flag.NewFlagSet("aptly-publish-source-add", flag.ExitOnError),
	}
	cmd.Flag.String("prefix", ".", "publishing prefix in the form of [<endpoint>:]<prefix>")
	cmd.Flag.String("component", "", "component names to add (for multi-component publishing, separate components with commas)")

	return cmd
}
