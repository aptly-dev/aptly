package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyPublishRemove(cmd *commander.Command, args []string) error {
	var err error

	if len(args) < 2 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	distribution := args[0]
	components := args[1:]

	param := context.Flags().Lookup("prefix").Value.String()
	if param == "" {
		param = "."
	}
	storage, prefix := deb.ParsePrefix(param)

	collectionFactory := context.NewCollectionFactory()
	published, err := collectionFactory.PublishedRepoCollection().ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	err = collectionFactory.PublishedRepoCollection().LoadComplete(published, collectionFactory)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	multiDist := context.Flags().Lookup("multi-dist").Value.Get().(bool)

	for _, component := range components {
		_, exists := published.Sources[component]
		if !exists {
			return fmt.Errorf("unable to update: '%s' is not a published component", component)
		}
		published.DropComponent(component)
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

	err = published.Publish(context.PackagePool(), context, collectionFactory, signer, context.Progress(), forceOverwrite, multiDist)
	if err != nil {
		return fmt.Errorf("unable to publish: %s", err)
	}

	err = collectionFactory.PublishedRepoCollection().Update(published)
	if err != nil {
		return fmt.Errorf("unable to save to DB: %s", err)
	}

	skipCleanup := context.Flags().Lookup("skip-cleanup").Value.Get().(bool)
	if !skipCleanup {
		publishedStorage := context.GetPublishedStorage(storage)

		err = collectionFactory.PublishedRepoCollection().CleanupPrefixComponentFiles(published.Prefix, components,
			publishedStorage, collectionFactory, context.Progress())
		if err != nil {
			return fmt.Errorf("unable to update: %s", err)
		}

		for _, component := range components {
			err = publishedStorage.RemoveDirs(filepath.Join(prefix, "dists", published.Distribution, component), context.Progress())
			if err != nil {
				return fmt.Errorf("unable to update: %s", err)
			}
		}
	}

	return err
}

func makeCmdPublishRemove() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishRemove,
		UsageLine: "remove <distribution> <component>...",
		Short:     "remove component from published repository",
		Long: `
Command removes one or multiple components from a published repository.

Example:

  $ aptly publish remove -prefix=filesystem:symlink:/debian wheezy contrib non-free
`,
		Flag: *flag.NewFlagSet("aptly-publish-remove", flag.ExitOnError),
	}
	cmd.Flag.String("gpg-key", "", "GPG key ID to use when signing the release")
	cmd.Flag.Var(&keyRingsFlag{}, "keyring", "GPG keyring to use (instead of default)")
	cmd.Flag.String("secret-keyring", "", "GPG secret keyring to use (instead of default)")
	cmd.Flag.String("passphrase", "", "GPG passphrase for the key (warning: could be insecure)")
	cmd.Flag.String("passphrase-file", "", "GPG passphrase-file for the key (warning: could be insecure)")
	cmd.Flag.Bool("batch", false, "run GPG with detached tty")
	cmd.Flag.Bool("skip-signing", false, "don't sign Release files with GPG")
	cmd.Flag.String("prefix", "", "publishing prefix in the form of [<endpoint>:]<prefix>")
	cmd.Flag.Bool("force-overwrite", false, "overwrite files in package pool in case of mismatch")
	cmd.Flag.Bool("skip-cleanup", false, "don't remove unreferenced files in prefix/component")
	cmd.Flag.Bool("multi-dist", false, "enable multiple packages with the same filename in different distributions")

	return cmd
}
