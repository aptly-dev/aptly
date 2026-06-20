package cmd

import (
	"fmt"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyPublishUpdate(cmd *commander.Command, args []string) error {
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

	var published *deb.PublishedRepo

	collectionFactory := context.NewCollectionFactory()
	published, err = collectionFactory.PublishedRepoCollection().ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	err = collectionFactory.PublishedRepoCollection().LoadComplete(published, collectionFactory)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	result, err := published.Update(collectionFactory, context.Progress())
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	signer, err := getSigner(context.Flags())
	if err != nil {
		return fmt.Errorf("unable to initialize GPG signer: %s", err)
	}

	forceOverwrite := context.Flags().Lookup("force-overwrite").Value.Get().(bool)
	if forceOverwrite {
		context.Progress().ColoredPrintf("@rWARNING@|: force overwrite mode enabled, aptly might corrupt other published repositories sharing " +
			"the same package pool.\n")
	}

	if context.Flags().IsSet("skip-contents") {
		published.SkipContents = context.Flags().Lookup("skip-contents").Value.Get().(bool)
	}

	if context.Flags().IsSet("skip-bz2") {
		published.SkipBz2 = context.Flags().Lookup("skip-bz2").Value.Get().(bool)
	}

	if context.Flags().IsSet("multi-dist") {
		published.MultiDist = context.Flags().Lookup("multi-dist").Value.Get().(bool)
	}

	err = published.Publish(context.PackagePool(), context, collectionFactory, signer, context.Progress(), forceOverwrite, context.SkelPath())
	if err != nil {
		return fmt.Errorf("unable to publish: %s", err)
	}

	err = collectionFactory.PublishedRepoCollection().Update(published)
	if err != nil {
		return fmt.Errorf("unable to save to DB: %s", err)
	}

	skipCleanup := context.Flags().Lookup("skip-cleanup").Value.Get().(bool)
	if !skipCleanup {
		cleanComponents := make([]string, 0, len(result.UpdatedSources)+len(result.RemovedSources))
		cleanComponents = append(append(cleanComponents, result.UpdatedComponents()...), result.RemovedComponents()...)
		err = collectionFactory.PublishedRepoCollection().CleanupPrefixComponentFiles(context, published, cleanComponents, collectionFactory, context.Progress())
		if err != nil {
			return fmt.Errorf("unable to update: %s", err)
		}
	}

	context.Progress().Printf("\nPublished %s repository %s has been updated successfully.\n", published.SourceKind, published.String())

	return err
}

func makeCmdPublishUpdate() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishUpdate,
		UsageLine: "update <distribution> [[<endpoint>:]<prefix>]",
		Short:     "update published repository",
		Long: `
The command updates updates a published repository after applying pending changes to the sources.

For published local repositories:

    * update to match local repository contents

For published snapshots:

    * switch components to new snapshot

The update happens in-place with minimum possible downtime for published repository.

For multiple component published repositories, all local repositories are updated.

Example:

    $ aptly publish update wheezy ppa
`,
		Flag: *flag.NewFlagSet("aptly-publish-update", flag.ExitOnError),
	}
	cmd.Flag.Var(&gpgKeyFlag{}, "gpg-key", "GPG key ID to use when signing the release (flag is repeatable, can be specified multiple times)")
	cmd.Flag.Var(&keyRingsFlag{}, "keyring", "GPG keyring to use (instead of default)")
	cmd.Flag.String("secret-keyring", "", "GPG secret keyring to use (instead of default)")
	cmd.Flag.String("passphrase", "", "GPG passphrase for the key (warning: could be insecure)")
	cmd.Flag.String("passphrase-file", "", "GPG passphrase-file for the key (warning: could be insecure)")
	cmd.Flag.Bool("batch", false, "run GPG with detached tty")
	cmd.Flag.Bool("skip-signing", false, "don't sign Release files with GPG")
	cmd.Flag.Bool("skip-contents", false, "don't generate Contents indexes")
	cmd.Flag.Bool("skip-bz2", false, "don't generate bzipped indexes")
	cmd.Flag.Bool("force-overwrite", false, "overwrite files in package pool in case of mismatch")
	cmd.Flag.Bool("skip-cleanup", false, "don't remove unreferenced files in prefix/component")
	cmd.Flag.Bool("multi-dist", false, "enable multiple packages with the same filename in different distributions")

	return cmd
}
