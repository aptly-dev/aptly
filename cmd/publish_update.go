package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
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
	storage, prefix := parsePrefix(param)

	var published *deb.PublishedRepo

	published, err = context.CollectionFactory().PublishedRepoCollection().ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	if published.SourceKind != "local" {
		return fmt.Errorf("unable to update: not a local repository publish")
	}

	err = context.CollectionFactory().PublishedRepoCollection().LoadComplete(published, context.CollectionFactory())
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	components := published.Components()
	for _, component := range components {
		published.UpdateLocalRepo(component)
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

	context.Progress().Printf("\nPublish for local repo %s has been successfully updated.\n", published.String())

	return err
}

func makeCmdPublishUpdate() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishUpdate,
		UsageLine: "update <distribution> [[<endpoint>:]<prefix>]",
		Short:     "update published local repository",
		Long: `
Command re-publishes (updates) published local repository. <distribution>
and <prefix> should be occupied with local repository published
using command aptly publish repo. Update happens in-place with
minimum possible downtime for published repository.

For multiple component published repositories, all local repositories
are updated.

Example:

    $ aptly publish update wheezy ppa
`,
		Flag: *flag.NewFlagSet("aptly-publish-update", flag.ExitOnError),
	}
	cmd.Flag.String("gpg-key", "", "GPG key ID to use when signing the release")
	cmd.Flag.Var(&keyRingsFlag{}, "keyring", "GPG keyring to use (instead of default)")
	cmd.Flag.String("secret-keyring", "", "GPG secret keyring to use (instead of default)")
	cmd.Flag.Bool("skip-signing", false, "don't sign Release files with GPG")
	cmd.Flag.Bool("force-overwrite", false, "overwrite files in package pool in case of mismatch")

	return cmd
}
