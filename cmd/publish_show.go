package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
)

func aptlyPublishShow(cmd *commander.Command, args []string) error {
	if len(args) < 1 || len(args) > 2 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	jsonFlag := cmd.Flag.Lookup("json").Value.Get().(bool)

	if jsonFlag {
		return aptlyPublishShowJson(cmd, args)
	}

	return aptlyPublishShowTxt(cmd, args)
}

func aptlyPublishShowTxt(cmd *commander.Command, args []string) error {
	var err error

	distribution := args[0]
	param := "."

	if len(args) == 2 {
		param = args[1]
	}

	storage, prefix := deb.ParsePrefix(param)

	collectionFactory := context.NewCollectionFactory()
	repo, err := collectionFactory.PublishedRepoCollection().ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	if repo.Storage != "" {
		fmt.Printf("Storage: %s\n", repo.Storage)
	}
	fmt.Printf("Prefix: %s\n", repo.Prefix)
	if repo.Distribution != "" {
		fmt.Printf("Distribution: %s\n", repo.Distribution)
	}
	fmt.Printf("Architectures: %s\n", strings.Join(repo.Architectures, " "))

	fmt.Printf("Sources:\n")
	for component, sourceID := range repo.Sources {
		var name string
		if repo.SourceKind == deb.SourceSnapshot {
			source, e := collectionFactory.SnapshotCollection().ByUUID(sourceID)
			if e != nil {
				continue
			}
			name = source.Name
		} else if repo.SourceKind == deb.SourceLocalRepo {
			source, e := collectionFactory.LocalRepoCollection().ByUUID(sourceID)
			if e != nil {
				continue
			}
			name = source.Name
		}

		if name != "" {
			fmt.Printf("  %s: %s [%s]\n", component, name, repo.SourceKind)
		}
	}

	return err
}

func aptlyPublishShowJson(cmd *commander.Command, args []string) error {
	var err error

	distribution := args[0]
	param := "."

	if len(args) == 2 {
		param = args[1]
	}

	storage, prefix := deb.ParsePrefix(param)

	repo, err := context.CollectionFactory().PublishedRepoCollection().ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	err = context.CollectionFactory().PublishedRepoCollection().LoadComplete(repo, context.CollectionFactory())
	if err != nil {
		return err
	}

	var output []byte
	if output, err = json.MarshalIndent(repo, "", "  "); err == nil {
		fmt.Println(string(output))
	}

	return err
}

func makeCmdPublishShow() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishShow,
		UsageLine: "show <distribution> [[<endpoint>:]<prefix>]",
		Short:     "shows details of published repository",
		Long: `
Command show displays full information of a published repository.

Example:

    $ aptly publish show wheezy
`,
	}

	cmd.Flag.Bool("json", false, "display record in JSON format")

	return cmd
}
