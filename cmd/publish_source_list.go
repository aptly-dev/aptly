package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyPublishSourceList(cmd *commander.Command, args []string) error {
	if len(args) != 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	prefix := context.Flags().Lookup("prefix").Value.String()
	distribution := args[0]
	storage, prefix := deb.ParsePrefix(prefix)

	published, err := context.NewCollectionFactory().PublishedRepoCollection().ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		return fmt.Errorf("unable to list: %s", err)
	}

	err = context.NewCollectionFactory().PublishedRepoCollection().LoadComplete(published, context.NewCollectionFactory())
	if err != nil {
		return err
	}

	if published.Revision == nil {
		return fmt.Errorf("unable to list: no source changes exist")
	}

	jsonFlag := cmd.Flag.Lookup("json").Value.Get().(bool)

	if jsonFlag {
		return aptlyPublishSourceListJSON(published)
	}

	return aptlyPublishSourceListTxt(published)
}

func aptlyPublishSourceListTxt(published *deb.PublishedRepo) error {
	revision := published.Revision

	fmt.Printf("Sources:\n")
	for _, component := range revision.Components() {
		name := revision.Sources[component]
		fmt.Printf("  %s: %s [%s]\n", component, name, published.SourceKind)
	}

	return nil
}

func aptlyPublishSourceListJSON(published *deb.PublishedRepo) error {
	revision := published.Revision

	output, err := json.MarshalIndent(revision.SourceList(), "", "  ")
	if err != nil {
		return fmt.Errorf("unable to list: %s", err)
	}

	fmt.Println(string(output))

	return nil
}

func makeCmdPublishSourceList() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishSourceList,
		UsageLine: "list <distribution>",
		Short:     "lists revision of published repository",
		Long: `
Command lists sources of a published repository.

Example:

    $ aptly publish source list wheezy
`,
		Flag: *flag.NewFlagSet("aptly-publish-source-list", flag.ExitOnError),
	}
	cmd.Flag.Bool("json", false, "display record in JSON format")
	cmd.Flag.String("prefix", ".", "publishing prefix in the form of [<endpoint>:]<prefix>")
	cmd.Flag.String("component", "", "component names to add (for multi-component publishing, separate components with commas)")

	return cmd
}
