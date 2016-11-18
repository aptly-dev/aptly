package cmd

import (
	"fmt"
	"strings"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/utils"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyPublishSnapshotOrRepo(cmd *commander.Command, args []string) error {
	var err error

	components := strings.Split(context.Flags().Lookup("component").Value.String(), ",")
	collectionFactory := context.NewCollectionFactory()

	if len(args) < len(components) || len(args) > len(components)+1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	var param string
	if len(args) == len(components)+1 {
		param = args[len(components)]
		args = args[0 : len(args)-1]
	} else {
		param = ""
	}
	storage, prefix := deb.ParsePrefix(param)

	var (
		sources = []interface{}{}
		message string
	)

	if cmd.Name() == "snapshot" { // nolint: goconst
		var (
			snapshot     *deb.Snapshot
			emptyWarning = false
			parts        = []string{}
		)

		for _, name := range args {
			snapshot, err = collectionFactory.SnapshotCollection().ByName(name)
			if err != nil {
				return fmt.Errorf("unable to publish: %s", err)
			}

			err = collectionFactory.SnapshotCollection().LoadComplete(snapshot)
			if err != nil {
				return fmt.Errorf("unable to publish: %s", err)
			}

			sources = append(sources, snapshot)
			parts = append(parts, snapshot.Name)

			if snapshot.NumPackages() == 0 {
				emptyWarning = true
			}
		}

		if len(parts) == 1 {
			message = fmt.Sprintf("Snapshot %s has", parts[0])
		} else {
			message = fmt.Sprintf("Snapshots %s have", strings.Join(parts, ", "))

		}

		if emptyWarning {
			context.Progress().Printf("Warning: publishing from empty source, architectures list should be complete, it can't be changed after publishing (use -architectures flag)\n")
		}
	} else if cmd.Name() == "repo" { // nolint: goconst
		var (
			localRepo    *deb.LocalRepo
			emptyWarning = false
			parts        = []string{}
		)

		for _, name := range args {
			localRepo, err = collectionFactory.LocalRepoCollection().ByName(name)
			if err != nil {
				return fmt.Errorf("unable to publish: %s", err)
			}

			err = collectionFactory.LocalRepoCollection().LoadComplete(localRepo)
			if err != nil {
				return fmt.Errorf("unable to publish: %s", err)
			}

			sources = append(sources, localRepo)
			parts = append(parts, localRepo.Name)

			if localRepo.NumPackages() == 0 {
				emptyWarning = true
			}
		}

		if len(parts) == 1 {
			message = fmt.Sprintf("Local repo %s has", parts[0])
		} else {
			message = fmt.Sprintf("Local repos %s have", strings.Join(parts, ", "))

		}

		if emptyWarning {
			context.Progress().Printf("Warning: publishing from empty source, architectures list should be complete, it can't be changed after publishing (use -architectures flag)\n")
		}
	} else {
		panic("unknown command")
	}

	distribution := context.Flags().Lookup("distribution").Value.String()
	origin := context.Flags().Lookup("origin").Value.String()
	notAutomatic := context.Flags().Lookup("notautomatic").Value.String()
	butAutomaticUpgrades := context.Flags().Lookup("butautomaticupgrades").Value.String()

	published, err := deb.NewPublishedRepo(storage, prefix, distribution, context.ArchitecturesList(), components, sources, collectionFactory)
	if err != nil {
		return fmt.Errorf("unable to publish: %s", err)
	}
	if origin != "" {
		published.Origin = origin
	}
	if notAutomatic != "" {
		published.NotAutomatic = notAutomatic
	}
	if butAutomaticUpgrades != "" {
		published.ButAutomaticUpgrades = butAutomaticUpgrades
	}
	published.Label = context.Flags().Lookup("label").Value.String()
	published.Suite = context.Flags().Lookup("suite").Value.String()

	published.SkipContents = context.Config().SkipContentsPublishing

	if context.Flags().IsSet("skip-contents") {
		published.SkipContents = context.Flags().Lookup("skip-contents").Value.Get().(bool)
	}

	if context.Flags().IsSet("acquire-by-hash") {
		published.AcquireByHash = context.Flags().Lookup("acquire-by-hash").Value.Get().(bool)
	}

	duplicate := collectionFactory.PublishedRepoCollection().CheckDuplicate(published)
	if duplicate != nil {
		collectionFactory.PublishedRepoCollection().LoadComplete(duplicate, collectionFactory)
		return fmt.Errorf("prefix/distribution already used by another published repo: %s", duplicate)
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

	err = published.Publish(context.PackagePool(), context, collectionFactory, signer, context.Progress(), forceOverwrite)
	if err != nil {
		return fmt.Errorf("unable to publish: %s", err)
	}

	err = collectionFactory.PublishedRepoCollection().Add(published)
	if err != nil {
		return fmt.Errorf("unable to save to DB: %s", err)
	}

	var repoComponents string
	prefix, repoComponents, distribution = published.Prefix, strings.Join(published.Components(), " "), published.Distribution
	if prefix == "." {
		prefix = ""
	} else if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	context.Progress().Printf("\n%s been successfully published.\n", message)

	if localStorage, ok := context.GetPublishedStorage(storage).(aptly.FileSystemPublishedStorage); ok {
		context.Progress().Printf("Please setup your webserver to serve directory '%s' with autoindexing.\n",
			localStorage.PublicPath())
	}

	context.Progress().Printf("Now you can add following line to apt sources:\n")
	context.Progress().Printf("  deb http://your-server/%s %s %s\n", prefix, distribution, repoComponents)
	if utils.StrSliceHasItem(published.Architectures, deb.ArchitectureSource) {
		context.Progress().Printf("  deb-src http://your-server/%s %s %s\n", prefix, distribution, repoComponents)
	}
	context.Progress().Printf("Don't forget to add your GPG key to apt with apt-key.\n")
	context.Progress().Printf("\nYou can also use `aptly serve` to publish your repositories over HTTP quickly.\n")

	return err
}

func makeCmdPublishSnapshot() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishSnapshotOrRepo,
		UsageLine: "snapshot <name> [[<endpoint>:]<prefix>]",
		Short:     "publish snapshot",
		Long: `
Command publishes snapshot as Debian repository ready to be consumed
by apt tools. Published repostiories appear under rootDir/public directory.
Valid GPG key is required for publishing.

Multiple component repository could be published by specifying several
components split by commas via -component flag and multiple snapshots
as the arguments:

    aptly publish snapshot -component=main,contrib snap-main snap-contrib

Example:

    $ aptly publish snapshot wheezy-main
`,
		Flag: *flag.NewFlagSet("aptly-publish-snapshot", flag.ExitOnError),
	}
	cmd.Flag.String("distribution", "", "distribution name to publish")
	cmd.Flag.String("component", "", "component name to publish (for multi-component publishing, separate components with commas)")
	cmd.Flag.String("gpg-key", "", "GPG key ID to use when signing the release")
	cmd.Flag.Var(&keyRingsFlag{}, "keyring", "GPG keyring to use (instead of default)")
	cmd.Flag.String("secret-keyring", "", "GPG secret keyring to use (instead of default)")
	cmd.Flag.String("passphrase", "", "GPG passphrase for the key (warning: could be insecure)")
	cmd.Flag.String("passphrase-file", "", "GPG passphrase-file for the key (warning: could be insecure)")
	cmd.Flag.Bool("batch", false, "run GPG with detached tty")
	cmd.Flag.Bool("skip-signing", false, "don't sign Release files with GPG")
	cmd.Flag.Bool("skip-contents", false, "don't generate Contents indexes")
	cmd.Flag.String("origin", "", "overwrite origin name to publish")
	cmd.Flag.String("notautomatic", "", "overwrite value for NotAutomatic field")
	cmd.Flag.String("butautomaticupgrades", "", "overwrite value for ButAutomaticUpgrades field")
	cmd.Flag.String("label", "", "label to publish")
	cmd.Flag.String("suite", "", "suite to publish (defaults to distribution)")
	cmd.Flag.Bool("force-overwrite", false, "overwrite files in package pool in case of mismatch")
	cmd.Flag.Bool("acquire-by-hash", false, "provide index files by hash")

	return cmd
}
