package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/commander"
	"github.com/smira/flag"
	"strings"
)

func aptlyPublishSwitch(cmd *commander.Command, args []string) error {
	var err error

	components := strings.Split(context.flags.Lookup("component").Value.String(), ",")

	if len(args) < len(components)+1 || len(args) > len(components)+2 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	distribution := args[0]
	param := "."

	var (
		names    []string
		snapshot *deb.Snapshot
	)

	if len(args) == len(components)+2 {
		param = args[1]
		names = args[2:]
	} else {
		names = args[1:]
	}

	storage, prefix := parsePrefix(param)

	var published *deb.PublishedRepo

	published, err = context.CollectionFactory().PublishedRepoCollection().ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	if published.SourceKind != "snapshot" {
		return fmt.Errorf("unable to update: not a snapshot publish")
	}

	err = context.CollectionFactory().PublishedRepoCollection().LoadComplete(published, context.CollectionFactory())
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	publishedComponents := published.Components()
	if len(components) == 1 && len(publishedComponents) == 1 && components[0] == "" {
		components = publishedComponents
	}

	if len(names) != len(components) {
		return fmt.Errorf("mismatch in number of components (%d) and snapshots (%d)", len(components), len(names))
	}

	for i, component := range components {
		snapshot, err = context.CollectionFactory().SnapshotCollection().ByName(names[i])
		if err != nil {
			return fmt.Errorf("unable to switch: %s", err)
		}

		err = context.CollectionFactory().SnapshotCollection().LoadComplete(snapshot)
		if err != nil {
			return fmt.Errorf("unable to switch: %s", err)
		}

		published.UpdateSnapshot(component, snapshot)
	}

	signer, err := getSigner(context.flags)
	if err != nil {
		return fmt.Errorf("unable to initialize GPG signer: %s", err)
	}

	forceOverwrite := context.flags.Lookup("force-overwrite").Value.Get().(bool)
	if forceOverwrite {
		context.Progress().ColoredPrintf("@rWARNING@|: force overwrite mode enabled, aptly might corrupt other published repositories sharing " +
			"the same package pool.\n")
	}

	err = published.Publish(context.PackagePool(), context, context.CollectionFactory(), signer, context.Progress(), forceOverwrite)
	if err != nil {
		return fmt.Errorf("unable to publish: %s", err)
	}

	err = context.CollectionFactory().PublishedRepoCollection().Update(published)
	if err != nil {
		return fmt.Errorf("unable to save to DB: %s", err)
	}

	err = context.CollectionFactory().PublishedRepoCollection().CleanupPrefixComponentFiles(published.Prefix, components,
		context.GetPublishedStorage(storage), context.CollectionFactory(), context.Progress())
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	context.Progress().Printf("\nPublish for snapshot %s has been successfully switched to new snapshot.\n", published.String())

	return err
}

func makeCmdPublishSwitch() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishSwitch,
		UsageLine: "switch <distribution> [[<endpoint>:]<prefix>] <new-snapshot>",
		Short:     "update published repository by switching to new snapshot",
		Long: `
Command switches in-place published repository with new snapshot contents. All
publishing parameters are preserved (architecture list, distribution,
component).

For multiple component repositories, flag -component should be given with
list of components to update. Corresponding snapshots should be given in the
same order, e.g.:

	aptly publish update -component=main,contrib wheezy wh-main wh-contrib

Example:

    $ aptly publish update wheezy ppa wheezy-7.5
`,
		Flag: *flag.NewFlagSet("aptly-publish-switch", flag.ExitOnError),
	}
	cmd.Flag.String("gpg-key", "", "GPG key ID to use when signing the release")
	cmd.Flag.Var(&keyRingsFlag{}, "keyring", "GPG keyring to use (instead of default)")
	cmd.Flag.String("secret-keyring", "", "GPG secret keyring to use (instead of default)")
	cmd.Flag.Bool("skip-signing", false, "don't sign Release files with GPG")
	cmd.Flag.String("component", "", "component names to update (for multi-component publishing, separate components with commas)")
	cmd.Flag.Bool("force-overwrite", false, "overwrite files in package pool in case of mismatch")

	return cmd
}
