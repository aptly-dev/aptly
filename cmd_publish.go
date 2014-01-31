package main

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
	"github.com/smira/aptly/utils"
	"sort"
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

	publishedCollecton := debian.NewPublishedRepoCollection(context.database)

	snapshotCollection := debian.NewSnapshotCollection(context.database)
	snapshot, err := snapshotCollection.ByName(name)
	if err != nil {
		return fmt.Errorf("unable to publish: %s", err)
	}

	err = snapshotCollection.LoadComplete(snapshot)
	if err != nil {
		return fmt.Errorf("unable to publish: %s", err)
	}

	var sourceRepo *debian.RemoteRepo

	if snapshot.SourceKind == "repo" && len(snapshot.SourceIDs) == 1 {
		repoCollection := debian.NewRemoteRepoCollection(context.database)

		sourceRepo, _ = repoCollection.ByUUID(snapshot.SourceIDs[0])
	}

	component := cmd.Flag.Lookup("component").Value.String()
	if component == "" {
		if sourceRepo != nil && len(sourceRepo.Components) == 1 {
			component = sourceRepo.Components[0]
		} else {
			component = "main"
		}
	}

	distribution := cmd.Flag.Lookup("distribution").Value.String()
	if distribution == "" {
		if sourceRepo != nil {
			distribution = sourceRepo.Distribution
		} else {
			return fmt.Errorf("unable to guess distribution name, please specify explicitly")
		}
	}

	signer := &utils.GpgSigner{}
	signer.SetKey(cmd.Flag.Lookup("gpg-key").Value.String())

	published, err := debian.NewPublishedRepo(prefix, distribution, component, context.architecturesList, snapshot)
	if err != nil {
		return fmt.Errorf("unable to publish: %s", err)
	}

	duplicate := publishedCollecton.CheckDuplicate(published)
	if duplicate != nil {
		publishedCollecton.LoadComplete(duplicate, snapshotCollection)
		return fmt.Errorf("prefix/distribution already used by another published repo: %s", duplicate)
	}

	packageCollection := debian.NewPackageCollection(context.database)
	err = published.Publish(context.packageRepository, packageCollection, signer)
	if err != nil {
		return fmt.Errorf("unable to publish: %s", err)
	}

	err = publishedCollecton.Add(published)
	if err != nil {
		return fmt.Errorf("unable to save to DB: %s", err)
	}

	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	fmt.Printf("\nSnapshot %s has been successfully published.\nPlease setup your webserver to serve directory '%s' with autoindexing.\n",
		snapshot.Name, context.packageRepository.PublicPath())
	fmt.Printf("Now you can add following line to apt sources:\n")
	fmt.Printf("  deb http://your-server/%s %s %s\n", prefix, distribution, component)
	fmt.Printf("Don't forget to add your GPG key to apt with apt-key.\n")
	fmt.Printf("\nYou can also use `aptly serve` to publish your repositories over HTTP quickly.\n")

	return err
}

func aptlyPublishList(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 0 {
		cmd.Usage()
		return err
	}

	publishedCollecton := debian.NewPublishedRepoCollection(context.database)
	snapshotCollection := debian.NewSnapshotCollection(context.database)

	if publishedCollecton.Len() == 0 {
		fmt.Printf("No snapshots have been published. Publish a snapshot by running `aptly publish snapshot ...`.\n")
		return err
	}

	published := make(sort.StringSlice, 0, publishedCollecton.Len())

	err = publishedCollecton.ForEach(func(repo *debian.PublishedRepo) error {
		err := publishedCollecton.LoadComplete(repo, snapshotCollection)
		if err != nil {
			return err
		}

		published = append(published, repo.String())
		return nil
	})

	if err != nil {
		return fmt.Errorf("unable to load list of repos: %s", err)
	}

	sort.Strings(published)

	fmt.Printf("Published repositories:\n")

	for _, description := range published {
		fmt.Printf("  * %s\n", description)
	}

	return err
}

func aptlyPublishDrop(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 1 || len(args) > 2 {
		cmd.Usage()
		return err
	}

	distribution := args[0]
	prefix := "."

	if len(args) == 2 {
		prefix = args[1]
	}

	publishedCollecton := debian.NewPublishedRepoCollection(context.database)

	err = publishedCollecton.Remove(context.packageRepository, prefix, distribution)
	if err != nil {
		return fmt.Errorf("unable to remove: %s", err)
	}

	fmt.Printf("\nPublished repositroy has been removed successfully.\n")

	return err
}

func makeCmdPublishSnapshot() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishSnapshot,
		UsageLine: "snapshot <name> [<prefix>]",
		Short:     "makes Debian repository out of snapshot",
		Long: `
Command publish oublishes snapshot as Debian repository ready to be used by apt tools.

ex.
	$ aptly publish snapshot wheezy-main
`,
		Flag: *flag.NewFlagSet("aptly-publish-snapshot", flag.ExitOnError),
	}
	cmd.Flag.String("distribution", "", "distribution name to publish")
	cmd.Flag.String("component", "", "component name to publish")
	cmd.Flag.String("gpg-key", "", "GPG key ID to use when signing the release")

	return cmd
}

func makeCmdPublishDrop() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishDrop,
		UsageLine: "drop <distribution> [<prefix>]",
		Short:     "removes files of published repository",
		Long: `
Command removes whatever has been published under specified prefix and distribution name.

ex.
	$ aptly publish drop wheezy
`,
		Flag: *flag.NewFlagSet("aptly-publish-drop", flag.ExitOnError),
	}

	return cmd
}

func makeCmdPublishList() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishList,
		UsageLine: "list",
		Short:     "displays list of published repositories",
		Long: `
Display command displays list of currently published snapshots with information about published root.

ex.
	$ aptly publish list
`,
		Flag: *flag.NewFlagSet("aptly-publish-list", flag.ExitOnError),
	}

	return cmd
}

func makeCmdPublish() *commander.Command {
	return &commander.Command{
		UsageLine: "publish",
		Short:     "manage published repositories",
		Subcommands: []*commander.Command{
			makeCmdPublishSnapshot(),
			makeCmdPublishList(),
			makeCmdPublishDrop(),
		},
		Flag: *flag.NewFlagSet("aptly-publish", flag.ExitOnError),
	}
}
