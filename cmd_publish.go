package main

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
	if len(args) != 2 {
		cmd.Usage()
		return err
	}

	name := args[0]
	prefix := args[1]

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

	var architecturesList []string
	architectures := cmd.Flag.Lookup("architectures").Value.String()
	if architectures != "" {
		architecturesList = strings.Split(architectures, ",")
	}

	signer := &utils.GpgSigner{}
	signer.SetKey(cmd.Flag.Lookup("gpg-key").Value.String())

	published := debian.NewPublishedRepo(prefix, distribution, component, architecturesList, snapshot)

	packageCollection := debian.NewPackageCollection(context.database)
	err = published.Publish(context.packageRepository, packageCollection, signer)

	return err
}

func makeCmdPublishSnapshot() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishSnapshot,
		UsageLine: "snapshot",
		Short:     "makes Debian repository out of snapshot",
		Long: `
Publishes snapshot as Debian repository ready to be used by apt tools.

ex:
  $ aptly publish snapshot <name> <prefix>
`,
		Flag: *flag.NewFlagSet("aptly-publish-snapshot", flag.ExitOnError),
	}
	cmd.Flag.String("distribution", "", "distribution name to publish")
	cmd.Flag.String("component", "", "component name to publish")
	cmd.Flag.String("architectures", "", "list of architectures to publish (comma-separated)")
	cmd.Flag.String("gpg-key", "", "GPG key ID to use when signing the release")

	return cmd
}

func makeCmdPublish() *commander.Command {
	return &commander.Command{
		UsageLine: "publish",
		Short:     "manage published repositories",
		Subcommands: []*commander.Command{
			makeCmdPublishSnapshot(),
		},
		Flag: *flag.NewFlagSet("aptly-publish", flag.ExitOnError),
	}
}
