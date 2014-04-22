package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyPublishSwitch(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 2 || len(args) > 3 {
		cmd.Usage()
		return err
	}

	distribution := args[0]
	prefix := "."

	var (
		name     string
		snapshot *deb.Snapshot
	)

	if len(args) == 3 {
		prefix = args[1]
		name = args[2]
	} else {
		name = args[1]
	}

	snapshot, err = context.CollectionFactory().SnapshotCollection().ByName(name)
	if err != nil {
		return fmt.Errorf("unable to switch: %s", err)
	}

	err = context.CollectionFactory().SnapshotCollection().LoadComplete(snapshot)
	if err != nil {
		return fmt.Errorf("unable to switch: %s", err)
	}

	var published *deb.PublishedRepo

	published, err = context.CollectionFactory().PublishedRepoCollection().ByPrefixDistribution(prefix, distribution)
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

	published.UpdateSnapshot(snapshot)

	signer, err := getSigner(context.flags)
	if err != nil {
		return fmt.Errorf("unable to initialize GPG signer: %s", err)
	}

	err = published.Publish(context.PackagePool(), context.PublishedStorage(), context.CollectionFactory(), signer, context.Progress())
	if err != nil {
		return fmt.Errorf("unable to publish: %s", err)
	}

	err = context.CollectionFactory().PublishedRepoCollection().Update(published)
	if err != nil {
		return fmt.Errorf("unable to save to DB: %s", err)
	}

	err = context.CollectionFactory().PublishedRepoCollection().CleanupPrefixComponentFiles(published.Prefix, published.Component,
		context.PublishedStorage(), context.CollectionFactory(), context.Progress())
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	context.Progress().Printf("\nPublish for snapshot %s has been successfully switched to new snapshot.\n", published.String())

	return err
}

func makeCmdPublishSwitch() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishSwitch,
		UsageLine: "switch <distribution> [<prefix>] <new-snapshot>",
		Short:     "update published repository by switching to new snapshot",
		Long: `
Command switches in-place published repository with new snapshot contents. All
publishing parameters are preserved (architecture list, distribution, component).

Example:

    $ aptly publish update wheezy ppa wheezy-7.5
`,
		Flag: *flag.NewFlagSet("aptly-publish-switch", flag.ExitOnError),
	}
	cmd.Flag.String("gpg-key", "", "GPG key ID to use when signing the release")
	cmd.Flag.Var(&keyRingsFlag{}, "keyring", "GPG keyring to use (instead of default)")
	cmd.Flag.String("secret-keyring", "", "GPG secret keyring to use (instead of default)")
	cmd.Flag.Bool("skip-signing", false, "don't sign Release files with GPG")

	return cmd
}
