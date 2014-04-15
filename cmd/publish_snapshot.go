package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/aptly/utils"
	"github.com/smira/commander"
	"github.com/smira/flag"
	"strings"
)

func aptlyPublishSnapshotOrRepo(cmd *commander.Command, args []string) error {
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

	var (
		source  interface{}
		message string
	)

	if cmd.Name() == "snapshot" {
		var snapshot *deb.Snapshot
		snapshot, err = context.CollectionFactory().SnapshotCollection().ByName(name)
		if err != nil {
			return fmt.Errorf("unable to publish: %s", err)
		}

		err = context.CollectionFactory().SnapshotCollection().LoadComplete(snapshot)
		if err != nil {
			return fmt.Errorf("unable to publish: %s", err)
		}

		source = snapshot
		message = fmt.Sprintf("Snapshot %s", snapshot.Name)
	} else if cmd.Name() == "repo" {
		var localRepo *deb.LocalRepo
		localRepo, err = context.CollectionFactory().LocalRepoCollection().ByName(name)
		if err != nil {
			return fmt.Errorf("unable to publish: %s", err)
		}

		err = context.CollectionFactory().LocalRepoCollection().LoadComplete(localRepo)
		if err != nil {
			return fmt.Errorf("unable to publish: %s", err)
		}

		source = localRepo
		message = fmt.Sprintf("Local repo %s", localRepo.Name)
	} else {
		panic("unknown command")
	}

	component := context.flags.Lookup("component").Value.String()
	distribution := context.flags.Lookup("distribution").Value.String()

	published, err := deb.NewPublishedRepo(prefix, distribution, component, context.ArchitecturesList(), source, context.CollectionFactory())
	if err != nil {
		return fmt.Errorf("unable to publish: %s", err)
	}
	published.Origin = cmd.Flag.Lookup("origin").Value.String()
	published.Label = cmd.Flag.Lookup("label").Value.String()

	duplicate := context.CollectionFactory().PublishedRepoCollection().CheckDuplicate(published)
	if duplicate != nil {
		context.CollectionFactory().PublishedRepoCollection().LoadComplete(duplicate, context.CollectionFactory())
		return fmt.Errorf("prefix/distribution already used by another published repo: %s", duplicate)
	}

	signer, err := getSigner(context.flags)
	if err != nil {
		return fmt.Errorf("unable to initialize GPG signer: %s", err)
	}

	err = published.Publish(context.PackagePool(), context.PublishedStorage(), context.CollectionFactory(), signer, context.Progress())
	if err != nil {
		return fmt.Errorf("unable to publish: %s", err)
	}

	err = context.CollectionFactory().PublishedRepoCollection().Add(published)
	if err != nil {
		return fmt.Errorf("unable to save to DB: %s", err)
	}

	prefix, component, distribution = published.Prefix, published.Component, published.Distribution
	if prefix == "." {
		prefix = ""
	} else if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	context.Progress().Printf("\n%s has been successfully published.\nPlease setup your webserver to serve directory '%s' with autoindexing.\n",
		message, context.PublishedStorage().PublicPath())
	context.Progress().Printf("Now you can add following line to apt sources:\n")
	context.Progress().Printf("  deb http://your-server/%s %s %s\n", prefix, distribution, component)
	if utils.StrSliceHasItem(published.Architectures, "source") {
		context.Progress().Printf("  deb-src http://your-server/%s %s %s\n", prefix, distribution, component)
	}
	context.Progress().Printf("Don't forget to add your GPG key to apt with apt-key.\n")
	context.Progress().Printf("\nYou can also use `aptly serve` to publish your repositories over HTTP quickly.\n")

	return err
}

func makeCmdPublishSnapshot() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishSnapshotOrRepo,
		UsageLine: "snapshot <name> [<prefix>]",
		Short:     "publish snapshot",
		Long: `
Command publishes snapshot as Debian repository ready to be consumed
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
	cmd.Flag.Var(&keyRingsFlag{}, "keyring", "GPG keyring to use (instead of default)")
	cmd.Flag.String("secret-keyring", "", "GPG secret keyring to use (instead of default)")
	cmd.Flag.Bool("skip-signing", false, "don't sign Release files with GPG")
	cmd.Flag.String("origin", "", "origin name to publish")
	cmd.Flag.String("label", "", "label to publish")

	return cmd
}
