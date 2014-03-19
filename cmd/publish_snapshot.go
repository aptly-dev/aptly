package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
	"github.com/smira/aptly/utils"
	"strings"
)

func aptlyPublishSnapshot(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 1 || len(args) > 2 {
		cmd.Usage()
		return err
	}

	name := args[0]

	var prefix string
	if len(args) == 2 {
		prefix = args[1]
	} else {
		prefix = ""
	}

	snapshot, err := context.collectionFactory.SnapshotCollection().ByName(name)
	if err != nil {
		return fmt.Errorf("unable to publish: %s", err)
	}

	err = context.collectionFactory.SnapshotCollection().LoadComplete(snapshot)
	if err != nil {
		return fmt.Errorf("unable to publish: %s", err)
	}

	component := cmd.Flag.Lookup("component").Value.String()
	distribution := cmd.Flag.Lookup("distribution").Value.String()

	published, err := debian.NewPublishedRepo(prefix, distribution, component, context.architecturesList, snapshot, context.collectionFactory)
	if err != nil {
		return fmt.Errorf("unable to publish: %s", err)
	}

	duplicate := context.collectionFactory.PublishedRepoCollection().CheckDuplicate(published)
	if duplicate != nil {
		context.collectionFactory.PublishedRepoCollection().LoadComplete(duplicate, context.collectionFactory)
		return fmt.Errorf("prefix/distribution already used by another published repo: %s", duplicate)
	}

	signer, err := getSigner(cmd)
	if err != nil {
		return fmt.Errorf("unable to initialize GPG signer: %s", err)
	}

	err = published.Publish(context.packagePool, context.publishedStorage, context.collectionFactory, signer, context.progress)
	if err != nil {
		return fmt.Errorf("unable to publish: %s", err)
	}

	err = context.collectionFactory.PublishedRepoCollection().Add(published)
	if err != nil {
		return fmt.Errorf("unable to save to DB: %s", err)
	}

	prefix, component, distribution = published.Prefix, published.Component, published.Distribution
	if prefix == "." {
		prefix = ""
	} else if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	context.progress.Printf("\nSnapshot %s has been successfully published.\nPlease setup your webserver to serve directory '%s' with autoindexing.\n",
		snapshot.Name, context.publishedStorage.PublicPath())
	context.progress.Printf("Now you can add following line to apt sources:\n")
	context.progress.Printf("  deb http://your-server/%s %s %s\n", prefix, distribution, component)
	if utils.StrSliceHasItem(published.Architectures, "source") {
		context.progress.Printf("  deb-src http://your-server/%s %s %s\n", prefix, distribution, component)
	}
	context.progress.Printf("Don't forget to add your GPG key to apt with apt-key.\n")
	context.progress.Printf("\nYou can also use `aptly serve` to publish your repositories over HTTP quickly.\n")

	return err
}

func makeCmdPublishSnapshot() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishSnapshot,
		UsageLine: "snapshot <name> [<prefix>]",
		Short:     "publish snapshot",
		Long: `
Command publish publishes snapshot as Debian repository ready to be consumed
by apt tools. Published repostiories appear under rootDir/public directory.
Valid GPG key is required for publishing.

Example:

    $ aptly publish snapshot wheezy-main
`,
		Flag: *flag.NewFlagSet("aptly-publish-snapshot", flag.ExitOnError),
	}
	cmd.Flag.String("distribution", "", "distribution name to publish")
	cmd.Flag.String("component", "", "component name to publish")
	cmd.Flag.String("gpg-key", "", "GPG key ID to use when signing the release")
	cmd.Flag.String("keyring", "", "GPG keyring to use (instead of default)")
	cmd.Flag.String("secret-keyring", "", "GPG secret keyring to use (instead of default)")
	cmd.Flag.Bool("skip-signing", false, "don't sign Release files with GPG")

	return cmd
}
