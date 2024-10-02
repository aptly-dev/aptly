package cmd

import (
	"fmt"
	"strings"

	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/utils"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyPublishAdd(cmd *commander.Command, args []string) error {
	var (
		err       error
		names     []string
		published *deb.PublishedRepo
	)

	components := strings.Split(context.Flags().Lookup("component").Value.String(), ",")

	if len(args) < len(components)+1 || len(args) > len(components)+2 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	distribution := args[0]
	param := "."

	if len(args) == len(components)+2 {
		param = args[1]
		names = args[2:]
	} else {
		names = args[1:]
	}

	if len(names) != len(components) {
		return fmt.Errorf("mismatch in number of components (%d) and sources (%d)", len(components), len(names))
	}

	storage, prefix := deb.ParsePrefix(param)

	collectionFactory := context.NewCollectionFactory()
	published, err = collectionFactory.PublishedRepoCollection().ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		return fmt.Errorf("unable to add: %s", err)
	}

	err = collectionFactory.PublishedRepoCollection().LoadComplete(published, collectionFactory)
	if err != nil {
		return fmt.Errorf("unable to add: %s", err)
	}

	publishedComponents := published.Components()

	if published.SourceKind == deb.SourceLocalRepo {
		localRepoCollection := collectionFactory.LocalRepoCollection()
		for i, component := range components {
			if utils.StrSliceHasItem(publishedComponents, component) {
				return fmt.Errorf("unable to add: component %s already exists in published local repository", component)
			}

			localRepo, err := localRepoCollection.ByName(names[i])
			if err != nil {
				return fmt.Errorf("unable to add: %s", err)
			}

			err = localRepoCollection.LoadComplete(localRepo)
			if err != nil {
				return fmt.Errorf("unable to add: %s", err)
			}

			published.UpsertLocalRepo(component, localRepo)
		}
	} else if published.SourceKind == deb.SourceSnapshot {
		snapshotCollection := collectionFactory.SnapshotCollection()
		for i, component := range components {
			if utils.StrSliceHasItem(publishedComponents, component) {
				return fmt.Errorf("unable to add: component %s already exists in published snapshot repository", component)
			}

			snapshot, err := snapshotCollection.ByName(names[i])
			if err != nil {
				return fmt.Errorf("unable to add: %s", err)
			}

			err = snapshotCollection.LoadComplete(snapshot)
			if err != nil {
				return fmt.Errorf("unable to add: %s", err)
			}

			published.UpsertSnapshot(component, snapshot)
		}
	} else {
		return fmt.Errorf("unknown published repository type")
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

	err = published.Publish(context.PackagePool(), context, collectionFactory, signer, context.Progress(), forceOverwrite)
	if err != nil {
		return fmt.Errorf("unable to publish: %s", err)
	}

	err = collectionFactory.PublishedRepoCollection().Update(published)
	if err != nil {
		return fmt.Errorf("unable to save to DB: %s", err)
	}

	context.Progress().Printf("\nPublished %s repository %s has been successfully updated by adding new source.\n", published.SourceKind, published.String())

	return err
}

func makeCmdPublishAdd() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishAdd,
		UsageLine: "add <distribution> [[<endpoint>:]<prefix>] <source>",
		Short:     "add package source to published repository",
		Long: `
The command adds (in place) one or multiple package sources to a published repository.
All publishing parameters are preserved (architecture list, distribution, ...).

The flag -component is mandatory. Use a comma-separated list of components,
if multiple components should be added. The number of given components must be
equal to the number of sources, e.g.:

	aptly publish add -component=main,contrib wheezy wheezy-main wheezy-contrib

Example:

	$ aptly publish add -component=contrib wheezy ppa wheezy-contrib

This command assigns the snapshot wheezy-contrib to the component contrib and
adds it as new package source to the published repository ppa/wheezy.
`,
		Flag: *flag.NewFlagSet("aptly-publish-add", flag.ExitOnError),
	}
	cmd.Flag.String("gpg-key", "", "GPG key ID to use when signing the release")
	cmd.Flag.Var(&keyRingsFlag{}, "keyring", "GPG keyring to use (instead of default)")
	cmd.Flag.String("secret-keyring", "", "GPG secret keyring to use (instead of default)")
	cmd.Flag.String("passphrase", "", "GPG passphrase for the key (warning: could be insecure)")
	cmd.Flag.String("passphrase-file", "", "GPG passphrase-file for the key (warning: could be insecure)")
	cmd.Flag.String("prefix", "", "publishing prefix in the form of [<endpoint>:]<prefix>")
	cmd.Flag.Bool("batch", false, "run GPG with detached tty")
	cmd.Flag.Bool("skip-signing", false, "don't sign Release files with GPG")
	cmd.Flag.Bool("skip-contents", false, "don't generate Contents indexes")
	cmd.Flag.Bool("skip-bz2", false, "don't generate bzipped indexes")
	cmd.Flag.String("component", "", "component names to add (for multi-component publishing, separate components with commas)")
	cmd.Flag.Bool("force-overwrite", false, "overwrite files in package pool in case of mismatch")
	cmd.Flag.Bool("multi-dist", false, "enable multiple packages with the same filename in different distributions")

	return cmd
}
